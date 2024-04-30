package testutil

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

type rwPipeConn struct {
	rw        *rwPipe
	closedMtx sync.Mutex
	closed    bool
}

func (c *rwPipeConn) Network() string {
	return "pipe"
}

func (c *rwPipeConn) String() string {
	return "pipe"
}

func (c *rwPipeConn) Read(b []byte) (n int, err error) {
	c.closedMtx.Lock()
	if c.closed {
		c.closedMtx.Unlock()
		return 0, net.ErrClosed
	}
	c.closedMtx.Unlock()

	return c.rw.Read(b)
}

func (c *rwPipeConn) Write(b []byte) (n int, err error) {
	c.closedMtx.Lock()
	if c.closed {
		c.closedMtx.Unlock()
		return 0, net.ErrClosed
	}
	c.closedMtx.Unlock()

	return c.rw.Write(b)
}

func (c *rwPipeConn) Close() error {
	c.closedMtx.Lock()
	defer c.closedMtx.Unlock()

	c.closed = true
	return c.rw.Close()
}

func (c *rwPipeConn) LocalAddr() net.Addr {
	return c
}

func (c *rwPipeConn) RemoteAddr() net.Addr {
	return c
}

func (c *rwPipeConn) SetDeadline(t time.Time) error {
	return errors.ErrUnsupported
}

func (c *rwPipeConn) SetReadDeadline(t time.Time) error {
	return errors.ErrUnsupported
}

func (c *rwPipeConn) SetWriteDeadline(t time.Time) error {
	return errors.ErrUnsupported
}

func PipeConn() (io.ReadWriteCloser, net.Conn) {
	a, b := newRWPipe()
	return a, &rwPipeConn{rw: b}
}

func newRWPipe() (a *rwPipe, b *rwPipe) {
	rr, rw := io.Pipe()
	wr, ww := io.Pipe()

	return &rwPipe{rr, ww}, &rwPipe{wr, rw}
}

type rwPipe struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func (r *rwPipe) Close() error {
	return errors.Join(r.r.Close(), r.w.Close())
}

func (r *rwPipe) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r *rwPipe) Write(p []byte) (n int, err error) {
	return r.w.Write(p)
}
