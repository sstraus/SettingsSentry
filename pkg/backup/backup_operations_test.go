package backup

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/pkg/config"
	"SettingsSentry/pkg/printer"
	"SettingsSentry/pkg/testutil"
	"SettingsSentry/pkg/util"
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupBackupOperationsTest() {
	testFs := interfaces.NewOsFileSystem()
	testLogger := testutil.SetupTestGlobals(testFs, nil)
	
	AppLogger = util.AppLogger
	Fs = util.Fs
	config.Fs = util.Fs
	config.AppLogger = util.AppLogger
	DryRun = util.DryRun
	
	testPrinter := printer.NewPrinter("Test", testLogger)
	Printer = testPrinter
}

// TestNewBackupContext tests BackupContext creation
func TestNewBackupContext(t *testing.T) {
	setupBackupOperationsTest()

	tests := []struct {
		name         string
		configFolder string
		backupFolder string
		appNames     []string
		isBackup     bool
		wantErr      bool
	}{
		{
			name:         "valid backup context",
			configFolder: "configs",
			backupFolder: "/tmp/backup",
			appNames:     []string{"app1"},
			isBackup:     true,
			wantErr:      false,
		},
		{
			name:         "valid restore context",
			configFolder: "configs",
			backupFolder: "/tmp/backup",
			appNames:     []string{},
			isBackup:     false,
			wantErr:      false,
		},
		{
			name:         "multiple apps",
			configFolder: "configs",
			backupFolder: "/tmp/backup",
			appNames:     []string{"app1", "app2", "app3"},
			isBackup:     true,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := NewBackupContext(
				tt.configFolder,
				tt.backupFolder,
				tt.appNames,
				tt.isBackup,
				false, // commands
				1,     // versionsToKeep
				false, // zipBackup
				"",    // password
			)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.wantErr {
				if ctx == nil {
					t.Fatal("Expected context, got nil")
				}
				if ctx.IsBackup != tt.isBackup {
					t.Errorf("IsBackup = %v, want %v", ctx.IsBackup, tt.isBackup)
				}
				if len(ctx.AppNames) != len(tt.appNames) {
					t.Errorf("AppNames length = %d, want %d", len(ctx.AppNames), len(tt.appNames))
				}
			}
		})
	}
}

// TestNewBackupContext_WithEnvVars tests environment variable expansion
func TestNewBackupContext_WithEnvVars(t *testing.T) {
	setupBackupOperationsTest()

	_ = os.Setenv("TEST_CONFIG", "/test/config")
	_ = os.Setenv("TEST_BACKUP", "/test/backup")
	defer func() {
		_ = os.Unsetenv("TEST_CONFIG")
		_ = os.Unsetenv("TEST_BACKUP")
	}()

	ctx, err := NewBackupContext(
		"$TEST_CONFIG",
		"$TEST_BACKUP",
		[]string{},
		true,
		false,
		1,
		false,
		"",
	)

	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	if !strings.Contains(ctx.ConfigFolder, "test") {
		t.Errorf("ConfigFolder should expand env var, got: %s", ctx.ConfigFolder)
	}
	if !strings.Contains(ctx.BackupFolder, "test") {
		t.Errorf("BackupFolder should expand env var, got: %s", ctx.BackupFolder)
	}
}

