package nanostore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// jsonFileStore implements the Store interface using a JSON file backend
type jsonFileStore struct {
	filePath string
	config   Config
	mu       sync.RWMutex
	data     *storeData
}

// storeData represents the in-memory data structure
type storeData struct {
	Documents []Document    `json:"documents"`
	Metadata  storeMetadata `json:"metadata"`
}

// storeMetadata contains store metadata
type storeMetadata struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// newJSONFileStore creates a new JSON file store
func newJSONFileStore(filePath string, config Config) (*jsonFileStore, error) {
	store := &jsonFileStore{
		filePath: filePath,
		config:   config,
		data: &storeData{
			Documents: []Document{},
			Metadata: storeMetadata{
				Version:   "1.0",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Try to load existing data (ignore if file doesn't exist)
	_ = store.load() // Ignore errors for now since load() is not implemented

	return store, nil
}

// load reads the JSON file into memory
func (s *jsonFileStore) load() error {
	// No locking here - caller must handle locking

	// Check if file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		// File doesn't exist yet, that's OK
		return nil
	}

	// Read the file
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Empty file is OK
	if len(data) == 0 {
		return nil
	}

	// Parse JSON
	var storeData storeData
	if err := json.Unmarshal(data, &storeData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	s.data = &storeData
	return nil
}

// save writes the in-memory data to the JSON file
func (s *jsonFileStore) save() error {
	// No locking here - caller must handle locking

	// Update metadata
	s.data.Metadata.UpdatedAt = time.Now()

	// Marshal to JSON with pretty printing
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to file atomically (write to temp file, then rename)
	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Rename temp file to actual file (atomic on most filesystems)
	if err := os.Rename(tmpFile, s.filePath); err != nil {
		os.Remove(tmpFile) // Clean up temp file
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// List returns documents based on the provided options
func (s *jsonFileStore) List(opts ListOptions) ([]Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// For now, just return all documents (no filtering/ordering/pagination yet)
	// We'll implement the full query engine in the next step

	// Make a copy of documents to avoid mutations
	result := make([]Document, len(s.data.Documents))
	copy(result, s.data.Documents)

	// Set UserFacingID to UUID for now (will be replaced with proper ID generation)
	for i := range result {
		result[i].UserFacingID = result[i].UUID
	}

	return result, nil
}

// Add creates a new document
func (s *jsonFileStore) Add(title string, dimensions map[string]interface{}) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate UUID
	docUUID := uuid.New().String()

	// Create document
	doc := Document{
		UUID:       docUUID,
		Title:      title,
		Body:       "", // Empty body by default
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Dimensions: make(map[string]interface{}),
	}

	// Apply dimension values
	for _, dimConfig := range s.config.Dimensions {
		if dimConfig.Type == Enumerated {
			// Check if value was provided
			if val, exists := dimensions[dimConfig.Name]; exists {
				// Validate the value
				strVal := fmt.Sprintf("%v", val)
				if !contains(dimConfig.Values, strVal) {
					return "", fmt.Errorf("invalid value %q for dimension %q", strVal, dimConfig.Name)
				}
				doc.Dimensions[dimConfig.Name] = strVal
			} else if dimConfig.DefaultValue != "" {
				// Use default value
				doc.Dimensions[dimConfig.Name] = dimConfig.DefaultValue
			}
		} else if dimConfig.Type == Hierarchical {
			// Handle parent reference
			if val, exists := dimensions[dimConfig.RefField]; exists {
				doc.Dimensions[dimConfig.RefField] = fmt.Sprintf("%v", val)
			}
		}
	}

	// Add to store
	s.data.Documents = append(s.data.Documents, doc)

	// Save to file
	if err := s.save(); err != nil {
		// Remove the document on save failure
		s.data.Documents = s.data.Documents[:len(s.data.Documents)-1]
		return "", fmt.Errorf("failed to save: %w", err)
	}

	return docUUID, nil
}

// Update modifies an existing document
func (s *jsonFileStore) Update(id string, updates UpdateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the document by UUID (for now, we'll implement smart ID resolution later)
	var found bool
	var docIndex int
	for i, doc := range s.data.Documents {
		if doc.UUID == id {
			found = true
			docIndex = i
			break
		}
	}

	if !found {
		return fmt.Errorf("document not found: %s", id)
	}

	// Apply updates
	doc := &s.data.Documents[docIndex]
	doc.UpdatedAt = time.Now()

	// Update title if provided
	if updates.Title != nil {
		doc.Title = *updates.Title
	}

	// Update body if provided
	if updates.Body != nil {
		doc.Body = *updates.Body
	}

	// Update dimensions if provided
	if updates.Dimensions != nil {
		// Validate dimension updates
		for dimName, value := range updates.Dimensions {
			// Find dimension config
			var dimConfig *DimensionConfig
			for _, dc := range s.config.Dimensions {
				if dc.Name == dimName || dc.RefField == dimName {
					dimConfig = &dc
					break
				}
			}

			if dimConfig == nil {
				return fmt.Errorf("unknown dimension: %s", dimName)
			}

			// Validate enumerated dimension values
			if dimConfig.Type == Enumerated && value != nil {
				strVal := fmt.Sprintf("%v", value)
				if !contains(dimConfig.Values, strVal) {
					return fmt.Errorf("invalid value %q for dimension %q", strVal, dimName)
				}
				doc.Dimensions[dimName] = strVal
			} else if dimConfig.Type == Hierarchical {
				// Store hierarchical dimension value
				if value != nil {
					doc.Dimensions[dimConfig.RefField] = fmt.Sprintf("%v", value)
				} else {
					delete(doc.Dimensions, dimConfig.RefField)
				}
			}
		}
	}

	// Save to file
	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}

	return nil
}

// ResolveUUID converts a user-facing ID to a UUID
func (s *jsonFileStore) ResolveUUID(userFacingID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// TODO: Implement canonical view ID resolution
	return "", errors.New("not implemented")
}

// Delete removes a document
func (s *jsonFileStore) Delete(id string, cascade bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.deleteInternal(id, cascade)
}

// deleteInternal is the internal delete method that doesn't lock
func (s *jsonFileStore) deleteInternal(id string, cascade bool) error {
	// Find the document
	var found bool
	var docIndex int
	for i, doc := range s.data.Documents {
		if doc.UUID == id {
			found = true
			docIndex = i
			break
		}
	}

	if !found {
		return fmt.Errorf("document not found: %s", id)
	}

	// Check for children if cascade is false
	if !cascade {
		// Find hierarchical dimension
		var hierDim *DimensionConfig
		for _, dim := range s.config.Dimensions {
			if dim.Type == Hierarchical {
				hierDim = &dim
				break
			}
		}

		if hierDim != nil {
			// Check if any document has this document as parent
			for _, doc := range s.data.Documents {
				if parentID, exists := doc.Dimensions[hierDim.RefField]; exists {
					if parentID == id {
						return fmt.Errorf("document has children and cascade is false")
					}
				}
			}
		}
	}

	// If cascade is true, delete all children
	if cascade {
		// Find hierarchical dimension
		var hierDim *DimensionConfig
		for _, dim := range s.config.Dimensions {
			if dim.Type == Hierarchical {
				hierDim = &dim
				break
			}
		}

		if hierDim != nil {
			// Collect child IDs to delete
			var childIDs []string
			for _, doc := range s.data.Documents {
				if parentID, exists := doc.Dimensions[hierDim.RefField]; exists {
					if parentID == id {
						childIDs = append(childIDs, doc.UUID)
					}
				}
			}

			// Recursively delete children (using internal method)
			for _, childID := range childIDs {
				if err := s.deleteInternal(childID, true); err != nil {
					return fmt.Errorf("failed to delete child %s: %w", childID, err)
				}
			}
		}
	}

	// Remove the document
	s.data.Documents = append(s.data.Documents[:docIndex], s.data.Documents[docIndex+1:]...)

	// Save to file
	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}

	return nil
}

// DeleteByDimension removes documents matching dimension filters
func (s *jsonFileStore) DeleteByDimension(filters map[string]interface{}) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// TODO: Implement bulk deletion by dimensions
	return 0, errors.New("not implemented")
}

// DeleteWhere removes documents matching a custom WHERE clause
func (s *jsonFileStore) DeleteWhere(whereClause string, args ...interface{}) (int, error) {
	// This method doesn't make sense for JSON store
	return 0, errors.New("DeleteWhere not supported in JSON store")
}

// UpdateByDimension updates documents matching dimension filters
func (s *jsonFileStore) UpdateByDimension(filters map[string]interface{}, updates UpdateRequest) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// TODO: Implement bulk update by dimensions
	return 0, errors.New("not implemented")
}

// UpdateWhere updates documents matching a custom WHERE clause
func (s *jsonFileStore) UpdateWhere(whereClause string, updates UpdateRequest, args ...interface{}) (int, error) {
	// This method doesn't make sense for JSON store
	return 0, errors.New("UpdateWhere not supported in JSON store")
}

// Close releases any resources
func (s *jsonFileStore) Close() error {
	// Save any pending changes
	return s.save()
}

// generateCanonicalView generates the canonical view for ID mapping
func (s *jsonFileStore) generateCanonicalView() ([]Document, error) {
	// TODO: Implement canonical view generation
	return nil, errors.New("not implemented")
}

// mapIDToUUID creates a mapping from user-facing IDs to UUIDs
func (s *jsonFileStore) mapIDToUUID() (map[string]string, error) {
	// TODO: Implement ID mapping based on canonical view
	return nil, errors.New("not implemented")
}

// contains checks if a slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
