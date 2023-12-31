package spop

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

func newProtocolClient(ctx context.Context, rw io.ReadWriter, handler Handler) *protocolClient {
	var c protocolClient
	c.rw = rw
	c.handler = handler
	c.ctx, c.ctxCancel = context.WithCancel(ctx)
	c.as = newAsyncScheduler(&c)
	return &c
}

type protocolClient struct {
	rw      io.ReadWriter
	handler Handler
	ctx     context.Context

	ctxCancel context.CancelFunc
	as        *asyncScheduler

	gotHello     bool
	maxFrameSize uint32
	engineID     string
}

func (c *protocolClient) Close() error {
	errDisconnect := (&AgentDisconnectFrame{
		ErrCode: ErrorUnknown,
	}).Write(c.rw)

	c.ctxCancel()

	return errors.Join(errDisconnect, c.ctx.Err())
}

func (c *protocolClient) frameHandler(f *frame) error {
	defer releaseFrame(f)

	switch f.frameType {
	case frameTypeIDHaproxyHello:
		return c.onHAProxyHello(f)
	case frameTypeIDNotify:
		return c.onNotify(f)
	case frameTypeIDHaproxyDisconnect:
		return c.onHAProxyDisconnect(f)
	default:
		return fmt.Errorf("unknown frame type: %d", f.frameType)
	}
}

func (c *protocolClient) Serve() error {
	for {
		f := acquireFrame()
		if _, err := f.ReadFrom(c.rw); err != nil {
			if c.ctx.Err() != nil {
				return c.ctx.Err()
			}

			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		c.as.schedule(f)
	}
}

const (
	version = "2.0"

	// maxFrameSize represents the maximum frame size allowed by this library
	// it also represents the maximum slice size that is allowed on stack
	maxFrameSize = 64<<10 - 1
)

func (c *protocolClient) onHAProxyHello(f *frame) error {
	if c.gotHello {
		panic("duplicate hello frame")
	}
	c.gotHello = true

	s := encoding.AcquireKVScanner(f.buf.ReadBytes(), -1)
	defer encoding.ReleaseKVScanner(s)

	k := encoding.AcquireKVEntry()
	defer encoding.ReleaseKVEntry(k)
	for s.Next(k) {
		switch {
		case k.NameEquals(helloKeyMaxFrameSize):
			c.maxFrameSize = uint32(k.ValueInt())
			if c.maxFrameSize > maxFrameSize {
				return fmt.Errorf("maxFrameSize bigger than maximum allowed size: %d < %d", maxFrameSize, c.maxFrameSize)
			}

		case k.NameEquals(helloKeyEngineID):
			//TODO: This does copy the engine id but yolo?
			c.engineID = string(k.ValueBytes())
		//case k.NameEquals(helloKeySupportedVersions):
		//case k.NameEquals(helloKeyCapabilities):
		case k.NameEquals(helloKeyHealthcheck):
			// as described in the protocol, close connection after hello
			// AGENT-HELLO + close()
			defer c.ctxCancel()
		}
	}

	if err := s.Error(); err != nil {
		return err
	}

	return (&AgentHelloFrame{
		Version:      version,
		MaxFrameSize: c.maxFrameSize,
		Capabilities: []string{capabilityNamePipelining, capabilityNameAsync},
	}).Write(c.rw)
}

func (c *protocolClient) runHandler(ctx context.Context, w *encoding.ActionWriter, m *encoding.Message, handler HandlerFunc) (err error) {
	didPanic := true
	defer func() {
		if didPanic {
			if e := recover(); e != nil {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				err = fmt.Errorf("spop: panic serving: %v\n%s", e, buf)
			}
		}
	}()
	handler(ctx, w, m)
	didPanic = false
	return
}

func (c *protocolClient) onNotify(f *frame) error {
	s := encoding.AcquireMessageScanner(f.buf.ReadBytes())
	defer encoding.ReleaseMessageScanner(s)

	m := encoding.AcquireMessage()
	defer encoding.ReleaseMessage(m)

	fn := func(w *encoding.ActionWriter) error {
		for s.Next(m) {
			err := c.runHandler(c.ctx, w, m, c.handler.HandleSPOE)
			if err != nil {
				return err
			}

			if err := m.KV.Discard(); err != nil {
				return err
			}
		}

		return s.Error()
	}

	return (&AckFrame{
		FrameID:              f.meta.FrameID,
		StreamID:             f.meta.StreamID,
		ActionWriterCallback: fn,
	}).Write(c.rw)
}

func (c *protocolClient) onHAProxyDisconnect(f *frame) error {
	//TODO: read disconnect reason and return error if required?
	return nil
}
