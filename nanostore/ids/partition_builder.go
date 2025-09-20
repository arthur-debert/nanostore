package ids

import (
	"fmt"

	"github.com/arthur-debert/nanostore/types"
)

// BuildPartitionForDocument creates a Partition from a document's dimensions
// This is used during ID generation to determine which partition a document belongs to
func BuildPartitionForDocument(doc types.Document, dimensionSet *types.DimensionSet) types.Partition {
	var values []types.DimensionValue

	// Build dimension values in order
	for _, dim := range dimensionSet.All() {
		value := ""

		switch dim.Type {
		case types.Enumerated:
			// Get value from dimensions map
			if v, exists := doc.Dimensions[dim.Name]; exists {
				value = fmt.Sprintf("%v", v)
			} else {
				// Use default value
				value = dim.DefaultValue
			}

		case types.Hierarchical:
			// For hierarchical dimensions, use the parent reference
			if v, exists := doc.Dimensions[dim.RefField]; exists {
				value = fmt.Sprintf("%v", v)
			}
		}

		if value != "" {
			values = append(values, types.DimensionValue{
				Dimension: dim.Name,
				Value:     value,
			})
		}
	}

	return types.Partition{
		Values:   values,
		Position: 0, // Position will be set during ID generation
	}
}
