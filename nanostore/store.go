package nanostore

import (
	"github.com/arthur-debert/nanostore/nanostore/internal/engine"
)

// storeAdapter wraps the internal engine to implement the public Store interface.
// With the refactored engine using concrete types, this adapter now serves as a
// clean translation layer between public and internal types without any runtime
// type assertions or interface{} usage.
type storeAdapter struct {
	engine engine.Engine // Now using the strongly-typed Engine interface
}

// newStore creates a new store instance (internal constructor)
func newStore(dbPath string) (Store, error) {
	eng, err := engine.New(dbPath)
	if err != nil {
		return nil, err
	}
	return &storeAdapter{engine: eng}, nil
}

// List returns documents based on the provided options
func (s *storeAdapter) List(opts ListOptions) ([]Document, error) {
	// Convert public ListOptions to internal engine.ListOptions
	// This is a clean type conversion at compile time, not runtime assertions
	engineOpts := engine.ListOptions{
		FilterByParent: opts.FilterByParent,
		FilterBySearch: opts.FilterBySearch,
	}

	// Convert Status enums to strings for internal use
	if len(opts.FilterByStatus) > 0 {
		engineOpts.FilterByStatus = make([]string, len(opts.FilterByStatus))
		for i, status := range opts.FilterByStatus {
			engineOpts.FilterByStatus[i] = string(status)
		}
	}

	// Get strongly-typed results from engine
	engineDocs, err := s.engine.List(engineOpts)
	if err != nil {
		return nil, err
	}

	// Convert internal Documents to public Documents
	// This is now a clean mapping between two concrete types
	docs := make([]Document, len(engineDocs))
	for i, engineDoc := range engineDocs {
		docs[i] = Document{
			UUID:         engineDoc.UUID,
			UserFacingID: engineDoc.UserFacingID,
			Title:        engineDoc.Title,
			Body:         engineDoc.Body,
			Status:       Status(engineDoc.Status), // Safe conversion of known values
			ParentUUID:   engineDoc.ParentUUID,
			CreatedAt:    engineDoc.CreatedAt,
			UpdatedAt:    engineDoc.UpdatedAt,
		}
	}

	return docs, nil
}

// Add creates a new document
func (s *storeAdapter) Add(title string, parentID *string) (string, error) {
	return s.engine.Add(title, parentID)
}

// Update modifies an existing document
func (s *storeAdapter) Update(id string, updates UpdateRequest) error {
	// Convert public UpdateRequest to internal engine.UpdateRequest
	// This is now a simple struct-to-struct mapping, no maps or runtime checks
	engineUpdate := engine.UpdateRequest{
		Title: updates.Title,
		Body:  updates.Body,
	}
	return s.engine.Update(id, engineUpdate)
}

// SetStatus changes the status of a document
func (s *storeAdapter) SetStatus(id string, status Status) error {
	return s.engine.SetStatus(id, string(status))
}

// ResolveUUID converts a user-facing ID to a UUID
func (s *storeAdapter) ResolveUUID(userFacingID string) (string, error) {
	return s.engine.ResolveUUID(userFacingID)
}

// Close releases any resources
func (s *storeAdapter) Close() error {
	return s.engine.Close()
}
