package nanostore

import (
	"time"

	"github.com/arthur-debert/nanostore/nanostore/internal/engine"
)

// storeAdapter wraps the internal engine to implement the public Store interface
type storeAdapter struct {
	engine interface {
		List(opts interface{}) ([]interface{}, error)
		Add(title string, parentID *string) (string, error)
		Update(id string, updates interface{}) error
		SetStatus(id string, status string) error
		ResolveUUID(userFacingID string) (string, error)
		Close() error
	}
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
	results, err := s.engine.List(opts)
	if err != nil {
		return nil, err
	}

	docs := make([]Document, 0, len(results))
	for _, result := range results {
		// Type assert the map from interface{}
		m, ok := result.(map[string]interface{})
		if !ok {
			continue
		}

		doc := Document{
			UUID:         m["uuid"].(string),
			UserFacingID: m["user_facing_id"].(string),
			Title:        m["title"].(string),
			Body:         m["body"].(string),
			Status:       Status(m["status"].(string)),
			CreatedAt:    time.Unix(m["created_at"].(int64), 0),
			UpdatedAt:    time.Unix(m["updated_at"].(int64), 0),
		}

		if parentUUID, ok := m["parent_uuid"].(string); ok {
			doc.ParentUUID = &parentUUID
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

// Add creates a new document
func (s *storeAdapter) Add(title string, parentID *string) (string, error) {
	return s.engine.Add(title, parentID)
}

// Update modifies an existing document
func (s *storeAdapter) Update(id string, updates UpdateRequest) error {
	// Convert UpdateRequest to map for the engine
	updateMap := make(map[string]*string)
	if updates.Title != nil {
		updateMap["title"] = updates.Title
	}
	if updates.Body != nil {
		updateMap["body"] = updates.Body
	}
	return s.engine.Update(id, updateMap)
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
