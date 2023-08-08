package peers

import (
	"bufio"
	"net"
)

type Config struct {
	UpdateHandler func(*EntryUpdate)
}

// A listener implements a network listener (net.Listener) for HAProxy connections.
type listener struct {
	net.Listener
	config *Config
}

func NewListener(inner net.Listener, config *Config) *listener {
	l := new(listener)
	l.Listener = inner
	l.config = config
	return l
}

func Listen(network, laddr string, config *Config) (*listener, error) {
	l, err := net.Listen(network, laddr)
	if err != nil {
		return nil, err
	}
	return NewListener(l, config), nil
}

func (l *listener) Accept() (*Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		conn:   c,
		r:      bufio.NewReader(c),
		config: l.config,
	}
	conn.handshakeFn = conn.peerHandshake
	return conn, nil
}
