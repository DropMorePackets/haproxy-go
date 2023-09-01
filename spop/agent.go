package spop

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/fionera/haproxy-go/pkg/encoding"
	"log"
	"net"
)

const (
	version = "2.0"

	// maxFrameSize represents the maximum frame size allowed by this library
	// it also represents the maximum slice size that is allowed on stack
	maxFrameSize = 64*1024 - 1
)

type Handler interface {
	HandleSPOE(*encoding.ActionWriter, *encoding.Message)
}

type HandlerFunc func(*encoding.ActionWriter, *encoding.Message)

func (h HandlerFunc) HandleSPOE(w *encoding.ActionWriter, m *encoding.Message) {
	h(w, m)
}

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

		bufConn := bufio.NewReadWriter(bufio.NewReader(nc), bufio.NewWriter(nc))

		c := &conn{
			bufConn:      bufConn,
			conn:         nc,
			maxFrameSize: maxFrameSize,
			handler:      a.Handler,
		}

		go c.serve()
	}
}

type conn struct {
	bufConn      *bufio.ReadWriter
	conn         net.Conn
	handler      Handler
	maxFrameSize uint32
	engineID     string
	gotHello     bool
	lastErr      error
}

const (
	helloKeyMaxFrameSize      = "max-frame-size"
	helloKeySupportedVersions = "supported-versions"
	helloKeyVersion           = "version"
	helloKeyCapabilities      = "capabilities"
	helloKeyHealthcheck       = "healthcheck"
	helloKeyEngineID          = "engine-id"

	capabilityNameAsync      = "async"
	capabilityNamePipelining = "pipelining"
)

func (c *conn) serve() {
	defer c.conn.Close()
	defer c.bufConn.Flush()
	defer c.writeDisconnect()

	f := acquireFrame()
	defer releaseFrame(f)

	for c.readFrame(f) {
		switch f.frameType {
		case frameTypeIDHaproxyHello:
			c.lastErr = c.onHello(f)
		case frameTypeIDNotify:
			c.lastErr = c.onNotify(f)
		case frameTypeIDHaproxyDisconnect:
			c.lastErr = c.onDisconnect(f)
			return
		default:
			panic(fmt.Errorf("unknown frame type: %d", f.frameType))
		}

		if c.lastErr != nil {
			log.Println(c.lastErr)
			//panic(c.lastErr)
		}

		c.bufConn.Flush()
	}
}

func (c *conn) writeFrame(f *frame) error {
	binary.BigEndian.PutUint32(f.length, uint32(f.buf.Len()))

	if _, err := c.bufConn.Write(f.length); err != nil {
		return err
	}

	if _, err := c.bufConn.Write(f.buf.ReadBytes()); err != nil {
		return err
	}

	return nil
}
