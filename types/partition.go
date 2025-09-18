package types

import (
	"fmt"
	"strings"
)

// DimensionValue represents a dimension:value pair
type DimensionValue struct {
	Dimension string
	Value     string
}

// String returns the string representation of a dimension:value pair
func (dv DimensionValue) String() string {
	return fmt.Sprintf("%s:%s", dv.Dimension, dv.Value)
}

// ParseDimensionValue parses a string like "status:pending" into a DimensionValue
func ParseDimensionValue(s string) (DimensionValue, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return DimensionValue{}, fmt.Errorf("invalid dimension:value format: %s", s)
	}

	return DimensionValue{
		Dimension: strings.TrimSpace(parts[0]),
		Value:     strings.TrimSpace(parts[1]),
	}, nil
}

// Partition represents a specific partition defined by dimension values
type Partition struct {
	// Values is an ordered list of dimension:value pairs
	// Order matches the dimension order in the configuration
	Values []DimensionValue

	// Position within this partition
	Position int
}

// String returns the full string representation of a partition
// Format: "dimension1:value1,dimension2:value2|position"
func (p Partition) String() string {
	var parts []string
	for _, dv := range p.Values {
		parts = append(parts, dv.String())
	}
	return fmt.Sprintf("%s|%d", strings.Join(parts, ","), p.Position)
}

// ParsePartition parses a partition string
func ParsePartition(s string) (Partition, error) {
	// Split by | to separate dimension values from position
	mainParts := strings.SplitN(s, "|", 2)
	if len(mainParts) != 2 {
		return Partition{}, fmt.Errorf("invalid partition format: missing position")
	}

	// Parse position
	var position int
	if _, err := fmt.Sscanf(mainParts[1], "%d", &position); err != nil {
		return Partition{}, fmt.Errorf("invalid position: %s", mainParts[1])
	}

	// Parse dimension values
	var values []DimensionValue
	if mainParts[0] != "" {
		dvParts := strings.Split(mainParts[0], ",")
		for _, dvStr := range dvParts {
			dv, err := ParseDimensionValue(dvStr)
			if err != nil {
				return Partition{}, err
			}
			values = append(values, dv)
		}
	}

	return Partition{
		Values:   values,
		Position: position,
	}, nil
}

// Key returns a unique key for this partition (without position)
// This is used for grouping documents into the same partition
func (p Partition) Key() string {
	var parts []string
	for _, dv := range p.Values {
		parts = append(parts, dv.String())
	}
	return strings.Join(parts, ",")
}

// HasDimension checks if this partition has a specific dimension
func (p Partition) HasDimension(dimension string) bool {
	for _, dv := range p.Values {
		if dv.Dimension == dimension {
			return true
		}
	}
	return false
}

// GetValue returns the value for a specific dimension
func (p Partition) GetValue(dimension string) (string, bool) {
	for _, dv := range p.Values {
		if dv.Dimension == dimension {
			return dv.Value, true
		}
	}
	return "", false
}

// PartitionMap is a collection of documents organized by partition
type PartitionMap map[string][]Document

// Add adds a document to the appropriate partition
func (pm PartitionMap) Add(partition Partition, doc Document) {
	key := partition.Key()
	pm[key] = append(pm[key], doc)
}

// Get returns all documents in a specific partition
func (pm PartitionMap) Get(partition Partition) []Document {
	return pm[partition.Key()]
}

// Count returns the number of documents in a specific partition
func (pm PartitionMap) Count(partition Partition) int {
	return len(pm[partition.Key()])
}

// BuildPartitionForDocument creates a Partition from a document's dimensions
func BuildPartitionForDocument(doc Document, dimensionSet *DimensionSet) Partition {
	var values []DimensionValue

	// Build dimension values in order
	for _, dim := range dimensionSet.All() {
		value := ""

		switch dim.Type {
		case Enumerated:
			// Get value from dimensions map
			if v, exists := doc.Dimensions[dim.Name]; exists {
				value = fmt.Sprintf("%v", v)
			} else {
				// Use default value
				value = dim.DefaultValue
			}

		case Hierarchical:
			// For hierarchical dimensions, use the parent reference
			if v, exists := doc.Dimensions[dim.RefField]; exists {
				value = fmt.Sprintf("%v", v)
			}
		}

		if value != "" {
			values = append(values, DimensionValue{
				Dimension: dim.Name,
				Value:     value,
			})
		}
	}

	return Partition{
		Values:   values,
		Position: 0, // Position will be set during ID generation
	}
}
