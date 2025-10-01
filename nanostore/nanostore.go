// Package nanostore provides a document and ID store library that uses JSON file storage
// to manage document storage and dynamically generate user-facing, hierarchical IDs.
//
// RECOMMENDED API: Use api.NewFromType[T]() for new applications. This package
// contains the deprecated Direct Store API. See docs/migration-direct-to-typed.md
// for migration guidance.
//
// The TypedStore API provides type safety, automatic configuration from struct tags,
// and a superior developer experience with compile-time checking.
package nanostore

import (
	"github.com/arthur-debert/nanostore/nanostore/export"
	"github.com/arthur-debert/nanostore/nanostore/store"
)

// Store is the main interface for document storage
//
// DEPRECATED: Direct usage of Store interface is deprecated.
// Use api.TypedStore[T] instead for type safety and better developer experience.
// See docs/migration-direct-to-typed.md for migration guide.
type Store = store.Store

// TestStore extends Store with testing utilities
//
// DEPRECATED: Direct usage of TestStore interface is deprecated.
// Use api.TypedStore[T].SetTimeFunc() instead.
// See docs/migration-direct-to-typed.md for migration guide.
type TestStore = store.TestStore

// New creates a new Store instance with the specified dimension configuration
// The store uses a JSON file backend with file locking for concurrent access
//
// DEPRECATED: This Direct Store API is deprecated in favor of the TypedStore API.
// Use api.NewFromType[T]() instead for type safety and automatic configuration.
//
// Migration example:
//
//	// Old (deprecated):
//	config := nanostore.Config{...}
//	store, err := nanostore.New(filePath, config)
//
//	// New (recommended):
//	type MyDoc struct {
//	  nanostore.Document
//	  Status string `values:"pending,done" default:"pending"`
//	}
//	store, err := api.NewFromType[MyDoc](filePath)
//
// See docs/migration-direct-to-typed.md for complete migration guide.
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
