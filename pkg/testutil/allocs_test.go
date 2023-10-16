package testutil

import (
	"testing"
)

func TestAllocTests(t *testing.T) {
	t.Run("without allocations", WithoutAllocations(func(t *testing.T) {
		v := make([]byte, 10)
		copy(v, []byte{1, 2, 3, 4})
	}))

	t.Run("with allocation", WithNAllocations(1, func(t *testing.T) {
		v := make([]byte, 10)
		v = append(v, 1)
	}))
}
