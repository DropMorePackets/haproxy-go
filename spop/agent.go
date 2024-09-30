package spop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
)

type Agent struct {
	Handler     Handler
	BaseContext context.Context
	Addr        string
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
			err = tcp.SetWriteBuffer(maxFrameSize * 4)
			if err != nil {
				return err
			}
			err = tcp.SetReadBuffer(maxFrameSize * 4)
			if err != nil {
				return err
			}
		}

		p := newProtocolClient(a.BaseContext, nc, as, a.Handler)
		go func() {
			defer nc.Close()
			defer p.Close()

			if err := p.Serve(); err != nil && !errors.Is(err, p.ctx.Err()) {
				log.Println(err)
			}
		}()
	}
}
