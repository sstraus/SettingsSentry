package backup

import (
	// Needed for setupBackupTestDependencies
	// Needed for setupBackupTestDependencies
	"SettingsSentry/pkg/config" // Needed for mocking GetHomeDirectory
	// Needed for setupBackupTestDependencies
	// Needed for setupBackupTestDependencies
	"archive/zip"
	"fmt" // Needed for TestProcessConfiguration_* tests
	"io"
	"os"
	"path/filepath"
	"strings" // Needed for TestCleanupOldVersions_Mixed
	"testing"
	"time" // Needed for TestProcessConfiguration_ZipRestore setup
)

// Helper function to create a dummy file for testing
func createDummyFile(t *testing.T, path string, content string) {
	t.Helper()
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatalf("Failed to create directory for dummy file %s: %v", path, err)
	}
	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy file %s: %v", path, err)
	}
}

// Helper function to verify zip content
func verifyZipContent(t *testing.T, zipPath string, expectedFiles map[string]string) {
	t.Helper()
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("Failed to open zip file %s: %v", zipPath, err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Errorf("Error closing zip reader for %s: %v", zipPath, err)
		}
	}()

	foundFiles := make(map[string]string)
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			// Optionally verify directory structure if needed
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Errorf("Failed to open file %s in zip: %v", f.Name, err)
			continue
		}
		contentBytes, err := io.ReadAll(rc)
		// Close rc *before* checking io.ReadAll error
		if closeErr := rc.Close(); closeErr != nil {
			t.Errorf("Error closing zip entry reader for %s: %v", f.Name, closeErr)
			// Continue processing other files even if close fails
		}
		if err != nil {
			t.Errorf("Failed to read file %s in zip: %v", f.Name, err)
			continue
		}
		foundFiles[f.Name] = string(contentBytes)
	}

	if len(foundFiles) != len(expectedFiles) {
		t.Errorf("Expected %d files in zip, found %d. Found: %v", len(expectedFiles), len(foundFiles), foundFiles)
	}

	for name, expectedContent := range expectedFiles {
		// Zip headers use forward slashes
		name = filepath.ToSlash(name)
		actualContent, ok := foundFiles[name]
		if !ok {
			t.Errorf("Expected file %s not found in zip", name)
		} else if actualContent != expectedContent {
			t.Errorf("Content mismatch for file %s. Expected '%s', got '%s'", name, expectedContent, actualContent)
		}
	}
}

func TestCreateZipArchive(t *testing.T) {
	tempDir := t.TempDir() // Use t.TempDir for automatic cleanup

	// Test case 1: Simple file
	srcDir1 := filepath.Join(tempDir, "src1")
	targetZip1 := filepath.Join(tempDir, "archive1.zip")
	createDummyFile(t, filepath.Join(srcDir1, "file1.txt"), "content1")

	err := createZipArchive(srcDir1, targetZip1)
	if err != nil {
		t.Fatalf("Test case 1 failed: createZipArchive returned error: %v", err)
	}
	verifyZipContent(t, targetZip1, map[string]string{"file1.txt": "content1"})

	// Test case 2: Directory structure
	srcDir2 := filepath.Join(tempDir, "src2")
	targetZip2 := filepath.Join(tempDir, "archive2.zip")
	createDummyFile(t, filepath.Join(srcDir2, "fileA.txt"), "contentA")
	createDummyFile(t, filepath.Join(srcDir2, "subdir", "fileB.txt"), "contentB")
	// Ensure subdir exists even if empty
	if err := os.MkdirAll(filepath.Join(srcDir2, "subdir", "empty_subdir"), 0755); err != nil {
		t.Fatalf("Failed to create empty subdir: %v", err)
	}

	err = createZipArchive(srcDir2, targetZip2)
	if err != nil {
		t.Fatalf("Test case 2 failed: createZipArchive returned error: %v", err)
	}
	verifyZipContent(t, targetZip2, map[string]string{
		"fileA.txt":        "contentA",
		"subdir/fileB.txt": "contentB",
		// Note: verifyZipContent currently only checks files, not directory entries
	})

	// Test case 3: Empty source directory
	srcDir3 := filepath.Join(tempDir, "src3")
	targetZip3 := filepath.Join(tempDir, "archive3.zip")
	if err := os.MkdirAll(srcDir3, 0755); err != nil {
		t.Fatalf("Failed to create empty src dir: %v", err)
	}

	err = createZipArchive(srcDir3, targetZip3)
	if err != nil {
		t.Fatalf("Test case 3 failed: createZipArchive returned error: %v", err)
	}
	verifyZipContent(t, targetZip3, map[string]string{})

	// Test case 4: Non-existent source directory (should error)
	targetZip4 := filepath.Join(tempDir, "archive4.zip")
	err = createZipArchive(filepath.Join(tempDir, "nonexistent"), targetZip4)
	if err == nil {
		t.Fatalf("Test case 4 failed: Expected error for non-existent source, got nil")
	}
}

