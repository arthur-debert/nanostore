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
	Title *string
	Body  *string
}
