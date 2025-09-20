package store

import (
	"github.com/arthur-debert/nanostore/nanostore/ids"
	"github.com/arthur-debert/nanostore/types"
)

// hybridCommandPreprocessor is for hybrid stores
type hybridCommandPreprocessor struct {
	store     *hybridJSONFileStore
	processor *ids.CommandPreprocessor
}

// newHybridCommandPreprocessor creates a new command preprocessor for hybrid stores
func newHybridCommandPreprocessor(store *hybridJSONFileStore) *hybridCommandPreprocessor {
	// Create field resolver
	fieldResolver := types.NewFieldResolver(store.dimensionSet)

	// Create the actual preprocessor with store as the ID resolver
	processor := ids.NewCommandPreprocessor(store, fieldResolver)

	return &hybridCommandPreprocessor{
		store:     store,
		processor: processor,
	}
}

// preprocessCommand delegates to the actual preprocessor
func (cp *hybridCommandPreprocessor) preprocessCommand(cmd interface{}) error {
	return cp.processor.PreprocessCommand(cmd)
}
