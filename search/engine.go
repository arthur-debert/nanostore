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
	var fieldMatches []FieldMatch
	var bestMatchType MatchType
	var maxScore float64

	// Set default highlight markers
	startMarker := options.HighlightStartMarker
	endMarker := options.HighlightEndMarker
	if startMarker == "" {
		startMarker = "**"
	}
	if endMarker == "" {
		endMarker = "**"
	}

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
		if fieldMatch := e.searchFieldDetailed(doc, field, query, options, startMarker, endMarker); fieldMatch != nil {
			fieldMatches = append(fieldMatches, *fieldMatch)
			if fieldMatch.FieldScore > maxScore {
				maxScore = fieldMatch.FieldScore
				if len(fieldMatch.Matches) > 0 {
					bestMatchType = fieldMatch.Matches[0].MatchType
				}
			}
		}
	}

	// No matches found
	if len(fieldMatches) == 0 {
		return nil
	}

	// Build result
	result := &SearchResult{
		Document:      doc,
		Score:         maxScore,
		MatchType:     bestMatchType,
		MatchedFields: make([]string, 0, len(fieldMatches)),
	}

	// Include detailed match data if requested
	if options.IncludeMatchDetails {
		result.FieldMatches = fieldMatches
	}

	// Build legacy highlights and matched fields
	if options.EnableHighlight {
		result.Highlights = make(map[string]string)
	}

	for _, fieldMatch := range fieldMatches {
		result.MatchedFields = append(result.MatchedFields, fieldMatch.FieldName)
		if options.EnableHighlight {
			result.Highlights[fieldMatch.FieldName] = fieldMatch.HighlightedText
		}
	}

	return result
}

// searchFieldDetailed searches within a specific field and returns detailed match information
func (e *Engine) searchFieldDetailed(doc types.Document, fieldName, query string, options SearchOptions, startMarker, endMarker string) *FieldMatch {
	var fieldValue string
	var baseMatchType MatchType

	// Get field value
	switch fieldName {
	case "title":
		fieldValue = doc.Title
		baseMatchType = MatchPartialTitle
	case "body":
		fieldValue = doc.Body
		baseMatchType = MatchPartialBody
	default:
		// Check if it's a dimension or custom data field
		if value, exists := doc.Dimensions[fieldName]; exists {
			fieldValue = fmt.Sprintf("%v", value)
			if strings.HasPrefix(fieldName, "_data.") {
				baseMatchType = MatchCustomData
			} else {
				baseMatchType = MatchDimension
			}
		} else {
			return nil // Field not found
		}
	}

	// Find all match positions
	matches := e.findMatches(fieldValue, query, options, baseMatchType)
	if len(matches) == 0 {
		return nil
	}

	// Calculate field score (highest match score)
	fieldScore := 0.0
	for _, match := range matches {
		if match.Score > fieldScore {
			fieldScore = match.Score
		}
	}

	// Generate highlighted text
	highlightedText := fieldValue
	if options.EnableHighlight {
		highlightedText = e.highlightMatchesWithMarkers(fieldValue, query, options.CaseSensitive, startMarker, endMarker)
	}

	return &FieldMatch{
		FieldName:       fieldName,
		OriginalText:    fieldValue,
		Matches:         matches,
		HighlightedText: highlightedText,
		FieldScore:      fieldScore,
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

// findMatches finds all occurrences of the query in the text and returns detailed match info
func (e *Engine) findMatches(text, query string, options SearchOptions, baseMatchType MatchType) []MatchInfo {
	var matches []MatchInfo

	// Prepare text for searching
	searchText := text
	searchQuery := query
	if !options.CaseSensitive {
		searchText = strings.ToLower(text)
		searchQuery = strings.ToLower(query)
	}

	queryLen := len(query)
	if queryLen == 0 {
		return matches
	}

	if options.ExactMatch {
		// For exact match, only one match is possible
		if searchText == searchQuery {
			matchType := baseMatchType
			// Update match type for exact matches
			switch baseMatchType {
			case MatchPartialTitle:
				matchType = MatchExactTitle
			case MatchPartialBody:
				matchType = MatchExactBody
			}

			matches = append(matches, MatchInfo{
				Start:     0,
				End:       len(text),
				Text:      text,
				Score:     1.0,
				MatchType: matchType,
			})
		}
	} else {
		// Find all substring matches
		for i := 0; i <= len(searchText)-queryLen; i++ {
			if searchText[i:i+queryLen] == searchQuery {
				matchText := text[i : i+queryLen]
				fieldName := ""
				switch baseMatchType {
				case MatchPartialTitle, MatchExactTitle:
					fieldName = "title"
				case MatchPartialBody, MatchExactBody:
					fieldName = "body"
				}
				score := e.calculateScore(text, query, fieldName)

				matches = append(matches, MatchInfo{
					Start:     i,
					End:       i + queryLen,
					Text:      matchText,
					Score:     score,
					MatchType: baseMatchType,
				})

				// Skip overlapping matches
				i += queryLen - 1
			}
		}
	}

	return matches
}

// highlightMatchesWithMarkers adds configurable highlight markers around matches
func (e *Engine) highlightMatchesWithMarkers(text, query string, caseSensitive bool, startMarker, endMarker string) string {
	if query == "" {
		return text
	}

	searchText := text
	searchQuery := query
	if !caseSensitive {
		searchText = strings.ToLower(text)
		searchQuery = strings.ToLower(query)
	}

	queryLen := len(query)
	if queryLen == 0 {
		return text
	}

	// Find all match positions first
	var matchPositions []int
	for i := 0; i <= len(searchText)-queryLen; i++ {
		if searchText[i:i+queryLen] == searchQuery {
			matchPositions = append(matchPositions, i)
			i += queryLen - 1 // Skip overlapping matches
		}
	}

	if len(matchPositions) == 0 {
		return text
	}

	// Build result string efficiently using strings.Builder
	var builder strings.Builder
	lastEnd := 0

	for _, start := range matchPositions {
		end := start + queryLen

		// Add text before match
		builder.WriteString(text[lastEnd:start])

		// Add highlighted match
		builder.WriteString(startMarker)
		builder.WriteString(text[start:end])
		builder.WriteString(endMarker)

		lastEnd = end
	}

	// Add remaining text after last match
	builder.WriteString(text[lastEnd:])

	return builder.String()
}
