package imports

import (
	"time"
)

// ImportData represents the complete import structure with documents and metadata
type ImportData struct {
	Documents []ImportDocument `json:"documents"`
	Metadata  ImportMetadata   `json:"metadata"`
}

// ImportDocument represents a document to be imported
type ImportDocument struct {
	// From file content - always takes precedence
	Title string `json:"title"`
	Body  string `json:"body"`

	// From file metadata or db.json - follows precedence rules
	UUID      *string    `json:"uuid,omitempty"`
	SimpleID  *string    `json:"simple_id,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	// Dimensions from metadata
	Dimensions map[string]interface{} `json:"dimensions,omitempty"`

	// For tracking source during import
	SourceFile string `json:"source_file"`
}

// ImportMetadata contains metadata about the import operation
type ImportMetadata struct {
	Version      string    `json:"version"`
	ImportedFrom string    `json:"imported_from"` // "directory", "zip", "export"
	ImportedAt   time.Time `json:"imported_at"`
}

// ImportOptions configures the import behavior
type ImportOptions struct {
	// SkipValidation bypasses validation checks (not recommended)
	SkipValidation bool `json:"skip_validation,omitempty"`

	// IgnoreDuplicateUUIDs continues import even if duplicate UUIDs are found
	// Instead of failing, it will generate new UUIDs for duplicates
	IgnoreDuplicateUUIDs bool `json:"ignore_duplicate_uuids,omitempty"`

	// DefaultDimensions provides default dimension values for imported documents
	// These are used when a document doesn't specify a value for a dimension
	DefaultDimensions map[string]interface{} `json:"default_dimensions,omitempty"`

	// DryRun performs validation without actually importing documents
	DryRun bool `json:"dry_run,omitempty"`
}

// ImportResult contains the results of an import operation
type ImportResult struct {
	// Imported contains successfully imported documents with their new UUIDs
	Imported []ImportedDocument `json:"imported"`

	// Failed contains documents that failed to import with error details
	Failed []FailedDocument `json:"failed"`

	// Warnings contains non-fatal issues encountered during import
	Warnings []string `json:"warnings"`

	// Summary statistics
	Summary ImportSummary `json:"summary"`
}

// ImportedDocument represents a successfully imported document
type ImportedDocument struct {
	OriginalUUID string `json:"original_uuid,omitempty"` // UUID from import data (if any)
	NewUUID      string `json:"new_uuid"`                // UUID in the store
	SimpleID     string `json:"simple_id"`               // Generated simple ID
	Title        string `json:"title"`
	SourceFile   string `json:"source_file"`
}

// FailedDocument represents a document that failed to import
type FailedDocument struct {
	SourceFile string `json:"source_file"`
	Title      string `json:"title"`
	Error      string `json:"error"`
}

// ImportSummary provides statistics about the import operation
type ImportSummary struct {
	TotalDocuments    int       `json:"total_documents"`
	SuccessfulImports int       `json:"successful_imports"`
	FailedImports     int       `json:"failed_imports"`
	WarningsCount     int       `json:"warnings_count"`
	DuplicateUUIDs    int       `json:"duplicate_uuids"`
	ProcessingTime    string    `json:"processing_time"`
	StartedAt         time.Time `json:"started_at"`
	CompletedAt       time.Time `json:"completed_at"`
}
