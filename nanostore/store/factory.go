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
