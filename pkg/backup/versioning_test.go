package backup

import (
	"SettingsSentry/interfaces"
	"SettingsSentry/logger"
	"SettingsSentry/pkg/printer"
	"SettingsSentry/pkg/util"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupVersioningTestDependencies() {
	testLogger, _ := logger.NewLogger("")
	testFs := newMockVersionFileSystem()
	testPrinter := printer.NewPrinter("Test", testLogger)
	util.InitGlobals(testLogger, testFs, nil, false, "")
	printer.AppLogger = testLogger
	AppLogger = util.AppLogger
	Fs = util.Fs
	DryRun = util.DryRun
	Printer = testPrinter
}

type mockVersionFileSystem struct {
	dirs       map[string][]os.DirEntry
	files      map[string][]byte
	fileInfo   map[string]os.FileInfo
	dirEntries map[string]bool
}

func newMockVersionFileSystem() *mockVersionFileSystem {
	return &mockVersionFileSystem{
		dirs:       make(map[string][]os.DirEntry),
		files:      make(map[string][]byte),
		fileInfo:   make(map[string]os.FileInfo),
		dirEntries: make(map[string]bool),
	}
}

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return m.isDir }
func (m *mockDirEntry) Type() os.FileMode          { return 0 }
func (m *mockDirEntry) Info() (os.FileInfo, error) { return nil, nil }

type mockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

func (m *mockVersionFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	entries, ok := m.dirs[dirname]
	if !ok {
		// Return an empty list if the directory doesn't exist
		return []os.DirEntry{}, nil
	}

	// Filter out entries that have been removed
	var validEntries []os.DirEntry
	for _, entry := range entries {
		fullPath := filepath.Join(dirname, entry.Name())
		if _, exists := m.dirEntries[fullPath]; exists {
			validEntries = append(validEntries, entry)
		}
	}

	return validEntries, nil
}

func (m *mockVersionFileSystem) Stat(name string) (os.FileInfo, error) {
	info, ok := m.fileInfo[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return info, nil
}

func (m *mockVersionFileSystem) MkdirAll(path string, perm os.FileMode) error {
	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		if err := m.MkdirAll(dir, perm); err != nil {
			return err
		}
	}

	m.fileInfo[path] = &mockFileInfo{
		name:    filepath.Base(path),
		mode:    perm | os.ModeDir,
		modTime: time.Now(),
		isDir:   true,
	}

	// Ensure parent directory has this as an entry
	parentDir := filepath.Dir(path)
	if parentDir != path {
		entry := &mockDirEntry{name: filepath.Base(path), isDir: true}
		m.dirs[parentDir] = append(m.dirs[parentDir], entry)
		m.dirEntries[filepath.Join(parentDir, filepath.Base(path))] = true
	}

	if _, ok := m.dirs[path]; !ok {
		m.dirs[path] = []os.DirEntry{}
		m.dirEntries[path] = true
	}

	return nil
}

func (m *mockVersionFileSystem) RemoveAll(path string) error {
	for storedPath := range m.dirs {
		if storedPath == path || strings.HasPrefix(storedPath, path+"/") {
			delete(m.dirs, storedPath)
			delete(m.dirEntries, storedPath)
		}
	}

	for storedPath := range m.files {
		if storedPath == path || strings.HasPrefix(storedPath, path+"/") {
			delete(m.files, storedPath)
		}
	}

	for storedPath := range m.fileInfo {
		if storedPath == path || strings.HasPrefix(storedPath, path+"/") {
			delete(m.fileInfo, storedPath)
			delete(m.dirEntries, storedPath)
		}
	}

	return nil
}

