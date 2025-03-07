package mocks

import (
	"SettingsSentry/interfaces"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MockFile implements the interfaces.File interface for testing
type MockFile struct {
	name     string
	content  *bytes.Buffer
	readOnly bool
	closed   bool
}

func (f *MockFile) Read(p []byte) (n int, err error) {
	if f.closed {
		return 0, os.ErrClosed
	}
	return f.content.Read(p)
}

func (f *MockFile) Write(p []byte) (n int, err error) {
	if f.closed {
		return 0, os.ErrClosed
	}
	if f.readOnly {
		return 0, os.ErrPermission
	}
	return f.content.Write(p)
}

func (f *MockFile) Close() error {
	if f.closed {
		return os.ErrClosed
	}
	f.closed = true
	return nil
}

// MockFileInfo implements os.FileInfo for testing
type MockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (fi *MockFileInfo) Name() string {
	return fi.name
}

func (fi *MockFileInfo) Size() int64 {
	return fi.size
}

func (fi *MockFileInfo) Mode() os.FileMode {
	return fi.mode
}

func (fi *MockFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi *MockFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi *MockFileInfo) Sys() interface{} {
	return nil
}

// MockDirEntry implements os.DirEntry for testing
type MockDirEntry struct {
	name  string
	isDir bool
}

func (e *MockDirEntry) Name() string {
	return e.name
}

func (e *MockDirEntry) IsDir() bool {
	return e.isDir
}

func (e *MockDirEntry) Type() os.FileMode {
	if e.isDir {
		return os.ModeDir
	}
	return 0
}

func (e *MockDirEntry) Info() (os.FileInfo, error) {
	return &MockFileInfo{
		name:    e.name,
		isDir:   e.isDir,
		mode:    e.Type(),
		modTime: time.Now(),
	}, nil
}

// MockFileSystem implements interfaces.FileSystem for testing
type MockFileSystem struct {
	files     map[string][]byte
	dirs      map[string]bool
	fileInfos map[string]*MockFileInfo
	mu        sync.RWMutex
}

// NewMockFileSystem creates a new MockFileSystem
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files:     make(map[string][]byte),
		dirs:      make(map[string]bool),
		fileInfos: make(map[string]*MockFileInfo),
	}
}

// AddFile adds a file to the mock file system
func (fs *MockFileSystem) AddFile(path string, content []byte, mode os.FileMode) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Normalize path
	path = filepath.Clean(path)

	// Add the file
	fs.files[path] = content

	// Add file info
	fs.fileInfos[path] = &MockFileInfo{
		name:    filepath.Base(path),
		size:    int64(len(content)),
		mode:    mode,
		modTime: time.Now(),
		isDir:   false,
	}

	// Ensure parent directories exist
	dir := filepath.Dir(path)
	for dir != "." && dir != "/" {
		fs.dirs[dir] = true
		fs.fileInfos[dir] = &MockFileInfo{
			name:    filepath.Base(dir),
			mode:    os.ModeDir | 0755,
			modTime: time.Now(),
			isDir:   true,
		}
		dir = filepath.Dir(dir)
	}
}

// AddDir adds a directory to the mock file system
func (fs *MockFileSystem) AddDir(path string, mode os.FileMode) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Normalize path
	path = filepath.Clean(path)

	// Add the directory
	fs.dirs[path] = true

	// Add directory info
	fs.fileInfos[path] = &MockFileInfo{
		name:    filepath.Base(path),
		mode:    os.ModeDir | mode,
		modTime: time.Now(),
		isDir:   true,
	}

	// Ensure parent directories exist
	dir := filepath.Dir(path)
	for dir != "." && dir != "/" {
		fs.dirs[dir] = true
		fs.fileInfos[dir] = &MockFileInfo{
			name:    filepath.Base(dir),
			mode:    os.ModeDir | 0755,
			modTime: time.Now(),
			isDir:   true,
		}
		dir = filepath.Dir(dir)
	}
}

// Open opens a file for reading
func (fs *MockFileSystem) Open(name string) (interfaces.File, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Normalize path
	name = filepath.Clean(name)

	// Check if the file exists
	content, exists := fs.files[name]
	if !exists {
		return nil, os.ErrNotExist
	}

	// Create a new buffer with the file content
	return &MockFile{
		name:     name,
		content:  bytes.NewBuffer(content),
		readOnly: true,
	}, nil
}

