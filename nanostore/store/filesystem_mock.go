package store

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MockFileSystem provides an in-memory implementation of FileSystem for testing
type MockFileSystem struct {
	mu    sync.RWMutex
	files map[string]*mockFile

	// Optional callbacks for simulating errors
	StatError      error
	ReadFileError  error
	WriteFileError error
	RenameError    error
	RemoveError    error
}

type mockFile struct {
	content []byte
	mode    fs.FileMode
	modTime time.Time
}

// mockFileInfo implements fs.FileInfo
type mockFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

func (fi mockFileInfo) Name() string       { return fi.name }
func (fi mockFileInfo) Size() int64        { return fi.size }
func (fi mockFileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi mockFileInfo) ModTime() time.Time { return fi.modTime }
func (fi mockFileInfo) IsDir() bool        { return fi.mode.IsDir() }
func (fi mockFileInfo) Sys() interface{}   { return nil }

// NewMockFileSystem creates a new mock file system
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string]*mockFile),
	}
}

// Stat implements FileSystem.Stat
func (fs *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if fs.StatError != nil {
		return nil, fs.StatError
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	file, exists := fs.files[name]
	if !exists {
		return nil, os.ErrNotExist
	}

	return mockFileInfo{
		name:    filepath.Base(name),
		size:    int64(len(file.content)),
		mode:    file.mode,
		modTime: file.modTime,
	}, nil
}

// ReadFile implements FileSystem.ReadFile
func (fs *MockFileSystem) ReadFile(name string) ([]byte, error) {
	if fs.ReadFileError != nil {
		return nil, fs.ReadFileError
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	file, exists := fs.files[name]
	if !exists {
		return nil, os.ErrNotExist
	}

	// Return a copy to prevent external modifications
	content := make([]byte, len(file.content))
	copy(content, file.content)
	return content, nil
}

// WriteFile implements FileSystem.WriteFile
func (fs *MockFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	if fs.WriteFileError != nil {
		return fs.WriteFileError
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Make a copy of the data to prevent external modifications
	content := make([]byte, len(data))
	copy(content, data)

	fs.files[name] = &mockFile{
		content: content,
		mode:    perm,
		modTime: time.Now(),
	}

	return nil
}

// Rename implements FileSystem.Rename
func (fs *MockFileSystem) Rename(oldpath, newpath string) error {
	if fs.RenameError != nil {
		return fs.RenameError
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	file, exists := fs.files[oldpath]
	if !exists {
		return os.ErrNotExist
	}

	// Move file to new location (overwrites if exists, like os.Rename)
	fs.files[newpath] = file
	delete(fs.files, oldpath)

	return nil
}

// Remove implements FileSystem.Remove
func (fs *MockFileSystem) Remove(name string) error {
	if fs.RemoveError != nil {
		return fs.RemoveError
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, exists := fs.files[name]; !exists {
		return os.ErrNotExist
	}

	delete(fs.files, name)
	return nil
}

// FileExists is a helper method for testing
func (fs *MockFileSystem) FileExists(name string) bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	_, exists := fs.files[name]
	return exists
}

// GetFileContent is a helper method for testing
func (fs *MockFileSystem) GetFileContent(name string) ([]byte, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	file, exists := fs.files[name]
	if !exists {
		return nil, false
	}

	// Return a copy
	content := make([]byte, len(file.content))
	copy(content, file.content)
	return content, true
}