func (m *mockVersionFileSystem) ReadFile(filename string) ([]byte, error) {
	data, ok := m.files[filename]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func (m *mockVersionFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	// Create parent directories if they don't exist
	dir := filepath.Dir(filename)
	if err := m.MkdirAll(dir, 0755); err != nil {
		return err
	}

	m.files[filename] = data

	m.fileInfo[filename] = &mockFileInfo{
		name:    filepath.Base(filename),
		size:    int64(len(data)),
		mode:    perm,
		modTime: time.Now(),
		isDir:   false,
	}

	entry := &mockDirEntry{name: filepath.Base(filename), isDir: false}
	m.dirs[dir] = append(m.dirs[dir], entry)
	m.dirEntries[filename] = true

	return nil
}

func (m *mockVersionFileSystem) Open(name string) (interfaces.File, error) {
	// Not needed for these tests
	return nil, nil
}

func (m *mockVersionFileSystem) Create(name string) (interfaces.File, error) {
	// Not needed for these tests
	return nil, nil
}

func (m *mockVersionFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (m *mockVersionFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

func (m *mockVersionFileSystem) Base(path string) string {
	return filepath.Base(path)
}

func (m *mockVersionFileSystem) Abs(path string) (string, error) {
	return path, nil
}

func (m *mockVersionFileSystem) EvalSymlinks(path string) (string, error) {
	return path, nil
}

func TestGetLatestVersionPath(t *testing.T) {
	setupVersioningTestDependencies()

	mockFs := util.Fs.(*mockVersionFileSystem)

	baseBackupPath := "/backups/TestApp"

	if err := mockFs.MkdirAll(baseBackupPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	_, err := GetLatestVersionPath(baseBackupPath)
	if err == nil {
		t.Error("Expected error when no versions exist, but got nil")
	}

	version1 := "20210101-120000"
	version2 := "20210102-120000" // More recent
	version3 := "20200101-120000" // Older

	if err := mockFs.MkdirAll(filepath.Join(baseBackupPath, version1), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := mockFs.MkdirAll(filepath.Join(baseBackupPath, version2), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := mockFs.MkdirAll(filepath.Join(baseBackupPath, version3), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Also create a non-version directory that should be ignored
	if err := mockFs.MkdirAll(filepath.Join(baseBackupPath, "not-a-version"), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	latestPath, err := GetLatestVersionPath(baseBackupPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedPath := filepath.Join(baseBackupPath, version2)
	if latestPath != expectedPath {
		t.Errorf("Expected latest version path to be %s, but got %s", expectedPath, latestPath)
	}
}

func TestCleanupOldVersions(t *testing.T) {
	setupVersioningTestDependencies()

	mockFs := util.Fs.(*mockVersionFileSystem)

	originalDryRun := util.DryRun
	defer func() { util.DryRun = originalDryRun }()

	util.DryRun = false

	baseBackupPath := "/backups/TestApp"

	if err := mockFs.MkdirAll(baseBackupPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	versions := []string{
		"20210101-120000", // 3rd newest
		"20210102-120000", // 2nd newest
		"20210103-120000", // newest
		"20200101-120000", // oldest
		"20200102-120000", // 2nd oldest
	}

	for _, version := range versions {
		if err := mockFs.MkdirAll(filepath.Join(baseBackupPath, version), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	versionsToKeep := 3
	err := CleanupOldVersions(baseBackupPath, versionsToKeep)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	entries, err := mockFs.ReadDir(baseBackupPath)
	if err != nil {
		t.Errorf("Unexpected error reading directory: %v", err)
	}

	if len(entries) != versionsToKeep {
		t.Errorf("Expected %d versions after cleanup, but got %d", versionsToKeep, len(entries))
	}

	expectedVersions := map[string]bool{
		"20210101-120000": true,
		"20210102-120000": true,
		"20210103-120000": true,
	}

	for _, entry := range entries {
		if !expectedVersions[entry.Name()] {
			t.Errorf("Unexpected version found after cleanup: %s", entry.Name())
		}
	}

	util.DryRun = true

	// Reset mockFs for the dry run test - Re-initialize util.Fs with a new mock
	mockFs = newMockVersionFileSystem()
	util.Fs = mockFs

	if err := mockFs.MkdirAll(baseBackupPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	versions = []string{
		"20210101-120000",
		"20210102-120000",
		"20210103-120000",
		"20200101-120000",
		"20200102-120000",
	}

	for _, version := range versions {
		if err := mockFs.MkdirAll(filepath.Join(baseBackupPath, version), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	err = CleanupOldVersions(baseBackupPath, versionsToKeep)
	if err != nil {
		t.Errorf("Unexpected error in dry run: %v", err)
	}

	// Check that we still have all versions (dry run shouldn't delete anything)
	entries, err = mockFs.ReadDir(baseBackupPath)
	if err != nil {
		t.Errorf("Unexpected error reading directory: %v", err)
	}

	if len(entries) != len(versions) {
		t.Errorf("Expected %d versions after dry run cleanup, but got %d", len(versions), len(entries))
	}
}

func TestDryRunMode(t *testing.T) {
	setupVersioningTestDependencies()

	originalDryRun := util.DryRun
	defer func() { util.DryRun = originalDryRun }()

	util.DryRun = true

	if !util.DryRun {
		t.Error("Dry run mode should be enabled")
	}

	util.DryRun = false

	if util.DryRun {
		t.Error("Dry run mode should be disabled")
	}
}
