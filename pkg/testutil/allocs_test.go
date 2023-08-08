package testutil

import (
	"testing"
	"unsafe"
)

func TestAllocTests(t *testing.T) {
	v := make([]byte, 10)
	t.Run("without allocations", WithoutAllocations(func(t *testing.T) {
		copy(v, []byte{1, 2, 3, 4})
	}))

	t.Run("with allocation", WithNAllocations(uint64(unsafe.Sizeof(v)), func(t *testing.T) {
		v = append(v, 1)
	}))
}
