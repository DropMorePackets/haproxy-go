package spop

import (
	"context"
	"fmt"
	"log"
	"net"
)

type Agent struct {
	Addr        string
	Handler     Handler
	BaseContext context.Context
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

	for {
		nc, err := l.Accept()
		if err != nil {
			return fmt.Errorf("accepting conn: %w", err)
		}

		p := newProtocolClient(a.BaseContext, nc, a.Handler)
		go func() {
			defer nc.Close()
			defer p.Close()

			if err := p.Serve(); err != nil {
				log.Println(err)
			}
		}()
	}
}
