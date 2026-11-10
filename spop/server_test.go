package spop

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
	"github.com/dropmorepackets/haproxy-go/pkg/testutil"
)

func TestFakeCon(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	negotiatedSize := uint32(maxFrameSize * 2)
	largeValue := make([]byte, maxFrameSize)

	pipe, pipeConn := testutil.PipeConn()
	defer pipe.Close()
	defer pipeConn.Close()
	peerDone := make(chan error, 1)
	go func() {
		if err := newHelloFrame(pipe, negotiatedSize); err != nil {
			peerDone <- err
			return
		}
		if err := readExpectedFrame(pipe, frameTypeIDAgentHello); err != nil {
			peerDone <- err
			return
		}

		if err := newNotifyFrame(pipe, largeValue); err != nil {
			peerDone <- err
			return
		}
		frameLen, err := readExpectedFrameWithLimit(pipe, frameTypeIDAck, negotiatedSize)
		if err != nil {
			peerDone <- err
			return
		}
		if frameLen <= maxFrameSize {
			peerDone <- fmt.Errorf("expected ACK above %d bytes, got %d", maxFrameSize, frameLen)
			return
		}
		peerDone <- nil
	}()

	handler := HandlerFunc(func(_ context.Context, w *encoding.ActionWriter, m *encoding.Message) {
		log.Println(m.NameBytes())
		if err := w.SetBinary(encoding.VarScopeTransaction, "result", largeValue); err != nil {
			t.Errorf("write action: %v", err)
		}
	})

	pc := newProtocolClient(ctx, pipeConn, newAsyncScheduler(), handler)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- pc.Serve()
	}()

	select {
	case err := <-peerDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("peer exchange timed out")
	}
	if err := pipe.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("serve protocol: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("protocol shutdown timed out")
	}
}

func readExpectedFrame(r io.Reader, expected frameType) error {
	_, err := readExpectedFrameWithLimit(r, expected, maxFrameSize)
	return err
}

func readExpectedFrameWithLimit(r io.Reader, expected frameType, limit uint32) (uint32, error) {
	f := acquireFrame()
	defer releaseFrame(f)

	if _, err := f.readFrom(r, limit); err != nil {
		return 0, err
	}
	if f.frameType != expected {
		return 0, fmt.Errorf("expected frame type %d, got %d", expected, f.frameType)
	}
	return binary.BigEndian.Uint32(f.length), nil
}

