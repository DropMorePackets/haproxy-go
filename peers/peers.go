package peers

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
)

type Peer struct {
	Addr        string
	Handler     Handler
	BaseContext context.Context
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

	for {
		nc, err := l.Accept()
		if err != nil {
			return fmt.Errorf("accepting conn: %w", err)
		}

		conn := &Conn{
			ctx:     a.BaseContext,
			conn:    nc,
			r:       bufio.NewReader(nc),
			handler: a.Handler,
		}

		go func() {
			defer nc.Close()

			if err := conn.Serve(); err != nil && err != conn.ctx.Err() {
				log.Println(err)
			}
		}()
	}
}
