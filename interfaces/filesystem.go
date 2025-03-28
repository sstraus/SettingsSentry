package interfaces

import (
	"io"
	"os"
	"path/filepath"
)

type FileSystem interface {
	Open(name string) (File, error)
	Create(name string) (File, error)
	ReadDir(dirname string) ([]os.DirEntry, error)
	MkdirAll(path string, perm os.FileMode) error
	RemoveAll(path string) error
	Stat(name string) (os.FileInfo, error)
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error

	Join(elem ...string) string
	Dir(path string) string
	Base(path string) string
	Abs(path string) (string, error)
	EvalSymlinks(path string) (string, error)
}

type File interface {
	io.Reader
	io.Writer
	io.Closer
}

type OsFileSystem struct{}

type OsFile struct {
	*os.File
}

func (fs *OsFileSystem) Open(name string) (File, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return &OsFile{File: file}, nil
}

func (fs *OsFileSystem) Create(name string) (File, error) {
	file, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	return &OsFile{File: file}, nil
}

func (fs *OsFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	return os.ReadDir(dirname)
}

func (fs *OsFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *OsFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (fs *OsFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (fs *OsFileSystem) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func (fs *OsFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

func (fs *OsFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (fs *OsFileSystem) Dir(path string) string {
	return filepath.Dir(path)
}

func (fs *OsFileSystem) Base(path string) string {
	return filepath.Base(path)
}

func (fs *OsFileSystem) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

func (fs *OsFileSystem) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

func NewOsFileSystem() *OsFileSystem {
	return &OsFileSystem{}
}