// TestPipelinedAcks sends several NOTIFY frames on one connection without
// waiting for their ACKs, as HAProxy does once pipelining is negotiated, and
// checks that the ACKs written by concurrent workers arrive as intact frames.
func TestPipelinedAcks(t *testing.T) {
	const (
		workers = 2
		rounds  = 10
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipe, pipeConn := testutil.PipeConn()
	defer pipe.Close()
	defer pipeConn.Close()

	// Every worker holds a frame of the same connection before any of them
	// writes its ACK.
	entered := make(chan struct{})
	release := make(chan struct{}, workers)
	handler := HandlerFunc(func(_ context.Context, w *encoding.ActionWriter, m *encoding.Message) {
		select {
		case entered <- struct{}{}:
		case <-ctx.Done():
			return
		}
		select {
		case <-release:
		case <-ctx.Done():
			return
		}

		k := encoding.AcquireKVEntry()
		defer encoding.ReleaseKVEntry(k)
		if !m.KV.Next(k) {
			t.Error("NOTIFY without payload")
			return
		}
		if err := w.SetBinary(encoding.VarScopeTransaction, "echo", k.ValueBytes()); err != nil {
			t.Errorf("write action: %v", err)
		}
	})

	as := &asyncScheduler{q: newQueue(workers * 2)}
	for i := 0; i < workers; i++ {
		go as.queueWorker()
	}

	pc := newProtocolClient(ctx, pipeConn, as, handler)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- pc.Serve()
	}()

	peerDone := make(chan error, 1)
	go func() {
		peerDone <- func() error {
			if err := newHelloFrame(pipe, maxFrameSize); err != nil {
				return err
			}
			if err := readExpectedFrame(pipe, frameTypeIDAgentHello); err != nil {
				return err
			}

			var frameID uint64
			for round := 0; round < rounds; round++ {
				want := make(map[uint64][]byte, workers)
				for i := 0; i < workers; i++ {
					frameID++
					want[frameID] = fmt.Appendf(nil, "frame %d", frameID)
					if err := writeNotifyFrame(pipe, frameID, frameID, want[frameID]); err != nil {
						return err
					}
				}
				for i := 0; i < workers; i++ {
					select {
					case <-entered:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				for i := 0; i < workers; i++ {
					release <- struct{}{}
				}
				// Let the workers block on the connection so their writes
				// compete once the ACKs are read.
				time.Sleep(10 * time.Millisecond)

				for i := 0; i < workers; i++ {
					if err := readEchoAck(pipe, want); err != nil {
						return fmt.Errorf("round %d: %w", round, err)
					}
				}
			}
			return nil
		}()
	}()

	select {
	case err := <-peerDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("timed out reading ACKs")
	}
	if err := pipe.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("serve protocol: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("protocol shutdown timed out")
	}
}

// readEchoAck reads an ACK and checks that it echoes the value sent in the
// NOTIFY frame with the same frame ID.
func readEchoAck(r io.Reader, want map[uint64][]byte) error {
	f := acquireFrame()
	defer releaseFrame(f)

	if _, err := f.ReadFrom(r); err != nil {
		return err
	}
	if f.frameType != frameTypeIDAck {
		return fmt.Errorf("expected frame type %d, got %d", frameTypeIDAck, f.frameType)
	}
	if f.meta.StreamID != f.meta.FrameID {
		return fmt.Errorf("unexpected stream ID %d for frame %d", f.meta.StreamID, f.meta.FrameID)
	}
	value, ok := want[f.meta.FrameID]
	if !ok {
		return fmt.Errorf("unexpected ACK for frame %d", f.meta.FrameID)
	}
	delete(want, f.meta.FrameID)

	aw := encoding.NewActionWriter(make([]byte, 64), 0)
	if err := aw.SetBinary(encoding.VarScopeTransaction, "echo", value); err != nil {
		return err
	}
	if !bytes.Equal(f.buf.ReadBytes(), aw.Bytes()) {
		return fmt.Errorf("ACK for frame %d does not echo %q", f.meta.FrameID, value)
	}
	return nil
}

func newNotifyFrame(wr io.Writer, value []byte) error {
	return writeNotifyFrame(wr, uint64(rand.Int63()), uint64(rand.Int63()), value)
}

func writeNotifyFrame(wr io.Writer, streamID, frameID uint64, value []byte) error {
	f := acquireFrame()
	defer releaseFrame(f)

	f.frameType = frameTypeIDNotify
	f.meta.StreamID = streamID
	f.meta.FrameID = frameID
	f.meta.Flags = frameFlagFin

	if err := f.encodeHeader(); err != nil {
		return err
	}
	f.buf.Grow(len(value) + 32)

	n, err := encoding.PutBytes(f.buf.WriteBytes(), []byte("example"))
	if err != nil {
		return err
	}
	f.buf.AdvanceW(n)
	f.buf.WriteNBytes(1)[0] = 1

	w := encoding.NewKVWriter(f.buf.WriteBytes(), 0)
	if err := w.SetBinary("payload", value); err != nil {
		return err
	}
	f.buf.AdvanceW(w.Off())

	binary.BigEndian.PutUint32(f.length, uint32(f.buf.Len()))
	wr.Write(f.length)
	wr.Write(f.buf.ReadBytes())

	return nil
}

func newHelloFrame(wr io.Writer, offer uint32) error {
	f := acquireFrame()
	defer releaseFrame(f)

	f.frameType = frameTypeIDHaproxyHello
	f.meta.StreamID = 0
	f.meta.FrameID = 0
	f.meta.Flags = frameFlagFin

	if err := f.encodeHeader(); err != nil {
		return err
	}

	w := encoding.AcquireKVWriter(f.buf.WriteBytes(), 0)
	defer encoding.ReleaseKVWriter(w)

	if err := w.SetString(helloKeySupportedVersions, version); err != nil {
		return err
	}
	if err := w.SetUInt32(helloKeyMaxFrameSize, offer); err != nil {
		return err
	}
	if err := w.SetString(helloKeyCapabilities, ""); err != nil {
		return err
	}

	// TODO
	if err := w.SetString(helloKeyEngineID, "random engine"); err != nil {
		return err
	}

	f.buf.AdvanceW(w.Off())

	binary.BigEndian.PutUint32(f.length, uint32(f.buf.Len()))
	wr.Write(f.length)
	wr.Write(f.buf.ReadBytes())

	return nil
}
