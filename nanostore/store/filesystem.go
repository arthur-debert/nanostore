package store

import (
	"io/fs"
	"os"
)

// FileSystem defines the interface for file system operations
// This abstraction allows for easy mocking in tests and potential
// alternative storage backends in the future.
type FileSystem interface {
	// Stat returns file info for the given path
	Stat(name string) (fs.FileInfo, error)

	// ReadFile reads the entire file and returns its contents
	ReadFile(name string) ([]byte, error)

	// WriteFile writes data to a file with the specified permissions
	WriteFile(name string, data []byte, perm fs.FileMode) error

	// Rename renames (moves) a file from oldpath to newpath
	Rename(oldpath, newpath string) error

	// Remove removes the named file
	Remove(name string) error
}

// OSFileSystem is the default implementation using the os package
type OSFileSystem struct{}

// Stat implements FileSystem.Stat
func (fs *OSFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

// ReadFile implements FileSystem.ReadFile
func (fs *OSFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// WriteFile implements FileSystem.WriteFile
func (fs *OSFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// Rename implements FileSystem.Rename
func (fs *OSFileSystem) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

// Remove implements FileSystem.Remove
func (fs *OSFileSystem) Remove(name string) error {
	return os.Remove(name)
}
