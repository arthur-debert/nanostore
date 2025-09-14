// Package nanostore provides a document and ID store library that uses SQLite
// to manage document storage and dynamically generate user-facing, contiguous IDs.
//
// This package replaces pkg/idm and parts of pkg/too/store with a cleaner,
// more focused approach to document management with configurable ID schemes.
package nanostore

import "time"

// Status represents the status of a document
type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
)

// Document represents a document in the store with its generated ID
type Document struct {
	UUID         string    // Stable internal identifier
	UserFacingID string    // Generated ID like "1", "c2", "1.2.c3"
	Title        string    // Document title
	Body         string    // Optional document body
	Status       Status    // Current status
	ParentUUID   *string   // UUID of parent document, if any
	CreatedAt    time.Time // Creation timestamp
	UpdatedAt    time.Time // Last update timestamp
}

// ListOptions configures how documents are listed
type ListOptions struct {
	// FilterByStatus limits results to specific statuses
	// If empty, all statuses are returned
	FilterByStatus []Status

	// FilterByParent limits results to children of a specific parent
	// Use nil for root documents only
	FilterByParent *string

	// FilterBySearch performs a text search on title and body
	FilterBySearch string
}

// UpdateRequest specifies fields to update on a document
type UpdateRequest struct {
	Title      *string
	Body       *string
	ParentID   *string           // Optional: new parent UUID (nil = no change, empty string = make root)
	Dimensions map[string]string // Optional: dimension values to update (e.g., "priority": "high")
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

	// Add creates a new document with the given title, optional parent, and dimension values
	// The dimensions map allows setting custom dimension values (e.g., "priority": "high")
	// Dimensions not specified will use their default values from the configuration
	// Returns the UUID of the created document
	Add(title string, parentID *string, dimensions map[string]string) (string, error)

	// Update modifies an existing document
	Update(id string, updates UpdateRequest) error

	// SetStatus changes the status of a document
	SetStatus(id string, status Status) error

	// ResolveUUID converts a user-facing ID (e.g., "1.2.c3") to a UUID
	ResolveUUID(userFacingID string) (string, error)

	// Delete removes a document and optionally its children
	// If cascade is true, all child documents are also deleted
	// If cascade is false and the document has children, an error is returned
	Delete(id string, cascade bool) error

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