// Package export provides functionality to export nanostore data to zip archives.
// It supports exporting by specific IDs, dimension filters, or custom queries.
//
// The export process happens in two main steps:
// 1. Generate export data - creates a JSON representation of all export content
// 2. Create archive - generates the actual zip file from the export data
//
// This design allows for comprehensive testing without file system operations
// and provides flexibility for different export scenarios.
package export

import (
	"fmt"

	"github.com/arthur-debert/nanostore/types"
)

// Export creates a complete export archive for the specified documents
// This is the main entry point for exporting data from a nanostore
//
// The function will:
// 1. Generate export data based on the provided options
// 2. Create a zip archive containing the database and all object files
// 3. Return the path to the created archive
//
// Example usage:
//
//	// Export all documents
//	archivePath, err := Export(store, ExportOptions{})
//
//	// Export specific documents
//	archivePath, err := Export(store, ExportOptions{IDs: []string{"1", "c2"}})
//
//	// Export by dimension filter
//	archivePath, err := Export(store, ExportOptions{
//	    DimensionFilters: map[string]interface{}{"status": "completed"},
//	})
func Export(store types.Store, options ExportOptions) (string, error) {
	// Generate the export data
	exportData, err := GenerateExportData(store, options)
	if err != nil {
		return "", fmt.Errorf("failed to generate export data: %w", err)
	}

	// Create the archive in a temporary directory
	archivePath, err := CreateExportArchiveToTempDir(exportData)
	if err != nil {
		return "", fmt.Errorf("failed to create export archive: %w", err)
	}

	return archivePath, nil
}

// ExportToPath creates an export archive at the specified path
// This variant allows you to specify exactly where the archive should be created
//
// Example usage:
//
//	err := ExportToPath(store, ExportOptions{}, "/path/to/my-export.zip")
func ExportToPath(store types.Store, options ExportOptions, outputPath string) error {
	// Generate the export data
	exportData, err := GenerateExportData(store, options)
	if err != nil {
		return fmt.Errorf("failed to generate export data: %w", err)
	}

	// Create the archive at the specified path
	err = CreateExportArchive(exportData, outputPath)
	if err != nil {
		return fmt.Errorf("failed to create export archive: %w", err)
	}

	return nil
}

// GetExportMetadata returns metadata about what would be exported without creating an archive
// This is useful for previewing exports or validating export options
//
// Returns information about:
// - Number of documents that would be exported
// - List of document IDs and titles
// - Estimated archive size (approximate)
func GetExportMetadata(store types.Store, options ExportOptions) (*ExportMetadata, error) {
	// Get the documents that would be exported
	documents, err := getDocumentsToExport(store, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents to export: %w", err)
	}

	metadata := &ExportMetadata{
		DocumentCount: len(documents),
		Documents:     make([]DocumentInfo, 0, len(documents)),
	}

	var totalContentSize int64
	for _, doc := range documents {
		docInfo := DocumentInfo{
			UUID:     doc.UUID,
			SimpleID: doc.SimpleID,
			Title:    doc.Title,
			Filename: generateFilename(doc),
		}
		metadata.Documents = append(metadata.Documents, docInfo)
		totalContentSize += int64(len(doc.Body))
	}

	// Rough estimate of archive size (content + overhead)
	metadata.EstimatedSizeBytes = totalContentSize + int64(len(documents)*200) // rough overhead per file

	return metadata, nil
}

// ExportMetadata contains information about an export operation
type ExportMetadata struct {
	DocumentCount      int            `json:"document_count"`
	Documents          []DocumentInfo `json:"documents"`
	EstimatedSizeBytes int64          `json:"estimated_size_bytes"`
}

// DocumentInfo contains basic information about a document in an export
type DocumentInfo struct {
	UUID     string `json:"uuid"`
	SimpleID string `json:"simple_id"`
	Title    string `json:"title"`
	Filename string `json:"filename"` // The filename it would have in the archive
}
