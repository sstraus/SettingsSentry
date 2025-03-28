package main

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"os"
	"path/filepath"
	"testing"
)

func TestConfig(t *testing.T) {
	// Test creating a Config struct directly
	config := Config{
		Name:                "TestApp",
		Files:               []string{".testconfig"},
		PreBackupCommands:   []string{"echo backup"},
		PostBackupCommands:  []string{},
		PreRestoreCommands:  []string{"echo restore"},
		PostRestoreCommands: []string{},
	}

	// Verify the Config struct
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
	// Create a temporary config file with comments
	tempDir, err := os.MkdirTemp("", "settingssentry-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

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

	// Test parsing the config
	// Create an iofs.FS rooted at the temp directory
	testFS := os.DirFS(tempDir)
	// Pass the FS and the relative path within the FS
	config, err := parseConfig(testFS, filepath.Base(configPath))
	if err != nil {
		t.Errorf("parseConfig() returned an error: %v", err)
	}

	// Verify the parsed config
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
	var err error
	appLogger, err = logger.NewLogger("")
	if err != nil {
		t.Fatalf("failed to initialize logger: %v", err)
	}
	fs = interfaces.NewOsFileSystem()
	// Test parsing a non-existent config file
	// Provide the root FS and the path relative to root
	_, err = parseConfig(os.DirFS("/"), "nonexistent/file.cfg")
	if err == nil {
		t.Errorf("parseConfig() did not return an error for a non-existent file")
	}
}
