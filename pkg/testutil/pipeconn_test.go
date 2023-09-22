package testutil

import (
	"bytes"
	"testing"
)

func TestPipeConn(t *testing.T) {
	rw, netConn := PipeConn()

	var a, b = []byte("abc"), []byte("def")
	go func() {
		if _, err := rw.Write(a); err != nil {
			t.Error(err)
		}

		if _, err := netConn.Write(b); err != nil {
			t.Error(err)
		}
	}()

	expectA := make([]byte, len(a))
	if _, err := netConn.Read(expectA); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(a, expectA) {
		t.Fatal("data doesnt match")
	}

	expectB := make([]byte, len(b))
	if _, err := rw.Read(expectB); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b, expectB) {
		t.Fatal("data doesnt match")
	}
}
