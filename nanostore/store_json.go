package nanostore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// jsonFileStore implements the Store and TestStore interfaces using a JSON file backend
type jsonFileStore struct {
	filePath string
	config   Config
	mu       sync.RWMutex
	data     *storeData
	// timeFunc is used to get the current time, defaults to time.Now
	// Can be overridden for testing
	timeFunc func() time.Time
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
		timeFunc: time.Now, // Default to time.Now
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

// SetTimeFunc sets a custom time function for testing
// This allows tests to provide deterministic timestamps
func (s *jsonFileStore) SetTimeFunc(fn func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timeFunc = fn
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
	s.data.Metadata.UpdatedAt = s.timeFunc()

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

	// Start with all documents
	result := make([]Document, 0, len(s.data.Documents))
	
	// Apply filters
	for _, doc := range s.data.Documents {
		// Check dimension filters
		if !s.matchesFilters(doc, opts.Filters) {
			continue
		}

		// Check text search filter
		if opts.FilterBySearch != "" && !s.matchesSearch(doc, opts.FilterBySearch) {
			continue
		}

		// Make a copy to avoid mutations
		docCopy := doc
		// Set SimpleID to UUID for now (will be replaced with proper ID generation)
		docCopy.SimpleID = doc.UUID
		result = append(result, docCopy)
	}

	// TODO: Apply ordering
	// TODO: Apply pagination (limit/offset)

	return result, nil
}

// matchesFilters checks if a document matches all the provided filters
func (s *jsonFileStore) matchesFilters(doc Document, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true // No filters means match all
	}

	for filterKey, filterValue := range filters {
		// Handle special filter for UUID
		if filterKey == "uuid" {
			if doc.UUID != fmt.Sprintf("%v", filterValue) {
				return false
			}
			continue
		}

		// Handle datetime filters and dimension filters
		var docValue interface{}
		var exists bool
		
		switch filterKey {
		case "created_at":
			docValue = doc.CreatedAt
			exists = true
		case "updated_at":
			docValue = doc.UpdatedAt
			exists = true
		default:
			// Check if it's a dimension filter
			docValue, exists = doc.Dimensions[filterKey]
			if !exists {
				// Document doesn't have this dimension
				// Check if it's a hierarchical dimension ref field
				found := false
				for _, dim := range s.config.Dimensions {
					if dim.Type == Hierarchical && dim.RefField == filterKey {
						// It's a hierarchical ref field
						if parentValue, ok := doc.Dimensions[dim.RefField]; ok {
							docValue = parentValue
							exists = true
							found = true
							break
						}
					}
				}
				if !found {
					return false
				}
			}
		}

		// Convert values to comparable strings
		docStr := s.valueToString(docValue)
		
		// Handle slice values (for "IN" style filtering)
		switch fv := filterValue.(type) {
		case []string:
			// Filter value is a slice, check if document value is in the slice
			found := false
			for _, v := range fv {
				if docStr == v {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		case []interface{}:
			// Filter value is a slice, check if document value is in the slice
			found := false
			for _, v := range fv {
				if docStr == s.valueToString(v) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		default:
			// Simple equality check
			filterStr := s.valueToString(filterValue)
			if docStr != filterStr {
				return false
			}
		}
	}

	return true
}

// matchesSearch checks if a document matches the search text
func (s *jsonFileStore) matchesSearch(doc Document, searchText string) bool {
	// Simple case-insensitive substring search in title and body
	searchLower := strings.ToLower(searchText)
	
	if strings.Contains(strings.ToLower(doc.Title), searchLower) {
		return true
	}
	
	if strings.Contains(strings.ToLower(doc.Body), searchLower) {
		return true
	}
	
	return false
}

// Add creates a new document
func (s *jsonFileStore) Add(title string, dimensions map[string]interface{}) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate UUID
	docUUID := uuid.New().String()

	// Create document
	now := s.timeFunc()
	doc := Document{
		UUID:       docUUID,
		Title:      title,
		Body:       "", // Empty body by default
		CreatedAt:  now,
		UpdatedAt:  now,
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
	doc.UpdatedAt = s.timeFunc()

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

// ResolveUUID converts a simple ID to a UUID
func (s *jsonFileStore) ResolveUUID(simpleID string) (string, error) {
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

// mapIDToUUID creates a mapping from simple IDs to UUIDs
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

// valueToString converts any value to a string for comparison
// Special handling for time.Time values to use RFC3339Nano format
func (s *jsonFileStore) valueToString(value interface{}) string {
	switch v := value.(type) {
	case time.Time:
		// Use RFC3339Nano for consistent datetime comparison with nanosecond precision
		return v.Format(time.RFC3339Nano)
	case string:
		// Check if it's a datetime string and normalize it
		// Try various datetime formats
		for _, format := range []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02",
		} {
			if t, err := time.Parse(format, v); err == nil {
				return t.Format(time.RFC3339Nano)
			}
		}
		// Not a datetime, return as-is
		return v
	default:
		return fmt.Sprintf("%v", value)
	}
}
