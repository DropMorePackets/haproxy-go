package testutil

import (
	"net"
	"testing"
)

func TCPListener(tb testing.TB) net.Listener {
	tb.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		_ = l.Close()
	})
	return l
}

func TCPPort(tb testing.TB) int {
	tb.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		tb.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
