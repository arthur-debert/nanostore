package types

import (
	"encoding/json"
	"fmt"
)

// DimensionType defines the type of dimension for ID partitioning
type DimensionType int

const (
	// Enumerated dimensions have predefined values (e.g., status, priority)
	Enumerated DimensionType = iota
	// Hierarchical dimensions create parent-child relationships
	Hierarchical
)

// String returns the string representation of the DimensionType
func (dt DimensionType) String() string {
	switch dt {
	case Enumerated:
		return "enumerated"
	case Hierarchical:
		return "hierarchical"
	default:
		return "unknown"
	}
}

// MarshalJSON implements json.Marshaler for DimensionType
func (dt DimensionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(dt.String())
}

// UnmarshalJSON implements json.Unmarshaler for DimensionType
func (dt *DimensionType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case "enumerated":
		*dt = Enumerated
	case "hierarchical":
		*dt = Hierarchical
	default:
		return fmt.Errorf("invalid dimension type: %q (must be 'enumerated' or 'hierarchical')", s)
	}

	return nil
}

// DimensionConfig defines a single dimension for ID partitioning
type DimensionConfig struct {
	// Name is the database column name and identifier for this dimension
	Name string `json:"name"`

	// Type specifies whether this is an enumerated or hierarchical dimension
	Type DimensionType `json:"type"`

	// Values lists the valid values for enumerated dimensions
	// Ignored for hierarchical dimensions
	Values []string `json:"values,omitempty"`

	// Prefixes maps values to their ID prefixes
	// For enumerated dimensions: value -> prefix (e.g., "completed" -> "c")
	// Ignored for hierarchical dimensions
	Prefixes map[string]string `json:"prefixes,omitempty"`

	// RefField specifies the foreign key field name for hierarchical dimensions
	// For hierarchical dimensions: typically "parent_uuid"
	// Ignored for enumerated dimensions
	RefField string `json:"ref_field,omitempty"`

	// DefaultValue specifies the default value for enumerated dimensions
	// Used when inserting new documents without explicit value
	DefaultValue string `json:"default_value,omitempty"`
}

// Config defines the overall configuration for the nanostore
type Config struct {
	// Dimensions defines the ID partitioning dimensions
	Dimensions []DimensionConfig `json:"dimensions"`

	// dimensionSet is the new internal representation
	// Will be populated from Dimensions during initialization
	dimensionSet *DimensionSet `json:"-"`
}

// GetEnumeratedDimensions returns all enumerated dimensions from the config
func (c Config) GetEnumeratedDimensions() []DimensionConfig {
	var enumerated []DimensionConfig
	for _, dim := range c.Dimensions {
		if dim.Type == Enumerated {
			enumerated = append(enumerated, dim)
		}
	}
	return enumerated
}

// GetHierarchicalDimensions returns all hierarchical dimensions from the config
func (c Config) GetHierarchicalDimensions() []DimensionConfig {
	var hierarchical []DimensionConfig
	for _, dim := range c.Dimensions {
		if dim.Type == Hierarchical {
			hierarchical = append(hierarchical, dim)
		}
	}
	return hierarchical
}

// GetDimension returns the dimension configuration by name
func (c Config) GetDimension(name string) (*DimensionConfig, bool) {
	for _, dim := range c.Dimensions {
		if dim.Name == name {
			return &dim, true
		}
	}
	return nil, false
}

// GetDimensionSet returns the dimension set, initializing it if needed
func (c *Config) GetDimensionSet() *DimensionSet {
	if c.dimensionSet == nil {
		c.dimensionSet = DimensionSetFromConfig(*c)
	}
	return c.dimensionSet
}
