package testutil

import (
	"testing"
)

const runsPerTest = 10

func WithoutAllocations(fn func(*testing.T)) func(t *testing.T) {
	return WithNAllocations(0, fn)
}

func WithNAllocations(n uint64, fn func(*testing.T)) func(t *testing.T) {
	return func(t *testing.T) {
		avg := testing.AllocsPerRun(runsPerTest, func() {
			fn(t)
		})

		// early exit when failed
		if t.Failed() {
			return
		}

		if uint64(avg) != n {
			t.Errorf("got %v allocs, want %d allocs", avg, n)
		}
	}
}
