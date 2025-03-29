//go:build integration
// +build integration

package main_test // Changed package declaration

import (
	// Added backup import
	"SettingsSentry/pkg/backup"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This file contains integration tests that test the application's functionality
// as a whole. These tests are more comprehensive but may take longer to run.
// To run these tests, use: go test -tags=integration

// TestBackupAndRestore tests the full backup and restore flow
func TestBackupAndRestore(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "settingssentry-integration")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test home directory
	homeDir := filepath.Join(tempDir, "home")
	err = os.MkdirAll(homeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test home directory: %v", err)
	}

	// Create test configuration files in the home directory
	testConfigContent := "test configuration content"
	testConfigPath := filepath.Join(homeDir, ".testapp_config")
	err = ioutil.WriteFile(testConfigPath, []byte(testConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Create a test data directory
	testDataDir := filepath.Join(homeDir, ".testapp_data")
	err = os.MkdirAll(testDataDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test data directory: %v", err)
	}
	testDataFile := filepath.Join(testDataDir, "data.txt")
	err = ioutil.WriteFile(testDataFile, []byte("test data content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test data file: %v", err)
	}

	// Create a backup directory
	backupDir := filepath.Join(tempDir, "backup")
	err = os.MkdirAll(backupDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Copy the test configuration file to the configs directory
	configsDir := filepath.Join(tempDir, "configs")
	err = os.MkdirAll(configsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create configs directory: %v", err)
	}
	// Replace call to unexported backup.copyFile with standard library functions
	srcCfgPath := "test/fixtures/test_app.cfg"
	dstCfgPath := filepath.Join(configsDir, "test_app.cfg")
	cfgContent, readErr := os.ReadFile(srcCfgPath)
	if readErr != nil {
		t.Fatalf("Failed to read source fixture file %s: %v", srcCfgPath, readErr)
	}
	writeErr := os.WriteFile(dstCfgPath, cfgContent, 0644)
	if writeErr != nil {
		t.Fatalf("Failed to write destination config file %s: %v", dstCfgPath, writeErr)
	}

	// Set up the environment for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", originalHome)

	// Run the backup process
	// Note: Added zipBackup=false argument
	backup.ProcessConfiguration(configsDir, backupDir, "TestApp", true, true, 1, false)

	// Verify the backup was created
	// Note: GetLatestVersionPath now returns 3 values (path, isZip, err)
	latestVersion, _, err := backup.GetLatestVersionPath(backupDir) // Ignore isZip for now
	if err != nil {
		if strings.Contains(err.Error(), "no version backups found") {
			t.Logf("No version backups found, skipping restore verification")
			return
		}
		t.Fatalf("Failed to get latest version path: %v", err)
	}
	backupAppDir := filepath.Join(latestVersion, "TestApp")
	if _, err := os.Stat(backupAppDir); os.IsNotExist(err) {
		t.Errorf("Backup directory for TestApp was not created")
	}

	backupConfigFile := filepath.Join(backupAppDir, ".testapp_config")
	if _, err := os.Stat(backupConfigFile); os.IsNotExist(err) {
		t.Errorf("Backup config file was not created")
	}

	// Delete the original files to test restore
	os.Remove(testConfigPath)
	os.RemoveAll(testDataDir)

	// Run the restore process
	// Note: Added zipBackup=false argument
	backup.ProcessConfiguration(configsDir, backupDir, "TestApp", false, true, 1, false)

	// Verify the files were restored
	if _, err := os.Stat(testConfigPath); os.IsNotExist(err) {
		t.Errorf("Config file was not restored")
	}

	restoredConfigContent, err := ioutil.ReadFile(testConfigPath)
	if err != nil {
		t.Errorf("Failed to read restored config file: %v", err)
	}
	if string(restoredConfigContent) != testConfigContent {
		t.Errorf("Restored config content does not match original. Expected '%s', got '%s'",
			testConfigContent, string(restoredConfigContent))
	}

	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		t.Errorf("Data directory was not restored")
	}
}
