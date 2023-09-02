package encoding

import (
	"fmt"
	"io"
)

var (
	ErrUnterminatedSequence = fmt.Errorf("unterminated sequence")
	ErrInsufficientSpace    = fmt.Errorf("insufficient space in buffer")
)

func ReadVarint(rd io.ByteReader) (int64, error) {
	b, err := rd.ReadByte()
	if err != nil {
		return 0, ErrUnterminatedSequence
	}

	val := int64(b)
	off := 1

	if val < 240 {
		return val, nil
	}

	r := uint(4)
	for {
		b, err := rd.ReadByte()
		if err != nil {
			return 0, ErrUnterminatedSequence
		}

		v := int64(b)
		val += v << r
		off++
		r += 7

		if v < 128 {
			break
		}
	}

	return val, nil
}

// Source: https://github.com/criteo/haproxy-spoe-go/blob/master/encoding.go
func PutVarint(b []byte, i int64) (int, error) {
	if len(b) == 0 {
		return 0, ErrInsufficientSpace
	}

	if i < 240 {
		b[0] = byte(i)
		return 1, nil
	}

	n := 0
	b[n] = byte(i) | 240
	n++
	i = (i - 240) >> 4
	for i >= 128 {
		if n > len(b)-1 {
			return 0, ErrInsufficientSpace
		}

		b[n] = byte(i) | 128
		n++
		i = (i - 128) >> 7
	}

	if n > len(b)-1 {
		return 0, ErrInsufficientSpace
	}

	b[n] = byte(i)
	n++

	return n, nil
}

// Source: https://github.com/criteo/haproxy-spoe-go/blob/master/encoding.go
func Varint(b []byte) (int64, int, error) {
	if len(b) == 0 {
		return 0, 0, ErrUnterminatedSequence
	}
	val := int64(b[0])
	off := 1

	if val < 240 {
		return val, 1, nil
	}

	r := uint(4)
	for {
		if off > len(b)-1 {
			return 0, 0, ErrUnterminatedSequence
		}

		v := int64(b[off])
		val += v << r
		off++
		r += 7

		if v < 128 {
			break
		}
	}

	return val, off, nil
}
