package main

import (
	"os"
	"testing"
)

// TestGetICloudFolderLocationErrors tests error cases for get_iCloud_folder_location
// We can't easily test the success case without mocking getHomeDirectory
func TestGetICloudFolderLocationErrors(t *testing.T) {
	// Save the original fs and restore it after the test
	originalFs := fs
	defer func() { fs = originalFs }()

	// Create a mock file system that always returns errors
	mockFs := &mockFileSystem{
		statFunc: func(path string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
	}
	fs = mockFs

	// Test case: iCloud folder not found
	_, err := get_iCloud_folder_location()
	if err == nil {
		t.Errorf("Expected error when iCloud folder doesn't exist, got nil")
	}
}
