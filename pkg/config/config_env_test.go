package config

import (
	"SettingsSentry/interfaces"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestExpandEnvVars(t *testing.T) {
	setupTestDependencies()

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
		result := ExpandEnvVars(tc.input)
		if result != tc.expected {
			t.Errorf("ExpandEnvVars(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestValidateConfig(t *testing.T) {
	setupTestDependencies()

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
			err := ValidateConfig(tc.config)
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
	setupTestDependencies()

	os.Setenv("TEST_APP_NAME", "EnvTestApp")
	os.Setenv("TEST_CONFIG_FILE", ".env_testconfig")
	os.Setenv("TEST_BACKUP_CMD", "echo env_backup")
	defer os.Unsetenv("TEST_APP_NAME")
	defer os.Unsetenv("TEST_CONFIG_FILE")
	defer os.Unsetenv("TEST_BACKUP_CMD")

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

	// Filesystem is now managed by setupTestDependencies and util.Fs
	// We still need the mock definition for the adapter, but don't assign to global fs.
	// The mock needs to read the actual temp file for env var expansion to work correctly.
	mockFsImpl := &mockFileSystem{
		readFileFunc: func(filename string) ([]byte, error) {
			// Read the actual temp file, not the static content string
			return os.ReadFile(filename)
		},
		statFunc: func(path string) (os.FileInfo, error) {
			// Use os.Stat for the actual temp file/dir
			return os.Stat(path)
		},
	}

	// Create adapter instance using the local mock implementation
	adapterInstance := &mockFSAdapter{mock: mockFsImpl}
	config, err := ParseConfig(adapterInstance, configPath)
	if err != nil {
		t.Errorf("ParseConfig() returned an error: %v", err)
	}

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

type mockFileSystem struct {
	readFileFunc func(filename string) ([]byte, error)
	statFunc     func(path string) (os.FileInfo, error)
}

type mockIOFSFile struct {
	content []byte
	offset  int64
	info    os.FileInfo // Can be nil if not needed/mocked
}

func (f *mockIOFSFile) Read(p []byte) (int, error) {
	if f.offset >= int64(len(f.content)) {
		return 0, io.EOF
	}
	n := copy(p, f.content[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *mockIOFSFile) Close() error {
	// No-op for mock
	return nil
}

func (f *mockIOFSFile) Stat() (os.FileInfo, error) {
	if f.info == nil {
		// Return a default or error if Stat is crucial and not mocked
		return nil, fmt.Errorf("mock Stat not implemented")
	}
	return f.info, nil
}

func (m *mockFileSystem) Abs(path string) (string, error) {
	panic("unimplemented")
}

func (m *mockFileSystem) RemoveAll(path string) error {
	panic("unimplemented")
}

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

func (m *mockFileSystem) OpenIOFS(name string) (iofs.File, error) {
	// Use readFileFunc to get content for the mock file
	content, err := m.ReadFile(name)
	if err != nil {
		// Propagate error like os.ErrNotExist
		return nil, err
	}
	// Use statFunc to get FileInfo for the mock file's Stat method
	info, _ := m.Stat(name) // Ignore error for simplicity in mock

	return &mockIOFSFile{content: content, info: info}, nil
}

// Open implements interfaces.FileSystem.Open (needed for assignment to global fs)
func (m *mockFileSystem) Open(name string) (interfaces.File, error) {
	// Simple implementation, as it won't be called by parseConfig in this test
	return nil, fmt.Errorf("mock Open (interfaces.File) not implemented/needed for this test")
}

func (m *mockFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	return nil, nil
}

func (m *mockFileSystem) EvalSymlinks(path string) (string, error) {
	return path, nil
}

type mockFSAdapter struct {
	mock *mockFileSystem
}

func (a *mockFSAdapter) Open(name string) (iofs.File, error) {
	// Call the specific method on the embedded mock
	return a.mock.OpenIOFS(name)
}

var _ iofs.FS = (*mockFSAdapter)(nil)