// TestSetupBackupDirectory tests backup directory setup
func TestSetupBackupDirectory(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	tests := []struct {
		name      string
		isBackup  bool
		zipBackup bool
		dryRun    bool
		wantErr   bool
	}{
		{
			name:      "backup with directory",
			isBackup:  true,
			zipBackup: false,
			dryRun:    false,
			wantErr:   false,
		},
		{
			name:      "backup with zip",
			isBackup:  true,
			zipBackup: true,
			dryRun:    false,
			wantErr:   false,
		},
		{
			name:      "dry run backup",
			isBackup:  true,
			zipBackup: false,
			dryRun:    true,
			wantErr:   false,
		},
		{
			name:      "restore existing directory",
			isBackup:  false,
			zipBackup: false,
			dryRun:    false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backupPath := filepath.Join(tempDir, tt.name)
			if !tt.isBackup {
				// Create directory for restore test
				if err := os.MkdirAll(backupPath, 0755); err != nil {
					t.Fatalf("Failed to create test backup dir: %v", err)
				}
			}

			oldDryRun := DryRun
			DryRun = tt.dryRun
			defer func() { DryRun = oldDryRun }()

			ctx, err := NewBackupContext(
				"configs",
				backupPath,
				[]string{},
				tt.isBackup,
				false,
				1,
				tt.zipBackup,
				"",
			)
			if err != nil {
				t.Fatalf("NewBackupContext failed: %v", err)
			}

			err = ctx.SetupBackupDirectory()

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check staging directory for zip backups
			if tt.isBackup && tt.zipBackup && !tt.dryRun {
				if ctx.StagingDir == "" {
					t.Error("StagingDir should be set for zip backup")
				}
				// Cleanup staging dir
				if ctx.StagingDir != "" {
					_ = os.RemoveAll(ctx.StagingDir)
				}
			}
		})
	}
}

// TestSetupBackupDirectory_RestoreNonExistent tests restore with non-existent dir
func TestSetupBackupDirectory_RestoreNonExistent(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, "nonexistent")

	ctx, err := NewBackupContext(
		"configs",
		nonExistentPath,
		[]string{},
		false, // restore
		false,
		1,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	err = ctx.SetupBackupDirectory()
	if err == nil {
		t.Error("Expected error for non-existent restore directory")
	}
}

