package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/arthur-debert/nanostore/types"
	"github.com/gofrs/flock"
	"github.com/google/uuid"
)

// JSONStorage implements the Storage interface using a JSON file
type JSONStorage struct {
	filePath string
	fileLock *flock.Flock
	mu       sync.RWMutex
}

// storeData represents the JSON file structure
type storeData struct {
	Documents []types.Document `json:"documents"`
	Metadata  metadata         `json:"metadata"`
}

// metadata contains storage metadata
type metadata struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewJSONStorage creates a new JSON file storage
func NewJSONStorage(filePath string) *JSONStorage {
	lockPath := filePath + ".lock"
	return &JSONStorage{
		filePath: filePath,
		fileLock: flock.New(lockPath),
	}
}

// LoadAll retrieves all documents from the JSON file
func (s *JSONStorage) LoadAll() ([]types.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Acquire file lock
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	locked, err := s.fileLock.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("could not acquire file lock")
	}
	defer func() { _ = s.fileLock.Unlock() }()

	// Check if file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		// File doesn't exist yet, return empty slice
		return []types.Document{}, nil
	}

	// Read the file
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Empty file is OK
	if len(data) == 0 {
		return []types.Document{}, nil
	}

	// Parse JSON
	var store storeData
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return store.Documents, nil
}

// Save persists a single document to storage
func (s *JSONStorage) Save(doc types.Document) (types.Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load current data
	store, err := s.loadLocked()
	if err != nil {
		return types.Document{}, err
	}

	// Generate UUID if not provided
	if doc.UUID == "" {
		doc.UUID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	doc.CreatedAt = now
	doc.UpdatedAt = now

	// Add document
	store.Documents = append(store.Documents, doc)
	store.Metadata.UpdatedAt = now

	// Save back
	if err := s.saveLocked(store); err != nil {
		return types.Document{}, err
	}

	return doc, nil
}

// Update modifies an existing document
func (s *JSONStorage) Update(uuid string, doc types.Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load current data
	store, err := s.loadLocked()
	if err != nil {
		return err
	}

	// Find document
	found := false
	for i, d := range store.Documents {
		if d.UUID == uuid {
			doc.UUID = uuid             // Preserve UUID
			doc.CreatedAt = d.CreatedAt // Preserve creation time
			doc.UpdatedAt = time.Now()
			store.Documents[i] = doc
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("document not found: %s", uuid)
	}

	store.Metadata.UpdatedAt = time.Now()
	return s.saveLocked(store)
}

// Delete removes a document by UUID
func (s *JSONStorage) Delete(uuid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load current data
	store, err := s.loadLocked()
	if err != nil {
		return err
	}

	// Find and remove document
	found := false
	for i, doc := range store.Documents {
		if doc.UUID == uuid {
			store.Documents = append(store.Documents[:i], store.Documents[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("document not found: %s", uuid)
	}

	store.Metadata.UpdatedAt = time.Now()
	return s.saveLocked(store)
}

// DeleteMultiple removes multiple documents
func (s *JSONStorage) DeleteMultiple(uuids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load current data
	store, err := s.loadLocked()
	if err != nil {
		return err
	}

	// Create a set for efficient lookup
	toDelete := make(map[string]bool)
	for _, uuid := range uuids {
		toDelete[uuid] = true
	}

	// Filter out documents to delete
	newDocs := make([]types.Document, 0, len(store.Documents))
	for _, doc := range store.Documents {
		if !toDelete[doc.UUID] {
			newDocs = append(newDocs, doc)
		}
	}

	store.Documents = newDocs
	store.Metadata.UpdatedAt = time.Now()
	return s.saveLocked(store)
}

// Close releases resources
func (s *JSONStorage) Close() error {
	// Clean up lock file
	lockPath := s.filePath + ".lock"
	_ = os.Remove(lockPath)
	return nil
}

// loadLocked loads data while holding the write lock
func (s *JSONStorage) loadLocked() (*storeData, error) {
	// Acquire file lock
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	locked, err := s.fileLock.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("could not acquire file lock")
	}
	defer func() { _ = s.fileLock.Unlock() }()

	// Check if file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		// Return empty store
		return &storeData{
			Documents: []types.Document{},
			Metadata: metadata{
				Version:   "1.0",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}, nil
	}

	// Read file
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Empty file
	if len(data) == 0 {
		return &storeData{
			Documents: []types.Document{},
			Metadata: metadata{
				Version:   "1.0",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}, nil
	}

	// Parse JSON
	var store storeData
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &store, nil
}

// saveLocked saves data while holding the write lock
func (s *JSONStorage) saveLocked(store *storeData) error {
	// Acquire file lock
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	locked, err := s.fileLock.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("could not acquire file lock")
	}
	defer func() { _ = s.fileLock.Unlock() }()

	// Marshal to JSON
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write atomically
	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Rename to final location
	if err := os.Rename(tmpFile, s.filePath); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}
