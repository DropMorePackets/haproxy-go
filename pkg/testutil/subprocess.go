package testutil

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func RunProcess(tb testing.TB, cmd string, args []string) {
	tb.Helper()

	cmdString := cmd
	if len(args) != 0 {
		cmdString += " " + strings.Join(args, " ")
	}

	var stdout, stderr bytes.Buffer
	c := exec.Command(cmd, args...)
	c.Stdout = &stdout
	c.Stderr = &stderr

	tb.Cleanup(func() {
		if c.Process == nil {
			return
		}

		if err := c.Process.Kill(); err != nil {
			tb.Errorf("while killing: %q: %v", cmdString, err)
		}

		// ignore the exit result
		_ = c.Wait()

		if tb.Failed() {
			tb.Logf("Subprocess %q stdout: \n%s", cmdString, stdout.String())
			tb.Logf("Subprocess %q stderr: \n%s", cmdString, stderr.String())
		}
	})

	if err := c.Start(); err != nil {
		tb.Fatalf("while running subprocess: %q: %v", cmdString, err)
	}
}
