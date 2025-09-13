package engine

import (
	"time"
)

// Document represents the internal document structure that matches the public API.
// This allows the engine to work with concrete types instead of interface{}.
type Document struct {
	UUID         string
	UserFacingID string
	Title        string
	Body         string
	Status       string // We use string internally, converted to Status type at the boundary
	ParentUUID   *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ListOptions represents the options for listing documents.
// This matches the public API structure to avoid interface{} usage.
type ListOptions struct {
	// FilterByStatus limits results to specific statuses
	FilterByStatus []string // We use string internally

	// FilterByParent limits results to children of a specific parent
	FilterByParent *string

	// FilterBySearch performs a text search on title and body
	FilterBySearch string
}

// UpdateRequest represents the fields that can be updated on a document.
// This provides type safety instead of using map[string]*string.
type UpdateRequest struct {
	Title *string
	Body  *string
}
