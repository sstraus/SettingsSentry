package main

import (
	"SettingsSentry/logger"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrinterPrint(t *testing.T) {
	// Save original stdout and logger output
	oldStdout := os.Stdout
	originalWriter := appLogger.GetCliLoggerWriter() // Use getter

	r, w, _ := os.Pipe()
	os.Stdout = w
	appLogger.SetCliLoggerOutput(w) // Redirect appLogger's output to pipe using setter

	defer func() {
		os.Stdout = oldStdout
		appLogger.SetCliLoggerOutput(originalWriter) // Restore original logger output using setter
	}()

	// Create a printer
	printer := NewPrinter("TestApp")

	// Test first print (should include app name)
	printer.Print("First message: %s", "hello")

	// Test subsequent print (should not include app name)
	printer.Print("Second message: %s", "world")

	// Reset printer
	printer.Reset()

	// Test print after reset (should include app name again)
	printer.Print("Third message: %s", "reset")

	// Close the writer to get all output
	w.Close()
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("io.Copy failed: %v", err)
	}
	output := buf.String()

	// Check that the output contains the expected messages
	if !strings.Contains(output, "First message: hello") {
		t.Errorf("Expected output to contain 'First message: hello', got: %s", output)
	}

	if !strings.Contains(output, "Second message: world") {
		t.Errorf("Expected output to contain 'Second message: world', got: %s", output)
	}

	if !strings.Contains(output, "Third message: reset") {
		t.Errorf("Expected output to contain 'Third message: reset', got: %s", output)
	}

	// Check that the app name appears twice (for the first print and after reset)
	appNameCount := strings.Count(output, "TestApp")
	if appNameCount != 2 {
		t.Errorf("Expected app name to appear 2 times, got %d times", appNameCount)
	}
}

func TestPrinterPrintWithLogger(t *testing.T) {
	// Save the original appLogger and restore it after the test
	originalLogger := appLogger
	defer func() { appLogger = originalLogger }()

	// Create a temporary log file
	tempDir, err := os.MkdirTemp("", "printer-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFilePath := tempDir + "/test.log"
	testLogger, err := logger.NewLogger(logFilePath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer testLogger.Close()

	// Set the global logger
	appLogger = testLogger

	// Create a printer
	printer := NewPrinter("TestApp")

	// Test first print (should include app name)
	printer.Print("First message: %s", "hello")

	// Test subsequent print (should not include app name)
	printer.Print("Second message: %s", "world")

	// Reset printer
	printer.Reset()

	// Test print after reset (should include app name again)
	printer.Print("Third message: %s", "reset")

	// Read the log file
	logContent, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	output := string(logContent)

	// Check that the output contains the expected messages
	if !strings.Contains(output, "First message: hello") {
		t.Errorf("Expected output to contain 'First message: hello', got: %s", output)
	}

	if !strings.Contains(output, "Second message: world") {
		t.Errorf("Expected output to contain 'Second message: world', got: %s", output)
	}

	if !strings.Contains(output, "Third message: reset") {
		t.Errorf("Expected output to contain 'Third message: reset', got: %s", output)
	}

	// Check that the app name appears twice (for the first print and after reset)
	appNameCount := strings.Count(output, "TestApp")
	if appNameCount != 2 {
		t.Errorf("Expected app name to appear 2 times, got %d times", appNameCount)
	}
}
