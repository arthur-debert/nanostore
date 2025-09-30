package imports

import (
	"fmt"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/types"
	"github.com/google/uuid"
)

// ProcessImportData imports documents from ImportData into the store
// This is the core import logic that validates and imports documents
func ProcessImportData(store types.Store, data ImportData, options ImportOptions) (*ImportResult, error) {
	startTime := time.Now()

	result := &ImportResult{
		Imported: make([]ImportedDocument, 0),
		Failed:   make([]FailedDocument, 0),
		Warnings: make([]string, 0),
		Summary: ImportSummary{
			TotalDocuments: len(data.Documents),
			StartedAt:      startTime,
		},
	}

	// Validate import data if not skipped
	if !options.SkipValidation {
		if err := validateImportData(data); err != nil {
			return nil, fmt.Errorf("import validation failed: %w", err)
		}
	}

	// Get existing UUIDs to check for duplicates
	existingUUIDs, err := getExistingUUIDs(store)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing UUIDs: %w", err)
	}

	// Track UUIDs being imported to detect duplicates within import data
	importingUUIDs := make(map[string]bool)

	// Process each document
	for _, doc := range data.Documents {
		if err := processDocument(store, doc, options, existingUUIDs, importingUUIDs, result); err != nil {
			// If it's a critical error, stop processing
			if !isContinuableError(err) {
				return result, err
			}
			// Otherwise, record the failure and continue
			result.Failed = append(result.Failed, FailedDocument{
				SourceFile: doc.SourceFile,
				Title:      doc.Title,
				Error:      err.Error(),
			})
		}
	}

	// Update summary
	result.Summary.SuccessfulImports = len(result.Imported)
	result.Summary.FailedImports = len(result.Failed)
	result.Summary.WarningsCount = len(result.Warnings)
	result.Summary.CompletedAt = time.Now()
	result.Summary.ProcessingTime = result.Summary.CompletedAt.Sub(startTime).String()

	return result, nil
}

// processDocument processes a single document for import
func processDocument(store types.Store, doc ImportDocument, options ImportOptions,
	existingUUIDs map[string]bool, importingUUIDs map[string]bool, result *ImportResult) error {

	// Validate document
	if err := validateDocument(doc); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Handle UUID
	var finalUUID string
	if doc.UUID != nil && *doc.UUID != "" {
		// Check for duplicate UUID
		if existingUUIDs[*doc.UUID] || importingUUIDs[*doc.UUID] {
			result.Summary.DuplicateUUIDs++
			if !options.IgnoreDuplicateUUIDs {
				return fmt.Errorf("duplicate UUID: %s", *doc.UUID)
			}
			// Generate new UUID
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Duplicate UUID %s found for document '%s', generating new UUID", *doc.UUID, doc.Title))
			finalUUID = uuid.New().String()
		} else {
			finalUUID = *doc.UUID
		}
	} else {
		// Generate new UUID
		finalUUID = uuid.New().String()
	}

	// Mark UUID as being imported
	importingUUIDs[finalUUID] = true

	// Prepare dimensions with defaults
	dimensions := prepareDimensions(doc.Dimensions, options.DefaultDimensions)

	// Add timestamps to dimensions if provided (as strings for compatibility)
	if doc.CreatedAt != nil {
		dimensions["_created_at"] = doc.CreatedAt.Format(time.RFC3339)
	}
	if doc.UpdatedAt != nil {
		dimensions["_updated_at"] = doc.UpdatedAt.Format(time.RFC3339)
	}

	// Handle body in dimensions
	if doc.Body != "" {
		dimensions["_body"] = doc.Body
	}

	// Dry run - don't actually import
	if options.DryRun {
		result.Imported = append(result.Imported, ImportedDocument{
			OriginalUUID: getOriginalUUID(doc),
			NewUUID:      finalUUID,
			SimpleID:     "[dry-run]",
			Title:        doc.Title,
			SourceFile:   doc.SourceFile,
		})
		return nil
	}

	// Perform the actual import
	createdUUID, err := store.Add(doc.Title, dimensions)
	if err != nil {
		return fmt.Errorf("failed to add document: %w", err)
	}

	// Get the created document to retrieve its SimpleID
	docs, err := store.List(types.ListOptions{
		Filters: map[string]interface{}{"uuid": createdUUID},
	})
	if err != nil || len(docs) == 0 {
		return fmt.Errorf("failed to retrieve created document")
	}

	result.Imported = append(result.Imported, ImportedDocument{
		OriginalUUID: getOriginalUUID(doc),
		NewUUID:      createdUUID,
		SimpleID:     docs[0].SimpleID,
		Title:        doc.Title,
		SourceFile:   doc.SourceFile,
	})

	return nil
}

// validateImportData validates the entire import data structure
func validateImportData(data ImportData) error {
	if len(data.Documents) == 0 {
		return fmt.Errorf("no documents to import")
	}

	// Check for duplicate UUIDs within import data
	uuidMap := make(map[string][]string) // UUID -> source files
	for _, doc := range data.Documents {
		if doc.UUID != nil && *doc.UUID != "" {
			uuidMap[*doc.UUID] = append(uuidMap[*doc.UUID], doc.SourceFile)
		}
	}

	for uuid, sources := range uuidMap {
		if len(sources) > 1 {
			return fmt.Errorf("duplicate UUID %s found in files: %s", uuid, strings.Join(sources, ", "))
		}
	}

	return nil
}

// validateDocument validates a single document
func validateDocument(doc ImportDocument) error {
	// Title is required
	if strings.TrimSpace(doc.Title) == "" {
		return fmt.Errorf("empty title")
	}

	// Validate dates if provided
	if doc.CreatedAt != nil && doc.CreatedAt.IsZero() {
		return fmt.Errorf("invalid created_at date")
	}
	if doc.UpdatedAt != nil && doc.UpdatedAt.IsZero() {
		return fmt.Errorf("invalid updated_at date")
	}

	// Validate UUID format if provided
	if doc.UUID != nil && *doc.UUID != "" {
		if _, err := uuid.Parse(*doc.UUID); err != nil {
			return fmt.Errorf("invalid UUID format: %s", *doc.UUID)
		}
	}

	return nil
}

// getExistingUUIDs retrieves all existing UUIDs from the store
func getExistingUUIDs(store types.Store) (map[string]bool, error) {
	allDocs, err := store.List(types.NewListOptions())
	if err != nil {
		return nil, err
	}

	uuids := make(map[string]bool)
	for _, doc := range allDocs {
		uuids[doc.UUID] = true
	}

	return uuids, nil
}

// prepareDimensions merges document dimensions with defaults
func prepareDimensions(docDimensions, defaultDimensions map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Start with defaults
	for k, v := range defaultDimensions {
		result[k] = v
	}

	// Override with document-specific dimensions
	for k, v := range docDimensions {
		result[k] = v
	}

	return result
}

// getOriginalUUID safely retrieves the original UUID from a document
func getOriginalUUID(doc ImportDocument) string {
	if doc.UUID != nil {
		return *doc.UUID
	}
	return ""
}

// isContinuableError determines if an error allows import to continue
func isContinuableError(err error) bool {
	// For now, validation errors and duplicate UUIDs are continuable
	// Other errors (like store errors) are not
	errStr := err.Error()
	return strings.Contains(errStr, "validation failed") ||
		strings.Contains(errStr, "duplicate UUID") ||
		strings.Contains(errStr, "empty title")
}
