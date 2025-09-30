package nanostore

import (
	imports "github.com/arthur-debert/nanostore/nanostore/import"
	"github.com/arthur-debert/nanostore/types"
)

// ImportFromPath imports documents from a file path (directory or zip file)
// This is a convenience function that wraps the import package functionality
func ImportFromPath(store types.Store, path string, options imports.ImportOptions) (*imports.ImportResult, error) {
	return imports.ImportFromPath(store, path, options)
}

// ProcessImportData imports documents from an ImportData structure
// This allows for programmatic import without reading from files
func ProcessImportData(store types.Store, data imports.ImportData, options imports.ImportOptions) (*imports.ImportResult, error) {
	return imports.ProcessImportData(store, data, options)
}

// DefaultImportOptions returns sensible default import options
func DefaultImportOptions() imports.ImportOptions {
	return imports.DefaultImportOptions()
}

// Re-export types for convenience
type ImportData = imports.ImportData
type ImportDocument = imports.ImportDocument
type ImportMetadata = imports.ImportMetadata
type ImportOptions = imports.ImportOptions
type ImportResult = imports.ImportResult
type ImportedDocument = imports.ImportedDocument
type FailedDocument = imports.FailedDocument
type ImportSummary = imports.ImportSummary
