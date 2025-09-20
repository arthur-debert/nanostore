// Package storage provides the persistence layer for nanostore.
// It defines interfaces for document storage and provides implementations
// for different storage backends.
package storage

import (
	"time"

	"github.com/arthur-debert/nanostore/types"
)

// StoreData represents the complete data structure stored in the backend
type StoreData struct {
	Documents []types.Document `json:"documents"`
	Metadata  Metadata         `json:"metadata"`
}

// Metadata contains storage metadata
type Metadata struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Storage defines the low-level interface for batch persistence.
// This interface handles loading and saving the entire document collection
// as a single unit, which matches the JSON file backend's natural behavior.
type Storage interface {
	// Load reads the entire store data from the backend
	Load() (*StoreData, error)

	// Save writes the entire store data to the backend
	Save(data *StoreData) error

	// Close releases any resources held by the storage
	Close() error
}
