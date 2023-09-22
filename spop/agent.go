package spop

import (
	"fmt"
	"log"
	"net"
)

type Agent struct {
	Addr    string
	Handler Handler
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

	for {
		nc, err := l.Accept()
		if err != nil {
			return fmt.Errorf("accepting conn: %w", err)
		}

		p := NewProtocolClient(nc, a.Handler)
		go func() {
			defer nc.Close()
			defer p.Close()

			if err := p.Serve(); err != nil {
				log.Println(err)
			}
		}()
	}
}
