package peers

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)

type Peer struct {
	Handler       Handler
	HandlerSource func() Handler
	BaseContext   context.Context
	Addr          string
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

		// Wrap the context to provide access to the underlying connection.
		// TODO(tim): Do we really want this?
		ctx := context.WithValue(a.BaseContext, connectionKey, nc)
		wmu := &sync.Mutex{}
		w := newWriter(nc, wmu)
		ctx = context.WithValue(ctx, writerKey, w)
		p := newProtocolClient(ctx, nc, a.HandlerSource(), wmu, w.bufferedWriter())
		go func() {
			defer nc.Close()
			defer p.Close()

			if err := p.Serve(); err != nil && err != p.ctx.Err() {
				log.Println(err)
			}
		}()
	}
}

type contextKey string

const (
	connectionKey = contextKey("connection")
	writerKey     = contextKey("writer")
)

// Connection returns the underlying connection used in calls
// to function in a Handler.
func Connection(ctx context.Context) net.Conn {
	return ctx.Value(connectionKey).(net.Conn)
}

// WriterFromContext returns the Writer associated with the current peer
// connection. Use this inside a Handler to push stick table updates back
// to HAProxy over the same connection that HAProxy established to us.
// Panics if called outside a handler context.
func WriterFromContext(ctx context.Context) *Writer {
	return ctx.Value(writerKey).(*Writer)
}
