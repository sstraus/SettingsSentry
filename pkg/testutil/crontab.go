package testutil

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// PreserveCrontab restores the account's crontab even when a test calls Fatal.
// Packages that mutate the crontab must run serially (go test -p 1).
func PreserveCrontab(t *testing.T) {
	t.Helper()
	cmd := exec.Command("crontab", "-l")
	var original, stderr bytes.Buffer
	cmd.Stdout = &original
	cmd.Stderr = &stderr
	err := cmd.Run()
	hadCrontab := err == nil
	var exitErr *exec.ExitError
	if err != nil && (!errors.As(err, &exitErr) || exitErr.ExitCode() != 1 || !strings.Contains(stderr.String(), "no crontab for")) {
		t.Fatalf("Cannot save original crontab: %v: %s", err, stderr.String())
	}

	t.Cleanup(func() {
		cmd := exec.Command("crontab", "-")
		if hadCrontab {
			cmd.Stdin = bytes.NewReader(original.Bytes())
		} else {
			cmd = exec.Command("crontab", "-r")
		}
		output, err := cmd.CombinedOutput()
		if !hadCrontab && errors.As(err, &exitErr) && exitErr.ExitCode() == 1 && strings.Contains(string(output), "no crontab for") {
			return
		}
		if err != nil {
			t.Errorf("Failed to restore original crontab: %v: %s", err, output)
		}
	})
}
