package backup

import (
	"SettingsSentry/interfaces"
	// "SettingsSentry/logger" // No longer needed directly
	"SettingsSentry/pkg/config"
	"SettingsSentry/pkg/printer"
	"SettingsSentry/pkg/testutil" // Added testutil
	"SettingsSentry/pkg/util"     // Keep util for Fs/AppLogger/DryRun access
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupBackupTestDependencies() {
	// Create the necessary FS implementation for this test context
	testFs := interfaces.NewOsFileSystem()

	// Use the shared helper, passing the OS FS and nil for CmdExecutor
	testLogger := testutil.SetupTestGlobals(testFs, nil)

	// Initialize package-specific dependencies using globals set by the helper
	AppLogger = util.AppLogger
	Fs = util.Fs
	config.Fs = util.Fs               // Initialize config.Fs as well
	config.AppLogger = util.AppLogger // Ensure config.AppLogger is also set
	DryRun = util.DryRun

	// Initialize printer specific to this package's tests
	testPrinter := printer.NewPrinter("Test", testLogger)
	Printer = testPrinter
}

func TestCopyFile(t *testing.T) {
	setupBackupTestDependencies()

	tempDir, err := os.MkdirTemp("", "settingssentry-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tempDir, err)
		}
	}()

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
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tempDir, err)
		}
	}()

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

func TestGetLatestVersionPath_Extended(t *testing.T) {
	setupBackupTestDependencies()

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		versions    []string
		isZip       []bool
		expectError bool
		expectedIdx int
	}{
		{
			name:        "single directory version",
			versions:    []string{"20240101-120000"},
			isZip:       []bool{false},
			expectError: false,
			expectedIdx: 0,
		},
		{
			name:        "multiple directory versions",
			versions:    []string{"20240101-120000", "20240102-120000", "20240103-120000"},
			isZip:       []bool{false, false, false},
			expectError: false,
			expectedIdx: 2, // Latest
		},
		{
			name:        "mixed zip and directory",
			versions:    []string{"20240101-120000", "20240102-120000"},
			isZip:       []bool{false, true},
			expectError: false,
			expectedIdx: 1, // Latest (zip)
		},
		{
			name:        "all zip versions",
			versions:    []string{"20240101-120000", "20240102-120000"},
			isZip:       []bool{true, true},
			expectError: false,
			expectedIdx: 1, // Latest zip
		},
		{
			name:        "no versions",
			versions:    []string{},
			isZip:       []bool{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testBackupDir := filepath.Join(tempDir, tt.name)
			err := os.MkdirAll(testBackupDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create test backup dir: %v", err)
			}

			// Create test versions
			for i, version := range tt.versions {
				if tt.isZip[i] {
					// Create zip file
					zipPath := filepath.Join(testBackupDir, version+".zip")
					f, err := os.Create(zipPath)
					if err != nil {
						t.Fatalf("Failed to create zip file: %v", err)
					}
					_ = f.Close()
				} else {
					// Create directory
					dirPath := filepath.Join(testBackupDir, version)
					err := os.MkdirAll(dirPath, 0755)
					if err != nil {
						t.Fatalf("Failed to create version directory: %v", err)
					}
				}
			}

			path, isZip, err := GetLatestVersionPath(testBackupDir)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				expectedVersion := tt.versions[tt.expectedIdx]
				if tt.isZip[tt.expectedIdx] {
					expectedVersion += ".zip"
				}
				expectedPath := filepath.Join(testBackupDir, expectedVersion)
				if path != expectedPath {
					t.Errorf("Path = %q, want %q", path, expectedPath)
				}
				if isZip != tt.isZip[tt.expectedIdx] {
					t.Errorf("isZip = %v, want %v", isZip, tt.isZip[tt.expectedIdx])
				}
			}
		})
	}
}

func TestGetLatestVersionPath_NonexistentDir(t *testing.T) {
	setupBackupTestDependencies()

	_, _, err := GetLatestVersionPath("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for nonexistent directory")
	}
}

