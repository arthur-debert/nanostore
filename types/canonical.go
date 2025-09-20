package types

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
