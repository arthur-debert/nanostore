// Package storage provides the persistence layer for nanostore.
// It defines interfaces for document storage and provides implementations
// for different storage backends.
package storage

import (
	"time"

	"github.com/arthur-debert/nanostore/types"
)

// Storage defines the low-level interface for document persistence.
// Implementations handle the actual storage mechanism (file, database, etc.)
// This interface focuses on simple CRUD operations without query logic.
type Storage interface {
	// LoadAll retrieves all documents from storage
	LoadAll() ([]types.Document, error)

	// Save persists a single document to storage
	// Returns the document with any storage-generated fields (e.g., timestamps)
	Save(doc types.Document) (types.Document, error)

	// Update modifies an existing document in storage
	Update(uuid string, doc types.Document) error

	// Delete removes a document from storage by UUID
	Delete(uuid string) error

	// DeleteMultiple removes multiple documents from storage
	DeleteMultiple(uuids []string) error

	// Close releases any resources held by the storage
	Close() error
}

// Transaction represents a storage transaction for atomic operations
type Transaction interface {
	Storage

	// Commit saves all changes in the transaction
	Commit() error

	// Rollback discards all changes in the transaction
	Rollback() error
}

// Metadata contains storage metadata
type Metadata struct {
	Version   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