func TestCleanupOldVersions_Extended(t *testing.T) {
	setupBackupTestDependencies()

	tempDir := t.TempDir()

	tests := []struct {
		name           string
		versions       []string
		isZip          []bool
		maxVersions    int
		expectedRemain int
	}{
		{
			name:           "keep 2 of 5",
			versions:       []string{"20240101-120000", "20240102-120000", "20240103-120000", "20240104-120000", "20240105-120000"},
			isZip:          []bool{false, false, false, false, false},
			maxVersions:    2,
			expectedRemain: 2,
		},
		{
			name:           "keep all when maxVersions is higher",
			versions:       []string{"20240101-120000", "20240102-120000"},
			isZip:          []bool{false, false},
			maxVersions:    5,
			expectedRemain: 2,
		},
		{
			name:           "maxVersions is zero",
			versions:       []string{"20240101-120000", "20240102-120000"},
			isZip:          []bool{false, false},
			maxVersions:    0,
			expectedRemain: 2,
		},
		{
			name:           "mixed zip and dir versions",
			versions:       []string{"20240101-120000", "20240102-120000", "20240103-120000", "20240104-120000"},
			isZip:          []bool{false, true, false, true},
			maxVersions:    2,
			expectedRemain: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testBackupDir := filepath.Join(tempDir, tt.name)
			err := os.MkdirAll(testBackupDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create test backup dir: %v", err)
			}

			// Create test versions
			for i, version := range tt.versions {
				if tt.isZip[i] {
					zipPath := filepath.Join(testBackupDir, version+".zip")
					f, err := os.Create(zipPath)
					if err != nil {
						t.Fatalf("Failed to create zip file: %v", err)
					}
					_ = f.Close()
				} else {
					dirPath := filepath.Join(testBackupDir, version)
					err := os.MkdirAll(dirPath, 0755)
					if err != nil {
						t.Fatalf("Failed to create version directory: %v", err)
					}
				}
			}

			err = CleanupOldVersions(testBackupDir, tt.maxVersions)
			if err != nil {
				t.Errorf("CleanupOldVersions failed: %v", err)
			}

			// Count remaining versions
			entries, err := os.ReadDir(testBackupDir)
			if err != nil {
				t.Fatalf("Failed to read backup directory: %v", err)
			}

			remaining := 0
			for _, entry := range entries {
				if entry.IsDir() || strings.HasSuffix(entry.Name(), ".zip") {
					remaining++
				}
			}

			if remaining != tt.expectedRemain {
				t.Errorf("Expected %d versions to remain, got %d", tt.expectedRemain, remaining)
			}
		})
	}
}

func TestCleanupOldVersions_NonexistentDir(t *testing.T) {
	setupBackupTestDependencies()

	// Should not error for nonexistent directory
	err := CleanupOldVersions("/nonexistent/path", 5)
	if err != nil {
		t.Errorf("CleanupOldVersions should handle nonexistent directory gracefully, got: %v", err)
	}
}

func TestCleanupOldVersions_DryRun(t *testing.T) {
	setupBackupTestDependencies()

	tempDir := t.TempDir()
	testBackupDir := filepath.Join(tempDir, "dryrun")
	err := os.MkdirAll(testBackupDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test backup dir: %v", err)
	}

	// Create 3 versions
	versions := []string{"20240101-120000", "20240102-120000", "20240103-120000"}
	for _, version := range versions {
		dirPath := filepath.Join(testBackupDir, version)
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create version directory: %v", err)
		}
	}

	// Enable dry run
	originalDryRun := DryRun
	DryRun = true
	defer func() { DryRun = originalDryRun }()

	err = CleanupOldVersions(testBackupDir, 1)
	if err != nil {
		t.Errorf("CleanupOldVersions failed: %v", err)
	}

	// All versions should still exist in dry run
	entries, err := os.ReadDir(testBackupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("In dry run, all versions should remain. Got %d, want 3", len(entries))
	}
}

func TestCopyFile_Errors(t *testing.T) {
	setupBackupTestDependencies()

	tempDir := t.TempDir()

	t.Run("nonexistent source", func(t *testing.T) {
		src := filepath.Join(tempDir, "nonexistent.txt")
		dst := filepath.Join(tempDir, "dest.txt")

		err := copyFile(src, dst)
		if err == nil {
			t.Error("Expected error for nonexistent source file")
		}
	})

	t.Run("invalid destination directory", func(t *testing.T) {
		src := filepath.Join(tempDir, "source.txt")
		_ = os.WriteFile(src, []byte("test"), 0644)

		// Try to write to a file as if it were a directory
		badDir := filepath.Join(tempDir, "file.txt")
		_ = os.WriteFile(badDir, []byte("block"), 0644)
		dst := filepath.Join(badDir, "dest.txt")

		err := copyFile(src, dst)
		if err == nil {
			t.Error("Expected error when destination directory is invalid")
		}
	})
}

func TestCopyDirectory_Errors(t *testing.T) {
	setupBackupTestDependencies()

	tempDir := t.TempDir()

	t.Run("nonexistent source directory", func(t *testing.T) {
		src := filepath.Join(tempDir, "nonexistent")
		dst := filepath.Join(tempDir, "dest")

		err := copyDirectory(src, dst)
		if err == nil {
			t.Error("Expected error for nonexistent source directory")
		}
	})
}
