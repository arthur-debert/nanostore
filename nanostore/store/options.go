package store

import "time"

// JSONFileStoreOption is a function that modifies JSONFileStore configuration
type JSONFileStoreOption func(*jsonFileStore)

// WithFileSystem sets a custom FileSystem implementation
func WithFileSystem(fs FileSystem) JSONFileStoreOption {
	return func(s *jsonFileStore) {
		s.fs = fs
	}
}

// WithFileLockFactory sets a custom FileLockFactory implementation
func WithFileLockFactory(factory FileLockFactory) JSONFileStoreOption {
	return func(s *jsonFileStore) {
		s.lockFactory = factory
	}
}

// WithTimeFunc sets a custom time function for testing
func WithTimeFunc(fn func() time.Time) JSONFileStoreOption {
	return func(s *jsonFileStore) {
		s.timeFunc = fn
	}
}
