package interfaces

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
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

// TestOsCommandExecutor_ShellInjectionVulnerability demonstrates that the current
// implementation is vulnerable to shell injection attacks via malicious config files
func TestOsCommandExecutor_ShellInjectionVulnerability(t *testing.T) {
	executor := NewOsCommandExecutor()

	// These test cases demonstrate how a malicious config file could execute
	// arbitrary commands if placed in backup_commands or pre_backup_commands
	vulnerabilityTests := []struct {
		name             string
		maliciousCommand string
		shouldBeBlocked  bool // After fix, these should be blocked
		description      string
	}{
		{
			name:             "command chaining with semicolon",
			maliciousCommand: "echo 'backup'; rm -rf /tmp/test_dir",
			shouldBeBlocked:  true,
			description:      "Attacker chains multiple commands",
		},
		{
			name:             "command substitution",
			maliciousCommand: "echo $(whoami)",
			shouldBeBlocked:  true,
			description:      "Attacker uses command substitution to exfiltrate data",
		},
		{
			name:             "pipe to malicious command",
			maliciousCommand: "cat /etc/passwd | head -n 1",
			shouldBeBlocked:  true,
			description:      "Attacker pipes sensitive data",
		},
		{
			name:             "background process",
			maliciousCommand: "sleep 0.1 &",
			shouldBeBlocked:  true,
			description:      "Attacker launches background process",
		},
		{
			name:             "redirect output to file",
			maliciousCommand: "echo 'malicious' > /tmp/injected.txt",
			shouldBeBlocked:  true,
			description:      "Attacker writes arbitrary files",
		},
	}

	for _, tt := range vulnerabilityTests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			// Currently, these commands will execute successfully,
			// demonstrating the vulnerability
			result := executor.Execute(tt.maliciousCommand, &stdout, &stderr)

			// Document current behavior: commands execute without validation
			t.Logf("CURRENT BEHAVIOR: Command '%s' executed with result=%v",
				tt.maliciousCommand, result)
			t.Logf("Description: %s", tt.description)

			// After fix: these commands should be blocked or sanitized
			// For now, we just document that this is a vulnerability
			if tt.shouldBeBlocked {
				t.Logf("SECURITY: This command should be blocked in a secure implementation")
			}
		})
	}
}

// TestOsCommandExecutor_SafeCommandExamples shows what safe commands should look like
func TestOsCommandExecutor_SafeCommandExamples(t *testing.T) {
	// These are examples of safe command patterns that should be allowed
	// after implementing proper command validation
	safeCommands := []string{
		"ls -la",
		"cp source.txt dest.txt",
		"mkdir -p /path/to/dir",
		"tar -czf backup.tar.gz source/",
	}

	_ = NewOsCommandExecutor()

	for _, cmd := range safeCommands {
		t.Run("safe:_"+cmd, func(t *testing.T) {
			// These commands should still work after implementing security fixes
			// (though they may need to be whitelisted or follow specific patterns)
			t.Logf("Safe command example: %s", cmd)
		})
	}
}

// TestOsCommandExecutor_GoroutineCompletion tests that ExecuteWithCallback
// properly waits for stdout/stderr reading goroutines to complete
func TestOsCommandExecutor_GoroutineCompletion(t *testing.T) {
	executor := NewOsCommandExecutor()

	// Test with a command that produces multiple lines of output
	// This increases the chance of catching the race condition
	command := "for i in 1 2 3 4 5; do echo \"line $i\"; done"

	var stdoutLines []string
	var mu sync.Mutex // Protect stdoutLines from concurrent access

	stdoutHandler := func(line string) {
		mu.Lock()
		stdoutLines = append(stdoutLines, line)
		mu.Unlock()
	}

	result := executor.ExecuteWithCallback(command, stdoutHandler, nil)

	if !result {
		t.Fatal("Command execution failed")
	}

	// After ExecuteWithCallback returns, ALL goroutines should have completed
	// and ALL output should be captured
	mu.Lock()
	lineCount := len(stdoutLines)
	mu.Unlock()

	// We expect exactly 5 lines
	if lineCount != 5 {
		t.Errorf("Expected 5 output lines, got %d. This suggests goroutines didn't complete before function returned", lineCount)
		t.Logf("Captured lines: %v", stdoutLines)
	}

	// Verify the content is complete
	expectedLines := []string{"line 1", "line 2", "line 3", "line 4", "line 5"}
	mu.Lock()
	for i, expected := range expectedLines {
		if i >= len(stdoutLines) || stdoutLines[i] != expected {
			t.Errorf("Line %d: expected %q, got incomplete output", i+1, expected)
		}
	}
	mu.Unlock()
}

// TestOsCommandExecutor_GoroutineRace uses multiple rapid executions
// to increase probability of catching race conditions
func TestOsCommandExecutor_GoroutineRace(t *testing.T) {
	executor := NewOsCommandExecutor()

	// Run multiple commands in sequence to stress test the goroutine synchronization
	for i := 0; i < 10; i++ {
		command := fmt.Sprintf("echo 'iteration %d'", i)

		var output string
		stdoutHandler := func(line string) {
			output = line
		}

		result := executor.ExecuteWithCallback(command, stdoutHandler, nil)
		if !result {
			t.Fatalf("Iteration %d: command failed", i)
		}

		expected := fmt.Sprintf("iteration %d", i)
		if output != expected {
			t.Errorf("Iteration %d: expected %q, got %q (possible race condition)", i, expected, output)
		}
	}
}