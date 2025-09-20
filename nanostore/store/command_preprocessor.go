package store

import (
	"github.com/arthur-debert/nanostore/nanostore/ids"
	"github.com/arthur-debert/nanostore/types"
)

// commandPreprocessor is a thin wrapper around the ids.CommandPreprocessor
// that implements the store's IDResolver interface
type commandPreprocessor struct {
	store     *jsonFileStore
	processor *ids.CommandPreprocessor
}

// newCommandPreprocessor creates a new command preprocessor
func newCommandPreprocessor(store *jsonFileStore) *commandPreprocessor {
	// Create field resolver
	fieldResolver := types.NewFieldResolver(store.dimensionSet)

	// Create the actual preprocessor with store as the ID resolver
	processor := ids.NewCommandPreprocessor(store, fieldResolver)

	return &commandPreprocessor{
		store:     store,
		processor: processor,
	}
}

// preprocessCommand delegates to the actual preprocessor
func (cp *commandPreprocessor) preprocessCommand(cmd interface{}) error {
	return cp.processor.PreprocessCommand(cmd)
}

// ResolveID implements the IDResolver interface for jsonFileStore
func (s *jsonFileStore) ResolveID(simpleID string) (string, error) {
	return s.resolveUUIDInternal(simpleID)
}
