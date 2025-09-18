package search

import "github.com/arthur-debert/nanostore/types"

// SearchOptions configures search behavior
type SearchOptions struct {
	// Query is the search term(s) to look for
	Query string

	// Fields specifies which fields to search in
	// Supported values: "title", "body", dimension names, "_data.fieldname"
	// Empty slice searches all fields
	Fields []string

	// CaseSensitive controls whether search is case-sensitive
	CaseSensitive bool

	// ExactMatch requires the entire field to match the query
	// When false, performs partial/substring matching
	ExactMatch bool

	// EnableHighlight includes highlighted match text in results
	EnableHighlight bool

	// MaxResults limits the number of search results
	// nil means no limit
	MaxResults *int
}

// SearchResult represents a search match with metadata
type SearchResult struct {
	// Document is the matched document
	Document types.Document

	// Score represents match relevance (0.0 to 1.0, higher is better)
	Score float64

	// Highlights contains highlighted text for each matched field
	// Key is field name, value is text with match markers
	Highlights map[string]string

	// MatchType describes where the match was found
	MatchType MatchType

	// MatchedFields lists all fields that contained matches
	MatchedFields []string
}

// MatchType indicates the type of match found
type MatchType string

const (
	MatchExactTitle   MatchType = "exact_title"
	MatchPartialTitle MatchType = "partial_title"
	MatchExactBody    MatchType = "exact_body"
	MatchPartialBody  MatchType = "partial_body"
	MatchDimension    MatchType = "dimension"
	MatchCustomData   MatchType = "custom_data"
)

// DocumentProvider defines the interface for accessing documents
// This allows for dependency injection and easy mocking in tests
type DocumentProvider interface {
	// GetDocuments returns all documents that match the given filters
	// This integrates with existing nanostore filtering
	GetDocuments(filters map[string]interface{}) ([]types.Document, error)
}

// Searcher defines the main search interface
type Searcher interface {
	// Search performs a search and returns ranked results
	Search(options SearchOptions, filters map[string]interface{}) ([]SearchResult, error)
}
