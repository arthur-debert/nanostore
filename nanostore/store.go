package nanostore

import (
	"github.com/arthur-debert/nanostore/nanostore/internal/engine"
)

// Engine defines the internal storage engine interface.
// It directly uses the public nanostore types to avoid redundant type definitions.
// This interface is defined here rather than in the engine package to avoid
// circular imports while still maintaining type safety.
type Engine interface {
	List(opts ListOptions) ([]Document, error)
	Add(title string, parentID *string, dimensions map[string]string) (string, error)
	Update(id string, updates UpdateRequest) error
	SetStatus(id string, status Status) error
	ResolveUUID(userFacingID string) (string, error)
	Delete(id string, cascade bool) error
	Close() error
}

// storeAdapter wraps the internal engine to implement the public Store interface.
// After eliminating redundant type definitions, this adapter is now a simple
// pass-through that exists solely to:
// 1. Hide the internal engine package from public API users
// 2. Maintain a clean separation between public interface and internal implementation
// All methods are now simple delegations with no type conversions needed.
type storeAdapter struct {
	engine Engine // Using the local Engine interface
}

// newStoreWithConfig creates a new store instance with custom configuration
func newStoreWithConfig(dbPath string, config Config) (Store, error) {
	// Create a new configurable store with the provided configuration
	eng, err := engine.New(dbPath, config)
	if err != nil {
		return nil, err
	}
	// No need to cast, it already implements Engine interface
	return &storeAdapter{engine: eng}, nil
}

// List returns documents based on the provided options
func (s *storeAdapter) List(opts ListOptions) ([]Document, error) {
	// Simple pass-through - no conversion needed!
	return s.engine.List(opts)
}

// Add creates a new document
func (s *storeAdapter) Add(title string, parentID *string, dimensions map[string]string) (string, error) {
	return s.engine.Add(title, parentID, dimensions)
}

// Update modifies an existing document
func (s *storeAdapter) Update(id string, updates UpdateRequest) error {
	// Simple pass-through - no conversion needed!
	return s.engine.Update(id, updates)
}

// SetStatus changes the status of a document
func (s *storeAdapter) SetStatus(id string, status Status) error {
	// Simple pass-through - Status type is now used throughout
	return s.engine.SetStatus(id, status)
}

// ResolveUUID converts a user-facing ID to a UUID
func (s *storeAdapter) ResolveUUID(userFacingID string) (string, error) {
	return s.engine.ResolveUUID(userFacingID)
}

// Delete removes a document and optionally its children
func (s *storeAdapter) Delete(id string, cascade bool) error {
	return s.engine.Delete(id, cascade)
}

// Close releases any resources
func (s *storeAdapter) Close() error {
	return s.engine.Close()
}
