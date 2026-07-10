package spop

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

func newProtocolClient(
	ctx context.Context,
	rw io.ReadWriter,
	as *asyncScheduler,
	framePool *framePool,
	handler Handler,
) *protocolClient {
	var c protocolClient
	c.rw = rw
	c.handler = handler
	c.ctx, c.ctxCancel = context.WithCancelCause(ctx)
	c.as = as
	c.framePool = framePool
	c.maxFrameSize = framePool.maxFrameSize
	return &c
}

type protocolClient struct {
	rw      io.ReadWriter
	handler Handler
	ctx     context.Context

	ctxCancel context.CancelCauseFunc
	as        *asyncScheduler
	framePool *framePool
	engineID  string

	closeOnce sync.Once

	maxFrameSize uint32

	gotHello bool
}

func (c *protocolClient) Close() error {
	if c.ctx.Err() != nil {
		return c.ctx.Err()
	}

	c.terminate(fmt.Errorf("closing client"))

	return nil
}

func (c *protocolClient) terminate(err error) {
	c.terminateWithCode(err, protocolErrorCode(err))
}

func (c *protocolClient) terminateWithCode(err error, code errorCode) {
	c.closeOnce.Do(func() {
		c.ctxCancel(err)

		if conn, ok := c.rw.(net.Conn); ok {
			if deadlineErr := conn.SetWriteDeadline(time.Now().Add(disconnectWriteTimeout)); deadlineErr == nil {
				_, _ = (&AgentDisconnectFrame{
					ErrCode: code,
				}).writeTo(conn, c.framePool, c.maxFrameSize)
			}
			_ = conn.Close()
			return
		}

		// A generic writer cannot provide a bounded best-effort write.
		if closer, ok := c.rw.(io.Closer); ok {
			_ = closer.Close()
		}
	})
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
	f, err := c.readFrame()
	if err != nil {
		err = c.readError(err)
		if err != nil {
			c.terminate(err)
		}
		return err
	}
	if f.frameType != frameTypeIDHaproxyHello {
		firstFrameType := f.frameType
		releaseFrame(f)
		err := newProtocolError(ErrorInvalid, "first frame must be HAPROXY-HELLO, got type %d", firstFrameType)
		c.terminate(err)
		return err
	}
	if err := c.frameHandler(f); err != nil {
		c.terminate(err)
		return err
	}
	if c.ctx.Err() != nil {
		return context.Cause(c.ctx)
	}

	for {
		f, err := c.readFrame()
		if err != nil {
			err = c.readError(err)
			if err != nil {
				c.terminate(err)
			}
			return err
		}

		c.as.schedule(f, c)
	}
}

func (c *protocolClient) readFrame() (*frame, error) {
	f := c.framePool.acquire()
	if _, err := f.readFrom(c.rw, c.maxFrameSize); err != nil {
		releaseFrame(f)
		return nil, err
	}
	return f, nil
}

func (c *protocolClient) readError(err error) error {
	if c.ctx.Err() != nil {
		return context.Cause(c.ctx)
	}
	if errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET) {
		return nil
	}
	return err
}

const (
	version = "2.0"

	minFrameSize           = 256
	disconnectWriteTimeout = 100 * time.Millisecond

	// DefaultMaxFrameSize is the maximum SPOP frame size used by an Agent when
	// MaxFrameSize is not configured.
	DefaultMaxFrameSize uint32 = 64<<10 - 1
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
	gotMaxFrameSize := false
	for s.Next(k) {
		switch {
		case k.NameEquals(helloKeyMaxFrameSize):
			if k.Type() != encoding.DataTypeUInt32 {
				return newProtocolError(ErrorBadFrameSize, "peer max frame size must be uint32")
			}
			gotMaxFrameSize = true
			peerMaxFrameSize := uint32(k.ValueInt())
			if peerMaxFrameSize < minFrameSize {
				return newProtocolError(
					ErrorBadFrameSize,
					"peer max frame size must be at least %d: %d",
					minFrameSize,
					peerMaxFrameSize,
				)
			}
			if binary.BigEndian.Uint32(f.length) > peerMaxFrameSize {
				return newProtocolError(
					ErrorBadFrameSize,
					"HAPROXY-HELLO frame length %d exceeds peer maximum %d",
					binary.BigEndian.Uint32(f.length),
					peerMaxFrameSize,
				)
			}
			if peerMaxFrameSize < c.maxFrameSize {
				c.maxFrameSize = peerMaxFrameSize
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
	if !gotMaxFrameSize {
		return newProtocolError(ErrorNoFrameSize, "HAPROXY-HELLO missing %q", helloKeyMaxFrameSize)
	}

	_, err := (&AgentHelloFrame{
		Version:      version,
		MaxFrameSize: c.maxFrameSize,
		Capabilities: []string{},
	}).writeTo(c.rw, c.framePool, c.maxFrameSize)
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
	}).writeTo(c.rw, c.framePool, c.maxFrameSize)
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
		err = context.Canceled
	default:
		err = fmt.Errorf("disconnect frame with code %d: %s", code, code)
	}

	c.terminateWithCode(err, code)
	if code == ErrorIO || code == ErrorTimeout || code == ErrorNone {
		return nil
	}
	return err
}