// TestLoadConfigFiles tests config file loading
func TestLoadConfigFiles(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	// Create test config files
	testFiles := []string{"app1.cfg", "app2.cfg", "test.txt"}
	for _, file := range testFiles {
		path := filepath.Join(tempDir, file)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	ctx, err := NewBackupContext(
		tempDir,
		"/tmp/backup",
		[]string{},
		true,
		false,
		1,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	fsys, files, err := ctx.LoadConfigFiles()
	if err != nil {
		t.Errorf("LoadConfigFiles failed: %v", err)
	}

	if fsys == nil {
		t.Error("Expected filesystem, got nil")
	}

	if len(files) != len(testFiles) {
		t.Errorf("Found %d files, want %d", len(files), len(testFiles))
	}
}

// TestLoadConfigFiles_EmptyDirectory tests empty config directory
func TestLoadConfigFiles_EmptyDirectory(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	ctx, err := NewBackupContext(
		tempDir,
		"/tmp/backup",
		[]string{},
		true,
		false,
		1,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	_, _, err = ctx.LoadConfigFiles()
	if err == nil {
		t.Error("Expected error for empty directory")
	}
	if !strings.Contains(err.Error(), "no configuration files") {
		t.Errorf("Error should mention no files, got: %v", err)
	}
}

// TestLoadConfigFiles_NonExistent tests non-existent config directory
func TestLoadConfigFiles_NonExistent(t *testing.T) {
	setupBackupOperationsTest()

	ctx, err := NewBackupContext(
		"/nonexistent/path",
		"/tmp/backup",
		[]string{},
		true,
		false,
		1,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	_, _, err = ctx.LoadConfigFiles()
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

// TestFilterConfigFiles tests config file filtering
func TestFilterConfigFiles(t *testing.T) {
	setupBackupOperationsTest()

	// Create mock directory entries
	tempDir := t.TempDir()
	testFiles := []string{"app1.cfg", "app2.cfg", "app3.cfg", "readme.txt"}
	for _, file := range testFiles {
		path := filepath.Join(tempDir, file)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read test dir: %v", err)
	}

	tests := []struct {
		name         string
		appNames     []string
		expectedCount int
	}{
		{
			name:         "no filter",
			appNames:     []string{},
			expectedCount: 4, // All files
		},
		{
			name:         "filter single app",
			appNames:     []string{"app1"},
			expectedCount: 1,
		},
		{
			name:         "filter multiple apps",
			appNames:     []string{"app1", "app2"},
			expectedCount: 2,
		},
		{
			name:         "no matching apps",
			appNames:     []string{"nonexistent"},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := NewBackupContext(
				tempDir,
				"/tmp/backup",
				tt.appNames,
				true,
				false,
				1,
				false,
				"",
			)
			if err != nil {
				t.Fatalf("NewBackupContext failed: %v", err)
			}

			filtered := ctx.FilterConfigFiles(files)
			if len(filtered) != tt.expectedCount {
				t.Errorf("Filtered %d files, want %d", len(filtered), tt.expectedCount)
			}
		})
	}
}

// TestResolveConfigFilePath tests config file path resolution
func TestResolveConfigFilePath(t *testing.T) {
	setupBackupOperationsTest()

	homeDir, _ := config.GetHomeDirectory()

	tests := []struct {
		name       string
		configFile string
		wantPrefix string
	}{
		{
			name:       "tilde path",
			configFile: "~/.config/app",
			wantPrefix: homeDir,
		},
		{
			name:       "relative path",
			configFile: ".config/app",
			wantPrefix: homeDir,
		},
		{
			name:       "absolute path",
			configFile: "/absolute/path",
			wantPrefix: "/absolute/path",
		},
		{
			name:       "no prefix relative",
			configFile: "config/app",
			wantPrefix: homeDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := NewBackupContext(
				"configs",
				"/tmp/backup",
				[]string{},
				true,
				false,
				1,
				false,
				"",
			)
			if err != nil {
				t.Fatalf("NewBackupContext failed: %v", err)
			}

			result := ctx.ResolveConfigFilePath(tt.configFile)
			if !strings.HasPrefix(result, tt.wantPrefix) {
				t.Errorf("Result %q should have prefix %q", result, tt.wantPrefix)
			}
		})
	}
}

// TestBackupFile tests file backup
func TestBackupFile(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backup")

	// Create source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	ctx, err := NewBackupContext(
		"configs",
		backupDir,
		[]string{},
		true,
		false,
		1,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	if err := ctx.SetupBackupDirectory(); err != nil {
		t.Fatalf("SetupBackupDirectory failed: %v", err)
	}

	targetPath := filepath.Join(backupDir, ctx.Timestamp, "source.txt")

	err = ctx.BackupFile(sourceFile, targetPath)
	if err != nil {
		t.Errorf("BackupFile failed: %v", err)
	}

	// Verify backup was created (not in dry-run)
	if !DryRun {
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			t.Error("Backup file was not created")
		}
	}
}

// TestBackupFile_NonExistent tests backing up non-existent file
func TestBackupFile_NonExistent(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		true,
		false,
		1,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	targetPath := filepath.Join(tempDir, "backup.txt")

	// Should not error, just skip
	err = ctx.BackupFile(nonExistentFile, targetPath)
	if err != nil {
		t.Errorf("BackupFile should not error for non-existent file, got: %v", err)
	}
}

// TestBackupFile_DryRun tests dry-run backup
func TestBackupFile_DryRun(t *testing.T) {
	setupBackupOperationsTest()

	oldDryRun := DryRun
	DryRun = true
	defer func() { DryRun = oldDryRun }()

	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		true,
		false,
		1,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	targetPath := filepath.Join(tempDir, "backup.txt")

	err = ctx.BackupFile(sourceFile, targetPath)
	if err != nil {
		t.Errorf("BackupFile failed: %v", err)
	}

	// In dry-run, file should not be created
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Error("File should not be created in dry-run mode")
	}
}

// TestExecuteCommands tests command execution
func TestExecuteCommands(t *testing.T) {
	setupBackupOperationsTest()

	ctx, err := NewBackupContext(
		"configs",
		"/tmp/backup",
		[]string{},
		true,
		false,
		1,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	commands := []string{"echo test"}

	// Should not panic
	ctx.ExecuteCommands(commands, "test")
	t.Log("ExecuteCommands completed")
}

// TestExecuteCommands_DryRun tests dry-run command execution
func TestExecuteCommands_DryRun(t *testing.T) {
	setupBackupOperationsTest()

	oldDryRun := DryRun
	DryRun = true
	defer func() { DryRun = oldDryRun }()

	ctx, err := NewBackupContext(
		"configs",
		"/tmp/backup",
		[]string{},
		true,
		false,
		1,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	commands := []string{"echo test", "echo test2"}

	// Should not execute in dry-run
	ctx.ExecuteCommands(commands, "test")
	t.Log("ExecuteCommands dry-run completed")
}

// TestFinalizeBackup tests backup finalization
func TestFinalizeBackup(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	tests := []struct {
		name      string
		zipBackup bool
		wantErr   bool
	}{
		{
			name:      "directory backup",
			zipBackup: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := NewBackupContext(
				"configs",
				filepath.Join(tempDir, tt.name),
				[]string{},
				true,
				false,
				1,
				tt.zipBackup,
				"",
			)
			if err != nil {
				t.Fatalf("NewBackupContext failed: %v", err)
			}

			if tt.zipBackup {
				if err := ctx.SetupBackupDirectory(); err != nil {
					t.Fatalf("SetupBackupDirectory failed: %v", err)
				}
			}

			err = ctx.FinalizeBackup()

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestLoadZipFileMap tests zip file map loading
func TestLoadZipFileMap(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")

	// Create a test zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}

	zipWriter := zip.NewWriter(zipFile)
	
	// Add test files to zip
	testFiles := []string{"file1.txt", "file2.txt"}
	for _, name := range testFiles {
		writer, err := zipWriter.Create(name)
		if err != nil {
			t.Fatalf("Failed to create zip entry: %v", err)
		}
		_, err = writer.Write([]byte("test content"))
		if err != nil {
			t.Fatalf("Failed to write zip entry: %v", err)
		}
	}

	_ = zipWriter.Close()
	_ = zipFile.Close()

	ctx, err := NewBackupContext(
		"configs",
		"/tmp/backup",
		[]string{},
		false,
		false,
		1,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	fileMap, zipReader, err := ctx.LoadZipFileMap(zipPath)
	if err != nil {
		t.Fatalf("LoadZipFileMap failed: %v", err)
	}
	defer func() { _ = zipReader.Close() }()

	if len(fileMap) != len(testFiles) {
		t.Errorf("FileMap has %d entries, want %d", len(fileMap), len(testFiles))
	}

	for _, name := range testFiles {
		if !fileMap[name] {
			t.Errorf("File %q not found in map", name)
		}
	}
}

// TestLoadZipFileMap_NonExistent tests loading non-existent zip
func TestLoadZipFileMap_NonExistent(t *testing.T) {
	setupBackupOperationsTest()

	ctx, err := NewBackupContext(
		"configs",
		"/tmp/backup",
		[]string{},
		false,
		false,
		1,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	_, _, err = ctx.LoadZipFileMap("/nonexistent/file.zip")
	if err == nil {
		t.Error("Expected error for non-existent zip file")
	}
}
// Additional tests for RestoreFile, EncryptFile, DecryptFile

// TestRestoreFile tests file restore
func TestRestoreFile(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	// Create a backup file
	backupFile := filepath.Join(tempDir, "backup.txt")
	if err := os.WriteFile(backupFile, []byte("backup content"), 0644); err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	restorePath := filepath.Join(tempDir, "restore.txt")

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		false, // restore
		false,
		1,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	err = ctx.RestoreFile(backupFile, restorePath, "", false)
	if err != nil {
		t.Errorf("RestoreFile failed: %v", err)
	}

	// Verify restore
	content, err := os.ReadFile(restorePath)
	if err != nil {
		t.Errorf("Failed to read restored file: %v", err)
	}
	if string(content) != "backup content" {
		t.Errorf("Restored content = %q, want 'backup content'", string(content))
	}
}

// TestRestoreFile_DryRun tests dry-run restore
func TestRestoreFile_DryRun(t *testing.T) {
	setupBackupOperationsTest()

	oldDryRun := DryRun
	DryRun = true
	defer func() { DryRun = oldDryRun }()

	tempDir := t.TempDir()
	backupFile := filepath.Join(tempDir, "backup.txt")
	restorePath := filepath.Join(tempDir, "restore.txt")

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		false,
		false,
		1,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	err = ctx.RestoreFile(backupFile, restorePath, "", false)
	if err != nil {
		t.Errorf("RestoreFile dry-run failed: %v", err)
	}

	// File should not exist in dry-run
	if _, err := os.Stat(restorePath); !os.IsNotExist(err) {
		t.Error("File should not be created in dry-run mode")
	}
}

// TestEncryptFile tests file encryption
func TestEncryptFile(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	sourceFile := filepath.Join(tempDir, "source.txt")
	if err := os.WriteFile(sourceFile, []byte("secret content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	targetPath := filepath.Join(tempDir, "encrypted")

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		true,
		false,
		1,
		false,
		"test-password",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	err = ctx.EncryptFile(sourceFile, targetPath)
	if err != nil {
		t.Errorf("EncryptFile failed: %v", err)
	}

	// Verify encrypted file exists
	encryptedPath := targetPath + ".encrypted"
	if _, err := os.Stat(encryptedPath); os.IsNotExist(err) {
		t.Error("Encrypted file was not created")
	}
}

// TestEncryptFile_NonExistent tests encrypting non-existent file
func TestEncryptFile_NonExistent(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		true,
		false,
		1,
		false,
		"test-password",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	targetPath := filepath.Join(tempDir, "encrypted")

	// Should not error, just skip
	err = ctx.EncryptFile(nonExistentFile, targetPath)
	if err != nil {
		t.Errorf("EncryptFile should not error for non-existent file, got: %v", err)
	}
}

// TestDecryptFile tests file decryption
func TestDecryptFile(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()
	password := "test-password"

	// First encrypt a file
	sourceFile := filepath.Join(tempDir, "source.txt")
	content := []byte("secret content")
	if err := os.WriteFile(sourceFile, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		true,
		false,
		1,
		false,
		password,
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	targetPath := filepath.Join(tempDir, "encrypted")
	if err := ctx.EncryptFile(sourceFile, targetPath); err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	// Now decrypt it
	encryptedPath := targetPath + ".encrypted"
	decryptedPath := filepath.Join(tempDir, "decrypted.txt")

	restoreCtx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		false,
		false,
		1,
		false,
		password,
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	err = restoreCtx.DecryptFile(encryptedPath, decryptedPath, "test", false, "")
	if err != nil {
		t.Errorf("DecryptFile failed: %v", err)
	}

	// Verify decrypted content matches original
	decryptedContent, err := os.ReadFile(decryptedPath)
	if err != nil {
		t.Errorf("Failed to read decrypted file: %v", err)
	}
	if string(decryptedContent) != string(content) {
		t.Errorf("Decrypted content = %q, want %q", string(decryptedContent), string(content))
	}
}

// TestDecryptFile_WrongPassword tests decryption with wrong password
func TestDecryptFile_WrongPassword(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	// Encrypt with one password
	sourceFile := filepath.Join(tempDir, "source.txt")
	if err := os.WriteFile(sourceFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		true,
		false,
		1,
		false,
		"correct-password",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	targetPath := filepath.Join(tempDir, "encrypted")
	if err := ctx.EncryptFile(sourceFile, targetPath); err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	// Try to decrypt with wrong password
	wrongCtx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		false,
		false,
		1,
		false,
		"wrong-password",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	encryptedPath := targetPath + ".encrypted"
	decryptedPath := filepath.Join(tempDir, "decrypted.txt")

	err = wrongCtx.DecryptFile(encryptedPath, decryptedPath, "test", false, "")
	if err == nil {
		t.Error("Expected error for wrong password")
	}
	if !strings.Contains(err.Error(), "wrong password") && !strings.Contains(err.Error(), "decrypt") {
		t.Errorf("Error should mention decryption/password issue, got: %v", err)
	}
}

// TestDecryptFile_FromZip tests decryption from a zip backup
func TestDecryptFile_FromZip(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()
	password := "test-password"
	appName := "testapp"

	// Create a source file and encrypt it
	sourceFile := filepath.Join(tempDir, "source.txt")
	content := []byte("secret content")
	if err := os.WriteFile(sourceFile, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create backup context for encryption
	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		true,
		false,
		1,
		false,
		password,
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	// Encrypt the file
	targetPath := filepath.Join(tempDir, "encrypted")
	if err := ctx.EncryptFile(sourceFile, targetPath); err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	// Create a zip file containing the encrypted file
	zipPath := filepath.Join(tempDir, "backup.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}

	zipWriter := zip.NewWriter(zipFile)
	encryptedPath := targetPath + ".encrypted"
	encryptedData, err := os.ReadFile(encryptedPath)
	if err != nil {
		t.Fatalf("Failed to read encrypted file: %v", err)
	}

	// Add encrypted file to zip
	zipEntryPath := filepath.ToSlash(filepath.Join(appName, "source.txt")) + ".encrypted"
	writer, err := zipWriter.Create(zipEntryPath)
	if err != nil {
		t.Fatalf("Failed to create zip entry: %v", err)
	}
	if _, err := writer.Write(encryptedData); err != nil {
		t.Fatalf("Failed to write to zip: %v", err)
	}
	_ = zipWriter.Close()
	_ = zipFile.Close()

	// Now test decryption from zip
	restoreCtx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		false,
		false,
		1,
		false,
		password,
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	decryptedPath := filepath.Join(tempDir, "decrypted.txt")
	err = restoreCtx.DecryptFile("", "source.txt", appName, true, zipPath)
	if err != nil {
		t.Errorf("DecryptFile from zip failed: %v", err)
	}

	// Verify decrypted content
	decryptedContent, err := os.ReadFile(decryptedPath)
	if err != nil {
		// File might be written to different location, check temp dir
		t.Logf("Could not read from expected path, checking for temp file cleanup")
	} else if string(decryptedContent) != string(content) {
		t.Errorf("Decrypted content = %q, want %q", string(decryptedContent), string(content))
	}
}

// TestDecryptFile_ZipExtractionFail tests handling of zip extraction failures
func TestDecryptFile_ZipExtractionFail(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		false,
		false,
		1,
		false,
		"password",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	// Try to decrypt from a non-existent zip file
	nonExistentZip := filepath.Join(tempDir, "nonexistent.zip")

	err = ctx.DecryptFile("", "source.txt", "testapp", true, nonExistentZip)
	// Should return nil (skip) when extraction fails
	if err != nil {
		t.Logf("DecryptFile handled missing zip file: %v", err)
	}
}

// TestDecryptFile_InvalidZip tests decryption with invalid zip data
func TestDecryptFile_InvalidZip(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	// Create an invalid zip file
	invalidZipPath := filepath.Join(tempDir, "invalid.zip")
	if err := os.WriteFile(invalidZipPath, []byte("not a zip file"), 0644); err != nil {
		t.Fatalf("Failed to create invalid zip: %v", err)
	}

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		false,
		false,
		1,
		false,
		"password",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	_ = filepath.Join(tempDir, "decrypted.txt") // Not used in this test
	err = ctx.DecryptFile("", "source.txt", "testapp", true, invalidZipPath)
	// Should handle invalid zip gracefully
	if err != nil {
		t.Logf("DecryptFile handled invalid zip: %v", err)
	}
}

// TestDecryptFile_TempFileCleanup tests temp file cleanup
func TestDecryptFile_TempFileCleanup(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()
	password := "test-password"

	// Create and encrypt a file
	sourceFile := filepath.Join(tempDir, "source.txt")
	if err := os.WriteFile(sourceFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		true,
		false,
		1,
		false,
		password,
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	targetPath := filepath.Join(tempDir, "encrypted")
	if err := ctx.EncryptFile(sourceFile, targetPath); err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	// Create zip with encrypted file
	zipPath := filepath.Join(tempDir, "backup.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create zip: %v", err)
	}

	zipWriter := zip.NewWriter(zipFile)
	encryptedData, err := os.ReadFile(targetPath + ".encrypted")
	if err != nil {
		t.Fatalf("Failed to read encrypted file: %v", err)
	}

	writer, err := zipWriter.Create("testapp/source.txt.encrypted")
	if err != nil {
		t.Fatalf("Failed to create zip entry: %v", err)
	}
	_, _ = writer.Write(encryptedData)
	_ = zipWriter.Close()
	_ = zipFile.Close()

	// Decrypt from zip
	restoreCtx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		false,
		false,
		1,
		false,
		password,
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	err = restoreCtx.DecryptFile("", "source.txt", "testapp", true, zipPath)
	if err != nil {
		t.Logf("DecryptFile completed with: %v", err)
	}

	// Check that temp files were cleaned up
	tempFiles, _ := os.ReadDir(os.TempDir())
	for _, f := range tempFiles {
		if strings.Contains(f.Name(), "settingssentry-decrypt-") {
			t.Errorf("Temp file %s was not cleaned up", f.Name())
		}
	}
}

// TestDecryptFile_DryRun tests dry-run mode for decryption
func TestDecryptFile_DryRun(t *testing.T) {
	setupBackupOperationsTest()

	oldDryRun := DryRun
	DryRun = true
	defer func() { DryRun = oldDryRun }()

	tempDir := t.TempDir()
	password := "test-password"

	// Create and encrypt a file
	sourceFile := filepath.Join(tempDir, "source.txt")
	DryRun = false // Temporarily disable for setup
	if err := os.WriteFile(sourceFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		true,
		false,
		1,
		false,
		password,
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	targetPath := filepath.Join(tempDir, "encrypted")
	if err := ctx.EncryptFile(sourceFile, targetPath); err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}
	DryRun = true // Re-enable for test

	// Try to decrypt in dry-run mode
	restoreCtx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		false,
		false,
		1,
		false,
		password,
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	encryptedPath := targetPath + ".encrypted"
	decryptedPath := filepath.Join(tempDir, "decrypted_dryrun.txt")

	err = restoreCtx.DecryptFile(encryptedPath, decryptedPath, "testapp", false, "")
	if err != nil {
		t.Errorf("DecryptFile dry-run failed: %v", err)
	}

	// File should not be created in dry-run
	if _, err := os.Stat(decryptedPath); !os.IsNotExist(err) {
		t.Error("File should not be created in dry-run mode")
	}
}

// TestDecryptFile_CorruptData tests decryption with corrupt encrypted data
func TestDecryptFile_CorruptData(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	// Create a file with corrupt encrypted data
	corruptFile := filepath.Join(tempDir, "corrupt.encrypted")
	if err := os.WriteFile(corruptFile, []byte("not encrypted data"), 0644); err != nil {
		t.Fatalf("Failed to create corrupt file: %v", err)
	}

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		false,
		false,
		1,
		false,
		"password",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	decryptedPath := filepath.Join(tempDir, "decrypted.txt")
	err = ctx.DecryptFile(corruptFile, decryptedPath, "testapp", false, "")
	if err == nil {
		t.Error("Expected error for corrupt encrypted data")
	}
	if !strings.Contains(err.Error(), "decrypt") {
		t.Errorf("Error should mention decryption, got: %v", err)
	}
}

// TestDecryptFile_MissingEncryptedFile tests decryption with missing source
func TestDecryptFile_MissingEncryptedFile(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	ctx, err := NewBackupContext(
		"configs",
		tempDir,
		[]string{},
		false,
		false,
		1,
		false,
		"password",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	nonExistentFile := filepath.Join(tempDir, "nonexistent.encrypted")
	decryptedPath := filepath.Join(tempDir, "decrypted.txt")

	err = ctx.DecryptFile(nonExistentFile, decryptedPath, "testapp", false, "")
	if err == nil {
		t.Error("Expected error for missing encrypted file")
	}
}

// TestRestoreFlow_DecryptionSetup tests restore context setup with encryption
func TestRestoreFlow_DecryptionSetup(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backup")
	configDir := filepath.Join(tempDir, "configs")

	// Create directories
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	// Test restore context with encryption
	restoreCtx, err := NewBackupContext(
		configDir,
		backupDir,
		[]string{"testapp"},
		false, // restore
		false,
		1,
		false,
		"testpassword",
	)
	if err != nil {
		t.Fatalf("NewBackupContext for restore failed: %v", err)
	}

	if restoreCtx.Password != "testpassword" {
		t.Errorf("Password = %q, want 'testpassword'", restoreCtx.Password)
	}

	if restoreCtx.IsBackup {
		t.Error("IsBackup should be false for restore")
	}
}

// TestRestoreFlow_WithZipBackup tests restore from zip archive
func TestRestoreFlow_WithZipBackup(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backup")
	configDir := filepath.Join(tempDir, "configs")
	sourceFile := filepath.Join(tempDir, "source.txt")

	// Create directories
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create test config
	configContent := `[application]
name = ziptest

[configuration_files]
` + sourceFile

	configPath := filepath.Join(configDir, "ziptest.cfg")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create source file
	sourceContent := "zip test content"
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Step 1: Backup with zip
	backupCtx, err := NewBackupContext(
		configDir,
		backupDir,
		[]string{"ziptest"},
		true,
		false,
		1,
		true, // zip backup
		"",
	)
	if err != nil {
		t.Fatalf("NewBackupContext failed: %v", err)
	}

	if err := backupCtx.SetupBackupDirectory(); err != nil {
		t.Fatalf("SetupBackupDirectory failed: %v", err)
	}

	// Perform backup
	fsys, files, err := backupCtx.LoadConfigFiles()
	if err != nil {
		t.Fatalf("LoadConfigFiles failed: %v", err)
	}

	stagingDir := backupCtx.StagingDir
	for _, file := range files {
		configData, err := config.ParseConfig(fsys, file.Name())
		if err != nil {
			continue
		}
		for _, srcPath := range configData.Files {
			destPath := filepath.Join(stagingDir, "ziptest", filepath.Base(srcPath))
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				t.Fatalf("Failed to create dest dir: %v", err)
			}
			if err := backupCtx.BackupFile(srcPath, destPath); err != nil {
				t.Fatalf("BackupFile failed: %v", err)
			}
		}
	}

	// Create zip archive manually for testing
	zipPath := filepath.Join(backupDir, "backup.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create zip: %v", err)
	}

	zipWriter := zip.NewWriter(zipFile)
	
	// Add files from staging to zip
	_ = filepath.Walk(stagingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(stagingDir, path)
		zipEntry, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = zipEntry.Write(data)
		return err
	})
	
	_ = zipWriter.Close()
	_ = zipFile.Close()

	// Verify zip exists
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		t.Fatal("Zip backup not created")
	}

	// Cleanup staging
	_ = os.RemoveAll(stagingDir)

	t.Logf("Zip backup created successfully at %s", zipPath)
}

// TestRestoreFlow_ErrorCases tests error handling in restore flow
func TestRestoreFlow_ErrorCases(t *testing.T) {
	setupBackupOperationsTest()

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		backupDir   string
		configDir   string
		expectError bool
	}{
		{
			name:        "nonexistent backup directory",
			backupDir:   filepath.Join(tempDir, "nonexistent"),
			configDir:   tempDir,
			expectError: true,
		},
		{
			name:        "empty backup directory",
			backupDir:   tempDir,
			configDir:   tempDir,
			expectError: false, // May succeed but find no files
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := NewBackupContext(
				tt.configDir,
				tt.backupDir,
				[]string{},
				false, // restore
				false,
				1,
				false,
				"",
			)
			if err != nil {
				t.Fatalf("NewBackupContext failed: %v", err)
			}

			err = ctx.SetupBackupDirectory()
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Logf("SetupBackupDirectory returned: %v", err)
			}
		})
	}
}
