// Package nanostore provides core types and interfaces for the nanostore document store.
//
// For creating and using document stores, use the api package:
//
//	import "github.com/arthur-debert/nanostore/nanostore/api"
//
//	type Task struct {
//	    nanostore.Document
//	    Status string `values:"pending,done" default:"pending"`
//	}
//
//	store, err := api.New[Task]("tasks.json")
//
// See docs/migration-to-v0.11.md for migration from previous versions.
package nanostore

import (
	"github.com/arthur-debert/nanostore/nanostore/export"
	"github.com/arthur-debert/nanostore/types"
)

// Export functions

// ExportOptions is an alias for export.ExportOptions
type ExportOptions = export.ExportOptions

// ExportMetadata is an alias for export.ExportMetadata
type ExportMetadata = export.ExportMetadata

// DocumentInfo is an alias for export.DocumentInfo
type DocumentInfo = export.DocumentInfo

// Export creates a complete export archive for the specified documents
// Returns the path to the created archive in a temporary directory
func Export(store types.Store, options ExportOptions) (string, error) {
	return export.Export(store, options)
}

// ExportToPath creates an export archive at the specified path
func ExportToPath(store types.Store, options ExportOptions, outputPath string) error {
	return export.ExportToPath(store, options, outputPath)
}

// GetExportMetadata returns metadata about what would be exported without creating an archive
func GetExportMetadata(store types.Store, options ExportOptions) (*ExportMetadata, error) {
	return export.GetExportMetadata(store, options)
}
