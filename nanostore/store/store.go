// Package store provides the main orchestrator for nanostore.
// It coordinates between storage, query processing, ID generation, and other components
// to provide the complete document store functionality.
package store

import (
	"time"

	"github.com/arthur-debert/nanostore/types"
)

// Store defines the public interface for the document store.
// It provides methods for managing documents with automatic ID generation,
// hierarchical organization, and flexible querying capabilities.
type Store interface {
	// List returns documents based on the provided options
	// The returned documents include generated user-facing IDs
	List(opts types.ListOptions) ([]types.Document, error)

	// Add creates a new document with the given title and dimension values
	// The dimensions map allows setting any dimension values, including:
	// - Enumerated dimensions (e.g., "status": "pending")
	// - Hierarchical dimensions (e.g., "parent_uuid": "parent-id")
	// Dimensions not specified will use their default values from the configuration
	// Returns the UUID of the created document
	Add(title string, dimensions map[string]interface{}) (string, error)

	// Update modifies an existing document
	Update(id string, updates types.UpdateRequest) error

	// ResolveUUID converts a simple ID (e.g., "1.2.c3") to a UUID
	ResolveUUID(simpleID string) (string, error)

	// Delete removes a document and optionally its children
	Delete(id string, cascade bool) error

	// DeleteByDimension removes all documents matching the specified dimension filters
	DeleteByDimension(filters map[string]interface{}) (int, error)

	// UpdateByDimension updates all documents matching the specified dimension filters
	UpdateByDimension(filters map[string]interface{}, updates types.UpdateRequest) (int, error)

	// DeleteWhere removes documents matching a custom WHERE clause
	DeleteWhere(whereClause string, args ...interface{}) (int, error)

	// UpdateWhere updates documents matching a custom WHERE clause
	UpdateWhere(whereClause string, updates types.UpdateRequest, args ...interface{}) (int, error)

	// UpdateByUUIDs updates multiple documents by their UUIDs in a single operation
	UpdateByUUIDs(uuids []string, updates types.UpdateRequest) (int, error)

	// DeleteByUUIDs deletes multiple documents by their UUIDs in a single operation
	DeleteByUUIDs(uuids []string) (int, error)

	// GetByID retrieves a single document by its UUID
	GetByID(id string) (*types.Document, error)

	// Close releases any resources held by the store
	Close() error
}

// TestStore extends Store with methods useful for testing
type TestStore interface {
	Store

	// SetTimeFunc sets a custom time function for deterministic timestamps
	SetTimeFunc(fn func() time.Time)
}
