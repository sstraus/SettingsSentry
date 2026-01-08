package backup

import (
	"SettingsSentry/interfaces"
	// "SettingsSentry/logger" // No longer needed directly
	"SettingsSentry/pkg/command"
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

// TestProcessConfiguration_CommandExecution tests that commands are only executed
// when the commands flag is true (security feature)
func TestProcessConfiguration_CommandExecution(t *testing.T) {
	// Setup with a REAL command executor so commands can actually run
	testFs := interfaces.NewOsFileSystem()
	realCmdExecutor := interfaces.NewOsCommandExecutor()
	testLogger := testutil.SetupTestGlobals(testFs, realCmdExecutor)

	// Initialize package-specific dependencies
	AppLogger = util.AppLogger
	Fs = util.Fs
	config.Fs = util.Fs
	config.AppLogger = util.AppLogger
	DryRun = util.DryRun

	// Initialize command package globals for command execution
	command.CmdExecutor = realCmdExecutor
	command.AppLogger = util.AppLogger

	testPrinter := printer.NewPrinter("Test", testLogger)
	Printer = testPrinter
	command.Printer = testPrinter

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	backupDir := filepath.Join(tempDir, "backups")
	mockHomeDir := filepath.Join(tempDir, "mockHome")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}
	if err := os.MkdirAll(mockHomeDir, 0755); err != nil {
		t.Fatalf("Failed to create mock home: %v", err)
	}

	// Mock GetHomeDirectory
	originalGetHomeDir := config.GetHomeDirectory
	config.GetHomeDirectory = func() (string, error) {
		return mockHomeDir, nil
	}
	defer func() { config.GetHomeDirectory = originalGetHomeDir }()

	// Create a marker file that will be deleted by the command
	markerFile := filepath.Join(tempDir, "command_executed.txt")
	if err := os.WriteFile(markerFile, []byte("marker"), 0644); err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	// Create a config file with a pre-backup command that creates a success marker
	successMarker := filepath.Join(tempDir, "command_success.txt")
	cfgContent := `[application]
name = CommandTest

[pre_backup_commands]
echo "command_executed" > ` + successMarker + `

[configuration_files]
~/test.txt`

	cfgPath := filepath.Join(configDir, "commandtest.cfg")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create dummy source file
	sourceFile := filepath.Join(mockHomeDir, "test.txt")
	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	t.Run("commands disabled by default", func(t *testing.T) {
		// Remove success marker if it exists
		os.Remove(successMarker)

		// Run backup with commands=false (default, secure behavior)
		ProcessConfiguration(configDir, backupDir, nil, true, false, 1, false, "")

		// Success marker should NOT exist (command was NOT executed)
		if _, err := os.Stat(successMarker); err == nil {
			t.Error("SECURITY FAIL: Command executed when commands flag was false!")
			t.Error("Commands should be disabled by default for security")
		} else if os.IsNotExist(err) {
			t.Log("PASS: Command not executed when flag is false (secure default)")
		}
	})

	t.Run("commands enabled with flag", func(t *testing.T) {
		// Remove success marker if it exists
		os.Remove(successMarker)

		// Run backup with commands=true
		ProcessConfiguration(configDir, backupDir, nil, true, true, 1, false, "")

		// Success marker SHOULD exist (command WAS executed)
		if _, err := os.Stat(successMarker); err == nil {
			t.Log("PASS: Command executed when flag is true")
		} else if os.IsNotExist(err) {
			t.Error("Command was not executed when commands flag was true")
			t.Logf("Success marker does not exist at: %s", successMarker)
		}
	})
}

// TestBackupContext_CommandExecutionControl tests the BackupContext approach
func TestBackupContext_CommandExecutionControl(t *testing.T) {
	setupBackupTestDependencies()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	backupDir := filepath.Join(tempDir, "backups")
	mockHomeDir := filepath.Join(tempDir, "mockHome")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}
	if err := os.MkdirAll(mockHomeDir, 0755); err != nil {
		t.Fatalf("Failed to create mock home: %v", err)
	}

	// Mock GetHomeDirectory
	originalGetHomeDir := config.GetHomeDirectory
	config.GetHomeDirectory = func() (string, error) {
		return mockHomeDir, nil
	}
	defer func() { config.GetHomeDirectory = originalGetHomeDir }()

	t.Run("context respects commands flag", func(t *testing.T) {
		// Test with commands=false
		ctx, err := NewBackupContext(configDir, backupDir, nil, true, false, 1, false, "")
		if err != nil {
			t.Fatalf("Failed to create context: %v", err)
		}
		if ctx.Commands {
			t.Error("BackupContext.Commands should be false when flag is false")
		}

		// Test with commands=true
		ctx2, err := NewBackupContext(configDir, backupDir, nil, true, true, 1, false, "")
		if err != nil {
			t.Fatalf("Failed to create context: %v", err)
		}
		if !ctx2.Commands {
			t.Error("BackupContext.Commands should be true when flag is true")
		}
	})
}

