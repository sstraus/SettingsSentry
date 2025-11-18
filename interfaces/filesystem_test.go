package interfaces

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOsFileSystem_CreateAndOpen(t *testing.T) {
	fs := NewOsFileSystem()
	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.txt")

	// Test Create
	file, err := fs.Create(testFile)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	content := []byte("test content")
	_, err = file.Write(content)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}
	file.Close()

	// Test Open
	file, err = fs.Open(testFile)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer file.Close()

	readContent := make([]byte, len(content))
	_, err = file.Read(readContent)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("Content mismatch: got %q, want %q", readContent, content)
	}
}

func TestOsFileSystem_ReadDir(t *testing.T) {
	fs := NewOsFileSystem()
	tempDir := t.TempDir()

	// Create test files
	testFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, name := range testFiles {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	entries, err := fs.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir() failed: %v", err)
	}

	if len(entries) != len(testFiles) {
		t.Errorf("ReadDir() returned %d entries, want %d", len(entries), len(testFiles))
	}

	// Verify all files are present
	foundFiles := make(map[string]bool)
	for _, entry := range entries {
		foundFiles[entry.Name()] = true
	}

	for _, name := range testFiles {
		if !foundFiles[name] {
			t.Errorf("Expected file %s not found in ReadDir() results", name)
		}
	}
}

func TestOsFileSystem_MkdirAll(t *testing.T) {
	fs := NewOsFileSystem()
	tempDir := t.TempDir()

	nestedDir := filepath.Join(tempDir, "level1", "level2", "level3")

	err := fs.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("MkdirAll() failed: %v", err)
	}

	info, err := os.Stat(nestedDir)
	if err != nil {
		t.Fatalf("Directory was not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("Created path is not a directory")
	}
}

func TestOsFileSystem_RemoveAll(t *testing.T) {
	fs := NewOsFileSystem()
	tempDir := t.TempDir()

	testDir := filepath.Join(tempDir, "to-remove")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create some files in the directory
	for i := 0; i < 3; i++ {
		path := filepath.Join(testDir, "file"+string(rune('1'+i))+".txt")
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	err := fs.RemoveAll(testDir)
	if err != nil {
		t.Fatalf("RemoveAll() failed: %v", err)
	}

	_, err = os.Stat(testDir)
	if !os.IsNotExist(err) {
		t.Error("Directory still exists after RemoveAll()")
	}
}

func TestOsFileSystem_Stat(t *testing.T) {
	fs := NewOsFileSystem()
	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	info, err := fs.Stat(testFile)
	if err != nil {
		t.Fatalf("Stat() failed: %v", err)
	}

	if info.IsDir() {
		t.Error("File incorrectly identified as directory")
	}

	if info.Size() != int64(len(content)) {
		t.Errorf("Size mismatch: got %d, want %d", info.Size(), len(content))
	}

	// Test non-existent file
	_, err = fs.Stat(filepath.Join(tempDir, "nonexistent.txt"))
	if err == nil {
		t.Error("Stat() should fail for non-existent file")
	}
}

func TestOsFileSystem_ReadFile(t *testing.T) {
	fs := NewOsFileSystem()
	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.txt")
	expected := []byte("test content for reading")
	if err := os.WriteFile(testFile, expected, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	content, err := fs.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(content) != string(expected) {
		t.Errorf("Content mismatch: got %q, want %q", content, expected)
	}
}

func TestOsFileSystem_WriteFile(t *testing.T) {
	fs := NewOsFileSystem()
	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.txt")
	expected := []byte("content to write")

	err := fs.WriteFile(testFile, expected, 0644)
	if err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(content) != string(expected) {
		t.Errorf("Content mismatch: got %q, want %q", content, expected)
	}
}

func TestOsFileSystem_PathOperations(t *testing.T) {
	fs := NewOsFileSystem()

	tests := []struct {
		name     string
		op       func() string
		expected string
	}{
		{
			name:     "Join paths",
			op:       func() string { return fs.Join("path", "to", "file.txt") },
			expected: filepath.Join("path", "to", "file.txt"),
		},
		{
			name:     "Dir of path",
			op:       func() string { return fs.Dir("/path/to/file.txt") },
			expected: "/path/to",
		},
		{
			name:     "Base of path",
			op:       func() string { return fs.Base("/path/to/file.txt") },
			expected: "file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.op()
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestOsFileSystem_Abs(t *testing.T) {
	fs := NewOsFileSystem()

	// Test with current directory
	absPath, err := fs.Abs(".")
	if err != nil {
		t.Fatalf("Abs() failed: %v", err)
	}

	if !filepath.IsAbs(absPath) {
		t.Errorf("Abs() returned non-absolute path: %s", absPath)
	}
}

func TestOsFileSystem_EvalSymlinks(t *testing.T) {
	fs := NewOsFileSystem()
	tempDir := t.TempDir()

	// Create a real file
	realFile := filepath.Join(tempDir, "real.txt")
	if err := os.WriteFile(realFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// For non-symlink, it should return the same path
	resolved, err := fs.EvalSymlinks(realFile)
	if err != nil {
		t.Fatalf("EvalSymlinks() failed: %v", err)
	}

	// The resolved path should be absolute
	if !filepath.IsAbs(resolved) {
		t.Errorf("EvalSymlinks() returned non-absolute path: %s", resolved)
	}
}

func TestNewOsFileSystem(t *testing.T) {
	fs := NewOsFileSystem()
	if fs == nil {
		t.Error("NewOsFileSystem() returned nil")
	}

	// Verify it implements the interface
	var _ FileSystem = fs
}

func TestOsFile_Interface(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	osFile := &OsFile{File: file}

	// Verify it implements the File interface
	var _ File = osFile

	// Test Write
	written, err := osFile.Write([]byte("test"))
	if err != nil {
		t.Errorf("Write() failed: %v", err)
	}
	if written != 4 {
		t.Errorf("Write() wrote %d bytes, want 4", written)
	}

	// Close and reopen for reading
	osFile.Close()

	file, err = os.Open(testFile)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer file.Close()

	osFile = &OsFile{File: file}

	// Test Read
	buffer := make([]byte, 4)
	read, err := osFile.Read(buffer)
	if err != nil {
		t.Errorf("Read() failed: %v", err)
	}
	if read != 4 {
		t.Errorf("Read() read %d bytes, want 4", read)
	}
	if string(buffer) != "test" {
		t.Errorf("Read() got %q, want %q", buffer, "test")
	}
}