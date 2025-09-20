package store

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DirEntry represents a directory entry (compatible with fs.DirEntry)
type DirEntry interface {
	Name() string
	IsDir() bool
	Type() fs.FileMode
	Info() (fs.FileInfo, error)
}

// FileSystemExt extends FileSystem with directory operations
type FileSystemExt interface {
	FileSystem

	// ReadDir reads the directory and returns entries
	ReadDir(name string) ([]fs.DirEntry, error)

	// MkdirAll creates a directory and all necessary parents
	MkdirAll(path string, perm fs.FileMode) error
}

// OSFileSystemExt is the extended OS implementation
type OSFileSystemExt struct {
	OSFileSystem
}

// ReadDir implements FileSystemExt.ReadDir
func (fs *OSFileSystemExt) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

// MkdirAll implements FileSystemExt.MkdirAll
func (fs *OSFileSystemExt) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

// MockFileSystemExt extends MockFileSystem with directory operations
type MockFileSystemExt struct {
	*MockFileSystem
}

// NewMockFileSystemExt creates a new extended mock file system
func NewMockFileSystemExt() *MockFileSystemExt {
	return &MockFileSystemExt{
		MockFileSystem: NewMockFileSystem(),
	}
}

// ReadDir implements FileSystemExt.ReadDir
func (mfs *MockFileSystemExt) ReadDir(name string) ([]fs.DirEntry, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	// Normalize the directory path
	dirPath := filepath.Clean(name)
	if dirPath == "." {
		dirPath = ""
	}

	entries := []fs.DirEntry{}
	seen := make(map[string]bool)

	for path := range mfs.files {
		// Check if this file is in the requested directory
		dir := filepath.Dir(path)
		if dir == dirPath || (dirPath == "" && dir == ".") {
			filename := filepath.Base(path)
			if !seen[filename] {
				seen[filename] = true
				entries = append(entries, &mockDirEntry{
					name:  filename,
					isDir: false, // In our simple mock, everything is a file
				})
			}
		}

		// Check for subdirectories
		if strings.HasPrefix(path, dirPath+string(filepath.Separator)) || dirPath == "" {
			relativePath := path
			if dirPath != "" {
				relativePath = strings.TrimPrefix(path, dirPath+string(filepath.Separator))
			}

			parts := strings.Split(relativePath, string(filepath.Separator))
			if len(parts) > 1 {
				// This is in a subdirectory
				subdir := parts[0]
				if !seen[subdir] {
					seen[subdir] = true
					entries = append(entries, &mockDirEntry{
						name:  subdir,
						isDir: true,
					})
				}
			}
		}
	}

	if len(entries) == 0 && len(mfs.files) > 0 {
		// Directory doesn't exist or is empty
		return nil, os.ErrNotExist
	}

	return entries, nil
}

// MkdirAll implements FileSystemExt.MkdirAll
func (mfs *MockFileSystemExt) MkdirAll(path string, perm fs.FileMode) error {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	// In our mock, we just create a marker file to indicate the directory exists
	markerPath := filepath.Join(path, ".dir")
	mfs.files[markerPath] = &mockFile{
		content: []byte{},
		mode:    perm | fs.ModeDir,
	}

	return nil
}

// mockDirEntry implements fs.DirEntry
type mockDirEntry struct {
	name  string
	isDir bool
}

func (e *mockDirEntry) Name() string { return e.name }
func (e *mockDirEntry) IsDir() bool  { return e.isDir }
func (e *mockDirEntry) Type() fs.FileMode {
	if e.isDir {
		return fs.ModeDir
	}
	return 0
}
func (e *mockDirEntry) Info() (fs.FileInfo, error) {
	return mockFileInfo{
		name: e.name,
		mode: e.Type(),
	}, nil
}