func TestExtractFromZip(t *testing.T) {
	tempDir := t.TempDir()

	// --- Setup: Create a test zip file ---
	srcDir := filepath.Join(tempDir, "source_for_zip")
	zipPath := filepath.Join(tempDir, "test_archive.zip")
	createDummyFile(t, filepath.Join(srcDir, "app1", "config.txt"), "app1_content")
	createDummyFile(t, filepath.Join(srcDir, "app1", "data", "file.dat"), "app1_data")
	createDummyFile(t, filepath.Join(srcDir, "app2", "settings.json"), "app2_settings")
	if err := os.MkdirAll(filepath.Join(srcDir, "app1", "empty_dir"), 0755); err != nil {
		t.Fatalf("Setup failed: Cannot create empty_dir: %v", err)
	}

	err := createZipArchive(srcDir, zipPath)
	if err != nil {
		t.Fatalf("Setup failed: Cannot create test zip archive: %v", err)
	}
	// --- End Setup ---

	// Test case 1: Extract single file
	extractDest1 := filepath.Join(tempDir, "extract1")
	entryPath1 := "app1/config.txt"
	destPath1 := filepath.Join(extractDest1, "my_config.txt")
	err = extractFromZip(zipPath, entryPath1, destPath1)
	if err != nil {
		t.Fatalf("Test case 1 failed: extractFromZip returned error: %v", err)
	}
	content1, err := os.ReadFile(destPath1)
	if err != nil {
		t.Fatalf("Test case 1 failed: Cannot read extracted file: %v", err)
	}
	if string(content1) != "app1_content" {
		t.Errorf("Test case 1 failed: Content mismatch. Expected 'app1_content', got '%s'", string(content1))
	}

	// Test case 2: Extract directory
	extractDest2 := filepath.Join(tempDir, "extract2")
	entryPath2 := "app1"
	destPath2 := filepath.Join(extractDest2, "restored_app1")
	err = extractFromZip(zipPath, entryPath2, destPath2)
	if err != nil {
		t.Fatalf("Test case 2 failed: extractFromZip returned error: %v", err)
	}
	// Verify extracted files within the directory
	content2a, err := os.ReadFile(filepath.Join(destPath2, "config.txt"))
	if err != nil || string(content2a) != "app1_content" {
		t.Errorf("Test case 2 failed: config.txt content mismatch or read error: %v", err)
	}
	content2b, err := os.ReadFile(filepath.Join(destPath2, "data", "file.dat"))
	if err != nil || string(content2b) != "app1_data" {
		t.Errorf("Test case 2 failed: data/file.dat content mismatch or read error: %v", err)
	}
	// Verify empty directory was created
	if _, err := os.Stat(filepath.Join(destPath2, "empty_dir")); os.IsNotExist(err) {
		t.Errorf("Test case 2 failed: empty_dir was not extracted")
	}

	// Test case 3: Extract non-existent entry
	extractDest3 := filepath.Join(tempDir, "extract3")
	entryPath3 := "app3/nonexistent.cfg"
	destPath3 := filepath.Join(extractDest3, "wont_be_created.cfg")
	err = extractFromZip(zipPath, entryPath3, destPath3)
	if err != nil {
		t.Fatalf("Test case 3 failed: extractFromZip returned error for non-existent entry: %v", err)
	}
	if _, err := os.Stat(destPath3); !os.IsNotExist(err) {
		t.Errorf("Test case 3 failed: Destination file %s was created for non-existent entry", destPath3)
	}

	// Test case 4: Extract from non-existent zip file (should error)
	err = extractFromZip(filepath.Join(tempDir, "nosuch.zip"), "any", "anywhere")
	if err == nil {
		t.Fatalf("Test case 4 failed: Expected error for non-existent zip file, got nil")
	}
}

