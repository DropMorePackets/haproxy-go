package spop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"runtime"
)

type Agent struct {
	Handler     Handler
	BaseContext context.Context
	Addr        string

	// MaxFrameSize is the maximum SPOP frame size, excluding the four-byte
	// length prefix. A zero value uses DefaultMaxFrameSize; values below 256
	// are invalid. Each pooled frame buffer is sized to this limit.
	MaxFrameSize uint32
}

func ListenAndServe(addr string, handler Handler) error {
	a := Agent{Addr: addr, Handler: handler}
	return a.ListenAndServe()
}

func (a *Agent) ListenAndServe() error {
	l, err := net.Listen("tcp", a.Addr)
	if err != nil {
		return fmt.Errorf("opening listener: %w", err)
	}
	defer l.Close()

	return a.Serve(l)
}

func (a *Agent) Serve(l net.Listener) error {
	a.Addr = l.Addr().String()
	if a.BaseContext == nil {
		a.BaseContext = context.Background()
	}

	maxFrameSize, err := a.configuredMaxFrameSize()
	if err != nil {
		return err
	}
	framePool := newFramePool(maxFrameSize)

	go func() {
		<-a.BaseContext.Done()
		l.Close()
	}()

	as := newAsyncScheduler()
	for {
		nc, err := l.Accept()
		if err != nil {
			return fmt.Errorf("accepting conn: %w", err)
		}

		if tcp, ok := nc.(*net.TCPConn); ok {
			err = tcp.SetWriteBuffer(math.MaxUint16) // 64KB seems like a fair buffer size
			if err != nil {
				return err
			}
			err = tcp.SetReadBuffer(math.MaxUint16) // 64KB seems like a fair buffer size
			if err != nil {
				return err
			}
		}

		p := newProtocolClient(a.BaseContext, nc, as, framePool, a.Handler)
		go func() {
			defer nc.Close()
			defer p.Close()

			// don't let panics inside the protocol kill the entire library
			if err := wrapPanic(p.Serve); err != nil && !errors.Is(err, p.ctx.Err()) {
				log.Println(err)
			}
		}()
	}
}

func (a *Agent) configuredMaxFrameSize() (uint32, error) {
	maxFrameSize := a.MaxFrameSize
	if maxFrameSize == 0 {
		maxFrameSize = DefaultMaxFrameSize
	}

	if maxFrameSize < minFrameSize {
		return 0, fmt.Errorf("max frame size must be at least %d: %d", minFrameSize, maxFrameSize)
	}
	if uint64(maxFrameSize) > uint64(^uint(0)>>1) {
		return 0, fmt.Errorf("max frame size exceeds platform limit: %d", maxFrameSize)
	}

	return maxFrameSize, nil
}

func wrapPanic(fn func() error) (err error) {
	didPanic := true
	defer func() {
		if didPanic {
			if e := recover(); e != nil {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				err = fmt.Errorf("spop: panic: %v\n%s", e, buf)
			}
		}
	}()
	err = fn()
	didPanic = false
	return
}
