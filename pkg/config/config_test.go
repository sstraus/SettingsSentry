package config

import (
	"SettingsSentry/interfaces"
	// "SettingsSentry/logger" // No longer needed directly
	"SettingsSentry/pkg/testutil" // Added testutil
	"SettingsSentry/pkg/util"     // Keep util for Fs/AppLogger access
	"os"
	"path/filepath"
	"strings"
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

func TestGetXDGConfigHome(t *testing.T) {
	setupTestDependencies()

	// Save original HOME env var
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	tests := []struct {
		name           string
		homeValue      string
		xdgValue       string
		expectError    bool
		expectedSuffix string
	}{
		{
			name:           "XDG_CONFIG_HOME set",
			homeValue:      "/home/testuser",
			xdgValue:       "/home/testuser/.config",
			expectError:    false,
			expectedSuffix: ".config",
		},
		{
			name:           "XDG_CONFIG_HOME not set, use default",
			homeValue:      "/home/testuser",
			xdgValue:       "",
			expectError:    false,
			expectedSuffix: ".config",
		},
		{
			name:        "HOME not set",
			homeValue:   "",
			xdgValue:    "",
			expectError: true,
		},
		{
			name:        "XDG_CONFIG_HOME outside home",
			homeValue:   "/home/testuser",
			xdgValue:    "/etc/config",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("HOME", tt.homeValue)
			if tt.xdgValue != "" {
				os.Setenv("XDG_CONFIG_HOME", tt.xdgValue)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}

			result, err := GetXDGConfigHome()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.expectedSuffix != "" && !strings.HasSuffix(result, tt.expectedSuffix) {
					t.Errorf("Expected result to end with %q, got %q", tt.expectedSuffix, result)
				}
			}
		})
	}

	// Cleanup
	os.Unsetenv("XDG_CONFIG_HOME")
}

func TestValidateConfig_Additional(t *testing.T) {
	setupTestDependencies()

	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config additional test",
			config: Config{
				Name:  "TestApp",
				Files: []string{".testconfig"},
			},
			expectError: false,
		},
		{
			name: "missing name additional",
			config: Config{
				Name:  "",
				Files: []string{".testconfig"},
			},
			expectError: true,
			errorMsg:    "application name is required",
		},
		{
			name: "no files additional",
			config: Config{
				Name:  "TestApp",
				Files: []string{},
			},
			expectError: true,
			errorMsg:    "at least one configuration file",
		},
		{
			name: "empty file path additional",
			config: Config{
				Name:  "TestApp",
				Files: []string{""},
			},
			expectError: true,
			errorMsg:    "empty configuration file path",
		},
		{
			name: "file path with whitespace only additional",
			config: Config{
				Name:  "TestApp",
				Files: []string{"   "},
			},
			expectError: true,
			errorMsg:    "empty configuration file path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message %q does not contain %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateConfig_TildeExpansion(t *testing.T) {
	setupTestDependencies()

	tests := []struct {
		name        string
		file        string
		expectError bool
	}{
		{
			name:        "tilde in path",
			file:        "~/.testconfig",
			expectError: false,
		},
		{
			name:        "tilde with glob",
			file:        "~/.config/*",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Name:  "TestApp",
				Files: []string{tt.file},
			}

			err := ValidateConfig(config)
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestExpandEnvVars_Additional(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envKey   string
		envValue string
		expected string
	}{
		{
			name:     "simple env var additional",
			input:    "$HOME/.config",
			envKey:   "HOME",
			envValue: "/home/user",
			expected: "/home/user/.config",
		},
		{
			name:     "env var with braces additional",
			input:    "${HOME}/.config",
			envKey:   "HOME",
			envValue: "/home/user",
			expected: "/home/user/.config",
		},
		{
			name:     "no env var additional",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "undefined env var additional",
			input:    "$UNDEFINED_VAR/path",
			expected: "/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			result := ExpandEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandEnvVars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseConfig_Sections(t *testing.T) {
	setupTestDependencies()

	// Set HOME for tests that need it
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		os.Setenv("HOME", "/tmp/testhome")
	}
	defer func() {
		if originalHome == "" {
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", originalHome)
		}
	}()

	tempDir := t.TempDir()

	tests := []struct {
		name           string
		configContent  string
		expectedName   string
		expectedFiles  int
		expectedPreBkp int
		expectedPostBkp int
		expectedPreRst  int
		expectedPostRst int
	}{
		{
			name: "all sections",
			configContent: `[application]
name = TestApp

[configuration_files]
.config1
.config2

[pre_backup_commands]
echo backup1
echo backup2

[post_backup_commands]
echo after backup

[pre_restore_commands]
echo restore1

[post_restore_commands]
echo after restore
`,
			expectedName:    "TestApp",
			expectedFiles:   2,
			expectedPreBkp:  2,
			expectedPostBkp: 1,
			expectedPreRst:  1,
			expectedPostRst: 1,
		},
		{
			name: "xdg_configuration_files section",
			configContent: `[application]
name = XDGApp

[xdg_configuration_files]
app/config.yml
`,
			expectedName:  "XDGApp",
			expectedFiles: 1,
		},
		{
			name: "legacy backup section",
			configContent: `[application]
name = LegacyApp

[files]
.oldconfig

[backup]
echo backup command
`,
			expectedName:   "LegacyApp",
			expectedFiles:  1,
			expectedPreBkp: 1,
		},
		{
			name: "legacy restore section",
			configContent: `[application]
name = RestoreApp

[files]
.restoreconfig

[restore]
echo restore command
`,
			expectedName:   "RestoreApp",
			expectedFiles:  1,
			expectedPreRst: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tempDir, "test.cfg")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}

			testFS := os.DirFS(tempDir)
			config, err := ParseConfig(testFS, "test.cfg")
			if err != nil {
				t.Fatalf("ParseConfig failed: %v", err)
			}

			if config.Name != tt.expectedName {
				t.Errorf("Name = %q, want %q", config.Name, tt.expectedName)
			}
			if len(config.Files) != tt.expectedFiles {
				t.Errorf("Files count = %d, want %d", len(config.Files), tt.expectedFiles)
			}
			if len(config.PreBackupCommands) != tt.expectedPreBkp {
				t.Errorf("PreBackupCommands count = %d, want %d", len(config.PreBackupCommands), tt.expectedPreBkp)
			}
			if len(config.PostBackupCommands) != tt.expectedPostBkp {
				t.Errorf("PostBackupCommands count = %d, want %d", len(config.PostBackupCommands), tt.expectedPostBkp)
			}
			if len(config.PreRestoreCommands) != tt.expectedPreRst {
				t.Errorf("PreRestoreCommands count = %d, want %d", len(config.PreRestoreCommands), tt.expectedPreRst)
			}
			if len(config.PostRestoreCommands) != tt.expectedPostRst {
				t.Errorf("PostRestoreCommands count = %d, want %d", len(config.PostRestoreCommands), tt.expectedPostRst)
			}
		})
	}
}

