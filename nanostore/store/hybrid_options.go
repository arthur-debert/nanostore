package store

import "time"

// HybridJSONFileStoreOption is a function that modifies hybrid store configuration
type HybridJSONFileStoreOption func(*hybridJSONFileStore)

// WithFileSystemExt sets a custom FileSystemExt implementation for hybrid store
func WithFileSystemExt(fs FileSystemExt) HybridJSONFileStoreOption {
	return func(s *hybridJSONFileStore) {
		s.fs = fs
	}
}

// WithHybridFileLockFactory sets a custom FileLockFactory for hybrid store
func WithHybridFileLockFactory(factory FileLockFactory) HybridJSONFileStoreOption {
	return func(s *hybridJSONFileStore) {
		s.lockFactory = factory
	}
}

// WithHybridTimeFunc sets a custom time function for hybrid store
func WithHybridTimeFunc(fn func() time.Time) HybridJSONFileStoreOption {
	return func(s *hybridJSONFileStore) {
		s.timeFunc = fn
	}
}

// WithEmbedSizeLimit sets the maximum size for embedded bodies
func WithEmbedSizeLimit(limit int64) HybridJSONFileStoreOption {
	return func(s *hybridJSONFileStore) {
		// Set the embed limit in metadata - this will be used when creating body storage
		if s.hybridData != nil {
			s.hybridData.Metadata.BodyStorageConfig.EmbedSizeLimit = limit
		}
		// Also update body storage if it already exists (for late configuration)
		if s.bodyStorage != nil {
			if hbs, ok := s.bodyStorage.(*HybridBodyStorage); ok {
				hbs.embedSizeLimit = limit
			}
		}
	}
}

// WithBodiesDir sets the subdirectory name for body files
func WithBodiesDir(dir string) HybridJSONFileStoreOption {
	return func(s *hybridJSONFileStore) {
		if s.hybridData != nil {
			s.hybridData.Metadata.BodyStorageConfig.BodiesDir = dir
		}
		// Also update body storage if it exists
		if s.bodyStorage != nil {
			if hbs, ok := s.bodyStorage.(*HybridBodyStorage); ok {
				hbs.bodiesDir = dir
			}
		}
	}
}
