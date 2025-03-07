package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPrinter(t *testing.T) {
	printer := NewPrinter("TestApp")
	if printer.printAppName != "\n\033[1mTestApp\033[0m -> " {
		t.Errorf("Expected printAppName to be '\n\033[1mTestApp\033[0m -> ', got '%s'", printer.printAppName)
	}
	if !printer.firstPrint {
		t.Errorf("Expected firstPrint to be true, got false")
	}
}

func TestPrinterReset(t *testing.T) {
	printer := NewPrinter("TestApp")
	printer.firstPrint = false
	printer.Reset()
	if !printer.firstPrint {
		t.Errorf("Expected firstPrint to be true after Reset(), got false")
	}
}

func TestGetHomeDirectory(t *testing.T) {
	homeDir, err := getHomeDirectory()
	if err != nil {
		t.Errorf("getHomeDirectory() returned an error: %v", err)
	}
	if homeDir == "" {
		t.Errorf("getHomeDirectory() returned an empty string")
	}
	// Check if the directory exists
	_, err = os.Stat(homeDir)
	if os.IsNotExist(err) {
		t.Errorf("Home directory %s does not exist", homeDir)
	}
}

func TestParseConfig(t *testing.T) {
	// Create a temporary config file
	tempDir, err := os.MkdirTemp("", "settingssentry-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configContent := `[application]
name = TestApp

[backup_commands]
test backup command

[restore_commands]
test restore command

[configuration_files]
.testconfig
`
	configPath := filepath.Join(tempDir, "test.cfg")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test parsing the config
	config, err := parseConfig(configPath)
	if err != nil {
		t.Errorf("parseConfig() returned an error: %v", err)
	}

	// Verify the parsed config
	if config.Name != "TestApp" {
		t.Errorf("Expected Name to be 'TestApp', got '%s'", config.Name)
	}
	if len(config.PreBackupCommands) != 1 || config.PreBackupCommands[0] != "test backup command" {
		t.Errorf("PreBackupCommands not parsed correctly, got %v", config.PreBackupCommands)
	}
	if len(config.PreRestoreCommands) != 1 || config.PreRestoreCommands[0] != "test restore command" {
		t.Errorf("PreRestoreCommands not parsed correctly, got %v", config.PreRestoreCommands)
	}
	if len(config.Files) != 1 || config.Files[0] != ".testconfig" {
		t.Errorf("Files not parsed correctly, got %v", config.Files)
	}
}

func TestCopyFile(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "settingssentry-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a source file
	srcContent := "test content"
	srcPath := filepath.Join(tempDir, "source.txt")
	err = os.WriteFile(srcPath, []byte(srcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Define destination path
	dstPath := filepath.Join(tempDir, "destination.txt")

	// Test copying the file
	err = copyFile(srcPath, dstPath)
	if err != nil {
		t.Errorf("copyFile() returned an error: %v", err)
	}

	// Verify the destination file
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Errorf("Failed to read destination file: %v", err)
	}
	if string(dstContent) != srcContent {
		t.Errorf("Destination file content does not match source. Expected '%s', got '%s'", srcContent, string(dstContent))
	}
}

func TestCopyDirectory(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "settingssentry-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a source directory structure
	srcDir := filepath.Join(tempDir, "src")
	err = os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory structure: %v", err)
	}

	// Create some files in the source directory
	files := map[string]string{
		"file1.txt":        "content of file 1",
		"subdir/file2.txt": "content of file 2",
	}

	for path, content := range files {
		filePath := filepath.Join(srcDir, path)
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Define destination directory
	dstDir := filepath.Join(tempDir, "dst")

	// Test copying the directory
	err = copyDirectory(srcDir, dstDir)
	if err != nil {
		t.Errorf("copyDirectory() returned an error: %v", err)
	}

	// Verify the destination directory structure and file contents
	for path, expectedContent := range files {
		filePath := filepath.Join(dstDir, path)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", path, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("File %s content does not match. Expected '%s', got '%s'", path, expectedContent, string(content))
		}
	}
}

// TestExecuteCommandLine is a simple test for the executeCommandLine function
// It only tests a simple command that should succeed on macOS
func TestExecuteCommandLine(t *testing.T) {
	// Initialize the global printer before calling executeCommandLine
	printer = NewPrinter("TestApp")

	result := executeCommandLine("echo test")
	if !result {
		t.Errorf("executeCommandLine() returned false for a simple echo command")
	}
}

func TestGetXDGConfigHome(t *testing.T) {
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", "/test/home")
	defer os.Setenv("HOME", originalHome)

	// Test case 1: XDG_CONFIG_HOME is set within home directory
	os.Setenv("XDG_CONFIG_HOME", "/test/home/xdg")
	xdgDir, err := getXDGConfigHome()
	if err != nil {
		t.Errorf("getXDGConfigHome() error = %v, want nil", err)
	}
	if xdgDir != "/test/home/xdg" {
		t.Errorf("getXDGConfigHome() got = %v, want /test/home/xdg", xdgDir)
	}
	os.Unsetenv("XDG_CONFIG_HOME")

	// Test case 2: XDG_CONFIG_HOME is not set
	xdgDir, err = getXDGConfigHome()
	if err != nil {
		t.Errorf("getXDGConfigHome() error = %v, want nil", err)
	}
	if xdgDir != "/test/home/.config" {
		t.Errorf("getXDGConfigHome() got = %v, want /test/home/.config", xdgDir)
	}

	// Test case 3: XDG_CONFIG_HOME is outside the home directory
	os.Setenv("XDG_CONFIG_HOME", "/outside/xdg")
	_, err = getXDGConfigHome()
	if err == nil {
		t.Errorf("getXDGConfigHome() error = %v, want error", err)
	}
	os.Unsetenv("XDG_CONFIG_HOME")
}
