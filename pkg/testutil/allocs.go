package testutil

import (
	"testing"
)

const runsPerTest = 10

func WithoutAllocations(tb testing.TB, fn func()) {
	WithNAllocations(tb, 0, fn)
}

func WithNAllocations(tb testing.TB, n uint64, fn func()) {
	avg := testing.AllocsPerRun(runsPerTest, fn)

	// early exit when failed
	if tb.Failed() {
		return
	}

	if uint64(avg) != n {
		tb.Errorf("got %v allocs, want %d allocs", avg, n)
	}
}
