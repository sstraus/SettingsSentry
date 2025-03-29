package config

import (
	"SettingsSentry/interfaces"
	// "SettingsSentry/logger" // No longer needed directly
	"SettingsSentry/pkg/testutil" // Added testutil
	"SettingsSentry/pkg/util"     // Keep util for Fs/AppLogger access
	"os"
	"path/filepath"
	"testing"
)

func setupTestDependencies() {
	// Create the necessary FS implementation for this test context
	testFs := interfaces.NewOsFileSystem()

	// Use the shared helper, passing the OS FS and nil for CmdExecutor
	_ = testutil.SetupTestGlobals(testFs, nil) // Logger is returned but not needed directly here

	// Initialize package-specific dependencies using globals set by the helper
	Fs = util.Fs
	AppLogger = util.AppLogger
}

func TestConfig(t *testing.T) {
	config := Config{
		Name:                "TestApp",
		Files:               []string{".testconfig"},
		PreBackupCommands:   []string{"echo backup"},
		PostBackupCommands:  []string{},
		PreRestoreCommands:  []string{"echo restore"},
		PostRestoreCommands: []string{},
	}

	if config.Name != "TestApp" {
		t.Errorf("Expected Name to be 'TestApp', got '%s'", config.Name)
	}
	if len(config.Files) != 1 || config.Files[0] != ".testconfig" {
		t.Errorf("Files not parsed correctly, got %v", config.Files)
	}
	if len(config.PreBackupCommands) != 1 || config.PreBackupCommands[0] != "echo backup" {
		t.Errorf("PreBackupCommands not parsed correctly, got %v", config.PreBackupCommands)
	}
	if len(config.PreRestoreCommands) != 1 || config.PreRestoreCommands[0] != "echo restore" {
		t.Errorf("PreRestoreCommands not parsed correctly, got %v", config.PreRestoreCommands)
	}
}

func TestParseConfigWithComments(t *testing.T) {
	setupTestDependencies()

	tempDir, err := os.MkdirTemp("", "settingssentry-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tempDir, err)
		}
	}()

	configContent := `[application]
# This is a comment
name = TestApp

[backup_commands]
# This is another comment
test backup command

[restore_commands]
; This is a semicolon comment
test restore command

[configuration_files]
# Comment before a config file
.testconfig
`
	configPath := filepath.Join(tempDir, "test.cfg")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	testFS := os.DirFS(tempDir)
	config, err := ParseConfig(testFS, filepath.Base(configPath))
	if err != nil {
		t.Errorf("ParseConfig() returned an error: %v", err)
	}

	if config.Name != "TestApp" {
		t.Errorf("Expected Name to be 'TestApp', got '%s'", config.Name)
	}
	if len(config.Files) != 1 || config.Files[0] != ".testconfig" {
		t.Errorf("Files not parsed correctly, got %v", config.Files)
	}
	if len(config.PreBackupCommands) != 1 || config.PreBackupCommands[0] != "test backup command" {
		t.Errorf("PreBackupCommands not parsed correctly, got %v", config.PreBackupCommands)
	}
	if len(config.PreRestoreCommands) != 1 || config.PreRestoreCommands[0] != "test restore command" {
		t.Errorf("PreRestoreCommands not parsed correctly, got %v", config.PreRestoreCommands)
	}
}

func TestParseConfigWithMissingFile(t *testing.T) {
	setupTestDependencies()

	_, err := ParseConfig(os.DirFS("/"), "nonexistent/file.cfg")
	if err == nil {
		t.Errorf("ParseConfig() did not return an error for a non-existent file")
	}
}
