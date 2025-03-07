package test

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTempDir creates a temporary directory for testing and returns its path
// The caller is responsible for cleaning up the directory using defer os.RemoveAll(tempDir)
func CreateTempDir(t *testing.T, prefix string) string {
	tempDir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	return tempDir
}

// CreateTestFile creates a test file with the given content
func CreateTestFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file %s: %v", name, err)
	}
	return path
}

// CreateTestDir creates a test directory
func CreateTestDir(t *testing.T, parent, name string) string {
	path := filepath.Join(parent, name)
	err := os.MkdirAll(path, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory %s: %v", name, err)
	}
	return path
}

// AssertFileExists checks if a file exists
func AssertFileExists(t *testing.T, path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist, but it doesn't", path)
	}
}

// AssertFileContent checks if a file has the expected content
func AssertFileContent(t *testing.T, path, expectedContent string) {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to read file %s: %v", path, err)
		return
	}
	if string(content) != expectedContent {
		t.Errorf("File %s content does not match. Expected '%s', got '%s'", 
			path, expectedContent, string(content))
	}
}

// WithEnvVar temporarily sets an environment variable for the duration of the test
func WithEnvVar(t *testing.T, key, value string, testFunc func()) {
	original, exists := os.LookupEnv(key)
	os.Setenv(key, value)
	defer func() {
		if exists {
			os.Setenv(key, original)
		} else {
			os.Unsetenv(key)
		}
	}()
	testFunc()
}
