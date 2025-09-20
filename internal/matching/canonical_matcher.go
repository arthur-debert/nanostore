package matching

import (
	"fmt"

	"github.com/arthur-debert/nanostore/types"
)

// CanonicalMatcher provides matching functionality for canonical views
type CanonicalMatcher struct {
	view         *types.CanonicalView
	dimensionSet *types.DimensionSet
}

// NewCanonicalMatcher creates a new matcher for the given canonical view
func NewCanonicalMatcher(view *types.CanonicalView, dimensionSet *types.DimensionSet) *CanonicalMatcher {
	return &CanonicalMatcher{
		view:         view,
		dimensionSet: dimensionSet,
	}
}

// Matches checks if a document matches the canonical view filters
func (m *CanonicalMatcher) Matches(doc types.Document) bool {
	if m.view == nil || len(m.view.Filters) == 0 {
		// No filters means everything matches
		return true
	}

	// Document must match ALL filters
	for _, filter := range m.view.Filters {
		dim, exists := m.dimensionSet.Get(filter.Dimension)
		if !exists {
			// Filter references unknown dimension
			return false
		}

		docValue := ""
		switch dim.Type {
		case types.Enumerated:
			if v, exists := doc.Dimensions[dim.Name]; exists {
				docValue = fmt.Sprintf("%v", v)
			} else {
				// Use default value
				docValue = dim.DefaultValue
			}
		case types.Hierarchical:
			// Hierarchical dimensions can have "*" as filter value
			if filter.Value == "*" {
				continue // Any value matches
			}
			if v, exists := doc.Dimensions[dim.RefField]; exists {
				docValue = fmt.Sprintf("%v", v)
			}
		}

		// Check if document value matches filter value
		if docValue != filter.Value && filter.Value != "*" {
			return false
		}
	}

	return true
}

// IsCanonicalValue checks if a value matches the canonical filter for a dimension
func (m *CanonicalMatcher) IsCanonicalValue(dimension, value string) bool {
	filterValue, hasFilter := m.view.GetFilterValue(dimension)
	if !hasFilter {
		// No filter means any value is canonical
		return true
	}

	// Check if value matches the filter
	return filterValue == value || filterValue == "*"
}

// ExtractFromPartition extracts canonical filters from a partition
// Returns dimension values that should be removed from the user-facing ID
func (m *CanonicalMatcher) ExtractFromPartition(partition types.Partition) []types.DimensionValue {
	var canonical []types.DimensionValue

	for _, dv := range partition.Values {
		// Only extract if there's a filter for this dimension AND the value matches
		if filterValue, hasFilter := m.view.GetFilterValue(dv.Dimension); hasFilter {
			// Skip wildcard filters (typically used for hierarchical dimensions)
			if filterValue == "*" {
				continue
			}
			if filterValue == dv.Value {
				canonical = append(canonical, dv)
			}
		}
	}

	return canonical
}
