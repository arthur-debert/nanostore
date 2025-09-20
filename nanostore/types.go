package nanostore

import (
	"fmt"
	"reflect"
	"time"

	"github.com/arthur-debert/nanostore/types"
)

// Re-export types from the types package for convenience
type DimensionType = types.DimensionType

const (
	Enumerated   = types.Enumerated
	Hierarchical = types.Hierarchical
)

// Document represents a document in the store with its generated ID
type Document struct {
	// These are default fields stored for every dodument
	UUID      string    // Stable internal identifier
	SimpleID  string    // Generated ID like "1", "c2", "1.2.c3"
	Title     string    // Document title
	Body      string    // Optional document body
	CreatedAt time.Time // Creation timestamp
	UpdatedAt time.Time // Last update timestamp
	// these are user defined dimensions (fields that define the partition)
	Dimensions map[string]interface{} // All dimension values and data (data prefixed with "_data.")
	// Users can add arbitrary extra fields here as needed, these are not used by nanostore itself but the api will work with them
}

// ListOptions configures how documents are listed
type ListOptions struct {
	// Filters allows filtering by any configured dimension
	// Key is dimension name, value can be a single value or slice of values
	// Example: {"status": []string{"active", "pending"}, "priority": "high"}
	Filters map[string]interface{}

	// FilterBySearch performs a text search on title and body
	// Empty string returns all documents (no filtering)
	FilterBySearch string

	// OrderBy specifies the order of results
	// Each OrderClause contains a field name and direction
	OrderBy []OrderClause

	// Limit specifies the maximum number of results to return
	// nil or negative values mean no limit
	// 0 returns no results
	Limit *int

	// Offset specifies the number of results to skip
	// nil or negative values mean no offset (start from beginning)
	// Values greater than result count return empty results
	Offset *int
}

// OrderClause represents a single ORDER BY clause
type OrderClause struct {
	Column     string
	Descending bool
}

// NewListOptions creates a new ListOptions with empty filters
func NewListOptions() ListOptions {
	return ListOptions{
		Filters: make(map[string]interface{}),
	}
}

// UpdateRequest specifies fields to update on a document
type UpdateRequest struct {
	Title      *string
	Body       *string
	Dimensions map[string]interface{} // Optional: dimension values to update (e.g., "status": "completed", "parent_uuid": "some-uuid")
}

