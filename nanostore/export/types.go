package export

import (
	"time"

	"github.com/arthur-debert/nanostore/formats"
)

// ExportData represents the complete export structure with all information
// needed to recreate the full export including database and file contents
type ExportData struct {
	ArchiveFilename string        `json:"archive-filename"`
	Contents        ExportContent `json:"contents"`
}

// ExportContent contains the database and object files to be exported
type ExportContent struct {
	DB      DatabaseFile `json:"db"`
	Objects []ObjectFile `json:"objects"`
}

// DatabaseFile represents the database export file
type DatabaseFile struct {
	Filename string      `json:"filename"`
	Contents interface{} `json:"contents"` // Full JSON object contents for the db
}

// ObjectFile represents an individual object file in the export
type ObjectFile struct {
	Filename string    `json:"filename"`
	Modified time.Time `json:"modified"`
	Created  time.Time `json:"created"`
	Content  string    `json:"content"` // Content as string
}

// ExportOptions configures what data to export
type ExportOptions struct {
	// IDs specifies explicit document IDs/UUIDs to export
	// Can be either simple IDs (e.g., "1", "c2") or UUIDs
	IDs []string `json:"ids,omitempty"`

	// FilterQuery specifies a custom WHERE clause for filtering documents
	// Should not include the "WHERE" keyword itself
	FilterQuery string `json:"filter_query,omitempty"`

	// FilterArgs provides arguments for the FilterQuery if it contains placeholders
	FilterArgs []interface{} `json:"filter_args,omitempty"`

	// DimensionFilters allows filtering by dimension values
	// Multiple filters are combined with AND
	DimensionFilters map[string]interface{} `json:"dimension_filters,omitempty"`

	// DocumentFormat specifies the format to use when serializing documents
	// If nil, defaults to formats.PlainText
	DocumentFormat *formats.DocumentFormat `json:"-"`
}
