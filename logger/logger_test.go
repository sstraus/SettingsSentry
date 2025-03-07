package logger

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	// Test case 1: Create logger without a log file
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

	// Test case 2: Create logger with a valid log file path
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

	// Test case 3: Create logger with a path in a non-existent directory
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

	// Check if the directory was created
	if _, err := os.Stat(nonExistentDir); os.IsNotExist(err) {
		t.Errorf("Expected directory %s to be created", nonExistentDir)
	}
	logger.Close()

	// Test case 4: Create logger with an invalid path (permission denied)
	// Skip this test on Windows as it's harder to create a permission denied scenario
	if os.Getenv("OS") != "Windows_NT" {
		invalidDir := filepath.Join(tempDir, "invalid")
		err = os.Mkdir(invalidDir, 0000) // No permissions
		if err != nil {
			t.Fatalf("Failed to create directory with no permissions: %v", err)
		}
		defer os.Chmod(invalidDir, 0755) // Restore permissions for cleanup

		logFilePath = filepath.Join(invalidDir, "test.log")
		_, err = NewLogger(logFilePath)
		if err == nil {
			t.Errorf("Expected an error for path with no permissions, got nil")
		}
	}
}

func TestLoggerLogf(t *testing.T) {
	// Create a temporary log file
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

	// Log a message
	testMessage := "Test log message %d"
	logger.Logf(testMessage, 123)

	// Read the log file
	content, err := ioutil.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Check if the message was logged
	if !strings.Contains(string(content), "Test log message 123") {
		t.Errorf("Expected log file to contain 'Test log message 123', got %s", string(content))
	}
}

func TestLoggerLog(t *testing.T) {
	// Create a temporary log file
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

	// Log a message
	logger.Log("Test", "log", "message")

	// Read the log file
	content, err := ioutil.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Check if the message was logged
	if !strings.Contains(string(content), "Test log message") {
		t.Errorf("Expected log file to contain 'Test log message', got %s", string(content))
	}
}

func TestLoggerClose(t *testing.T) {
	// Create a temporary log file
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

	// Get the file descriptor before closing
	fd := logger.logFile.Fd()

	// Close the logger
	logger.Close()

	// Try to write to the file after closing (should fail)
	_, err = logger.logFile.Write([]byte("Test"))
	if err == nil {
		t.Errorf("Expected an error when writing to a closed file, got nil")
	}

	// Check if the file descriptor is invalid
	if fd == 0 {
		t.Errorf("Expected file descriptor to be non-zero")
	}
}