// DimensionConfig defines a single dimension for ID partitioning and document organization.
//
// Dimensions are the core organizing principle in nanostore, determining how documents
// are partitioned for ID generation and how hierarchical relationships are established.
// Each dimension represents a discrete aspect of document classification.
//
//	Dimension Types
//
//	Enumerated Dimensions
//
// Enumerated dimensions have a finite, predefined set of valid values. They are used
// for categorical data like status, priority, or type. Enumerated dimensions support:
//
// - Value Validation: Only predefined values are accepted
// - Prefix Mapping: Values can be mapped to single-character prefixes for compact IDs
// - Default Values: Canonical values that are omitted from short IDs
// - types.Partition Creation: Documents with the same value belong to the same partition
//
// Example enumerated dimension:
//
//		{
//		    Name: "status",
//		    Type: Enumerated,
//		    Values: []string{"active", "pending", "completed", "cancelled"},
//		    Prefixes: map[string]string{
//		        "pending": "p",
//		        "completed": "c",
//		        "cancelled": "x",
//		        // "active" has no prefix (default/canonical value)
//		    },
//		    DefaultValue: "active",
//		}
//
//	 Hierarchical Dimensions
//
// Hierarchical dimensions create parent-child relationships between documents,
// enabling tree-like organizational structures. They support:
//
// - Parent References: Documents can reference other documents as parents
// - Nested IDs: Child documents inherit parent ID prefixes (e.g., "1.2.3")
// - Cascade Operations: Deletion can optionally cascade to children
// - Unlimited Depth: No artificial limits on hierarchy depth
//
// Example hierarchical dimension:
//
//		{
//		    Name: "location",
//		    Type: Hierarchical,
//		    RefField: "parent_id",
//		}
//
//	 ID Generation Impact
//
// Dimensions directly influence how SimpleIDs are generated:
//
// 1. types.Partition Formation: Documents with identical dimension values form partitions
// 2. Position Assignment: Documents get sequential positions within their partition
// 3. Prefix Application: Enumerated values become prefixes in the final ID
// 4. Hierarchical Paths: Parent-child relationships create dot-separated ID segments
//
// Example ID generation with multiple dimensions:
//
//   - Document: status=completed, priority=high, parent=1, position=3
//
//   - Generated ID: "1.ch3" (parent=1, completed="c", high="h", position=3)
//
//     Configuration Validation
//
// The system enforces several validation rules:
//
// - Unique Names: No duplicate dimension names
// - Valid Values: Non-empty values for enumerated dimensions
// - Prefix Conflicts: No duplicate prefixes across dimensions
// - Reference Fields: Hierarchical dimensions must specify RefField
// - Default Validation: DefaultValue must be in Values list
// - Type Consistency: Type-specific fields must be properly configured
//
//	Performance Considerations
//
// - Dimension Count: Limited to 7 dimensions for optimal performance
// - Value Count: No limit on enumerated values, but affects validation time
// - Prefix Length: Single-character prefixes recommended for compact IDs
// - Hierarchy Depth: Deep hierarchies may impact ID resolution performance
//
//	Best Practices
//
// 1. Canonical Values: Use empty prefixes for most common values
// 2. Meaningful Prefixes: Choose intuitive single-character prefixes
// 3. Stable Configuration: Avoid changing dimension configs after deployment
// 4. Logical Ordering: Order dimensions by importance/frequency
// 5. RefField Naming: Use consistent naming (e.g., "parent_id", "parent_uuid")
type DimensionConfig struct {
	// Name is the database column name and identifier for this dimension.
	// Must be unique across all dimensions in the configuration.
	// Used in partition keys, filtering, and API operations.
	//
	// Examples: "status", "priority", "category", "location"
	Name string

	// Type specifies whether this is an enumerated or hierarchical dimension.
	// This fundamentally changes how the dimension behaves:
	// - Enumerated: Fixed set of values with optional prefixes
	// - Hierarchical: Parent-child relationships with unlimited values
	Type DimensionType

	// Values lists the valid values for enumerated dimensions.
	// Each value represents a possible state or category.
	// Order is preserved but doesn't affect functionality.
	// Ignored for hierarchical dimensions.
	//
	// Example: []string{"draft", "review", "approved", "published"}
	Values []string

	// Prefixes maps enumerated values to their single-character ID prefixes.
	// Prefixes allow values to be represented compactly in SimpleIDs.
	// Values without prefixes are considered "canonical" and omitted from IDs.
	// Ignored for hierarchical dimensions.
	//
	// Example: map[string]string{
	//     "draft": "d",
	//     "review": "r",
	//     "published": "p",
	//     // "approved" has no prefix (canonical value)
	// }
	Prefixes map[string]string

	// RefField specifies the foreign key field name for hierarchical dimensions.
	// This field in document dimensions will contain the parent document's UUID.
	// The field name is used for:
	// - Storing parent references in document data
	// - Command preprocessing to resolve parent SimpleIDs
	// - Cascade deletion operations
	// Ignored for enumerated dimensions.
	//
	// Common values: "parent_id", "parent_uuid", "location_parent"
	RefField string

	// DefaultValue specifies the default value for enumerated dimensions.
	// Used when creating documents without an explicit value for this dimension.
	// Must be present in the Values list.
	// Canonical views often use default values to define "normal" document states.
	// Ignored for hierarchical dimensions.
	//
	// Example: "active" for a status dimension with values ["active", "archived"]
	DefaultValue string
}