func TestGetLatestVersionPath_Mixed(t *testing.T) {
	// Use real filesystem for this test
	setupBackupTestDependencies() // Sets up util.Fs = OsFileSystem

	tempDir := t.TempDir()
	baseBackupPath := filepath.Join(tempDir, "mixed_backups")
	if err := os.MkdirAll(baseBackupPath, 0755); err != nil {
		t.Fatalf("Failed to create base backup dir: %v", err)
	}

	// Timestamps
	ts1 := "20230101-100000"
	ts2 := "20230101-110000"
	ts3 := "20230101-120000"

	// Case 1: No versions yet
	_, _, err := GetLatestVersionPath(baseBackupPath)
	if err == nil {
		t.Fatalf("Case 1 Failed: Expected error for no versions, got nil")
	}

	// Case 2: Add a directory
	dir1 := filepath.Join(baseBackupPath, ts1)
	if err := os.Mkdir(dir1, 0755); err != nil {
		t.Fatalf("Failed to create dir1: %v", err)
	}
	latestPath, isZip, err := GetLatestVersionPath(baseBackupPath)
	if err != nil {
		t.Fatalf("Case 2 Failed: GetLatestVersionPath errored: %v", err)
	}
	if isZip || latestPath != dir1 {
		t.Errorf("Case 2 Failed: Expected dir '%s' (isZip=false), got '%s' (isZip=%t)", dir1, latestPath, isZip)
	}

	// Case 3: Add a newer zip file
	zip2 := filepath.Join(baseBackupPath, ts2+".zip")
	createDummyFile(t, zip2, "zip content") // Create dummy zip file
	latestPath, isZip, err = GetLatestVersionPath(baseBackupPath)
	if err != nil {
		t.Fatalf("Case 3 Failed: GetLatestVersionPath errored: %v", err)
	}
	if !isZip || latestPath != zip2 {
		t.Errorf("Case 3 Failed: Expected zip '%s' (isZip=true), got '%s' (isZip=%t)", zip2, latestPath, isZip)
	}

	// Case 4: Add an even newer directory
	dir3 := filepath.Join(baseBackupPath, ts3)
	if err := os.Mkdir(dir3, 0755); err != nil {
		t.Fatalf("Failed to create dir3: %v", err)
	}
	latestPath, isZip, err = GetLatestVersionPath(baseBackupPath)
	if err != nil {
		t.Fatalf("Case 4 Failed: GetLatestVersionPath errored: %v", err)
	}
	if isZip || latestPath != dir3 {
		t.Errorf("Case 4 Failed: Expected dir '%s' (isZip=false), got '%s' (isZip=%t)", dir3, latestPath, isZip)
	}
}

func TestCleanupOldVersions_Mixed(t *testing.T) {
	// Use real filesystem for this test
	setupBackupTestDependencies() // Sets up util.Fs = OsFileSystem

	tempDir := t.TempDir()
	baseBackupPath := filepath.Join(tempDir, "cleanup_mixed_backups")
	if err := os.MkdirAll(baseBackupPath, 0755); err != nil {
		t.Fatalf("Failed to create base backup dir: %v", err)
	}

	// Timestamps (out of order)
	ts := []string{
		"20230101-120000", // Keep (Dir)
		"20230101-100000", // Remove (Zip)
		"20230101-130000", // Keep (Zip)
		"20230101-090000", // Remove (Dir)
		"20230101-110000", // Keep (Dir)
	}

	// Create mix of dirs and zips
	if err := os.Mkdir(filepath.Join(baseBackupPath, ts[0]), 0755); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}
	createDummyFile(t, filepath.Join(baseBackupPath, ts[1]+".zip"), "zip1")
	createDummyFile(t, filepath.Join(baseBackupPath, ts[2]+".zip"), "zip2")
	if err := os.Mkdir(filepath.Join(baseBackupPath, ts[3]), 0755); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}
	if err := os.Mkdir(filepath.Join(baseBackupPath, ts[4]), 0755); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	// --- Test Cleanup (Keep 3) ---
	versionsToKeep := 3
	err := CleanupOldVersions(baseBackupPath, versionsToKeep)
	if err != nil {
		t.Fatalf("CleanupOldVersions errored: %v", err)
	}

	// Verify remaining entries
	entries, err := os.ReadDir(baseBackupPath)
	if err != nil {
		t.Fatalf("Failed to read backup dir after cleanup: %v", err)
	}

	if len(entries) != versionsToKeep {
		t.Errorf("Expected %d entries after cleanup, found %d", versionsToKeep, len(entries))
	}

	// Check that the correct 3 latest versions remain
	expectedRemaining := map[string]bool{
		ts[0]: true, // 20230101-120000 (Dir)
		ts[2]: true, // 20230101-130000.zip (Zip)
		ts[4]: true, // 20230101-110000 (Dir)
	}

	foundRemaining := make(map[string]bool)
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".zip")
		if !expectedRemaining[name] {
			t.Errorf("Unexpected entry found after cleanup: %s", entry.Name())
		}
		foundRemaining[name] = true
	}

	if len(foundRemaining) != versionsToKeep {
		t.Errorf("Did not find all expected remaining versions. Found: %v", foundRemaining)
	}

	// Check that the removed ones are gone
	if _, err := os.Stat(filepath.Join(baseBackupPath, ts[1]+".zip")); !os.IsNotExist(err) {
		t.Errorf("Expected %s.zip to be removed, but it still exists", ts[1])
	}
	if _, err := os.Stat(filepath.Join(baseBackupPath, ts[3])); !os.IsNotExist(err) {
		t.Errorf("Expected %s to be removed, but it still exists", ts[3])
	}
}

