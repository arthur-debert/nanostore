package migration

import (
	"time"

	"github.com/arthur-debert/nanostore/types"
)

// MessageLevel represents the severity of a message
type MessageLevel int

const (
	LevelDebug MessageLevel = iota
	LevelInfo
	LevelWarning
	LevelError
)

// Message represents a single output message from a migration
type Message struct {
	Level   MessageLevel
	Text    string
	Details map[string]interface{} // Optional structured data
}

// Result encapsulates the outcome of a migration operation
type Result struct {
	Success      bool
	Code         int // 0 = success, >0 = specific error codes
	Messages     []Message
	ModifiedDocs []string // UUIDs of modified documents
	Stats        Stats    // Operation statistics
}

// Stats provides migration statistics
type Stats struct {
	TotalDocs    int
	ModifiedDocs int
	SkippedDocs  int
	Duration     time.Duration
}

// MigrationContext holds the state for a migration operation
type MigrationContext struct {
	Documents []types.Document
	Config    types.Config
	DryRun    bool
}

// Options configures migration behavior
type Options struct {
	DryRun      bool
	Verbose     bool
	IsDataField bool      // For add field operation
	FieldType   FieldType // Specifies which field type to operate on
}

// FieldType specifies which type of field to operate on
type FieldType int

const (
	FieldTypeAuto      FieldType = iota // Default: operate on field found (dimension or data)
	FieldTypeDimension                  // Only operate on dimension fields
	FieldTypeData                       // Only operate on data fields
	FieldTypeBoth                       // Explicitly operate on both dimension and data fields
)

// Error codes
const (
	CodeSuccess = iota
	CodeValidationError
	CodeExecutionError
	CodePartialFailure
)

// Command represents a migration command
type Command interface {
	// Validate checks if the command can be executed
	Validate(ctx *MigrationContext) []Message

	// Execute runs the migration
	Execute(ctx *MigrationContext) *Result

	// Description returns a human-readable description
	Description() string
}
