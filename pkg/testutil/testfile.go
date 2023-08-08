package testutil

import (
	"os"
	"testing"
)

// TempFile creates a temporary file that just needs to be deleted with
// os.Remove(f)
func TempFile(t *testing.T, name, content string) string {
	f, err := os.CreateTemp("", name)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	return f.Name()
}
