package peers

import (
	"context"
	"fmt"
	"log"
	"net"
)

type Peer struct {
	Addr          string
	Handler       Handler
	HandlerSource func() Handler
	BaseContext   context.Context
}

func ListenAndServe(addr string, handler Handler) error {
	a := Peer{Addr: addr, Handler: handler}
	return a.ListenAndServe()
}

func (a *Peer) ListenAndServe() error {
	l, err := net.Listen("tcp", a.Addr)
	if err != nil {
		return fmt.Errorf("opening listener: %w", err)
	}
	defer l.Close()

	return a.Serve(l)
}

func (a *Peer) Serve(l net.Listener) error {
	a.Addr = l.Addr().String()
	if a.BaseContext == nil {
		a.BaseContext = context.Background()
	}

	go func() {
		<-a.BaseContext.Done()
		l.Close()
	}()

	if a.Handler != nil && a.HandlerSource != nil {
		return fmt.Errorf("cannot set Handler and HandlerSource at the same time")
	}

	if a.Handler != nil {
		a.HandlerSource = func() Handler {
			return a.Handler
		}
	}

	for {
		nc, err := l.Accept()
		if err != nil {
			return fmt.Errorf("accepting conn: %w", err)
		}

		p := newProtocolClient(a.BaseContext, nc, a.HandlerSource())
		go func() {
			defer nc.Close()
			defer p.Close()

			if err := p.Serve(); err != nil && err != p.ctx.Err() {
				log.Println(err)
			}
		}()
	}
}
