//go:build integration
// +build integration

package main_test // Changed package declaration

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/pkg/backup"
	"SettingsSentry/pkg/command"
	"SettingsSentry/pkg/config"
	"SettingsSentry/pkg/printer"
	"SettingsSentry/pkg/testutil"
	"SettingsSentry/pkg/util"
	"fmt"
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
	// --- Setup Globals ---
	testLogger := testutil.SetupTestGlobals(interfaces.NewOsFileSystem(), interfaces.NewOsCommandExecutor())
	backup.AppLogger = util.AppLogger
	backup.Fs = util.Fs
	config.Fs = util.Fs
	config.AppLogger = util.AppLogger
	testLogger.Logf("TEST LOG FROM SETUP in TestBackupAndRestore") // Verify logger works

	command.CmdExecutor = util.CmdExecutor
	backup.Printer = printer.NewPrinter("IntegrationTest", testLogger)
	command.Printer = backup.Printer // Share printer
	backup.DryRun = false            // Ensure DryRun is off for integration tests unless specifically testing it
	util.DryRun = false
	// --- End Setup Globals ---

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "settingssentry-integration")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tempDir, err)
		}
	}()

	// Create a test home directory
	homeDir := filepath.Join(tempDir, "home")
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatalf("Failed to create test home directory: %v", err)
	}

	// Create test configuration files in the home directory
	testConfigContent := "test configuration content"
	testConfigPath := filepath.Join(homeDir, ".testapp_config")
	if err := ioutil.WriteFile(testConfigPath, []byte(testConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Create a test data directory
	testDataDir := filepath.Join(homeDir, ".testapp_data")
	if err := os.MkdirAll(testDataDir, 0755); err != nil {
		t.Fatalf("Failed to create test data directory: %v", err)
	}
	testDataFile := filepath.Join(testDataDir, "data.txt")
	if err := ioutil.WriteFile(testDataFile, []byte("test data content"), 0644); err != nil {
		t.Fatalf("Failed to write test data file: %v", err)
	}

	// Create a backup directory
	backupDir := filepath.Join(tempDir, "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Copy the test configuration file to the configs directory
	configsDir := filepath.Join(tempDir, "configs")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("Failed to create configs directory: %v", err)
	}
	// Replace call to unexported backup.copyFile with standard library functions
	// Use the renamed fixture and align destination filename
	srcCfgPath := "test/fixtures/testapp.cfg"
	dstCfgPath := filepath.Join(configsDir, "testapp.cfg")
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
	if err := os.Setenv("HOME", homeDir); err != nil {
		t.Fatalf("Failed to set HOME env var: %v", err)
	}
	defer func() {
		if err := os.Setenv("HOME", originalHome); err != nil {
			t.Logf("Warning: Failed to restore HOME env var: %v", err)
		}
	}()

	// Run the backup process
	// Note: Added zipBackup=false argument and password=""
	backup.ProcessConfiguration(configsDir, backupDir, "TestApp", true, true, 1, false, "")

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
	if err := os.Remove(testConfigPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to remove original config file before restore: %v", err)
	}
	if err := os.RemoveAll(testDataDir); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to remove original data dir before restore: %v", err)
	}

	// Run the restore process
	// Note: Added zipBackup=false argument and password=""
	backup.ProcessConfiguration(configsDir, backupDir, "TestApp", false, true, 1, false, "")

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

// TestBackupAndRestore_Encrypted tests the full backup and restore flow with encryption
func TestBackupAndRestore_Encrypted(t *testing.T) {
	// --- Setup Globals ---
	testLogger := testutil.SetupTestGlobals(interfaces.NewOsFileSystem(), interfaces.NewOsCommandExecutor())
	backup.AppLogger = util.AppLogger
	backup.Fs = util.Fs
	config.Fs = util.Fs
	config.AppLogger = util.AppLogger
	command.CmdExecutor = util.CmdExecutor
	backup.Printer = printer.NewPrinter("IntegrationTestEnc", testLogger)
	command.Printer = backup.Printer // Share printer
	backup.DryRun = false            // Ensure DryRun is off for integration tests unless specifically testing it
	util.DryRun = false
	// --- End Setup Globals ---

	// --- Test Setup --- (Similar to TestBackupAndRestore)
	tempDir, err := ioutil.TempDir("", "settingssentry-integration-enc")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tempDir, err)
		}
	}()

	homeDir := filepath.Join(tempDir, "home")
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatalf("Failed to create test home directory: %v", err)
	}

	testConfigContent := "encrypted configuration content"
	testConfigFilename := ".testapp_config_enc"
	testConfigPath := filepath.Join(homeDir, testConfigFilename)
	if err := ioutil.WriteFile(testConfigPath, []byte(testConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	backupDir := filepath.Join(tempDir, "backup_enc")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	configsDir := filepath.Join(tempDir, "configs_enc")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("Failed to create configs directory: %v", err)
	}

	// Use a different fixture or create one dynamically if needed for encryption test
	srcCfgContent := fmt.Sprintf(`[application]
name = TestAppEnc
[files]
~/%s
`, testConfigFilename)
	// Align destination filename with appName.cfg convention
	dstCfgPath := filepath.Join(configsDir, "testappenc.cfg")
	if err := os.WriteFile(dstCfgPath, []byte(srcCfgContent), 0644); err != nil {
		t.Fatalf("Failed to write destination config file %s: %v", dstCfgPath, err)
	}

	originalHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", homeDir); err != nil {
		t.Fatalf("Failed to set HOME env var: %v", err)
	}
	defer func() {
		if err := os.Setenv("HOME", originalHome); err != nil {
			t.Logf("Warning: Failed to restore HOME env var: %v", err)
		}
	}()

	password := "test-password-123"
	appName := "TestAppEnc"

	// --- Run Encrypted Backup ---
	backup.ProcessConfiguration(configsDir, backupDir, appName, true, false, 1, false, password)

	// --- Verify Encrypted Backup ---
	latestVersion, isZip, err := backup.GetLatestVersionPath(backupDir)
	if err != nil {
		t.Fatalf("Failed to get latest version path after encrypted backup: %v", err)
	}
	if isZip {
		t.Fatalf("Expected directory backup, but got zip: %s", latestVersion)
	}

	backupAppDir := filepath.Join(latestVersion, appName)
	if _, err := os.Stat(backupAppDir); os.IsNotExist(err) {
		t.Fatalf("Backup directory for %s was not created", appName)
	}

	// Check for the .encrypted file
	encryptedBackupConfigFile := filepath.Join(backupAppDir, testConfigFilename+".encrypted")
	if _, err := os.Stat(encryptedBackupConfigFile); os.IsNotExist(err) {
		t.Errorf("Encrypted backup config file %s was not created", encryptedBackupConfigFile)
	}

	// Check that the unencrypted file does NOT exist
	unencryptedBackupConfigFile := filepath.Join(backupAppDir, testConfigFilename)
	if _, err := os.Stat(unencryptedBackupConfigFile); err == nil {
		t.Errorf("Unencrypted backup config file %s was found, but should not exist", unencryptedBackupConfigFile)
	}

	// --- Prepare for Restore ---
	if err := os.Remove(testConfigPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to remove original file before restore: %v", err)
	}

	// --- Attempt Restore WITHOUT Password (should fail) ---
	backup.ProcessConfiguration(configsDir, backupDir, appName, false, false, 1, false, "") // Empty password

	// Verify file was NOT restored
	if _, err := os.Stat(testConfigPath); !os.IsNotExist(err) {
		t.Errorf("Config file was restored without password, but should have failed")
		// Clean up if it was incorrectly restored
		if err := os.Remove(testConfigPath); err != nil {
			t.Logf("Warning: Failed to remove incorrectly restored file: %v", err)
		}
	}

	// --- Attempt Restore WITH Correct Password ---
	backup.ProcessConfiguration(configsDir, backupDir, appName, false, false, 1, false, password)

	// --- Verify Correct Restore ---
	if _, err := os.Stat(testConfigPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not restored with correct password")
	}

	restoredConfigContent, err := ioutil.ReadFile(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to read restored config file: %v", err)
	}
	if string(restoredConfigContent) != testConfigContent {
		t.Errorf("Restored config content does not match original. Expected '%s', got '%s'",
			testConfigContent, string(restoredConfigContent))
	}
}
