package interfaces

import (
	"bytes"
	"strings"
	"testing"
)

func TestOsCommandExecutor_Execute(t *testing.T) {
	executor := NewOsCommandExecutor()

	tests := []struct {
		name          string
		command       string
		expectSuccess bool
	}{
		{
			name:          "successful echo command",
			command:       "echo 'test output'",
			expectSuccess: true,
		},
		{
			name:          "successful true command",
			command:       "true",
			expectSuccess: true,
		},
		{
			name:          "failing false command",
			command:       "false",
			expectSuccess: false,
		},
		{
			name:          "nonexistent command",
			command:       "nonexistentcommand12345",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			result := executor.Execute(tt.command, &stdout, &stderr)

			if result != tt.expectSuccess {
				t.Errorf("Execute(%q) = %v, want %v", tt.command, result, tt.expectSuccess)
			}

			if tt.expectSuccess && tt.command == "echo 'test output'" {
				output := strings.TrimSpace(stdout.String())
				if !strings.Contains(output, "test output") {
					t.Errorf("Expected output to contain 'test output', got: %s", output)
				}
			}
		})
	}
}

func TestOsCommandExecutor_ExecuteWithCallback(t *testing.T) {
	executor := NewOsCommandExecutor()

	tests := []struct {
		name           string
		command        string
		expectSuccess  bool
		expectStdout   string
		expectStderr   bool
	}{
		{
			name:           "echo to stdout",
			command:        "echo 'callback test'",
			expectSuccess:  true,
			expectStdout:   "callback test",
			expectStderr:   false,
		},
		{
			name:           "multiple lines",
			command:        "echo 'line1' && echo 'line2'",
			expectSuccess:  true,
			expectStdout:   "line",
			expectStderr:   false,
		},
		{
			name:           "error to stderr",
			command:        "echo 'error message' >&2",
			expectSuccess:  true,
			expectStdout:   "",
			expectStderr:   true,
		},
		{
			name:           "failing command",
			command:        "false",
			expectSuccess:  false,
			expectStdout:   "",
			expectStderr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdoutLines, stderrLines []string

			stdoutHandler := func(line string) {
				stdoutLines = append(stdoutLines, line)
			}

			stderrHandler := func(line string) {
				stderrLines = append(stderrLines, line)
			}

			result := executor.ExecuteWithCallback(tt.command, stdoutHandler, stderrHandler)

			if result != tt.expectSuccess {
				t.Errorf("ExecuteWithCallback(%q) = %v, want %v", tt.command, result, tt.expectSuccess)
			}

			if tt.expectStdout != "" {
				foundOutput := false
				for _, line := range stdoutLines {
					if strings.Contains(line, tt.expectStdout) {
						foundOutput = true
						break
					}
				}
				if !foundOutput {
					t.Errorf("Expected stdout to contain %q, got: %v", tt.expectStdout, stdoutLines)
				}
			}

			if tt.expectStderr && len(stderrLines) == 0 {
				t.Errorf("Expected stderr output, got none")
			}
		})
	}
}

func TestOsCommandExecutor_ExecuteWithCallback_NilHandlers(t *testing.T) {
	executor := NewOsCommandExecutor()

	// Test with nil handlers - should not panic
	result := executor.ExecuteWithCallback("echo 'test'", nil, nil)
	if !result {
		t.Errorf("ExecuteWithCallback with nil handlers failed unexpectedly")
	}
}

func TestNewOsCommandExecutor(t *testing.T) {
	executor := NewOsCommandExecutor()
	if executor == nil {
		t.Error("NewOsCommandExecutor() returned nil")
	}

	// Verify it implements the interface
	var _ CommandExecutor = executor
}