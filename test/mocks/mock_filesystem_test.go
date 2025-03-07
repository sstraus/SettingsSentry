package mocks

import (
	"testing"
	"time"
)

func TestMockFileInfo(t *testing.T) {
	// Create a mock file system
	fs := NewMockFileSystem()
	
	// Add a file to the mock file system
	fs.AddFile("test.txt", []byte("test content"), 0644)
	
	// Get file info
	fileInfo, err := fs.Stat("test.txt")
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}
	
	// Check that ModTime returns a time.Time value
	modTime := fileInfo.ModTime()
	if modTime.IsZero() {
		t.Error("ModTime should not be zero")
	}
	
	// Verify the type is time.Time
	_, ok := interface{}(modTime).(time.Time)
	if !ok {
		t.Error("ModTime should return a time.Time value")
	}
}
