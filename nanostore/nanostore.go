// Package nanostore provides a document and ID store library that uses JSON file storage
// to manage document storage and dynamically generate user-facing, hierarchical IDs.
//
// This package provides a clean approach to document management with configurable
// ID schemes, automatic hierarchical organization, and human-friendly ID generation.
package nanostore

import (
	"github.com/arthur-debert/nanostore/nanostore/export"
	"github.com/arthur-debert/nanostore/nanostore/store"
)

// Store is the main interface for document storage
type Store = store.Store

// TestStore extends Store with testing utilities
type TestStore = store.TestStore

// New creates a new Store instance with the specified dimension configuration
// The store uses a JSON file backend with file locking for concurrent access
func New(filePath string, config Config) (Store, error) {
	return store.New(filePath, &config)
}

// Export functions

// ExportOptions is an alias for export.ExportOptions
type ExportOptions = export.ExportOptions

// ExportMetadata is an alias for export.ExportMetadata
type ExportMetadata = export.ExportMetadata

// DocumentInfo is an alias for export.DocumentInfo
type DocumentInfo = export.DocumentInfo

// Export creates a complete export archive for the specified documents
// Returns the path to the created archive in a temporary directory
func Export(store Store, options ExportOptions) (string, error) {
	return export.Export(store, options)
}

// ExportToPath creates an export archive at the specified path
func ExportToPath(store Store, options ExportOptions, outputPath string) error {
	return export.ExportToPath(store, options, outputPath)
}

// GetExportMetadata returns metadata about what would be exported without creating an archive
func GetExportMetadata(store Store, options ExportOptions) (*ExportMetadata, error) {
	return export.GetExportMetadata(store, options)
}
