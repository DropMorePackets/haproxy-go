package peers

import (
	"bufio"
	"fmt"
	"net"
)

type Peer struct {
	Addr    string
	Handler Handler
}

type Handler interface {
	Update(*EntryUpdate)
}

type HandlerFunc func(*EntryUpdate)

func (h HandlerFunc) Update(u *EntryUpdate) {
	h(u)
}

func ListenAndServe(addr string, h Handler) error {
	a := Peer{Addr: addr, Handler: h}
	return a.ListenAndServe()
}

func (p *Peer) ListenAndServe() error {
	l, err := net.Listen("tcp", p.Addr)
	if err != nil {
		return fmt.Errorf("opening listener: %w", err)
	}
	defer l.Close()

	return p.Serve(l)
}

func (p *Peer) Serve(l net.Listener) error {
	p.Addr = l.Addr().String()

	for {
		c, err := l.Accept()
		if err != nil {
			return fmt.Errorf("accepting conn: %w", err)
		}

		conn := &Conn{
			conn:    c,
			r:       bufio.NewReader(c),
			handler: p.Handler,
		}

		go conn.serve()
	}
}
