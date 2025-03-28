package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger, err := NewLogger("")
	if err != nil {
		t.Errorf("NewLogger() with empty path returned an error: %v", err)
	}
	if logger.logFile != nil {
		t.Errorf("Expected logFile to be nil, got %v", logger.logFile)
	}
	if logger.enabled {
		t.Errorf("Expected enabled to be false, got true")
	}
	logger.Close()

	tempDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFilePath := filepath.Join(tempDir, "test.log")
	logger, err = NewLogger(logFilePath)
	if err != nil {
		t.Errorf("NewLogger() with valid path returned an error: %v", err)
	}
	if logger.logFile == nil {
		t.Errorf("Expected logFile to not be nil")
	}
	if !logger.enabled {
		t.Errorf("Expected enabled to be true, got false")
	}
	logger.Close()

	nonExistentDir := filepath.Join(tempDir, "non-existent")
	logFilePath = filepath.Join(nonExistentDir, "test.log")
	logger, err = NewLogger(logFilePath)
	if err != nil {
		t.Errorf("NewLogger() with non-existent directory returned an error: %v", err)
	}
	if logger.logFile == nil {
		t.Errorf("Expected logFile to not be nil")
	}
	if !logger.enabled {
		t.Errorf("Expected enabled to be true, got false")
	}

	if _, err := os.Stat(nonExistentDir); os.IsNotExist(err) {
		t.Errorf("Expected directory %s to be created", nonExistentDir)
	}
	logger.Close()

	// Skip this test on Windows as it's harder to create a permission denied scenario
	if os.Getenv("OS") != "Windows_NT" {
		invalidDir := filepath.Join(tempDir, "invalid")
		err = os.Mkdir(invalidDir, 0000) // No permissions
		if err != nil {
			t.Fatalf("Failed to create directory with no permissions: %v", err)
		}
		defer func() {
			err := os.Chmod(invalidDir, 0755)
			if err != nil {
				t.Errorf("Failed to restore permissions for cleanup: %v", err)
			}
		}()

		logFilePath = filepath.Join(invalidDir, "test.log")
		_, err = NewLogger(logFilePath)
		if err == nil {
			t.Errorf("Expected an error for path with no permissions, got nil")
		}
	}
}

func TestLoggerLogf(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFilePath := filepath.Join(tempDir, "test.log")
	logger, err := NewLogger(logFilePath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	testMessage := "Test log message %d"
	logger.Logf(testMessage, 123)

	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "Test log message 123") {
		t.Errorf("Expected log file to contain 'Test log message 123', got %s", string(content))
	}
}

func TestLoggerLog(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFilePath := filepath.Join(tempDir, "test.log")
	logger, err := NewLogger(logFilePath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	logger.Log("Test", "log", "message")

	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "Test log message") {
		t.Errorf("Expected log file to contain 'Test log message', got %s", string(content))
	}
}

func TestLoggerClose(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFilePath := filepath.Join(tempDir, "test.log")
	logger, err := NewLogger(logFilePath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	fd := logger.logFile.Fd()

	logger.Close()

	// Try to write to the file after closing (should fail)
	_, err = logger.logFile.Write([]byte("Test"))
	if err == nil {
		t.Errorf("Expected an error when writing to a closed file, got nil")
	}

	if fd == 0 {
		t.Errorf("Expected file descriptor to be non-zero")
	}
}
