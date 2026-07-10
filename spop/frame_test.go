package spop

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestFrame_ReadFrom_ExceedsMaxFrameSize(t *testing.T) {
	writeFrameLength := func(length uint32) *bytes.Buffer {
		var buf bytes.Buffer
		lengthBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(lengthBytes, length)
		buf.Write(lengthBytes)
		return &buf
	}

	assertError := func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "frame length") || !strings.Contains(err.Error(), "exceeds maximum") {
			t.Errorf("error should mention frame length exceeding maximum, got: %v", err)
		}
	}

	t.Run("rejects frame length exceeding maxFrameSize", func(t *testing.T) {
		buf := writeFrameLength(369295622) // production panic value

		f := acquireFrame()
		defer releaseFrame(f)

		_, err := f.ReadFrom(buf)
		assertError(t, err)
	})

	t.Run("accepts frame length at maxFrameSize boundary", func(t *testing.T) {
		buf := writeFrameLength(DefaultMaxFrameSize)

		frameData := make([]byte, DefaultMaxFrameSize)
		frameData[0] = byte(frameTypeIDHaproxyHello)
		binary.BigEndian.PutUint32(frameData[1:5], 0)
		frameData[5] = 0 // streamID varint
		frameData[6] = 0 // frameID varint
		buf.Write(frameData)

		f := acquireFrame()
		defer releaseFrame(f)

		_, err := f.ReadFrom(buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.frameType != frameTypeIDHaproxyHello {
			t.Errorf("expected frameType %v, got %v", frameTypeIDHaproxyHello, f.frameType)
		}
	})

	t.Run("rejects frame length one byte over maxFrameSize", func(t *testing.T) {
		buf := writeFrameLength(DefaultMaxFrameSize + 1)

		f := acquireFrame()
		defer releaseFrame(f)

		_, err := f.ReadFrom(buf)
		assertError(t, err)
	})

	t.Run("accepts configured frame length above default", func(t *testing.T) {
		const configuredMaxFrameSize uint32 = 262140
		buf := writeFrameLength(configuredMaxFrameSize)

		frameData := make([]byte, configuredMaxFrameSize)
		frameData[0] = byte(frameTypeIDHaproxyHello)
		binary.BigEndian.PutUint32(frameData[1:5], 0)
		frameData[5] = 0
		frameData[6] = 0
		buf.Write(frameData)

		f := newFramePool(configuredMaxFrameSize).acquire()
		defer releaseFrame(f)

		if _, err := f.ReadFrom(buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects frame above configured maximum", func(t *testing.T) {
		const configuredMaxFrameSize uint32 = 262140
		buf := writeFrameLength(configuredMaxFrameSize + 1)

		f := newFramePool(configuredMaxFrameSize).acquire()
		defer releaseFrame(f)

		_, err := f.ReadFrom(buf)
		assertError(t, err)
	})
}
