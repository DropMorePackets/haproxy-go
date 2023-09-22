package testutil

import "io"

func NewRepeatReader(data []byte) *RepeatReader {
	return &RepeatReader{data: data}
}

type RepeatReader struct {
	data   []byte
	offset int
}

func (r *RepeatReader) Read(p []byte) (n int, err error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}

	for n < len(p) {
		remaining := len(r.data) - r.offset
		if remaining == 0 {
			r.offset = 0
			continue
		}

		// Calculate how many bytes to copy
		toCopy := min(len(p)-n, remaining)
		copy(p[n:], r.data[r.offset:r.offset+toCopy])
		n += toCopy
		r.offset += toCopy
	}

	return n, nil
}
