package search

import (
	"github.com/arthur-debert/nanostore/types"
)

// NanostoreAdapter adapts a nanostore Store to work as a DocumentProvider
type NanostoreAdapter struct {
	store types.Store
}

// NewNanostoreAdapter creates a new adapter for a nanostore Store
func NewNanostoreAdapter(store types.Store) *NanostoreAdapter {
	return &NanostoreAdapter{
		store: store,
	}
}

// GetDocuments implements DocumentProvider by using the store's List method
func (a *NanostoreAdapter) GetDocuments(filters map[string]interface{}) ([]types.Document, error) {
	opts := types.ListOptions{
		Filters: filters,
	}
	return a.store.List(opts)
}

// SearchWithStore is a convenience function to search using a nanostore Store directly
func SearchWithStore(store types.Store, options SearchOptions, filters map[string]interface{}) ([]SearchResult, error) {
	adapter := NewNanostoreAdapter(store)
	engine := NewEngine(adapter)
	return engine.Search(options, filters)
}
