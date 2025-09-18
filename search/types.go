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

	// HighlightStartMarker is the marker to insert before matches (default "**")
	HighlightStartMarker string

	// HighlightEndMarker is the marker to insert after matches (default "**")
	HighlightEndMarker string

	// IncludeMatchDetails includes structured match position data in results
	IncludeMatchDetails bool

	// MaxResults limits the number of search results
	// nil means no limit
	MaxResults *int
}

// MatchInfo represents a single match occurrence within a field
type MatchInfo struct {
	// Start is the character position where the match begins
	Start int

	// End is the character position where the match ends (exclusive)
	End int

	// Text is the actual matched text from the original content
	Text string

	// Score is the relevance score for this specific match (0.0 to 1.0)
	Score float64

	// MatchType describes the type of this specific match
	MatchType MatchType
}

// FieldMatch represents all matches within a specific field
type FieldMatch struct {
	// FieldName is the name of the field that was searched
	FieldName string

	// OriginalText is the unmodified field content
	OriginalText string

	// Matches contains detailed information about each match occurrence
	Matches []MatchInfo

	// HighlightedText is the field content with highlight markers inserted
	// Only populated if EnableHighlight is true
	HighlightedText string

	// FieldScore is the overall relevance score for this field (0.0 to 1.0)
	FieldScore float64
}

// SearchResult represents a search match with metadata
type SearchResult struct {
	// Document is the matched document
	Document types.Document

	// Score represents overall match relevance (0.0 to 1.0, higher is better)
	Score float64

	// Highlights contains highlighted text for each matched field (legacy)
	// Key is field name, value is text with match markers
	// Deprecated: Use FieldMatches for more detailed information
	Highlights map[string]string

	// FieldMatches contains detailed match information for each field
	// Only populated if IncludeMatchDetails is true
	FieldMatches []FieldMatch

	// MatchType describes the primary/best match type found
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
