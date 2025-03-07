package main

import (
	"SettingsSentry/logger"
	"os"
	"testing"
)

// TestGetICloudFolderLocationErrors tests error cases for get_iCloud_folder_location
// We can't easily test the success case without mocking getHomeDirectory
func TestGetICloudFolderLocationErrors(t *testing.T) {
	// Initialize the global logger properly
	var loggerErr error
	appLogger, loggerErr = logger.NewLogger("")
	if loggerErr != nil {
		t.Fatalf("Failed to initialize logger: %v", loggerErr)
	}

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
	_, iCloudErr := get_iCloud_folder_location()
	if iCloudErr == nil {
		t.Errorf("Expected error when iCloud folder doesn't exist, got nil")
	}
}
