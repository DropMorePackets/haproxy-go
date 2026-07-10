package spop

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/dropmorepackets/haproxy-go/pkg/buffer"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

const uint32Len = 4

type framePool struct {
	pool         sync.Pool
	maxFrameSize uint32
}

func newFramePool(maxFrameSize uint32) *framePool {
	p := &framePool{maxFrameSize: maxFrameSize}
	p.pool.New = func() any {
		return &frame{
			buf:    buffer.NewSliceBuffer(int(maxFrameSize)),
			length: make([]byte, uint32Len),
			pool:   p,
		}
	}
	return p
}

var defaultFramePool = newFramePool(DefaultMaxFrameSize)

func (p *framePool) acquire() *frame {
	return p.pool.Get().(*frame)
}

func acquireFrame() *frame {
	return defaultFramePool.acquire()
}

func releaseFrame(f *frame) {
	f.buf.Reset()
	f.frameType = 0
	f.meta = frameMetadata{}

	f.pool.pool.Put(f)
}

type frameMetadata struct {
	Flags    frameFlag
	StreamID uint64
	FrameID  uint64
}

type frame struct {
	buf  *buffer.SliceBuffer
	pool *framePool

	length []byte
	meta   frameMetadata

	frameType frameType
}

func (f *frame) ReadFrom(r io.Reader) (int64, error) {
	return f.readFrom(r, f.pool.maxFrameSize)
}

func (f *frame) readFrom(r io.Reader, maxFrameSize uint32) (int64, error) {
	if _, err := io.ReadFull(r, f.length); err != nil {
		return 0, fmt.Errorf("reading frame length: %w", err)
	}
	frameLen := binary.BigEndian.Uint32(f.length)

	if maxFrameSize > f.pool.maxFrameSize {
		maxFrameSize = f.pool.maxFrameSize
	}
	if frameLen > maxFrameSize {
		return int64(len(f.length)), newProtocolError(
			ErrorTooBig,
			"frame length %d exceeds maximum %d",
			frameLen,
			maxFrameSize,
		)
	}

	f.buf.Reset()
	dataBuf := f.buf.WriteNBytes(int(frameLen))

	// read full frame into buffer
	n, err := io.ReadFull(r, dataBuf)
	if err != nil {
		return int64(n + len(f.length)), fmt.Errorf("reading frame payload: %w", err)
	}

	return int64(n + len(f.length)), f.decodeHeader()
}

func (f *frame) WriteTo(w io.Writer) (int64, error) {
	return f.writeTo(w, f.pool.maxFrameSize)
}

func (f *frame) writeTo(w io.Writer, maxFrameSize uint32) (int64, error) {
	if maxFrameSize > f.pool.maxFrameSize {
		maxFrameSize = f.pool.maxFrameSize
	}
	frameLen := uint32(f.buf.Len())
	if frameLen > maxFrameSize {
		return 0, newProtocolError(
			ErrorTooBig,
			"frame length %d exceeds maximum %d",
			frameLen,
			maxFrameSize,
		)
	}

	binary.BigEndian.PutUint32(f.length, frameLen)

	if n, err := w.Write(f.length); err != nil {
		return int64(n), err
	}

	n, err := w.Write(f.buf.ReadBytes())
	return int64(n + len(f.length)), err
}

func (f *frame) encodeHeader() error {
	f.buf.WriteNBytes(1)[0] = byte(f.frameType)

	binary.BigEndian.PutUint32(f.buf.WriteNBytes(uint32Len), uint32(f.meta.Flags))

	n, err := encoding.PutVarint(f.buf.WriteBytes(), f.meta.StreamID)
	if err != nil {
		return err
	}
	f.buf.AdvanceW(n)

	n, err = encoding.PutVarint(f.buf.WriteBytes(), f.meta.FrameID)
	if err != nil {
		return err
	}
	f.buf.AdvanceW(n)

	return nil
}

func (f *frame) decodeHeader() error {
	// We don't need to validate here,
	// there is validation further down the chain
	f.frameType = frameType(f.buf.ReadNBytes(1)[0])

	f.meta.Flags = frameFlag(binary.BigEndian.Uint32(f.buf.ReadNBytes(uint32Len)))

	streamID, n, err := encoding.Varint(f.buf.ReadBytes())
	if err != nil {
		return err
	}

	f.meta.StreamID = streamID
	f.buf.AdvanceR(n)

	frameID, n, err := encoding.Varint(f.buf.ReadBytes())
	if err != nil {
		return err
	}
	f.meta.FrameID = frameID
	f.buf.AdvanceR(n)

	return nil
}