// Config defines the overall configuration for the nanostore
type Config struct {
	// Dimensions defines the ID partitioning dimensions
	Dimensions []DimensionConfig

	// dimensionSet is the new internal representation
	// Will be populated from Dimensions during initialization
	dimensionSet *types.DimensionSet
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
func (c *Config) GetDimensionSet() *types.DimensionSet {
	if c.dimensionSet == nil {
		var dims []types.Dimension
		for _, dimConfig := range c.Dimensions {
			dim := types.Dimension{
				Name:         dimConfig.Name,
				Type:         dimConfig.Type,
				Values:       dimConfig.Values,
				Prefixes:     dimConfig.Prefixes,
				DefaultValue: dimConfig.DefaultValue,
				RefField:     dimConfig.RefField,
			}
			dims = append(dims, dim)
		}
		c.dimensionSet = types.NewDimensionSet(dims)
	}
	return c.dimensionSet
}

// ValidateConfig validates a configuration
func ValidateConfig(config Config) error {
	return config.GetDimensionSet().Validate()
}

// Store defines the public interface for the document store
type Store interface {
	// List returns documents based on the provided options
	// The returned documents include generated user-facing IDs
	List(opts ListOptions) ([]Document, error)

	// Add creates a new document with the given title and dimension values
	// The dimensions map allows setting any dimension values, including:
	// - Enumerated dimensions (e.g., "status": "pending")
	// - Hierarchical dimensions (e.g., "parent_uuid": "parent-id")
	// Dimensions not specified will use their default values from the configuration
	// Returns the UUID of the created document
	Add(title string, dimensions map[string]interface{}) (string, error)

	// Update modifies an existing document
	Update(id string, updates UpdateRequest) error

	// ResolveUUID converts a simple ID (e.g., "1.2.c3") to a UUID
	ResolveUUID(simpleID string) (string, error)

	// Delete removes a document and optionally its children
	// If cascade is true, all child documents are also deleted
	// If cascade is false and the document has children, an error is returned
	Delete(id string, cascade bool) error

	// DeleteByDimension removes all documents matching dimension filters
	// For example: DeleteByDimension(map[string]interface{}{"status": "archived"})
	// Multiple filters are combined with AND
	// Returns the number of documents deleted
	DeleteByDimension(filters map[string]interface{}) (int, error)

	// DeleteWhere removes all documents matching a custom WHERE clause
	// The where clause should not include the "WHERE" keyword itself
	// For example: DeleteWhere("status = 'archived' AND priority = 'low'")
	// Use with caution as it allows arbitrary SQL conditions
	// Returns the number of documents deleted
	DeleteWhere(whereClause string, args ...interface{}) (int, error)

	// UpdateByDimension updates all documents matching dimension filters
	// For example: UpdateByDimension(map[string]interface{}{"status": "pending"}, UpdateRequest{Title: &newTitle})
	// Multiple filters are combined with AND
	// Returns the number of documents updated
	UpdateByDimension(filters map[string]interface{}, updates UpdateRequest) (int, error)

	// UpdateWhere updates all documents matching a custom WHERE clause
	// The where clause should not include the "WHERE" keyword itself
	// For example: UpdateWhere("created_at < ?", UpdateRequest{...}, time.Now().AddDate(0, -1, 0))
	// Use with caution as it allows arbitrary SQL conditions
	// Returns the number of documents updated
	UpdateWhere(whereClause string, updates UpdateRequest, args ...interface{}) (int, error)

	// Close releases any resources held by the store
	Close() error
}

// ToTypesDocument converts a local Document to types.Document
func ToTypesDocument(doc Document) types.Document {
	return types.Document{
		UUID:       doc.UUID,
		SimpleID:   doc.SimpleID,
		Title:      doc.Title,
		Body:       doc.Body,
		Dimensions: doc.Dimensions,
		CreatedAt:  doc.CreatedAt,
		UpdatedAt:  doc.UpdatedAt,
	}
}

// ValidateSimpleType ensures a dimension value is a simple type (string, number, bool)
func ValidateSimpleType(value interface{}, dimensionName string) error {
	if value == nil {
		return nil
	}

	// Check the type using reflection
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return nil
	case reflect.Slice, reflect.Array:
		return fmt.Errorf("dimension '%s' cannot be an array/slice type, got %T", dimensionName, value)
	case reflect.Map:
		return fmt.Errorf("dimension '%s' cannot be a map type, got %T", dimensionName, value)
	case reflect.Struct:
		// Allow time.Time as it's commonly used
		if _, ok := value.(time.Time); ok {
			return nil
		}
		return fmt.Errorf("dimension '%s' cannot be a struct type, got %T", dimensionName, value)
	case reflect.Ptr, reflect.Interface:
		// Dereference and check the underlying type
		if v.IsNil() {
			return nil
		}
		return ValidateSimpleType(v.Elem().Interface(), dimensionName)
	default:
		return fmt.Errorf("dimension '%s' must be a simple type (string, number, or bool), got %T", dimensionName, value)
	}
}