func TestParseConfig_XDGPaths(t *testing.T) {
	setupTestDependencies()

	// Set HOME for XDG tests
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		os.Setenv("HOME", "/tmp/testhome")
	}
	defer func() {
		if originalHome == "" {
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", originalHome)
		}
	}()

	tempDir := t.TempDir()

	configContent := `[application]
name = XDGTest

[xdg_configuration_files]
myapp/config.yml
`
	configPath := filepath.Join(tempDir, "xdg.cfg")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	testFS := os.DirFS(tempDir)
	config, err := ParseConfig(testFS, "xdg.cfg")
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if len(config.Files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(config.Files))
	}

	// File should be relative to home, not absolute
	if strings.HasPrefix(config.Files[0], "/") {
		t.Errorf("XDG file path should not be absolute: %s", config.Files[0])
	}
}

func TestParseConfig_InvalidXDGPath(t *testing.T) {
	setupTestDependencies()

	tempDir := t.TempDir()

	configContent := `[application]
name = BadXDG

[xdg_configuration_files]
/absolute/path
`
	configPath := filepath.Join(tempDir, "bad.cfg")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	testFS := os.DirFS(tempDir)
	_, err = ParseConfig(testFS, "bad.cfg")
	if err == nil {
		t.Error("Expected error for absolute path in xdg_configuration_files")
	}
	if !strings.Contains(err.Error(), "absolute path") {
		t.Errorf("Error should mention absolute path, got: %v", err)
	}
}

func TestGetHomeDirectory(t *testing.T) {
	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	t.Run("HOME set", func(t *testing.T) {
		os.Setenv("HOME", "/home/testuser")
		home, err := GetHomeDirectory()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if home != "/home/testuser" {
			t.Errorf("Expected /home/testuser, got %s", home)
		}
	})

	t.Run("HOME not set", func(t *testing.T) {
		os.Unsetenv("HOME")
		_, err := GetHomeDirectory()
		if err == nil {
			t.Error("Expected error when HOME is not set")
		}
	})
}

func TestGetICloudFolderLocation(t *testing.T) {
	setupTestDependencies()

	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	t.Run("iCloud folder exists", func(t *testing.T) {
		// Create temp directory structure
		tempDir := t.TempDir()
		os.Setenv("HOME", tempDir)

		icloudPath := filepath.Join(tempDir, "Library", "Mobile Documents", "com~apple~CloudDocs")
		err := os.MkdirAll(icloudPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create iCloud path: %v", err)
		}

		result, err := GetICloudFolderLocation()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result == "" {
			t.Error("Expected non-empty result")
		}
	})

	t.Run("iCloud folder not found", func(t *testing.T) {
		tempDir := t.TempDir()
		os.Setenv("HOME", tempDir)

		_, err := GetICloudFolderLocation()
		if err == nil {
			t.Error("Expected error when iCloud folder doesn't exist")
		}
	})

	t.Run("Fs not initialized", func(t *testing.T) {
		originalFs := Fs
		Fs = nil
		defer func() { Fs = originalFs }()

		_, err := GetICloudFolderLocation()
		if err == nil {
			t.Error("Expected error when Fs is nil")
		}
	})
}
