package engine

import "github.com/arthur-debert/nanostore/nanostore/types"

// New creates a new configurable store with the given configuration
func New(dbPath string, config types.Config) (*configurableStore, error) {
	return NewConfigurable(dbPath, config)
}
