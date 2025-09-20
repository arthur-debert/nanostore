package types

// FieldResolver provides dimension-aware field resolution
type FieldResolver struct {
	dimensionSet *DimensionSet
}

// NewFieldResolver creates a new field resolver
func NewFieldResolver(dimensionSet *DimensionSet) *FieldResolver {
	return &FieldResolver{
		dimensionSet: dimensionSet,
	}
}

// IsReferenceField determines if a field name is a hierarchical reference field
func (fr *FieldResolver) IsReferenceField(fieldName string) bool {
	// Check all hierarchical dimensions for ref fields
	for _, dim := range fr.dimensionSet.Hierarchical() {
		if dim.RefField == fieldName {
			return true
		}
	}
	return false
}

// GetDimensionForRefField returns the dimension that uses the given reference field
func (fr *FieldResolver) GetDimensionForRefField(fieldName string) *Dimension {
	for _, dim := range fr.dimensionSet.Hierarchical() {
		if dim.RefField == fieldName {
			dimCopy := dim
			return &dimCopy
		}
	}
	return nil
}
