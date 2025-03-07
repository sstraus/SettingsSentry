package interfaces

import (
	"io"
	"os"
	"path/filepath"
)

// FileSystem defines an interface for file system operations
type FileSystem interface {
	// File operations
	Open(name string) (File, error)
	Create(name string) (File, error)
	ReadDir(dirname string) ([]os.DirEntry, error)
	MkdirAll(path string, perm os.FileMode) error
	RemoveAll(path string) error
	Stat(name string) (os.FileInfo, error)
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	
	// Path operations
	Join(elem ...string) string
	Dir(path string) string
	Base(path string) string
	Abs(path string) (string, error)
	EvalSymlinks(path string) (string, error)
}

// File defines an interface for file operations
type File interface {
	io.Reader
	io.Writer
	io.Closer
}

// OsFileSystem is the concrete implementation of FileSystem using OS functions
type OsFileSystem struct{}

// OsFile is the concrete implementation of File using OS file
type OsFile struct {
	*os.File
}

// Open opens a file using OS functions
func (fs *OsFileSystem) Open(name string) (File, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return &OsFile{File: file}, nil
}

// Create creates a file using OS functions
func (fs *OsFileSystem) Create(name string) (File, error) {
	file, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	return &OsFile{File: file}, nil
}

// ReadDir reads a directory using OS functions
func (fs *OsFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	return os.ReadDir(dirname)
}

// MkdirAll creates directories using OS functions
func (fs *OsFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// RemoveAll removes directories using OS functions
func (fs *OsFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// Stat returns file info using OS functions
func (fs *OsFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

// ReadFile reads a file using OS functions
func (fs *OsFileSystem) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// WriteFile writes to a file using OS functions
func (fs *OsFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

// Join joins path elements using filepath.Join
func (fs *OsFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Dir returns the directory portion of a path using filepath.Dir
func (fs *OsFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

// Base returns the base portion of a path using filepath.Base
func (fs *OsFileSystem) Base(path string) string {
	return filepath.Base(path)
}

// Abs returns the absolute path using filepath.Abs
func (fs *OsFileSystem) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

// EvalSymlinks evaluates symbolic links using filepath.EvalSymlinks
func (fs *OsFileSystem) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

// NewOsFileSystem creates a new OsFileSystem
func NewOsFileSystem() *OsFileSystem {
	return &OsFileSystem{}
}
