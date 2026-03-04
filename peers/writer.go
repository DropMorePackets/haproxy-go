package peers

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/dropmorepackets/haproxy-go/peers/sticktable"
	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

// Writer sends stick table updates over an existing peer connection.
// It is safe for concurrent use. Obtain a Writer from a handler's context
// using WriterFromContext.
type Writer struct {
	bw  *bufio.Writer
	mu  *sync.Mutex
	buf []byte // reusable scratch buffer for marshaling

	nextUpdateID uint32
}

func newWriter(w io.Writer, mu *sync.Mutex) *Writer {
	bw := bufio.NewWriterSize(w, 64*1024)
	return &Writer{
		bw:  bw,
		mu:  mu,
		buf: make([]byte, 65536),
	}
}

// bufferedWriter returns the underlying bufio.Writer so the protocol
// client can share the same buffered output (under the shared mutex).
func (w *Writer) bufferedWriter() *bufio.Writer {
	return w.bw
}

// Flush flushes any buffered data to the underlying connection.
// The caller must hold the mutex or call this after a batch of writes.
func (w *Writer) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.bw.Flush()
}

// writeMessage writes a peer protocol message. Messages with type >= 128
// include a varint-encoded data length prefix before the payload.
// Caller must NOT hold the mutex — this method acquires it.
func (w *Writer) writeMessage(class MessageClass, msgType byte, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writeMessageLocked(class, msgType, data)
}

// writeMessageLocked writes a peer protocol message.
// Caller MUST hold the mutex.
func (w *Writer) writeMessageLocked(class MessageClass, msgType byte, data []byte) error {
	var lenBuf [10]byte
	var lenBytes int
	if msgType >= 128 {
		n, err := encoding.PutVarint(lenBuf[:], uint64(len(data)))
		if err != nil {
			return fmt.Errorf("encoding data length: %w", err)
		}
		lenBytes = n
	}

	// Write header (class + type)
	if _, err := w.bw.Write([]byte{byte(class), msgType}); err != nil {
		return err
	}

	// Write length prefix if present
	if lenBytes > 0 {
		if _, err := w.bw.Write(lenBuf[:lenBytes]); err != nil {
			return err
		}
	}

	// Write payload
	if len(data) > 0 {
		if _, err := w.bw.Write(data); err != nil {
			return err
		}
	}

	return nil
}

// SendTableDefinition sends a stick table definition message.
// This must be called before sending entry updates for a table so
// that the remote peer knows which table the updates refer to.
func (w *Writer) SendTableDefinition(def *sticktable.Definition) error {
	var buf [4096]byte
	n, err := def.Marshal(buf[:])
	if err != nil {
		return fmt.Errorf("marshaling table definition: %w", err)
	}

	if err := w.writeMessage(
		MessageClassStickTableUpdates,
		byte(StickTableUpdateMessageTypeStickTableDefinition),
		buf[:n],
	); err != nil {
		return err
	}

	return w.Flush()
}

// SendTableSwitch sends a table switch message to select a previously
// defined table by its sender table ID.
func (w *Writer) SendTableSwitch(tableID uint64) error {
	var buf [10]byte
	n, err := encoding.PutVarint(buf[:], tableID)
	if err != nil {
		return fmt.Errorf("encoding table ID: %w", err)
	}

	if err := w.writeMessage(
		MessageClassStickTableUpdates,
		byte(StickTableUpdateMessageTypeStickTableSwitch),
		buf[:n],
	); err != nil {
		return err
	}

	return w.Flush()
}

// marshalEntry marshals a single entry update into buf and returns the
// byte count. The updateID is written first, followed by optional expiry,
// key and data values.
func marshalEntry(buf []byte, entry *sticktable.EntryUpdate, updateID uint32) (int, error) {
	offset := 0

	binary.BigEndian.PutUint32(buf[offset:], updateID)
	offset += 4

	if entry.WithExpiry {
		binary.BigEndian.PutUint32(buf[offset:], entry.Expiry)
		offset += 4
	}

	n, err := entry.Key.Marshal(buf[offset:], entry.StickTable.KeyLength)
	offset += n
	if err != nil {
		return offset, fmt.Errorf("marshaling entry key: %w", err)
	}

	for _, data := range entry.Data {
		n, err := data.Marshal(buf[offset:])
		offset += n
		if err != nil {
			return offset, fmt.Errorf("marshaling entry data: %w", err)
		}
	}

	return offset, nil
}

// SendEntry sends a stick table entry update with an automatically
// assigned update ID. The message type is chosen based on the entry's
// WithExpiry flag:
//   - WithExpiry=false: Entry Update (0x80)
//   - WithExpiry=true:  Update Timed (0x85)
//
// Note: for bulk operations, prefer SendEntries which batches writes and flushes once.
func (w *Writer) SendEntry(entry *sticktable.EntryUpdate) error {
	w.mu.Lock()
	updateID := w.nextUpdateID
	w.nextUpdateID++

	msgType := StickTableUpdateMessageTypeEntryUpdate
	if entry.WithExpiry {
		msgType = StickTableUpdateMessageTypeUpdateTimed
	}

	offset, err := marshalEntry(w.buf, entry, updateID)
	if err != nil {
		w.mu.Unlock()
		return fmt.Errorf("marshaling entry update: %w", err)
	}

	if err := w.writeMessageLocked(
		MessageClassStickTableUpdates,
		byte(msgType),
		w.buf[:offset],
	); err != nil {
		w.mu.Unlock()
		return err
	}

	err = w.bw.Flush()
	w.mu.Unlock()
	return err
}

// SendEntries sends multiple stick table entry updates in a single
// locked batch. This is significantly faster than calling SendEntry
// in a loop because it acquires the mutex once, marshals and writes
// all entries into the buffer, then flushes once.
func (w *Writer) SendEntries(entries []*sticktable.EntryUpdate) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, entry := range entries {
		updateID := w.nextUpdateID
		w.nextUpdateID++

		msgType := StickTableUpdateMessageTypeEntryUpdate
		if entry.WithExpiry {
			msgType = StickTableUpdateMessageTypeUpdateTimed
		}

		offset, err := marshalEntry(w.buf, entry, updateID)
		if err != nil {
			return fmt.Errorf("marshaling entry update: %w", err)
		}

		if err := w.writeMessageLocked(
			MessageClassStickTableUpdates,
			byte(msgType),
			w.buf[:offset],
		); err != nil {
			return err
		}
	}

	return w.bw.Flush()
}
