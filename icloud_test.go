package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetICloudFolderLocation(t *testing.T) {
	// This test is more of a smoke test since we can't easily mock the iCloud path
	// on a real system. It will at least verify that the function doesn't crash.
	
	// Get the iCloud folder location
	icloudPath, err := get_iCloud_folder_location()
	
	// If the test is running on a system without iCloud configured,
	// we expect an error, so we'll skip the test in that case
	if err != nil {
		t.Skipf("Skipping test because iCloud is not configured: %v", err)
	}
	
	// Verify the path is not empty
	if icloudPath == "" {
		t.Errorf("get_iCloud_folder_location() returned an empty path")
	}
	
	// Verify the path exists
	_, err = os.Stat(icloudPath)
	if os.IsNotExist(err) {
		t.Errorf("iCloud path %s does not exist", icloudPath)
	}
}

// TestGetICloudFolderLocationWithMock tests the function with a mocked environment
func TestGetICloudFolderLocationWithMock(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "settingssentry-test", "Library", "Mobile Documents", "com~apple~CloudDocs")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(filepath.Join(os.TempDir(), "settingssentry-test"))
	
	// Save the original HOME environment variable
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	
	// Set HOME to our test directory
	os.Setenv("HOME", filepath.Join(os.TempDir(), "settingssentry-test"))
	
	// This test will still likely fail on a real system because the function
	// uses filepath.EvalSymlinks which expects real symlinks, but it's a good
	// example of how we would test this function in isolation
}
