package main

import (
	"SettingsSentry/interfaces"
	"os"
	"path/filepath"
	"testing"
)

func TestExpandEnvVars(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("HOME_DIR", "/home/user")
	defer os.Unsetenv("TEST_VAR")
	defer os.Unsetenv("HOME_DIR")

	testCases := []struct {
		input    string
		expected string
	}{
		{"${TEST_VAR}", "test_value"},
		{"prefix_${TEST_VAR}_suffix", "prefix_test_value_suffix"},
		{"${HOME_DIR}/config", "/home/user/config"},
		{"no_vars_here", "no_vars_here"},
		{"${NONEXISTENT_VAR}", ""},
		{"~/${TEST_VAR}", "~/test_value"},
	}

	for _, tc := range testCases {
		result := expandEnvVars(tc.input)
		if result != tc.expected {
			t.Errorf("expandEnvVars(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "Valid config",
			config: Config{
				Name:                "TestApp",
				Files:               []string{".testconfig"},
				PreBackupCommands:   []string{"echo backup"},
				PostBackupCommands:  []string{},
				PreRestoreCommands:  []string{"echo restore"},
				PostRestoreCommands: []string{},
			},
			expectError: false,
		},
		{
			name: "Missing app name",
			config: Config{
				Name:                "",
				Files:               []string{".testconfig"},
				PreBackupCommands:   []string{"echo backup"},
				PostBackupCommands:  []string{},
				PreRestoreCommands:  []string{"echo restore"},
				PostRestoreCommands: []string{},
			},
			expectError: true,
		},
		{
			name: "Missing config files",
			config: Config{
				Name:                "TestApp",
				Files:               []string{},
				PreBackupCommands:   []string{"echo backup"},
				PostBackupCommands:  []string{},
				PreRestoreCommands:  []string{"echo restore"},
				PostRestoreCommands: []string{},
			},
			expectError: true,
		},
		{
			name: "Empty config file path",
			config: Config{
				Name:                "TestApp",
				Files:               []string{""},
				PreBackupCommands:   []string{"echo backup"},
				PostBackupCommands:  []string{},
				PreRestoreCommands:  []string{"echo restore"},
				PostRestoreCommands: []string{},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateConfig(tc.config)
			if tc.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestParseConfigWithEnvVars(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_APP_NAME", "EnvTestApp")
	os.Setenv("TEST_CONFIG_FILE", ".env_testconfig")
	os.Setenv("TEST_BACKUP_CMD", "echo env_backup")
	defer os.Unsetenv("TEST_APP_NAME")
	defer os.Unsetenv("TEST_CONFIG_FILE")
	defer os.Unsetenv("TEST_BACKUP_CMD")

	// Create a temporary config file
	tempDir, err := os.MkdirTemp("", "settingssentry-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configContent := `[application]
name = ${TEST_APP_NAME}

[backup_commands]
${TEST_BACKUP_CMD}

[restore_commands]
echo restore

[configuration_files]
${TEST_CONFIG_FILE}
`
	configPath := tempDir + "/test_env.cfg"
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Mock the filesystem for testing
	originalFs := fs
	defer func() { fs = originalFs }()

	fs = &mockFileSystem{
		readFileFunc: func(filename string) ([]byte, error) {
			if filename == configPath {
				return []byte(configContent), nil
			}
			return nil, os.ErrNotExist
		},
		statFunc: func(path string) (os.FileInfo, error) {
			return nil, nil
		},
	}

	// Test parsing the config
	config, err := parseConfig(configPath)
	if err != nil {
		t.Errorf("parseConfig() returned an error: %v", err)
	}

	// Verify the parsed config with environment variables expanded
	if config.Name != "EnvTestApp" {
		t.Errorf("Expected Name to be 'EnvTestApp', got '%s'", config.Name)
	}
	if len(config.PreBackupCommands) != 1 || config.PreBackupCommands[0] != "echo env_backup" {
		t.Errorf("PreBackupCommands not parsed correctly, got %v", config.PreBackupCommands)
	}
	if len(config.Files) != 1 || config.Files[0] != ".env_testconfig" {
		t.Errorf("Files not parsed correctly, got %v", config.Files)
	}
}

// Simple mock filesystem for testing
type mockFileSystem struct {
	readFileFunc func(filename string) ([]byte, error)
	statFunc     func(path string) (os.FileInfo, error)
}

// Abs implements interfaces.FileSystem.
func (m *mockFileSystem) Abs(path string) (string, error) {
	panic("unimplemented")
}

// RemoveAll implements interfaces.FileSystem.
func (m *mockFileSystem) RemoveAll(path string) error {
	panic("unimplemented")
}

// WriteFile implements interfaces.FileSystem.
func (m *mockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	panic("unimplemented")
}

func (m *mockFileSystem) ReadFile(filename string) ([]byte, error) {
	return m.readFileFunc(filename)
}

func (m *mockFileSystem) Stat(path string) (os.FileInfo, error) {
	return m.statFunc(path)
}

func (m *mockFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (m *mockFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

func (m *mockFileSystem) Base(path string) string {
	return filepath.Base(path)
}

func (m *mockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

func (m *mockFileSystem) Create(name string) (interfaces.File, error) {
	return nil, nil
}

func (m *mockFileSystem) Open(name string) (interfaces.File, error) {
	return nil, nil
}

func (m *mockFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	return nil, nil
}

func (m *mockFileSystem) EvalSymlinks(path string) (string, error) {
	return path, nil
}
