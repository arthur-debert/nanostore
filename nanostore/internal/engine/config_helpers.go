package engine

import "github.com/arthur-debert/nanostore/nanostore/types"

// GetEnumeratedDimensions returns all enumerated dimensions from the config
func GetEnumeratedDimensions(c types.Config) []types.DimensionConfig {
	var enumerated []types.DimensionConfig
	for _, dim := range c.Dimensions {
		if dim.Type == types.Enumerated {
			enumerated = append(enumerated, dim)
		}
	}
	return enumerated
}

// GetHierarchicalDimensions returns all hierarchical dimensions from the config
func GetHierarchicalDimensions(c types.Config) []types.DimensionConfig {
	var hierarchical []types.DimensionConfig
	for _, dim := range c.Dimensions {
		if dim.Type == types.Hierarchical {
			hierarchical = append(hierarchical, dim)
		}
	}
	return hierarchical
}

// GetDimension returns the dimension configuration by name
func GetDimension(c types.Config, name string) (*types.DimensionConfig, bool) {
	for _, dim := range c.Dimensions {
		if dim.Name == name {
			return &dim, true
		}
	}
	return nil, false
}
