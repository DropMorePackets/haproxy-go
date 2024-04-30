package spop

import (
	"context"
	"encoding/binary"
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
	ctx, cancel := context.WithCancel(context.Background())

	pipe, pipeConn := testutil.PipeConn()
	go func() {
		defer cancel()

		if err := newHelloFrame(pipe); err != nil {
			t.Error(err)
			return
		}

		if err := newNotifyFrame(pipe); err != nil {
			t.Error(err)
			return
		}
	}()

	go func() {
		defer cancel()

		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
			t.Error("timeout")
		}
	}()

	handler := HandlerFunc(func(_ context.Context, _ *encoding.ActionWriter, m *encoding.Message) {
		log.Println(m.NameBytes())
		cancel()
	})

	pc := newProtocolClient(context.Background(), pipeConn, handler)
	defer pc.Close()
	defer pipe.Close()
	go pc.Serve()

	<-ctx.Done()
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

	if err := w.SetUInt32(helloKeyMaxFrameSize, maxFrameSize); err != nil {
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
