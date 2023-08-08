package testutil

import (
	"runtime"
	"testing"
)

func WithoutAllocations(fn func(*testing.T)) func(t *testing.T) {
	return WithNAllocations(0, fn)
}

func WithNAllocations(n uint64, fn func(*testing.T)) func(t *testing.T) {
	var m runtime.MemStats
	return func(t *testing.T) {
		runtime.ReadMemStats(&m)
		prev := m.TotalAlloc

		fn(t)

		// early exit when failed
		if t.Failed() {
			return
		}

		runtime.ReadMemStats(&m)
		after := m.TotalAlloc

		if after-prev != n {
			t.Errorf("%d bytes got allocated, should be %d", after-prev, n)
		}
	}
}
