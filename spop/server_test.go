package spop

import (
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

	pipe, pipeConn := testutil.PipeConn()
	defer pipe.Close()
	defer pipeConn.Close()
	peerDone := make(chan error, 1)
	go func() {
		if err := newHelloFrame(pipe); err != nil {
			peerDone <- err
			return
		}
		if err := readExpectedFrame(pipe, frameTypeIDAgentHello); err != nil {
			peerDone <- err
			return
		}

		if err := newNotifyFrame(pipe); err != nil {
			peerDone <- err
			return
		}
		if err := readExpectedFrame(pipe, frameTypeIDAck); err != nil {
			peerDone <- err
			return
		}
		peerDone <- nil
	}()

	handler := HandlerFunc(func(_ context.Context, _ *encoding.ActionWriter, m *encoding.Message) {
		log.Println(m.NameBytes())
	})

	pc := newProtocolClient(ctx, pipeConn, newTestAsyncScheduler(), defaultFramePool, handler)
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
	f := acquireFrame()
	defer releaseFrame(f)

	if _, err := f.ReadFrom(r); err != nil {
		return err
	}
	if f.frameType != expected {
		return fmt.Errorf("expected frame type %d, got %d", expected, f.frameType)
	}
	return nil
}

func newNotifyFrame(wr io.Writer) error {
	f := acquireFrame()
	defer releaseFrame(f)

	f.frameType = frameTypeIDNotify
	f.meta.StreamID = uint64(rand.Int63())
	f.meta.FrameID = uint64(rand.Int63())
	f.meta.Flags = frameFlagFin

	if err := f.encodeHeader(); err != nil {
		return err
	}

	n, err := encoding.PutBytes(f.buf.WriteBytes(), []byte("example"))
	if err != nil {
		return err
	}
	f.buf.AdvanceW(n)
	f.buf.WriteNBytes(1)[0] = 0

	//TODO Write message
	//w := encoding.AcquireActionWriter(f.buf.WriteBytes(), 0)
	//defer encoding.ReleaseActionWriter(w)

	//f.buf.AdvanceW(w.Off())

	binary.BigEndian.PutUint32(f.length, uint32(f.buf.Len()))
	wr.Write(f.length)
	wr.Write(f.buf.ReadBytes())

	return nil
}

func newHelloFrame(wr io.Writer) error {
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
	if err := w.SetUInt32(helloKeyMaxFrameSize, DefaultMaxFrameSize); err != nil {
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
