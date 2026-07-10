package spop

import (
	"context"
	"errors"
	"fmt"
	"io"
	"syscall"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

func newProtocolClient(ctx context.Context, rw io.ReadWriter, as *asyncScheduler, handler Handler) *protocolClient {
	var c protocolClient
	c.rw = rw
	c.handler = handler
	c.ctx, c.ctxCancel = context.WithCancelCause(ctx)
	c.as = as
	return &c
}

type protocolClient struct {
	rw      io.ReadWriter
	handler Handler
	ctx     context.Context

	ctxCancel context.CancelCauseFunc
	as        *asyncScheduler

	engineID     string
	maxFrameSize uint32

	gotHello bool
}

func (c *protocolClient) Close() error {
	if c.ctx.Err() != nil {
		return c.ctx.Err()
	}

	// We ignore any error since the disconnect frame is delivered on
	// best effort anyway.
	_, _ = (&AgentDisconnectFrame{
		ErrCode: ErrorUnknown,
	}).WriteTo(c.rw)

	c.ctxCancel(fmt.Errorf("closing client"))

	return nil
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
		limit := uint32(maxFrameSize)
		if c.gotHello {
			limit = c.maxFrameSize
		}

		f := acquireFrame()
		if _, err := f.readFrom(c.rw, limit); err != nil {
			releaseFrame(f)
			if c.ctx.Err() != nil {
				return context.Cause(c.ctx)
			}

			if errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET) {
				return nil
			}

			return err
		}

		if !c.gotHello {
			if f.frameType != frameTypeIDHaproxyHello {
				firstFrameType := f.frameType
				releaseFrame(f)
				return fmt.Errorf("first frame must be HAPROXY-HELLO, got type %d", firstFrameType)
			}
			if err := c.frameHandler(f); err != nil {
				return err
			}
			if c.ctx.Err() != nil {
				return context.Cause(c.ctx)
			}
			continue
		}

		c.as.schedule(f, c)
	}
}

const (
	version = "2.0"

	// maxFrameSize is the initial buffer size and pre-negotiation frame limit.
	maxFrameSize = 64<<10 - 1

	// HAProxy advertises tune.bufsize-4, and tune.bufsize is bounded by a C int.
	maxHAProxyFrameSize = 1<<31 - 1
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
			if c.maxFrameSize < 256 {
				return fmt.Errorf("maxFrameSize smaller than minimum allowed size: %d", c.maxFrameSize)
			}
			if c.maxFrameSize > maxHAProxyFrameSize {
				return fmt.Errorf("maxFrameSize exceeds HAProxy maximum: %d", c.maxFrameSize)
			}

		case k.NameEquals(helloKeyEngineID):
			// Engine ID allocation is necessary since we need to store it beyond the lifetime
			// of the KVEntry/Scanner. The underlying bytes will be reused by the frame pool.
			c.engineID = string(k.ValueBytes())
		//case k.NameEquals(helloKeySupportedVersions):
		//case k.NameEquals(helloKeyCapabilities):
		case k.NameEquals(helloKeyHealthcheck):
			// as described in the protocol, close connection after hello
			// AGENT-HELLO + close()
			defer c.ctxCancel(nil)
		}
	}

	if err := s.Error(); err != nil {
		return err
	}
	if c.maxFrameSize == 0 {
		return fmt.Errorf("HAPROXY-HELLO missing %q", helloKeyMaxFrameSize)
	}

	_, err := (&AgentHelloFrame{
		Version:      version,
		MaxFrameSize: c.maxFrameSize,
		Capabilities: []string{},
	}).WriteTo(c.rw)
	return err
}

func (c *protocolClient) onNotify(f *frame) error {
	s := encoding.AcquireMessageScanner(f.buf.ReadBytes())
	defer encoding.ReleaseMessageScanner(s)

	m := encoding.AcquireMessage()
	defer encoding.ReleaseMessage(m)

	fn := func(w *encoding.ActionWriter) error {
		for s.Next(m) {
			err := wrapPanic(func() error {
				c.handler.HandleSPOE(c.ctx, w, m)
				return nil
			})
			if err != nil {
				return err
			}

			if err := m.KV.Discard(); err != nil {
				return err
			}
		}

		return s.Error()
	}

	_, err := (&AckFrame{
		FrameID:              f.meta.FrameID,
		StreamID:             f.meta.StreamID,
		ActionWriterCallback: fn,
	}).writeTo(c.rw, c.maxFrameSize)
	return err
}

func (c *protocolClient) onHAProxyDisconnect(f *frame) error {
	if f.buf.Len() == 0 {
		return fmt.Errorf("disconnect frame without content")
	}

	s := encoding.AcquireKVScanner(f.buf.ReadBytes(), -1)
	defer encoding.ReleaseKVScanner(s)

	k := encoding.AcquireKVEntry()
	defer encoding.ReleaseKVEntry(k)

	var (
		code errorCode
	)

	for s.Next(k) {
		switch {
		case k.NameEquals("status-code"):
			code = errorCode(k.ValueInt())
		case k.NameEquals("message"):
			// We don't really care about the message since they should all be
			// defined in the errorCode type.
		default:
			return fmt.Errorf("unexpected kv entry in disconnect frame: %q", k.NameBytes())
		}
	}

	var err error
	switch code {
	// HAProxy returns an IO error when it doesn't require a connection
	// anymore.
	case ErrorIO, ErrorTimeout, ErrorNone:
	default:
		err = fmt.Errorf("disconnect frame with code %d: %s", code, code)
	}

	c.ctxCancel(err)
	return err
}
