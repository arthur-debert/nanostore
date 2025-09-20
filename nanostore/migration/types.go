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
	DryRun  bool
	Verbose bool
}

// Error codes
const (
	CodeSuccess = iota
	CodeValidationError
	CodeExecutionError
	CodePartialFailure
)
