package spop

import (
	"encoding/binary"
	"github.com/fionera/haproxy-go/pkg/buffer"
	"sync"

	"github.com/fionera/haproxy-go/pkg/encoding"
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
	f.err = nil

	framePool.Put(f)
}

type frameFlag uint32

const (
	frameFlagFin  frameFlag = 1
	frameFlagAbrt frameFlag = 2
)

type frameType byte

const (
	// Frames sent by HAProxy
	frameTypeIDHaproxyHello      frameType = 1
	frameTypeIDHaproxyDisconnect frameType = 2
	frameTypeIDNotify            frameType = 3

	// Frames sent by the agents
	frameTypeIDAgentHello      frameType = 101
	frameTypeIDAgentDisconnect frameType = 102
	frameTypeIDAck             frameType = 103
)

type frameMetadata struct {
	Flags    frameFlag
	StreamID int64
	FrameID  int64
}

type frame struct {
	length []byte
	buf    *buffer.SliceBuffer

	frameType frameType
	meta      frameMetadata
	err       error
}

func (f *frame) writeHeader() {
	f.buf.WriteNBytes(1)[0] = byte(f.frameType)

	binary.BigEndian.PutUint32(f.buf.WriteNBytes(uint32Len), uint32(f.meta.Flags))

	var n int
	n, f.err = encoding.PutVarint(f.buf.WriteBytes(), f.meta.StreamID)
	f.buf.AdvanceW(n)

	n, f.err = encoding.PutVarint(f.buf.WriteBytes(), f.meta.FrameID)
	f.buf.AdvanceW(n)
}

func (f *frame) readHeader() {
	// We don't need to validate here,
	// there is validation further down the chain
	f.frameType = frameType(f.buf.ReadNBytes(1)[0])

	f.meta.Flags = frameFlag(binary.BigEndian.Uint32(f.buf.ReadNBytes(uint32Len)))

	var n int
	f.meta.StreamID, n, f.err = encoding.Varint(f.buf.ReadBytes())
	f.buf.AdvanceR(n)

	f.meta.FrameID, n, f.err = encoding.Varint(f.buf.ReadBytes())
	f.buf.AdvanceR(n)
}
