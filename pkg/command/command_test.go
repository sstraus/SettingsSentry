package command

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/pkg/printer"
	"SettingsSentry/pkg/testutil" // Added testutil
	"errors"
	"io"
	"testing"
)

func setupTestDependencies() {
	// Use the shared helper, passing nil for FS and CmdExecutor as they aren't needed here
	testLogger := testutil.SetupTestGlobals(nil, nil)

	// Initialize package-specific dependencies
	testPrinter := printer.NewPrinter("Test", testLogger)
	Printer = testPrinter
	// printer.AppLogger is set via util.InitGlobals inside SetupTestGlobals
}

func TestSafeExecute(t *testing.T) {
	setupTestDependencies()

	err := SafeExecute("successful operation", func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error for successful operation, got: %v", err)
	}

	expectedErr := errors.New("test error")
	err = SafeExecute("error operation", func() error {
		return expectedErr
	})
	if err != expectedErr {
		t.Errorf("Expected error '%v', got: %v", expectedErr, err)
	}

	// We expect SafeExecute to recover and potentially log, but not crash the test
	// We also don't expect an error return value from the panic itself
	err = SafeExecute("panic operation", func() error {
		panic("test panic")
	})
	if err != nil {
		t.Errorf("Expected no error after panic recovery, got: %v", err)
	}
}

func TestExecuteCommandLine_Success(t *testing.T) {
	setupTestDependencies()

	// Mock command executor that always succeeds
	mockExecutor := &MockCommandExecutor{shouldSucceed: true}
	CmdExecutor = mockExecutor

	result := ExecuteCommandLine("echo test")
	if !result {
		t.Error("ExecuteCommandLine should return true for successful command")
	}

	if !mockExecutor.called {
		t.Error("Command executor should have been called")
	}
}

func TestExecuteCommandLine_Failure(t *testing.T) {
	setupTestDependencies()

	// Mock command executor that always fails
	mockExecutor := &MockCommandExecutor{shouldSucceed: false}
	CmdExecutor = mockExecutor

	result := ExecuteCommandLine("failing command")
	if result {
		t.Error("ExecuteCommandLine should return false for failed command")
	}
}

func TestExecuteCommandLine_EmptyCommand(t *testing.T) {
	setupTestDependencies()

	mockExecutor := &MockCommandExecutor{shouldSucceed: true}
	CmdExecutor = mockExecutor

	result := ExecuteCommandLine("")
	if !result {
		t.Error("ExecuteCommandLine should return true for empty command")
	}

	if mockExecutor.called {
		t.Error("Command executor should not be called for empty command")
	}
}

func TestExecuteCommandLine_NilExecutor(t *testing.T) {
	setupTestDependencies()

	CmdExecutor = nil

	result := ExecuteCommandLine("test command")
	if result {
		t.Error("ExecuteCommandLine should return false when executor is nil")
	}
}

func TestExecuteCommandLine_WithCallbacks(t *testing.T) {
	setupTestDependencies()

	mockExecutor := &MockCommandExecutor{
		shouldSucceed: true,
		stdoutLines:   []string{"output line 1", "output line 2"},
		stderrLines:   []string{"error line 1"},
	}
	CmdExecutor = mockExecutor

	result := ExecuteCommandLine("test command")
	if !result {
		t.Error("ExecuteCommandLine should return true for successful command")
	}
}

func TestExecuteCommandLine_NoPrinter(t *testing.T) {
	setupTestDependencies()

	// Set Printer to nil
	Printer = nil
	mockExecutor := &MockCommandExecutor{shouldSucceed: true}
	CmdExecutor = mockExecutor

	// Should not panic
	result := ExecuteCommandLine("test command")
	if !result {
		t.Error("ExecuteCommandLine should work without Printer")
	}
}

// MockCommandExecutor for testing
type MockCommandExecutor struct {
	shouldSucceed bool
	called        bool
	stdoutLines   []string
	stderrLines   []string
}

func (m *MockCommandExecutor) Execute(commandLine string, stdout, stderr io.Writer) bool {
	m.called = true
	return m.shouldSucceed
}

func (m *MockCommandExecutor) ExecuteWithCallback(commandLine string, stdoutHandler, stderrHandler interfaces.OutputHandler) bool {
	m.called = true
	
	// Simulate stdout output
	for _, line := range m.stdoutLines {
		if stdoutHandler != nil {
			stdoutHandler(line)
		}
	}
	
	// Simulate stderr output
	for _, line := range m.stderrLines {
		if stderrHandler != nil {
			stderrHandler(line)
		}
	}
	
	return m.shouldSucceed
}
