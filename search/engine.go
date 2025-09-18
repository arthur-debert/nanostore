package search

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arthur-debert/nanostore/types"
)

// Engine implements the Searcher interface
type Engine struct {
	provider DocumentProvider
}

// NewEngine creates a new search engine with the given document provider
func NewEngine(provider DocumentProvider) *Engine {
	return &Engine{
		provider: provider,
	}
}

// Search performs a search and returns ranked results
func (e *Engine) Search(options SearchOptions, filters map[string]interface{}) ([]SearchResult, error) {
	if options.Query == "" {
		return []SearchResult{}, nil
	}

	// Get documents from provider (may already be filtered)
	documents, err := e.provider.GetDocuments(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	// Search through documents
	var results []SearchResult
	query := options.Query
	if !options.CaseSensitive {
		query = strings.ToLower(query)
	}

	for _, doc := range documents {
		if result := e.searchDocument(doc, query, options); result != nil {
			results = append(results, *result)
		}
	}

	// Sort by score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Apply max results limit
	if options.MaxResults != nil && *options.MaxResults > 0 && len(results) > *options.MaxResults {
		results = results[:*options.MaxResults]
	}

	return results, nil
}

// searchDocument searches a single document and returns a result if it matches
func (e *Engine) searchDocument(doc types.Document, query string, options SearchOptions) *SearchResult {
	var matches []fieldMatch
	var bestMatchType MatchType
	var maxScore float64

	// Determine which fields to search
	fieldsToSearch := options.Fields
	if len(fieldsToSearch) == 0 {
		fieldsToSearch = []string{"title", "body"}
		// Add all dimension keys that don't start with "_data."
		for key := range doc.Dimensions {
			if !strings.HasPrefix(key, "_data.") {
				fieldsToSearch = append(fieldsToSearch, key)
			} else {
				// Add _data fields with their prefix
				fieldsToSearch = append(fieldsToSearch, key)
			}
		}
	}

	// Search each field
	for _, field := range fieldsToSearch {
		if match := e.searchField(doc, field, query, options); match != nil {
			matches = append(matches, *match)
			if match.score > maxScore {
				maxScore = match.score
				bestMatchType = match.matchType
			}
		}
	}

	// No matches found
	if len(matches) == 0 {
		return nil
	}

	// Build result
	result := &SearchResult{
		Document:      doc,
		Score:         maxScore,
		MatchType:     bestMatchType,
		MatchedFields: make([]string, 0, len(matches)),
	}

	// Collect highlights and matched fields
	if options.EnableHighlight {
		result.Highlights = make(map[string]string)
	}

	for _, match := range matches {
		result.MatchedFields = append(result.MatchedFields, match.field)
		if options.EnableHighlight {
			result.Highlights[match.field] = match.highlightedText
		}
	}

	return result
}

// fieldMatch represents a match within a specific field
type fieldMatch struct {
	field           string
	matchType       MatchType
	score           float64
	highlightedText string
}

// searchField searches within a specific field of a document
func (e *Engine) searchField(doc types.Document, fieldName, query string, options SearchOptions) *fieldMatch {
	var fieldValue string
	var matchType MatchType

	// Get field value
	switch fieldName {
	case "title":
		fieldValue = doc.Title
		matchType = MatchPartialTitle
	case "body":
		fieldValue = doc.Body
		matchType = MatchPartialBody
	default:
		// Check if it's a dimension or custom data field
		if value, exists := doc.Dimensions[fieldName]; exists {
			fieldValue = fmt.Sprintf("%v", value)
			if strings.HasPrefix(fieldName, "_data.") {
				matchType = MatchCustomData
			} else {
				matchType = MatchDimension
			}
		} else {
			return nil // Field not found
		}
	}

	// Prepare field value for comparison
	searchValue := fieldValue
	if !options.CaseSensitive {
		searchValue = strings.ToLower(searchValue)
	}

	// Check for match
	var matched bool
	var score float64

	if options.ExactMatch {
		matched = searchValue == query
		if matched {
			score = 1.0
			// Update match type for exact matches
			switch fieldName {
			case "title":
				matchType = MatchExactTitle
			case "body":
				matchType = MatchExactBody
			}
		}
	} else {
		matched = strings.Contains(searchValue, query)
		if matched {
			// Calculate score based on match quality
			score = e.calculateScore(searchValue, query, fieldName)
		}
	}

	if !matched {
		return nil
	}

	// Generate highlighted text
	highlightedText := fieldValue
	if options.EnableHighlight {
		highlightedText = e.highlightMatches(fieldValue, query, options.CaseSensitive)
	}

	return &fieldMatch{
		field:           fieldName,
		matchType:       matchType,
		score:           score,
		highlightedText: highlightedText,
	}
}

// calculateScore computes a relevance score for a match
func (e *Engine) calculateScore(fieldValue, query, fieldName string) float64 {
	baseScore := 0.5

	// Boost score for title matches
	if fieldName == "title" {
		baseScore = 0.8
	}

	// Boost for exact substring match vs partial
	if strings.Contains(fieldValue, query) {
		baseScore += 0.2
	}

	// Boost if match is at the beginning
	if strings.HasPrefix(fieldValue, query) {
		baseScore += 0.2
	}

	// Boost if query takes up a large portion of the field
	if len(query) > 0 {
		coverage := float64(len(query)) / float64(len(fieldValue))
		if coverage > 0.5 {
			baseScore += 0.1
		}
	}

	// Ensure score doesn't exceed 1.0
	if baseScore > 1.0 {
		baseScore = 1.0
	}

	return baseScore
}

// highlightMatches adds highlight markers around matches
func (e *Engine) highlightMatches(text, query string, caseSensitive bool) string {
	if query == "" {
		return text
	}

	searchText := text
	searchQuery := query
	if !caseSensitive {
		searchText = strings.ToLower(text)
		searchQuery = strings.ToLower(query)
	}

	// Find all occurrences and highlight them
	result := text
	queryLen := len(query)

	// Work backwards to maintain indices
	for i := len(searchText) - queryLen; i >= 0; i-- {
		if i+queryLen <= len(searchText) && searchText[i:i+queryLen] == searchQuery {
			// Insert highlight markers
			result = result[:i] + "**" + result[i:i+queryLen] + "**" + result[i+queryLen:]
		}
	}

	return result
}
