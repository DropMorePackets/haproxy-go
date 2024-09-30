package testutil

import (
	"testing"
)

func TestAllocTests(t *testing.T) {
	t.Run("without allocations", func(t *testing.T) {
		WithoutAllocations(t, func() {
			v := make([]byte, 10)
			copy(v, []byte{1, 2, 3, 4})
		})
	})

	t.Run("with allocation", func(t *testing.T) {
		WithNAllocations(t, 1, func() {
			v := make([]byte, 10)
			v = append(v, 1)
			_ = v
		})
	})
}
