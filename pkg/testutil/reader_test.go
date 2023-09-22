package testutil

import (
	"bytes"
	"testing"
)

func TestRepeatReader_Read(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		out  []byte
	}{
		{name: "same size", data: []byte{1, 2, 3}, out: []byte{1, 2, 3}},
		{name: "double size", data: []byte{1, 2, 3}, out: []byte{1, 2, 3, 1, 2, 3}},
		{name: "huge size", data: []byte{1, 2, 3}, out: bytes.Repeat([]byte{1, 2, 3}, 123)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RepeatReader{
				data: tt.data,
			}
			buf := make([]byte, len(tt.out))
			_, err := r.Read(buf)
			if err != nil {
				t.Errorf("Read() error = %v", err)
				return
			}
			if !bytes.Equal(buf, tt.out) {
				t.Errorf("Equal(): %v != %v", buf, tt.out)
				return
			}
		})
	}
}