func TestProcessConfiguration_ZipBackup(t *testing.T) {
	// Use real filesystem
	setupBackupTestDependencies()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	backupDir := filepath.Join(tempDir, "backups")
	mockHomeDir := filepath.Join(tempDir, "mockHome") // Create a mock home

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.MkdirAll(mockHomeDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// Mock GetHomeDirectory from config package
	originalGetHomeDir := config.GetHomeDirectory
	config.GetHomeDirectory = func() (string, error) {
		return mockHomeDir, nil
	}
	defer func() { config.GetHomeDirectory = originalGetHomeDir }()

	// Create dummy source file inside mock home
	sourceFileName := ".myapp_settings"
	sourceFileContent := "my app settings content"
	sourceFilePathInHome := filepath.Join(mockHomeDir, sourceFileName)
	createDummyFile(t, sourceFilePathInHome, sourceFileContent)

	// Create dummy config file using relative path
	cfgContent := fmt.Sprintf(`[application]
name = MyZipApp
[files]
~/%s
`, sourceFileName) // Use relative path starting with ~/
	cfgPath := filepath.Join(configDir, "myzipapp.cfg")
	createDummyFile(t, cfgPath, cfgContent)

	// --- Perform Zip Backup ---
	ProcessConfiguration(configDir, backupDir, "", true, false, 1, true) // zipBackup = true

	// --- Verification ---
	// Find the created zip file (should be only one entry)
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry in backup dir, found %d", len(entries))
	}
	backupEntry := entries[0]
	if backupEntry.IsDir() || !strings.HasSuffix(backupEntry.Name(), ".zip") {
		t.Fatalf("Expected a zip file, found: %s", backupEntry.Name())
	}

	// Verify zip content
	zipPath := filepath.Join(backupDir, backupEntry.Name())
	expectedZipContent := map[string]string{
		"MyZipApp/.myapp_settings": "my app settings content",
	}
	verifyZipContent(t, zipPath, expectedZipContent)
}

func TestProcessConfiguration_ZipRestore(t *testing.T) {
	// Use real filesystem
	setupBackupTestDependencies()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	backupDir := filepath.Join(tempDir, "backups")
	mockHomeDir := filepath.Join(tempDir, "mockHome") // Use a mock home for restore destination
	restoreDestDir := mockHomeDir                     // Restore directly into mock home

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.MkdirAll(mockHomeDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// Mock GetHomeDirectory
	originalGetHomeDir := config.GetHomeDirectory
	config.GetHomeDirectory = func() (string, error) {
		return mockHomeDir, nil
	}
	defer func() { config.GetHomeDirectory = originalGetHomeDir }()

	// --- Setup: Create source, config, and ZIP backup ---
	appName := "RestoreZipApp"
	sourceFileName := ".myrestore_settings"
	sourceFileContent := "restore zip content"

	// Config file points to the *final* restore destination path relative to home
	cfgContent := fmt.Sprintf(`[application]
name = %s
[files]
~/%s
`, appName, sourceFileName) // Use relative path starting with ~/
	cfgPath := filepath.Join(configDir, "restorezipapp.cfg")
	createDummyFile(t, cfgPath, cfgContent)

	// Create a zip backup manually for the test
	zipTimestamp := time.Now().Format("20060102-150405")
	zipFileName := zipTimestamp + ".zip"
	zipFilePath := filepath.Join(backupDir, zipFileName)
	stagingDir := filepath.Join(tempDir, "staging_for_restore_test")
	stagedFilePath := filepath.Join(stagingDir, appName, sourceFileName)
	createDummyFile(t, stagedFilePath, sourceFileContent)
	if err := createZipArchive(stagingDir, zipFilePath); err != nil {
		t.Fatalf("Setup failed: Could not create test zip backup: %v", err)
	}
	// Clean up staging dir used for setup, checking error
	if err := os.RemoveAll(stagingDir); err != nil {
		t.Errorf("Failed to remove staging dir %s during setup cleanup: %v", stagingDir, err)
	}

	// Ensure the restore destination does NOT exist before restore
	restoreFilePathFinal := filepath.Join(restoreDestDir, sourceFileName)
	if err := os.Remove(restoreFilePathFinal); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Setup failed: Could not remove pre-existing restore destination: %v", err)
	}

	// --- Perform Restore ---
	// We pass zipBackup=false because restore determines type automatically
	ProcessConfiguration(configDir, backupDir, "", false, false, 1, false)

	// --- Verification ---
	// Check if the file was restored to the correct destination
	restoredContent, err := os.ReadFile(restoreFilePathFinal)
	if err != nil {
		t.Fatalf("Restore failed: Cannot read restored file at %s: %v", restoreFilePathFinal, err)
	}
	if string(restoredContent) != sourceFileContent {
		t.Errorf("Restore failed: Content mismatch. Expected '%s', got '%s'", sourceFileContent, string(restoredContent))
	}
}
