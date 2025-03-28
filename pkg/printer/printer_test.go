package printer

import (
	"SettingsSentry/logger"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrinterPrint(t *testing.T) {
	testLogger, _ := logger.NewLogger("")
	oldWriter := testLogger.GetCliLoggerWriter()

	r, w, _ := os.Pipe()
	testLogger.SetCliLoggerOutput(w)

	defer func() {
		testLogger.SetCliLoggerOutput(oldWriter)
		testLogger.Close()
	}()

	printer := NewPrinter("TestApp", testLogger)

	printer.Print("First message: %s", "hello")
	printer.Print("Second message: %s", "world")

	printer.Reset()

	printer.Print("Third message: %s", "reset")

	w.Close()
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("io.Copy failed: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "First message: hello") {
		t.Errorf("Expected output to contain 'First message: hello', got: %s", output)
	}

	if !strings.Contains(output, "Second message: world") {
		t.Errorf("Expected output to contain 'Second message: world', got: %s", output)
	}

	if !strings.Contains(output, "Third message: reset") {
		t.Errorf("Expected output to contain 'Third message: reset', got: %s", output)
	}

	appNameCount := strings.Count(output, "TestApp")
	if appNameCount != 2 {
		t.Errorf("Expected app name to appear 2 times, got %d times", appNameCount)
	}
}

func TestPrinterPrintWithLogger(t *testing.T) {
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

	printer := NewPrinter("TestApp", testLogger)

	printer.Print("First message: %s", "hello")
	printer.Print("Second message: %s", "world")

	printer.Reset()

	printer.Print("Third message: %s", "reset")

	logContent, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	output := string(logContent)

	if !strings.Contains(output, "First message: hello") {
		t.Errorf("Expected output to contain 'First message: hello', got: %s", output)
	}

	if !strings.Contains(output, "Second message: world") {
		t.Errorf("Expected output to contain 'Second message: world', got: %s", output)
	}

	if !strings.Contains(output, "Third message: reset") {
		t.Errorf("Expected output to contain 'Third message: reset', got: %s", output)
	}

	appNameCount := strings.Count(output, "TestApp")
	if appNameCount != 2 {
		t.Errorf("Expected app name to appear 2 times, got %d times", appNameCount)
	}
}
