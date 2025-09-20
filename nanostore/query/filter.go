package query

import (
	"fmt"
	"strings"

	"github.com/arthur-debert/nanostore/types"
)

// matchesFilters checks if a document matches all the provided filters
func (p *processor) matchesFilters(doc types.Document, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true // No filters means match all
	}

	for filterKey, filterValue := range filters {
		// Handle special filter for UUID
		if filterKey == "uuid" {
			if doc.UUID != fmt.Sprintf("%v", filterValue) {
				return false
			}
			continue
		}

		// Handle datetime filters and dimension filters
		var docValue interface{}
		var exists bool

		switch filterKey {
		case "created_at":
			docValue = doc.CreatedAt
			exists = true
		case "updated_at":
			docValue = doc.UpdatedAt
			exists = true
		default:
			// Check if it's a dimension filter
			docValue, exists = doc.Dimensions[filterKey]
			if !exists {
				// Try with _data prefix for non-dimension fields
				docValue, exists = doc.Dimensions["_data."+filterKey]
				if !exists {
					// Document doesn't have this dimension or data field
					// Check if it's a hierarchical dimension ref field
					found := false
					for _, dim := range p.dimensionSet.Hierarchical() {
						if dim.RefField == filterKey {
							// It's a hierarchical ref field
							if parentValue, ok := doc.Dimensions[dim.RefField]; ok {
								docValue = parentValue
								exists = true
								found = true
								break
							}
						}
					}
					if !found {
						return false
					}
				}
			}
		}

		// Convert values to comparable strings
		docStr := valueToString(docValue)

		// Handle slice values (for "IN" style filtering)
		switch fv := filterValue.(type) {
		case []string:
			// Filter value is a slice, check if document value is in the slice
			found := false
			for _, v := range fv {
				if docStr == v {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		case []interface{}:
			// Filter value is a slice, check if document value is in the slice
			found := false
			for _, v := range fv {
				if docStr == valueToString(v) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		default:
			// Simple equality check
			filterStr := valueToString(filterValue)
			if docStr != filterStr {
				return false
			}
		}
	}

	return true
}

// matchesSearch checks if a document matches the search text
func (p *processor) matchesSearch(doc types.Document, searchText string) bool {
	// Simple case-insensitive substring search in title and body
	searchLower := strings.ToLower(searchText)

	if strings.Contains(strings.ToLower(doc.Title), searchLower) {
		return true
	}

	if strings.Contains(strings.ToLower(doc.Body), searchLower) {
		return true
	}

	return false
}

// MatchesFilters implements the Processor interface method
func (p *processor) MatchesFilters(doc types.Document, filters map[string]interface{}) bool {
	return p.matchesFilters(doc, filters)
}
