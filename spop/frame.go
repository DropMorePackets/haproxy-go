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

var framePool = sync.Pool{
	New: func() any {
		return &frame{
			length: make([]byte, uint32Len),
			buf:    buffer.NewSliceBuffer(maxFrameSize),
		}
	},
}

func acquireFrame() *frame {
	return framePool.Get().(*frame)
}

func releaseFrame(f *frame) {
	f.buf.Reset()
	f.frameType = 0
	f.meta = frameMetadata{}

	framePool.Put(f)
}

type frameMetadata struct {
	Flags    frameFlag
	StreamID uint64
	FrameID  uint64
}

type frame struct {
	buf *buffer.SliceBuffer

	length []byte
	meta   frameMetadata

	frameType frameType
}

func (f *frame) ReadFrom(r io.Reader) (int64, error) {
	return f.readFrom(r, maxFrameSize)
}

func (f *frame) readFrom(r io.Reader, limit uint32) (int64, error) {
	if _, err := io.ReadFull(r, f.length); err != nil {
		return 0, fmt.Errorf("reading frame length: %w", err)
	}
	frameLen := binary.BigEndian.Uint32(f.length)

	if frameLen > limit {
		return int64(len(f.length)), fmt.Errorf("frame length %d exceeds maximum %d", frameLen, limit)
	}

	f.buf.Reset()
	f.buf.Grow(int(frameLen))
	dataBuf := f.buf.WriteNBytes(int(frameLen))

	// read full frame into buffer
	n, err := io.ReadFull(r, dataBuf)
	if err != nil {
		return int64(n + len(f.length)), fmt.Errorf("reading frame payload: %w", err)
	}

	return int64(n + len(f.length)), f.decodeHeader()
}

func (f *frame) WriteTo(w io.Writer) (int64, error) {
	return f.writeTo(w, nil)
}

func (f *frame) writeTo(w io.Writer, payload []byte) (int64, error) {
	frameLen := uint64(f.buf.Len()) + uint64(len(payload))
	if frameLen > uint64(^uint32(0)) {
		return 0, fmt.Errorf("frame length %d exceeds protocol limit", frameLen)
	}
	binary.BigEndian.PutUint32(f.length, uint32(frameLen))

	n, err := w.Write(f.length)
	written := int64(n)
	if err != nil {
		return written, err
	}

	n, err = w.Write(f.buf.ReadBytes())
	written += int64(n)
	if err != nil || len(payload) == 0 {
		return written, err
	}

	n, err = w.Write(payload)
	return written + int64(n), err
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
