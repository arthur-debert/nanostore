package export

import (
	"fmt"
	"time"

	"github.com/arthur-debert/nanostore/formats"
	"github.com/arthur-debert/nanostore/nanostore/storage"
	"github.com/arthur-debert/nanostore/types"
)

// GenerateExportData creates the export data structure for a given set of documents
// This function generates the complete JSON representation that describes the export,
// including all database content and individual object files
func GenerateExportData(store types.Store, options ExportOptions) (*ExportData, error) {
	// Set default format if not specified
	if options.DocumentFormat == nil {
		options.DocumentFormat = formats.PlainText
	}
	// Get the documents to export based on options
	documents, err := getDocumentsToExport(store, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents to export: %w", err)
	}

	// Get the full database structure
	// We need to access the storage layer to get complete data
	storeData, err := getStoreData(store)
	if err != nil {
		return nil, fmt.Errorf("failed to get store data: %w", err)
	}

	// Generate archive filename with timestamp
	timestamp := time.Now().Format("2006-01-02T15:04:05")
	archiveFilename := fmt.Sprintf("export-%s.zip", timestamp)

	// Create database file representation
	dbFile := DatabaseFile{
		Filename: "db.json",
		Contents: storeData,
	}

	// Create object files for each document
	objectFiles := make([]ObjectFile, 0, len(documents))
	for _, doc := range documents {
		filename := generateFilename(doc, options.DocumentFormat)

		// Extract metadata (all fields except title and body)
		metadata := extractMetadata(doc)

		// Serialize document using the specified format
		content := options.DocumentFormat.Serialize(doc.Title, doc.Body, metadata)
		objectFile := ObjectFile{
			Filename: filename,
			Modified: doc.UpdatedAt,
			Created:  doc.CreatedAt,
			Content:  content,
		}
		objectFiles = append(objectFiles, objectFile)
	}

	// Create the complete export data structure
	exportData := &ExportData{
		ArchiveFilename: archiveFilename,
		Contents: ExportContent{
			DB:      dbFile,
			Objects: objectFiles,
		},
	}

	return exportData, nil
}

// getDocumentsToExport retrieves documents based on the export options
func getDocumentsToExport(store types.Store, options ExportOptions) ([]types.Document, error) {
	// If specific IDs are provided, resolve and filter by them
	if len(options.IDs) > 0 {
		return getDocumentsByIDs(store, options.IDs)
	}

	// If dimension filters are provided, use them
	if len(options.DimensionFilters) > 0 {
		listOpts := types.NewListOptions()
		listOpts.Filters = options.DimensionFilters
		return store.List(listOpts)
	}

	// If a custom filter query is provided, use DeleteWhere logic but for listing
	// For now, we'll use List with no filters to get all documents
	// TODO: Implement custom WHERE clause filtering when that functionality is available
	if options.FilterQuery != "" {
		// This would require extending the Store interface to support custom WHERE in List
		return nil, fmt.Errorf("custom filter queries not yet implemented")
	}

	// Default: export all documents
	return store.List(types.NewListOptions())
}

// getDocumentsByIDs retrieves specific documents by their IDs (either SimpleID or UUID)
func getDocumentsByIDs(store types.Store, ids []string) ([]types.Document, error) {
	var documents []types.Document

	for _, id := range ids {
		// Try to resolve as SimpleID first
		uuid, err := store.ResolveUUID(id)
		if err != nil {
			// If resolution fails, treat as UUID
			uuid = id
		}

		// Get all documents and filter by UUID
		// This is not optimal but works with the current interface
		allDocs, err := store.List(types.NewListOptions())
		if err != nil {
			return nil, fmt.Errorf("failed to list documents: %w", err)
		}

		// Find the document with matching UUID
		for _, doc := range allDocs {
			if doc.UUID == uuid {
				documents = append(documents, doc)
				break
			}
		}
	}

	return documents, nil
}

// getStoreData extracts the complete store data structure
// This requires accessing the storage layer, which we'll need to expose or work around
func getStoreData(store types.Store) (*storage.StoreData, error) {
	// For now, we'll create a StoreData from the documents we can access
	// In a real implementation, we'd need access to the storage layer
	allDocs, err := store.List(types.NewListOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to list all documents: %w", err)
	}

	storeData := &storage.StoreData{
		Documents: allDocs,
		Metadata: storage.Metadata{
			Version:   "1.0",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	return storeData, nil
}

// extractMetadata extracts all metadata from a document (everything except title and body)
func extractMetadata(doc types.Document) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Add standard fields
	metadata["uuid"] = doc.UUID
	metadata["simple_id"] = doc.SimpleID
	metadata["created_at"] = doc.CreatedAt
	metadata["updated_at"] = doc.UpdatedAt

	// Add all dimensions
	for key, value := range doc.Dimensions {
		metadata[key] = value
	}

	return metadata
}
