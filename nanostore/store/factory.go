package store

import (
	"github.com/arthur-debert/nanostore/internal/validation"
	"github.com/arthur-debert/nanostore/types"
)

// Config represents the store configuration
type Config interface {
	// GetDimensionSet returns the dimension configuration
	GetDimensionSet() *types.DimensionSet
}

// New creates a new Store instance with the specified dimension configuration
// The store uses a JSON file backend with file locking for concurrent access
func New(filePath string, config Config) (Store, error) {
	// First validate the configuration
	if err := validation.Validate(config.GetDimensionSet()); err != nil {
		return nil, err
	}
	return newJSONFileStore(filePath, config)
}

// NewWithOptions creates a new Store instance with custom options
// This is useful for testing with mock file systems and locks
func NewWithOptions(filePath string, config Config, opts ...JSONFileStoreOption) (Store, error) {
	// First validate the configuration
	if err := validation.Validate(config.GetDimensionSet()); err != nil {
		return nil, err
	}
	return newJSONFileStore(filePath, config, opts...)
}

// NewHybrid creates a new Store instance with hybrid body storage
// Bodies larger than embedSizeLimit will be stored in separate files
func NewHybrid(filePath string, config Config, embedSizeLimit int64) (Store, error) {
	// First validate the configuration
	if err := validation.Validate(config.GetDimensionSet()); err != nil {
		return nil, err
	}
	return newHybridJSONFileStore(filePath, config)
}

// NewHybridWithOptions creates a new hybrid Store instance with custom options
func NewHybridWithOptions(filePath string, config Config, opts ...HybridJSONFileStoreOption) (Store, error) {
	// First validate the configuration
	if err := validation.Validate(config.GetDimensionSet()); err != nil {
		return nil, err
	}
	return newHybridJSONFileStore(filePath, config, opts...)
}
