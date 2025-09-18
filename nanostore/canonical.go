package nanostore

import (
	"fmt"
	"strings"
)

// CanonicalFilter represents a filter for the canonical view
type CanonicalFilter struct {
	Dimension string
	Value     string
}

// CanonicalView defines which documents appear in the canonical (default) view
type CanonicalView struct {
	// Filters define the dimension filters for the canonical view
	// Only documents matching ALL filters appear in the canonical view
	Filters []CanonicalFilter
}

// NewCanonicalView creates a new canonical view with the given filters
func NewCanonicalView(filters ...CanonicalFilter) *CanonicalView {
	return &CanonicalView{
		Filters: filters,
	}
}

// String returns a string representation of the canonical view
func (cv *CanonicalView) String() string {
	if cv == nil || len(cv.Filters) == 0 {
		return "canonical:*"
	}

	var parts []string
	for _, f := range cv.Filters {
		parts = append(parts, fmt.Sprintf("%s:%s", f.Dimension, f.Value))
	}
	return fmt.Sprintf("canonical:%s", strings.Join(parts, ","))
}

// Matches checks if a document matches the canonical view filters
func (cv *CanonicalView) Matches(doc Document, dimensionSet *DimensionSet) bool {
	if cv == nil || len(cv.Filters) == 0 {
		// No filters means everything matches
		return true
	}

	// Document must match ALL filters
	for _, filter := range cv.Filters {
		dim, exists := dimensionSet.Get(filter.Dimension)
		if !exists {
			// Filter references unknown dimension
			return false
		}

		docValue := ""
		switch dim.Type {
		case Enumerated:
			if v, exists := doc.Dimensions[dim.Name]; exists {
				docValue = fmt.Sprintf("%v", v)
			} else {
				// Use default value
				docValue = dim.DefaultValue
			}
		case Hierarchical:
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

// GetFilterValue returns the filter value for a specific dimension
func (cv *CanonicalView) GetFilterValue(dimension string) (string, bool) {
	if cv == nil {
		return "", false
	}

	for _, f := range cv.Filters {
		if f.Dimension == dimension {
			return f.Value, true
		}
	}
	return "", false
}

// HasFilter checks if the canonical view has a filter for a specific dimension
func (cv *CanonicalView) HasFilter(dimension string) bool {
	_, exists := cv.GetFilterValue(dimension)
	return exists
}

// IsCanonicalValue checks if a value matches the canonical filter for a dimension
func (cv *CanonicalView) IsCanonicalValue(dimension, value string) bool {
	filterValue, hasFilter := cv.GetFilterValue(dimension)
	if !hasFilter {
		// No filter means any value is canonical
		return true
	}

	// Check if value matches the filter
	return filterValue == value || filterValue == "*"
}

// ExtractFromPartition extracts canonical filters from a partition
// Returns dimension values that should be removed from the user-facing ID
func (cv *CanonicalView) ExtractFromPartition(partition Partition) []DimensionValue {
	var canonical []DimensionValue

	for _, dv := range partition.Values {
		// Only extract if there's a filter for this dimension AND the value matches
		if filterValue, hasFilter := cv.GetFilterValue(dv.Dimension); hasFilter {
			if filterValue == dv.Value || filterValue == "*" {
				canonical = append(canonical, dv)
			}
		}
	}

	return canonical
}

// ConfigWithCanonicalView extends Config with canonical view information
type ConfigWithCanonicalView struct {
	Config
	CanonicalView *CanonicalView
}

// GetCanonicalView returns the canonical view, creating a default if needed
func (c *ConfigWithCanonicalView) GetCanonicalView() *CanonicalView {
	if c.CanonicalView == nil {
		// Default canonical view based on dimension defaults
		var filters []CanonicalFilter
		for _, dim := range c.GetDimensionSet().Enumerated() {
			if dim.DefaultValue != "" {
				filters = append(filters, CanonicalFilter{
					Dimension: dim.Name,
					Value:     dim.DefaultValue,
				})
			}
		}
		// Hierarchical dimensions default to "*" (any value)
		for _, dim := range c.GetDimensionSet().Hierarchical() {
			filters = append(filters, CanonicalFilter{
				Dimension: dim.Name,
				Value:     "*",
			})
		}
		c.CanonicalView = NewCanonicalView(filters...)
	}
	return c.CanonicalView
}
