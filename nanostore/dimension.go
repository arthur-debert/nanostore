package nanostore

import (
	"fmt"
)

// DimensionMetadata holds metadata about a dimension
type DimensionMetadata struct {
	// Order in which this dimension appears in partitioning
	// Lower order dimensions are stronger (e.g., parent=0, status=1)
	Order int

	// Whether this dimension affects the canonical view
	// If false, this dimension's value must match canonical filter
	IsCanonical bool
}

// Dimension represents a single dimension in the partitioning system
type Dimension struct {
	// Name is the identifier for this dimension
	Name string

	// Type specifies whether this is enumerated or hierarchical
	Type DimensionType

	// Metadata about this dimension's role
	Meta DimensionMetadata

	// For enumerated dimensions
	Values       []string          // Valid values
	Prefixes     map[string]string // Value to prefix mapping
	DefaultValue string            // Default when not specified

	// For hierarchical dimensions
	RefField string // Foreign key field name (e.g., "parent_uuid")
}

// IsValid checks if a value is valid for this dimension
func (d *Dimension) IsValid(value string) bool {
	if d.Type != Enumerated {
		return true // Hierarchical dimensions accept any value
	}

	for _, v := range d.Values {
		if v == value {
			return true
		}
	}
	return false
}

// GetPrefix returns the prefix for a given value
func (d *Dimension) GetPrefix(value string) string {
	if d.Type != Enumerated {
		return ""
	}

	prefix, exists := d.Prefixes[value]
	if !exists {
		return ""
	}
	return prefix
}

// HasPrefix checks if this dimension has any prefixes defined
func (d *Dimension) HasPrefix() bool {
	return len(d.Prefixes) > 0
}

// DimensionSet represents an ordered collection of dimensions
type DimensionSet struct {
	dimensions []Dimension
	byName     map[string]*Dimension
}

// NewDimensionSet creates a new dimension set from a slice of dimensions
func NewDimensionSet(dims []Dimension) *DimensionSet {
	ds := &DimensionSet{
		dimensions: make([]Dimension, len(dims)),
		byName:     make(map[string]*Dimension),
	}

	copy(ds.dimensions, dims)

	for i := range ds.dimensions {
		ds.byName[ds.dimensions[i].Name] = &ds.dimensions[i]
	}

	return ds
}

// Get returns a dimension by name
func (ds *DimensionSet) Get(name string) (*Dimension, bool) {
	dim, exists := ds.byName[name]
	return dim, exists
}

// All returns all dimensions in order
func (ds *DimensionSet) All() []Dimension {
	return ds.dimensions
}

// Enumerated returns only enumerated dimensions
func (ds *DimensionSet) Enumerated() []Dimension {
	var result []Dimension
	for _, dim := range ds.dimensions {
		if dim.Type == Enumerated {
			result = append(result, dim)
		}
	}
	return result
}

// Hierarchical returns only hierarchical dimensions
func (ds *DimensionSet) Hierarchical() []Dimension {
	var result []Dimension
	for _, dim := range ds.dimensions {
		if dim.Type == Hierarchical {
			result = append(result, dim)
		}
	}
	return result
}

// Count returns the number of dimensions
func (ds *DimensionSet) Count() int {
	return len(ds.dimensions)
}

// Validate checks the dimension set for consistency
func (ds *DimensionSet) Validate() error {
	if ds.Count() == 0 {
		return fmt.Errorf("at least one dimension must be configured")
	}

	// Enforce dimension limit for performance
	const maxDimensions = 7
	if ds.Count() > maxDimensions {
		return fmt.Errorf("too many dimensions: %d (maximum %d)", ds.Count(), maxDimensions)
	}

	// Check for duplicate names
	seen := make(map[string]bool)
	for _, dim := range ds.dimensions {
		if seen[dim.Name] {
			return fmt.Errorf("duplicate dimension name: %s", dim.Name)
		}
		seen[dim.Name] = true
	}

	// Track prefixes to check for conflicts
	prefixesSeen := make(map[string]string)

	for _, dim := range ds.dimensions {
		// Validate dimension name
		if dim.Name == "" {
			return fmt.Errorf("dimension name cannot be empty")
		}

		// Check for reserved column names
		if isReservedColumnName(dim.Name) {
			return fmt.Errorf("'%s' is a reserved column name", dim.Name)
		}

		// Validate based on dimension type
		switch dim.Type {
		case Enumerated:
			if err := validateEnumeratedDim(&dim, prefixesSeen); err != nil {
				return err
			}
		case Hierarchical:
			if err := validateHierarchicalDim(&dim); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid dimension type %d for %s", dim.Type, dim.Name)
		}
	}

	return nil
}

// validateEnumeratedDim validates an enumerated dimension
func validateEnumeratedDim(dim *Dimension, prefixesSeen map[string]string) error {
	// Must have at least one value
	if len(dim.Values) == 0 {
		return fmt.Errorf("dimension %s: enumerated dimensions must have at least one value", dim.Name)
	}

	// Check for duplicate values
	valuesSeen := make(map[string]bool)
	for _, value := range dim.Values {
		if value == "" {
			return fmt.Errorf("dimension %s: values cannot be empty", dim.Name)
		}
		if valuesSeen[value] {
			return fmt.Errorf("dimension %s: duplicate value '%s'", dim.Name, value)
		}
		valuesSeen[value] = true
	}

	// Validate default value if specified
	if dim.DefaultValue != "" {
		if !dim.IsValid(dim.DefaultValue) {
			return fmt.Errorf("dimension %s: default value '%s' is not in the list of valid values", dim.Name, dim.DefaultValue)
		}
	}

	// Validate prefixes
	for value, prefix := range dim.Prefixes {
		if !dim.IsValid(value) {
			return fmt.Errorf("dimension %s: prefix defined for invalid value '%s'", dim.Name, value)
		}
		if prefix == "" {
			return fmt.Errorf("dimension %s: empty prefix for value '%s'", dim.Name, value)
		}

		// Check for prefix conflicts
		if existingDim, exists := prefixesSeen[prefix]; exists {
			return fmt.Errorf("dimension %s: prefix '%s' conflicts with dimension %s", dim.Name, prefix, existingDim)
		}
		prefixesSeen[prefix] = dim.Name
	}

	return nil
}

// validateHierarchicalDim validates a hierarchical dimension
func validateHierarchicalDim(dim *Dimension) error {
	// Must have RefField
	if dim.RefField == "" {
		return fmt.Errorf("dimension %s: hierarchical dimensions must specify RefField", dim.Name)
	}

	// Should not have values or prefixes
	if len(dim.Values) > 0 {
		return fmt.Errorf("dimension %s: hierarchical dimensions should not have values", dim.Name)
	}
	if len(dim.Prefixes) > 0 {
		return fmt.Errorf("dimension %s: hierarchical dimensions should not have prefixes", dim.Name)
	}

	return nil
}

// Helper function to convert old config to new dimension set
// This will be used during migration
func dimensionSetFromConfig(config Config) *DimensionSet {
	dims := make([]Dimension, len(config.Dimensions))

	for i, dc := range config.Dimensions {
		dims[i] = Dimension{
			Name:         dc.Name,
			Type:         dc.Type,
			Values:       dc.Values,
			Prefixes:     dc.Prefixes,
			DefaultValue: dc.DefaultValue,
			RefField:     dc.RefField,
			Meta: DimensionMetadata{
				Order:       i,
				IsCanonical: true, // Will be updated when we add canonical view
			},
		}
	}

	return NewDimensionSet(dims)
}
