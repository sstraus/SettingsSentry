package backup

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"SettingsSentry/pkg/config"
	"SettingsSentry/pkg/printer"
	"SettingsSentry/pkg/util"
	"os"
	"path/filepath"
	"testing"
)

func setupBackupTestDependencies() {
	testLogger, _ := logger.NewLogger("")
	testFs := interfaces.NewOsFileSystem()
	testPrinter := printer.NewPrinter("Test", testLogger)
	util.InitGlobals(testLogger, testFs, nil, false, "")
	// Initialize Fs and AppLogger in both backup and config packages
	AppLogger = util.AppLogger
	Fs = util.Fs
	config.Fs = util.Fs               // Initialize config.Fs as well
	config.AppLogger = util.AppLogger // Ensure config.AppLogger is also set
	DryRun = util.DryRun
	Printer = testPrinter
}

func TestCopyFile(t *testing.T) {
	setupBackupTestDependencies()

	tempDir, err := os.MkdirTemp("", "settingssentry-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	srcContent := "test content"
	srcPath := filepath.Join(tempDir, "source.txt")
	err = os.WriteFile(srcPath, []byte(srcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	dstPath := filepath.Join(tempDir, "destination.txt")

	err = copyFile(srcPath, dstPath)
	if err != nil {
		t.Errorf("copyFile() returned an error: %v", err)
	}

	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Errorf("Failed to read destination file: %v", err)
	}
	if string(dstContent) != srcContent {
		t.Errorf("Destination file content does not match source. Expected '%s', got '%s'", srcContent, string(dstContent))
	}
}

func TestCopyDirectory(t *testing.T) {
	setupBackupTestDependencies()

	tempDir, err := os.MkdirTemp("", "settingssentry-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	srcDir := filepath.Join(tempDir, "src")
	err = os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory structure: %v", err)
	}

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

	dstDir := filepath.Join(tempDir, "dst")

	err = copyDirectory(srcDir, dstDir)
	if err != nil {
		t.Errorf("copyDirectory() returned an error: %v", err)
	}

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
