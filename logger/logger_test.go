package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLogger_NoLogFile(t *testing.T) {
	logger, err := NewLogger("")
	if err != nil {
		t.Fatalf("NewLogger(\"\") failed: %v", err)
	}
	defer logger.Close()

	if logger.enabled {
		t.Error("Logger should not be enabled when no log file is provided")
	}

	if logger.logger != nil {
		t.Error("File logger should be nil when no log file is provided")
	}

	if logger.cliLogger == nil {
		t.Error("CLI logger should not be nil")
	}
}

func TestNewLogger_WithLogFile(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := NewLogger(logFile)
	if err != nil {
		t.Fatalf("NewLogger(%q) failed: %v", logFile, err)
	}
	defer logger.Close()

	if !logger.enabled {
		t.Error("Logger should be enabled when log file is provided")
	}

	if logger.logger == nil {
		t.Error("File logger should not be nil when log file is provided")
	}

	if logger.logFile == nil {
		t.Error("Log file should not be nil when log file is provided")
	}

	// Verify log file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestNewLogger_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "nested", "dir", "test.log")

	logger, err := NewLogger(logFile)
	if err != nil {
		t.Fatalf("NewLogger(%q) failed: %v", logFile, err)
	}
	defer logger.Close()

	// Verify directory was created
	logDir := filepath.Dir(logFile)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("Log directory was not created")
	}
}

func TestLogger_Logf(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := NewLogger(logFile)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer logger.Close()

	// Capture CLI output
	var cliOutput bytes.Buffer
	logger.SetCliLoggerOutput(&cliOutput)

	testMsg := "Test message with %s"
	testArg := "formatting"
	logger.Logf(testMsg, testArg)

	// Check CLI output
	cliStr := cliOutput.String()
	expectedCli := "Test message with formatting"
	if !strings.Contains(cliStr, expectedCli) {
		t.Errorf("CLI output does not contain expected message.\nGot: %s\nWant substring: %s", cliStr, expectedCli)
	}

	// Read log file
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	fileStr := string(content)
	if !strings.Contains(fileStr, expectedCli) {
		t.Errorf("Log file does not contain expected message.\nGot: %s\nWant substring: %s", fileStr, expectedCli)
	}
}

func TestLogger_Log(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := NewLogger(logFile)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer logger.Close()

	// Capture CLI output
	var cliOutput bytes.Buffer
	logger.SetCliLoggerOutput(&cliOutput)

	testMsg := "Simple log message"
	logger.Log(testMsg)

	// Check CLI output
	cliStr := cliOutput.String()
	if !strings.Contains(cliStr, testMsg) {
		t.Errorf("CLI output does not contain expected message.\nGot: %s\nWant substring: %s", cliStr, testMsg)
	}

	// Read log file
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	fileStr := string(content)
	if !strings.Contains(fileStr, testMsg) {
		t.Errorf("Log file does not contain expected message.\nGot: %s\nWant substring: %s", fileStr, testMsg)
	}
}

func TestLogger_LogErrorf(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := NewLogger(logFile)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer logger.Close()

	// Capture CLI output
	var cliOutput bytes.Buffer
	logger.SetCliLoggerOutput(&cliOutput)

	testMsg := "Error message with %s"
	testArg := "details"
	returnedErr := logger.LogErrorf(testMsg, testArg)

	// Check returned error
	if returnedErr == nil {
		t.Error("LogErrorf should return an error")
	}

	expectedErrStr := "Error: Error message with details"
	if returnedErr.Error() != expectedErrStr {
		t.Errorf("Returned error message incorrect.\nGot: %s\nWant: %s", returnedErr.Error(), expectedErrStr)
	}

	// Check CLI output (should contain red color codes)
	cliStr := cliOutput.String()
	if !strings.Contains(cliStr, "Error:") {
		t.Errorf("CLI output does not contain 'Error:'.\nGot: %s", cliStr)
	}
	if !strings.Contains(cliStr, "Error message with details") {
		t.Errorf("CLI output does not contain error message.\nGot: %s", cliStr)
	}
	// Check for ANSI color codes
	if !strings.Contains(cliStr, "\033[31m") {
		t.Error("CLI output does not contain red color code")
	}
	if !strings.Contains(cliStr, "\033[0m") {
		t.Error("CLI output does not contain color reset code")
	}

	// Read log file (should not have color codes)
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	fileStr := string(content)
	if !strings.Contains(fileStr, "Error: Error message with details") {
		t.Errorf("Log file does not contain expected error message.\nGot: %s", fileStr)
	}
	// File should NOT have color codes
	if strings.Contains(fileStr, "\033[31m") {
		t.Error("Log file should not contain color codes")
	}
}

func TestLogger_GetCliLoggerWriter(t *testing.T) {
	logger, err := NewLogger("")
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer logger.Close()

	writer := logger.GetCliLoggerWriter()
	if writer == nil {
		t.Error("GetCliLoggerWriter() returned nil")
	}

	// Test writing to the writer
	testMsg := "test message"
	n, err := writer.Write([]byte(testMsg))
	if err != nil {
		t.Errorf("Write to CLI logger writer failed: %v", err)
	}
	if n != len(testMsg) {
		t.Errorf("Write returned %d, want %d", n, len(testMsg))
	}
}

func TestLogger_SetCliLoggerOutput(t *testing.T) {
	logger, err := NewLogger("")
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer logger.Close()

	var buffer bytes.Buffer
	logger.SetCliLoggerOutput(&buffer)

	testMsg := "redirected output"
	logger.Logf("%s", testMsg)

	output := buffer.String()
	if !strings.Contains(output, testMsg) {
		t.Errorf("Output was not redirected correctly.\nGot: %s\nWant substring: %s", output, testMsg)
	}
}

func TestLogger_Close(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := NewLogger(logFile)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}

	// Write something before closing
	logger.Logf("test message")

	// Close should not panic
	logger.Close()

	// Verify file exists and has content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file after close: %v", err)
	}

	if !strings.Contains(string(content), "test message") {
		t.Error("Log file does not contain expected content after close")
	}
}

func TestLogger_LogfWithoutFile(t *testing.T) {
	logger, err := NewLogger("")
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer logger.Close()

	// Capture CLI output
	var cliOutput bytes.Buffer
	logger.SetCliLoggerOutput(&cliOutput)

	testMsg := "CLI only message"
	logger.Logf("%s", testMsg)

	// Check CLI output
	output := cliOutput.String()
	if !strings.Contains(output, testMsg) {
		t.Errorf("CLI output does not contain expected message.\nGot: %s\nWant substring: %s", output, testMsg)
	}
}

func TestLogger_MultipleWrites(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := NewLogger(logFile)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer logger.Close()

	messages := []string{"First message", "Second message", "Third message"}
	for _, msg := range messages {
		logger.Logf("%s", msg)
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	fileStr := string(content)
	for _, msg := range messages {
		if !strings.Contains(fileStr, msg) {
			t.Errorf("Log file does not contain message: %s", msg)
		}
	}
}