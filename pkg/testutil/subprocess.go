package testutil

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func WithProcess(cmd string, args []string, f func(t *testing.T)) func(t *testing.T) {
	cmdString := cmd
	if len(args) != 0 {
		cmdString += " " + strings.Join(args, " ")
	}

	return func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		c := exec.Command(cmd, args...)
		c.Stdout = &stdout
		c.Stderr = &stderr

		defer func() {
			if c.Process == nil {
				return
			}

			if err := c.Process.Kill(); err != nil {
				t.Errorf("while killing: %q: %v", cmdString, err)
			}

			// ignore the exit result
			_ = c.Wait()

			if t.Failed() {
				t.Logf("Subprocess %q stdout: \n%s", cmdString, stdout.String())
				t.Logf("Subprocess %q stderr: \n%s", cmdString, stderr.String())
			}
		}()

		if err := c.Start(); err != nil {
			t.Fatalf("while running subprocess: %q: %v", cmdString, err)
		}

		f(t)

	}
}
