// Package types defines the core types used throughout the nanostore library.
// This package exists to prevent circular dependencies between the public API
// and internal engine packages while maintaining a single source of truth for
// all type definitions.
package types

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
