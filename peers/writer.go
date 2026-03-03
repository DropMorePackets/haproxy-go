package peers

import (
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
	w  io.Writer
	mu sync.Mutex

	nextUpdateID uint32
}

func newWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// writeMessage writes a peer protocol message. Messages with type >= 128
// include a varint-encoded data length prefix before the payload.
func (w *Writer) writeMessage(class MessageClass, msgType byte, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var lenBuf [10]byte
	var lenBytes int
	if msgType >= 128 {
		n, err := encoding.PutVarint(lenBuf[:], uint64(len(data)))
		if err != nil {
			return fmt.Errorf("encoding data length: %w", err)
		}
		lenBytes = n
	}

	// Build the complete message in a single buffer to send atomically.
	msg := make([]byte, 2+lenBytes+len(data))
	msg[0] = byte(class)
	msg[1] = msgType
	copy(msg[2:], lenBuf[:lenBytes])
	copy(msg[2+lenBytes:], data)

	_, err := w.w.Write(msg)
	return err
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

	return w.writeMessage(
		MessageClassStickTableUpdates,
		byte(StickTableUpdateMessageTypeStickTableDefinition),
		buf[:n],
	)
}

// SendTableSwitch sends a table switch message to select a previously
// defined table by its sender table ID.
func (w *Writer) SendTableSwitch(tableID uint64) error {
	var buf [10]byte
	n, err := encoding.PutVarint(buf[:], tableID)
	if err != nil {
		return fmt.Errorf("encoding table ID: %w", err)
	}

	return w.writeMessage(
		MessageClassStickTableUpdates,
		byte(StickTableUpdateMessageTypeStickTableSwitch),
		buf[:n],
	)
}

// SendEntry sends a stick table entry update with an automatically
// assigned update ID. The message type is chosen based on the entry's
// WithExpiry flag:
//   - WithExpiry=false: Entry Update (0x80)
//   - WithExpiry=true:  Update Timed (0x85)
func (w *Writer) SendEntry(entry *sticktable.EntryUpdate) error {
	entry.WithLocalUpdateID = true
	entry.LocalUpdateID = w.nextUpdateID
	w.nextUpdateID++

	msgType := StickTableUpdateMessageTypeEntryUpdate
	if entry.WithExpiry {
		msgType = StickTableUpdateMessageTypeUpdateTimed
	}

	var buf [65536]byte
	n, err := entry.Marshal(buf[:])
	if err != nil {
		return fmt.Errorf("marshaling entry update: %w", err)
	}

	return w.writeMessage(
		MessageClassStickTableUpdates,
		byte(msgType),
		buf[:n],
	)
}
