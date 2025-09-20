// Package query provides query processing functionality for nanostore.
// It handles filtering, sorting, pagination, and ID assignment for documents.
package query

import (
	"github.com/arthur-debert/nanostore/nanostore/ids"
	"github.com/arthur-debert/nanostore/types"
)

// Processor handles query execution against a set of documents
type Processor interface {
	// Execute runs a query against documents and returns the results
	Execute(docs []types.Document, opts types.ListOptions) ([]types.Document, error)

	// MatchesFilters checks if a document matches the given filters
	MatchesFilters(doc types.Document, filters map[string]interface{}) bool
}

// processor implements the Processor interface
type processor struct {
	dimensionSet *types.DimensionSet
	idGenerator  *ids.IDGenerator
}

// NewProcessor creates a new query processor
func NewProcessor(dimensionSet *types.DimensionSet, idGenerator *ids.IDGenerator) Processor {
	return &processor{
		dimensionSet: dimensionSet,
		idGenerator:  idGenerator,
	}
}

// Execute runs the query and returns filtered, sorted, and paginated results
func (p *processor) Execute(docs []types.Document, opts types.ListOptions) ([]types.Document, error) {
	// Start with all documents
	result := make([]types.Document, 0, len(docs))

	// Apply filters
	for _, doc := range docs {
		// Check dimension filters
		if !p.matchesFilters(doc, opts.Filters) {
			continue
		}

		// Check text search filter
		if opts.FilterBySearch != "" && !p.matchesSearch(doc, opts.FilterBySearch) {
			continue
		}

		// Make a copy to avoid mutations
		docCopy := doc
		result = append(result, docCopy)
	}

	// Apply ordering
	if len(opts.OrderBy) > 0 {
		p.sortDocuments(result, opts.OrderBy)
	}

	// Generate SimpleIDs using the ID generator
	// We need ALL documents for proper ID generation (not just filtered ones)
	idMap := p.idGenerator.GenerateIDs(docs)

	// Create reverse mapping (UUID -> SimpleID)
	uuidToID := make(map[string]string)
	for simpleID, uuid := range idMap {
		uuidToID[uuid] = simpleID
	}

	// Assign SimpleIDs to results
	for i := range result {
		if simpleID, exists := uuidToID[result[i].UUID]; exists {
			result[i].SimpleID = simpleID
		} else {
			// Fallback to UUID if not found (shouldn't happen)
			result[i].SimpleID = result[i].UUID
		}
	}

	// Apply pagination
	if opts.Offset != nil && *opts.Offset > 0 {
		if *opts.Offset >= len(result) {
			result = []types.Document{}
		} else {
			result = result[*opts.Offset:]
		}
	}

	if opts.Limit != nil && *opts.Limit > 0 {
		if *opts.Limit < len(result) {
			result = result[:*opts.Limit]
		}
	}

	return result, nil
}