// Create creates a new file for writing
func (fs *MockFileSystem) Create(name string) (interfaces.File, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Normalize path
	name = filepath.Clean(name)

	// Check if the directory exists
	dir := filepath.Dir(name)
	if dir != "." && !fs.dirs[dir] {
		return nil, os.ErrNotExist
	}

	// Create a new empty file
	fs.files[name] = []byte{}
	fs.fileInfos[name] = &MockFileInfo{
		name:    filepath.Base(name),
		size:    0,
		mode:    0644,
		modTime: time.Now(),
		isDir:   false,
	}

	// Return a file handle
	buffer := bytes.NewBuffer([]byte{})
	file := &MockFile{
		name:     name,
		content:  buffer,
		readOnly: false,
	}

	// Update the file content when the file is closed
	return file, nil
}

// ReadDir reads a directory
func (fs *MockFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Check if directory exists
	if !fs.dirs[dirname] && dirname != "." {
		return nil, os.ErrNotExist
	}

	// Get all files and directories in this directory
	entries := []os.DirEntry{}
	
	// Add all files in this directory
	for path := range fs.files {
		dir := filepath.Dir(path)
		if dir == dirname || (dirname == "." && dir == ".") {
			base := filepath.Base(path)
			entries = append(entries, &MockDirEntry{
				name:  base,
				isDir: false,
			})
		}
	}

	// Add all directories in this directory
	for dir := range fs.dirs {
		parentDir := filepath.Dir(dir)
		if parentDir == dirname || (dirname == "." && parentDir == ".") {
			base := filepath.Base(dir)
			entries = append(entries, &MockDirEntry{
				name:  base,
				isDir: true,
			})
		}
	}

	return entries, nil
}

// MkdirAll creates a directory and all parent directories
func (fs *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Normalize path
	path = filepath.Clean(path)

	// Add the directory and all parent directories
	fs.dirs[path] = true
	fs.fileInfos[path] = &MockFileInfo{
		name:    filepath.Base(path),
		mode:    os.ModeDir | perm,
		modTime: time.Now(),
		isDir:   true,
	}

	// Ensure parent directories exist
	dir := filepath.Dir(path)
	for dir != "." && dir != "/" {
		fs.dirs[dir] = true
		fs.fileInfos[dir] = &MockFileInfo{
			name:    filepath.Base(dir),
			mode:    os.ModeDir | 0755,
			modTime: time.Now(),
			isDir:   true,
		}
		dir = filepath.Dir(dir)
	}

	return nil
}

// RemoveAll removes a file or directory and all its contents
func (fs *MockFileSystem) RemoveAll(path string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Normalize path
	path = filepath.Clean(path)

	// Remove the file or directory
	delete(fs.files, path)
	delete(fs.dirs, path)
	delete(fs.fileInfos, path)

	// Remove all children
	prefix := path + "/"
	for filePath := range fs.files {
		if strings.HasPrefix(filePath, prefix) {
			delete(fs.files, filePath)
			delete(fs.fileInfos, filePath)
		}
	}

	for dirPath := range fs.dirs {
		if strings.HasPrefix(dirPath, prefix) {
			delete(fs.dirs, dirPath)
			delete(fs.fileInfos, dirPath)
		}
	}

	return nil
}

// Stat returns file info
func (fs *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Normalize path
	name = filepath.Clean(name)

	// Check if the file or directory exists
	info, exists := fs.fileInfos[name]
	if !exists {
		return nil, os.ErrNotExist
	}

	return info, nil
}

// ReadFile reads a file
func (fs *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Normalize path
	filename = filepath.Clean(filename)

	// Check if the file exists
	content, exists := fs.files[filename]
	if !exists {
		return nil, os.ErrNotExist
	}

	return content, nil
}

// WriteFile writes a file
func (fs *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Normalize path
	filename = filepath.Clean(filename)

	// Check if the directory exists
	dir := filepath.Dir(filename)
	if dir != "." && !fs.dirs[dir] {
		return os.ErrNotExist
	}

	// Write the file
	fs.files[filename] = data
	fs.fileInfos[filename] = &MockFileInfo{
		name:    filepath.Base(filename),
		size:    int64(len(data)),
		mode:    perm,
		modTime: time.Now(),
		isDir:   false,
	}

	return nil
}

// Join joins path elements
func (fs *MockFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Dir returns the directory portion of a path
func (fs *MockFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

// Base returns the base portion of a path
func (fs *MockFileSystem) Base(path string) string {
	return filepath.Base(path)
}

// Abs returns the absolute path
func (fs *MockFileSystem) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

// EvalSymlinks evaluates symbolic links
func (fs *MockFileSystem) EvalSymlinks(path string) (string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Normalize path
	path = filepath.Clean(path)

	// Check if the file or directory exists
	_, exists := fs.fileInfos[path]
	if !exists {
		return "", os.ErrNotExist
	}

	return path, nil
}
