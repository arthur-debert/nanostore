package query

import (
	"sort"

	"github.com/arthur-debert/nanostore/types"
)

// sortDocuments sorts documents according to the order clauses
func (p *processor) sortDocuments(docs []types.Document, orderBy []types.OrderClause) {
	sort.Slice(docs, func(i, j int) bool {
		for _, clause := range orderBy {
			// Get values for comparison
			valI := p.getDocumentValue(docs[i], clause.Column)
			valJ := p.getDocumentValue(docs[j], clause.Column)

			// Convert to comparable strings
			strI := valueToString(valI)
			strJ := valueToString(valJ)

			// Compare
			if strI < strJ {
				return !clause.Descending
			} else if strI > strJ {
				return clause.Descending
			}
			// If equal, continue to next order clause
		}
		return false // All equal
	})
}

// getDocumentValue retrieves a value from a document by field name
func (p *processor) getDocumentValue(doc types.Document, column string) interface{} {
	switch column {
	case "uuid":
		return doc.UUID
	case "simple_id", "simpleid":
		return doc.SimpleID
	case "title":
		return doc.Title
	case "body":
		return doc.Body
	case "created_at":
		return doc.CreatedAt
	case "updated_at":
		return doc.UpdatedAt
	default:
		// Check if it's a dimension
		if val, exists := doc.Dimensions[column]; exists {
			return val
		}
		// Try with _data prefix for non-dimension fields (transparent ordering support)
		if val, exists := doc.Dimensions["_data."+column]; exists {
			return val
		}
		// Return empty string for non-existent fields
		return ""
	}
}
