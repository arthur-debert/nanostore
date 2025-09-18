package types

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

// DimensionSetFromConfig converts old config to new dimension set
func DimensionSetFromConfig(config Config) *DimensionSet {
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