// TestBackupConfigNamePathTraversal tests that malicious config names with path traversal
// sequences cannot write backups outside the intended backup directory
func TestBackupConfigNamePathTraversal(t *testing.T) {
	setupBackupTestDependencies()

	// Create temp directories
	tempDir, err := os.MkdirTemp("", "settingssentry-pathtraversal-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tempDir, err)
		}
	}()

	configDir := filepath.Join(tempDir, "configs")
	backupDir := filepath.Join(tempDir, "backups")
	homeDir := filepath.Join(tempDir, "home")
	testFile := filepath.Join(homeDir, "test.conf")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatalf("Failed to create home dir: %v", err)
	}

	// Create a test file to backup
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		configName  string
		description string
	}{
		{
			name:        "path traversal with parent directories",
			configName:  "../../../malicious",
			description: "Attacker tries to write backup outside backup folder using ../../../",
		},
		{
			name:        "path traversal with absolute-like path",
			configName:  "../../etc/shadow",
			description: "Attacker tries to overwrite system files",
		},
		{
			name:        "legitimate config name",
			configName:  "myapp",
			description: "Normal config name should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create malicious config file
			configContent := `[Application]
name=` + tt.configName + `

[Files]
~/test.conf
`
			configPath := filepath.Join(configDir, "malicious.cfg")
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}
			defer os.Remove(configPath)

			// Mock GetHomeDirectory
			originalGetHomeDir := config.GetHomeDirectory
			config.GetHomeDirectory = func() (string, error) {
				return homeDir, nil
			}
			defer func() { config.GetHomeDirectory = originalGetHomeDir }()

			// Create backup context
			ctx, err := NewBackupContext(configDir, backupDir, nil, true, false, 0, false, "")
			if err != nil {
				t.Fatalf("Failed to create backup context: %v", err)
			}

			// Setup backup directory
			if err := ctx.SetupBackupDirectory(); err != nil {
				t.Fatalf("Failed to setup backup directory: %v", err)
			}

			// Load config files
			currentFS, files, err := ctx.LoadConfigFiles()
			if err != nil {
				t.Fatalf("Failed to load config files: %v", err)
			}

			// Process the config file
			for _, file := range files {
				if !strings.HasSuffix(file.Name(), ".cfg") {
					continue
				}

				cfg, err := config.ParseConfig(currentFS, file.Name())
				if err != nil {
					t.Logf("Failed to parse config: %v", err)
					continue
				}

				// Sanitize config name to prevent path traversal attacks
				// This simulates what ProcessConfiguration does in production code
				sanitizedName := sanitizeConfigName(cfg.Name)
				t.Logf("Original cfg.Name: %s, Sanitized: %s", cfg.Name, sanitizedName)

				// Process each file in the config
				for _, configFile := range cfg.Files {
					configFile = ctx.ResolveConfigFilePath(configFile)

					var targetPath string
					if ctx.ZipBackup {
						targetPath = Fs.Join(ctx.StagingDir, sanitizedName, Fs.Base(configFile))
					} else {
						targetPath = Fs.Join(ctx.BackupFolder, ctx.Timestamp, sanitizedName, Fs.Base(configFile))
					}

					// Security check: ensure targetPath is within backupDir
					absBackupDir, err := filepath.Abs(backupDir)
					if err != nil {
						t.Fatalf("Failed to get absolute backup dir: %v", err)
					}

					absTargetPath, err := filepath.Abs(targetPath)
					if err != nil {
						t.Fatalf("Failed to get absolute target path: %v", err)
					}

					relPath, err := filepath.Rel(absBackupDir, absTargetPath)
					if err != nil || strings.HasPrefix(relPath, "..") {
						t.Errorf("%s: Backup path escapes backup directory!\n  Config name: %s\n  Target path: %s\n  Backup dir: %s\n  Relative: %s",
							tt.description, cfg.Name, absTargetPath, absBackupDir, relPath)
					} else {
						t.Logf("✓ Backup path safely within backup directory: %s", relPath)
					}
				}
			}
		})
	}
}

// TestProcessConfiguration_PartialBackupFailure tests error aggregation
func TestProcessConfiguration_PartialBackupFailure(t *testing.T) {
	tempDir, backupDir := setup

BackupTestDirs(t)
	defer os.RemoveAll(tempDir)
	defer os.RemoveAll(backupDir)

	// Create config with multiple files, some will fail
	configDir := filepath.Join(tempDir, "configs")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a config that references both existing and non-accessible files
	configContent := `[application]
name = TestPartialFailure

[configuration_files]
good_file.txt
/root/inaccessible.txt
another_good.txt
`
	configPath := filepath.Join(configDir, "partial.cfg")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create the good files
	goodFile1 := filepath.Join(tempDir, "good_file.txt")
	goodFile2 := filepath.Join(tempDir, "another_good.txt")
	os.WriteFile(goodFile1, []byte("content1"), 0644)
	os.WriteFile(goodFile2, []byte("content2"), 0644)

	// Run backup - should aggregate errors
	err := ProcessConfiguration(configDir, backupDir, []string{}, true, false, 1, false, "")
	
	// Should return error indicating partial failure
	if err == nil {
		t.Error("ProcessConfiguration should return error for partial backup failure")
	}
	
	// Error message should mention failed files
	errMsg := err.Error()
	if !strings.Contains(errMsg, "failed") && !strings.Contains(errMsg, "error") {
		t.Errorf("Error message should indicate failures, got: %s", errMsg)
	}

	// Good files should still be backed up
	timestamp := getLatestTimestamp(t, backupDir)
	goodBackup1 := filepath.Join(backupDir, timestamp, "TestPartialFailure", "good_file.txt")
	goodBackup2 := filepath.Join(backupDir, timestamp, "TestPartialFailure", "another_good.txt")
	
	if _, err := os.Stat(goodBackup1); os.IsNotExist(err) {
		t.Error("Good file 1 should be backed up despite other failures")
	}
	if _, err := os.Stat(goodBackup2); os.IsNotExist(err) {
		t.Error("Good file 2 should be backed up despite other failures")
	}
}

func getLatestTimestamp(t *testing.T, backupDir string) string {
	entries, err := os.ReadDir(backupDir)
	if err != nil || len(entries) == 0 {
		t.Fatal("No backup directories found")
	}
	return entries[len(entries)-1].Name()
}
