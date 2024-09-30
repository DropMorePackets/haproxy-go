package testutil

import (
	"os"
	"testing"
)

// TempFile creates a temporary file that just needs to be deleted with
// os.Remove(f)
func TempFile(tb testing.TB, name, content string) string {
	tb.Helper()

	f, err := os.CreateTemp("", name)
	if err != nil {
		tb.Fatal(err)
	}

	if _, err := f.WriteString(content); err != nil {
		tb.Fatal(err)
	}

	if err := f.Close(); err != nil {
		tb.Fatal(err)
	}

	tb.Cleanup(func() {
		os.Remove(f.Name())
	})

	return f.Name()
}
