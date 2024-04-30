package buffer

type SliceBuffer struct {
	buf         []byte
	readOffset  int
	writeOffset int
}

func NewSliceBuffer(size int) *SliceBuffer {
	return &SliceBuffer{
		buf: make([]byte, size),
	}
}

func NewSliceBufferWithSlice(b []byte) *SliceBuffer {
	return &SliceBuffer{
		buf:         b,
		writeOffset: len(b),
	}
}

func (s *SliceBuffer) Reset() {
	s.readOffset = 0
	s.writeOffset = 0
}

func (s *SliceBuffer) ReadBytes() []byte {
	return s.buf[s.readOffset:s.writeOffset]
}

func (s *SliceBuffer) WriteBytes() []byte {
	return s.buf[s.writeOffset:]
}

func (s *SliceBuffer) AdvanceR(n int) {
	s.readOffset += n
}

func (s *SliceBuffer) AdvanceW(n int) {
	s.writeOffset += n
}

func (s *SliceBuffer) WriteNBytes(n int) []byte {
	s.writeOffset += n
	return s.buf[s.writeOffset-n : s.writeOffset]
}

func (s *SliceBuffer) ReadNBytes(n int) []byte {
	s.readOffset += n
	return s.buf[s.readOffset-n : s.readOffset]
}

func (s *SliceBuffer) Len() int {
	return s.writeOffset - s.readOffset
}
