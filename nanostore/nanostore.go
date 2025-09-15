// Package nanostore provides a document and ID store library that uses SQLite
// to manage document storage and dynamically generate user-facing, contiguous IDs.
//
// This package replaces pkg/idm and parts of pkg/too/store with a cleaner,
// more focused approach to document management with configurable ID schemes.
package nanostore

import "time"

// Document represents a document in the store with its generated ID
type Document struct {
	UUID         string                 // Stable internal identifier
	UserFacingID string                 // Generated ID like "1", "c2", "1.2.c3"
	Title        string                 // Document title
	Body         string                 // Optional document body
	Dimensions   map[string]interface{} // All dimension values for this document
	CreatedAt    time.Time              // Creation timestamp
	UpdatedAt    time.Time              // Last update timestamp
}

// GetParentUUID returns the parent UUID from hierarchical dimension, if it exists
func (d *Document) GetParentUUID() *string {
	// Check common parent field names
	parentFields := []string{"parent_uuid", "parent"}
	for _, field := range parentFields {
		if parent, ok := d.Dimensions[field].(string); ok && parent != "" {
			return &parent
		}
	}
	return nil
}

// ListOptions configures how documents are listed
type ListOptions struct {
	// Filters allows filtering by any configured dimension
	// Key is dimension name, value can be a single value or slice of values
	// Example: {"status": []string{"active", "pending"}, "priority": "high"}
	Filters map[string]interface{}

	// FilterBySearch performs a text search on title and body
	FilterBySearch string
}

// NewListOptions creates a new ListOptions with empty filters
func NewListOptions() ListOptions {
	return ListOptions{
		Filters: make(map[string]interface{}),
	}
}

// WithParentFilter adds a parent filter to ListOptions
func (opts ListOptions) WithParentFilter(parentUUID *string) ListOptions {
	if opts.Filters == nil {
		opts.Filters = make(map[string]interface{})
	}
	if parentUUID != nil {
		opts.Filters["parent_uuid"] = *parentUUID
	} else {
		opts.Filters["parent_uuid"] = nil
	}
	return opts
}

// UpdateRequest specifies fields to update on a document
type UpdateRequest struct {
	Title      *string
	Body       *string
	Dimensions map[string]string // Optional: dimension values to update (e.g., "status": "completed", "parent_uuid": "some-uuid")
}

// DimensionType defines the type of dimension for ID partitioning
type DimensionType int

const (
	// Enumerated dimensions have predefined values (e.g., status, priority)
	Enumerated DimensionType = iota
	// Hierarchical dimensions create parent-child relationships
	Hierarchical
)

// DimensionConfig defines a single dimension for ID partitioning
type DimensionConfig struct {
	// Name is the database column name and identifier for this dimension
	Name string

	// Type specifies whether this is an enumerated or hierarchical dimension
	Type DimensionType

	// Values lists the valid values for enumerated dimensions
	// Ignored for hierarchical dimensions
	Values []string

	// Prefixes maps values to their ID prefixes
	// For enumerated dimensions: value -> prefix (e.g., "completed" -> "c")
	// Ignored for hierarchical dimensions
	Prefixes map[string]string

	// RefField specifies the foreign key field name for hierarchical dimensions
	// For hierarchical dimensions: typically "parent_uuid"
	// Ignored for enumerated dimensions
	RefField string

	// DefaultValue specifies the default value for enumerated dimensions
	// Used when inserting new documents without explicit value
	DefaultValue string
}

// Config defines the overall configuration for the nanostore
type Config struct {
	// Dimensions defines the ID partitioning dimensions
	Dimensions []DimensionConfig
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

	// ResolveUUID converts a user-facing ID (e.g., "1.2.c3") to a UUID
	ResolveUUID(userFacingID string) (string, error)

	// Delete removes a document and optionally its children
	// If cascade is true, all child documents are also deleted
	// If cascade is false and the document has children, an error is returned
	Delete(id string, cascade bool) error

	// DeleteByDimension removes all documents matching a specific dimension value
	// For example: DeleteByDimension("status", "archived") or DeleteByDimension("priority", "low")
	// Returns the number of documents deleted
	DeleteByDimension(dimension string, value string) (int, error)

	// DeleteWhere removes all documents matching a custom WHERE clause
	// The where clause should not include the "WHERE" keyword itself
	// For example: DeleteWhere("status = 'archived' AND priority = 'low'")
	// Use with caution as it allows arbitrary SQL conditions
	// Returns the number of documents deleted
	DeleteWhere(whereClause string, args ...interface{}) (int, error)

	// Close releases any resources held by the store
	Close() error
}

// New creates a new Store instance with the specified dimension configuration
// Use ":memory:" for an in-memory database (useful for testing)
func New(dbPath string, config Config) (Store, error) {
	// First validate the configuration
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}
	return newConfigurableStore(dbPath, config)
}
