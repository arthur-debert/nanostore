package nanostore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/nanostore/ids"
	"github.com/arthur-debert/nanostore/types"
	"github.com/gofrs/flock"
	"github.com/google/uuid"
)

// jsonFileStore implements the Store and TestStore interfaces using a JSON file backend
type jsonFileStore struct {
	filePath      string
	config        Config
	dimensionSet  *types.DimensionSet
	canonicalView *types.CanonicalView
	idGenerator   *ids.IDGenerator
	preprocessor  *commandPreprocessor
	lockManager   *lockManager
	fileLock      *flock.Flock // Cross-process file locking
	data          *storeData
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
	// Create a file lock for the data file
	// Use a separate lock file to avoid issues with file replacement during save
	lockPath := filePath + ".lock"
	fileLock := flock.New(lockPath)

	// Create canonical view from config
	// Default canonical view based on dimension defaults
	var filters []types.CanonicalFilter
	for _, dim := range config.GetDimensionSet().Enumerated() {
		if dim.DefaultValue != "" {
			filters = append(filters, types.CanonicalFilter{
				Dimension: dim.Name,
				Value:     dim.DefaultValue,
			})
		}
	}
	// Hierarchical dimensions default to "*" (any value)
	for _, dim := range config.GetDimensionSet().Hierarchical() {
		filters = append(filters, types.CanonicalFilter{
			Dimension: dim.Name,
			Value:     "*",
		})
	}
	canonicalView := types.NewCanonicalView(filters...)

	store := &jsonFileStore{
		filePath:      filePath,
		config:        config,
		dimensionSet:  config.GetDimensionSet(),
		canonicalView: canonicalView,
		idGenerator:   ids.NewIDGenerator(config.GetDimensionSet(), canonicalView),
		lockManager:   newLockManager(),
		fileLock:      fileLock,
		timeFunc:      time.Now, // Default to time.Now
		data: &storeData{
			Documents: []Document{},
			Metadata: storeMetadata{
				Version:   "1.0",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Initialize preprocessor
	store.preprocessor = newCommandPreprocessor(store)

	// Try to load existing data with lock
	if err := store.loadWithLock(); err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	return store, nil
}

// SetTimeFunc sets a custom time function for testing
// This allows tests to provide deterministic timestamps
func (s *jsonFileStore) SetTimeFunc(fn func() time.Time) {
	_ = s.lockManager.execute(writeOperation, func() error {
		s.timeFunc = fn
		return nil
	})
}

// Constants for file locking
const (
	lockTimeout    = 3 * time.Second
	lockMaxRetries = 3
	lockRetryDelay = 100 * time.Millisecond
)

// acquireLock attempts to acquire an exclusive file lock with retry logic
func (s *jsonFileStore) acquireLock(ctx context.Context) error {
	for i := 0; i < lockMaxRetries; i++ {
		locked, err := s.fileLock.TryLockContext(ctx, lockRetryDelay)
		if err != nil {
			return fmt.Errorf("failed to acquire lock: %w", err)
		}
		if locked {
			return nil
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(lockRetryDelay):
			// Continue to next retry
		}
	}

	return fmt.Errorf("failed to acquire lock after %d attempts", lockMaxRetries)
}

// releaseLock releases the file lock
func (s *jsonFileStore) releaseLock() error {
	return s.fileLock.Unlock()
}

// loadWithLock loads the data file with proper locking
func (s *jsonFileStore) loadWithLock() error {
	ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()

	// Acquire file lock
	if err := s.acquireLock(ctx); err != nil {
		return err
	}
	defer func() { _ = s.releaseLock() }()

	// Load data while holding the lock
	return s.load()
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

// saveWithLock saves the data with proper locking
func (s *jsonFileStore) saveWithLock() error {
	ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()

	// Acquire file lock
	if err := s.acquireLock(ctx); err != nil {
		return err
	}
	defer func() { _ = s.releaseLock() }()

	// Save data while holding the lock
	return s.save()
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
		_ = os.Remove(tmpFile) // Clean up temp file
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// List returns documents based on the provided options
func (s *jsonFileStore) List(opts ListOptions) ([]Document, error) {
	var result []Document
	err := s.lockManager.execute(readOperation, func() error {
		// Start with all documents
		result = make([]Document, 0, len(s.data.Documents))

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

		// Apply ordering
		if len(opts.OrderBy) > 0 {
			s.sortDocuments(result, opts.OrderBy)
		}

		// Generate SimpleIDs using the new ID generator
		// Get all documents for ID generation (not just the filtered ones)
		// GenerateIDs now handles copying internally, so we can pass the slice directly
		idMap := s.idGenerator.GenerateIDs(s.data.Documents)

		// Create reverse mapping (SimpleID -> UUID)
		uuidToID := make(map[string]string)
		for simpleID, uuid := range idMap {
			uuidToID[uuid] = simpleID
		}

		// Assign SimpleIDs to results
		for i := range result {
			if simpleID, exists := uuidToID[result[i].UUID]; exists {
				result[i].SimpleID = simpleID
			} else {
				// Fallback to UUID if not found (shouldn't happen)
				result[i].SimpleID = result[i].UUID
			}
		}

		// Apply pagination
		if opts.Offset != nil && *opts.Offset > 0 {
			if *opts.Offset >= len(result) {
				result = []Document{}
			} else {
				result = result[*opts.Offset:]
			}
		}

		if opts.Limit != nil && *opts.Limit > 0 {
			if *opts.Limit < len(result) {
				result = result[:*opts.Limit]
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
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
				// Try with _data prefix for non-dimension fields
				docValue, exists = doc.Dimensions["_data."+filterKey]
				if !exists {
					// Document doesn't have this dimension or data field
					// Check if it's a hierarchical dimension ref field
					found := false
					for _, dim := range s.dimensionSet.Hierarchical() {
						if dim.RefField == filterKey {
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
	// Preprocess command to resolve IDs in dimensions
	cmd := &AddCommand{
		Title:      title,
		Dimensions: dimensions,
	}
	if err := s.preprocessor.preprocessCommand(cmd); err != nil {
		return "", fmt.Errorf("preprocessing failed: %w", err)
	}

	result, err := s.lockManager.executeWithResult(writeOperation, func() (interface{}, error) {

		// Generate UUID
		docUUID := uuid.New().String()

		// Create document
		now := s.timeFunc()
		doc := Document{
			UUID:       docUUID,
			Title:      cmd.Title,
			Body:       "", // Empty body by default
			CreatedAt:  now,
			UpdatedAt:  now,
			Dimensions: make(map[string]interface{}),
		}

		// Validate all provided dimensions are simple types
		for name, value := range cmd.Dimensions {
			// Skip validation for _data fields - they can be any type
			if strings.HasPrefix(name, "_data.") {
				continue
			}
			if err := ValidateSimpleType(value, name); err != nil {
				return "", err
			}
		}

		// Apply dimension values
		for _, dimConfig := range s.dimensionSet.All() {
			switch dimConfig.Type {
			case Enumerated:
				// Check if value was provided
				if val, exists := cmd.Dimensions[dimConfig.Name]; exists {
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
			case Hierarchical:
				// Handle parent reference
				// ID resolution already handled by preprocessor
				if val, exists := cmd.Dimensions[dimConfig.RefField]; exists {
					doc.Dimensions[dimConfig.RefField] = fmt.Sprintf("%v", val)
				}
			}
		}

		// Also store any _data prefixed values directly
		for key, value := range cmd.Dimensions {
			if strings.HasPrefix(key, "_data.") {
				doc.Dimensions[key] = value
			}
		}

		// Add to store
		s.data.Documents = append(s.data.Documents, doc)

		// Save to file
		if err := s.saveWithLock(); err != nil {
			// Remove the document on save failure
			s.data.Documents = s.data.Documents[:len(s.data.Documents)-1]
			return "", fmt.Errorf("failed to save: %w", err)
		}

		return docUUID, nil
	})

	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// Update modifies an existing document
func (s *jsonFileStore) Update(id string, updates UpdateRequest) error {
	// Preprocess command to resolve IDs
	cmd := &UpdateCommand{
		ID:      id,
		Request: updates,
	}
	if err := s.preprocessor.preprocessCommand(cmd); err != nil {
		return fmt.Errorf("preprocessing failed: %w", err)
	}

	return s.lockManager.execute(writeOperation, func() error {
		// Find the document by UUID
		var found bool
		var docIndex int
		for i, doc := range s.data.Documents {
			if doc.UUID == cmd.ID {
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
		if cmd.Request.Title != nil {
			doc.Title = *cmd.Request.Title
		}

		// Update body if provided
		if cmd.Request.Body != nil {
			doc.Body = *cmd.Request.Body
		}

		// Update dimensions if provided
		if cmd.Request.Dimensions != nil {
			// Validate all dimensions are simple types
			for name, value := range cmd.Request.Dimensions {
				// Skip validation for _data fields - they can be any type
				if strings.HasPrefix(name, "_data.") {
					continue
				}
				if value != nil {
					if err := ValidateSimpleType(value, name); err != nil {
						return err
					}
				}
			}

			// First handle _data prefixed values (no validation needed)
			for dimName, value := range cmd.Request.Dimensions {
				if strings.HasPrefix(dimName, "_data.") {
					if value != nil {
						doc.Dimensions[dimName] = value
					} else {
						delete(doc.Dimensions, dimName)
					}
				}
			}

			// Then validate and process dimension updates
			for dimName, value := range cmd.Request.Dimensions {
				// Skip _data prefixed fields (already handled)
				if strings.HasPrefix(dimName, "_data.") {
					continue
				}

				// Find dimension config
				dim, found := s.dimensionSet.Get(dimName)
				var dimConfig *DimensionConfig
				if found {
					dimConfig = &DimensionConfig{
						Name:         dim.Name,
						Type:         dim.Type,
						Values:       dim.Values,
						Prefixes:     dim.Prefixes,
						DefaultValue: dim.DefaultValue,
						RefField:     dim.RefField,
					}
				} else {
					// Try by RefField for hierarchical dimensions
					for _, dc := range s.dimensionSet.Hierarchical() {
						if dc.RefField == dimName {
							dimConfig = &DimensionConfig{
								Name:         dc.Name,
								Type:         dc.Type,
								Values:       dc.Values,
								Prefixes:     dc.Prefixes,
								DefaultValue: dc.DefaultValue,
								RefField:     dc.RefField,
							}
							break
						}
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
					// ID resolution already handled by preprocessor
					if value != nil {
						doc.Dimensions[dimConfig.RefField] = fmt.Sprintf("%v", value)
					} else {
						delete(doc.Dimensions, dimConfig.RefField)
					}
				}
			}
		}

		// Save to file
		if err := s.saveWithLock(); err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}

		return nil
	})
}

// ResolveUUID converts a simple ID to a UUID
func (s *jsonFileStore) ResolveUUID(simpleID string) (string, error) {
	result, err := s.lockManager.executeWithResult(readOperation, func() (interface{}, error) {
		return s.resolveUUIDInternal(simpleID)
	})
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// resolveUUIDInternal is the internal version that doesn't take locks
func (s *jsonFileStore) resolveUUIDInternal(simpleID string) (string, error) {
	// Get all documents
	allDocs := make([]Document, len(s.data.Documents))
	copy(allDocs, s.data.Documents)

	// Use the ID generator to resolve the ID
	// Convert to types.Document
	return s.idGenerator.ResolveID(simpleID, allDocs)
}

// Delete removes a document
func (s *jsonFileStore) Delete(id string, cascade bool) error {
	// Preprocess command to resolve IDs
	cmd := &DeleteCommand{
		ID:      id,
		Cascade: cascade,
	}
	if err := s.preprocessor.preprocessCommand(cmd); err != nil {
		return fmt.Errorf("preprocessing failed: %w", err)
	}

	return s.lockManager.execute(writeOperation, func() error {
		return s.deleteInternal(cmd.ID, cmd.Cascade)
	})
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
		hierDims := s.dimensionSet.Hierarchical()
		if len(hierDims) > 0 {
			hierDim := &DimensionConfig{
				Name:         hierDims[0].Name,
				Type:         hierDims[0].Type,
				Values:       hierDims[0].Values,
				Prefixes:     hierDims[0].Prefixes,
				DefaultValue: hierDims[0].DefaultValue,
				RefField:     hierDims[0].RefField,
			}
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
		hierDims := s.dimensionSet.Hierarchical()
		if len(hierDims) > 0 {
			hierDim := &DimensionConfig{
				Name:         hierDims[0].Name,
				Type:         hierDims[0].Type,
				Values:       hierDims[0].Values,
				Prefixes:     hierDims[0].Prefixes,
				DefaultValue: hierDims[0].DefaultValue,
				RefField:     hierDims[0].RefField,
			}
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
	if err := s.saveWithLock(); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}

	return nil
}

// DeleteByDimension removes documents matching dimension filters
func (s *jsonFileStore) DeleteByDimension(filters map[string]interface{}) (int, error) {
	result, err := s.lockManager.executeWithResult(writeOperation, func() (interface{}, error) {
		// Find all documents matching the filters
		toDelete := []int{}
		for i, doc := range s.data.Documents {
			if s.matchesFilters(doc, filters) {
				toDelete = append(toDelete, i)
			}
		}

		// Delete in reverse order to preserve indices
		deletedCount := 0
		for i := len(toDelete) - 1; i >= 0; i-- {
			idx := toDelete[i]
			s.data.Documents = append(s.data.Documents[:idx], s.data.Documents[idx+1:]...)
			deletedCount++
		}

		// Save changes if any documents were deleted
		if deletedCount > 0 {
			if err := s.saveWithLock(); err != nil {
				return 0, fmt.Errorf("failed to save after deletion: %w", err)
			}
		}

		return deletedCount, nil
	})

	if err != nil {
		return 0, err
	}
	return result.(int), nil
}

// DeleteWhere removes documents matching a custom WHERE clause
func (s *jsonFileStore) DeleteWhere(whereClause string, args ...interface{}) (int, error) {
	// This method doesn't make sense for JSON store
	return 0, errors.New("DeleteWhere not supported in JSON store")
}

// UpdateByDimension updates documents matching dimension filters
func (s *jsonFileStore) UpdateByDimension(filters map[string]interface{}, updates UpdateRequest) (int, error) {
	result, err := s.lockManager.executeWithResult(writeOperation, func() (interface{}, error) {

		// Validate update dimensions if provided
		if updates.Dimensions != nil {
			for name, value := range updates.Dimensions {
				// Skip validation for _data fields - they can be any type
				if strings.HasPrefix(name, "_data.") {
					continue
				}
				if value != nil {
					if err := ValidateSimpleType(value, name); err != nil {
						return 0, err
					}
				}
			}
		}

		// Find and update all matching documents
		updatedCount := 0
		now := s.timeFunc()

		for i := range s.data.Documents {
			if s.matchesFilters(s.data.Documents[i], filters) {
				doc := &s.data.Documents[i]
				doc.UpdatedAt = now

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
					// First handle _data prefixed values (no validation needed)
					for dimName, value := range updates.Dimensions {
						if strings.HasPrefix(dimName, "_data.") {
							if value != nil {
								doc.Dimensions[dimName] = value
							} else {
								delete(doc.Dimensions, dimName)
							}
						}
					}

					// Then validate and process dimension updates
					for dimName, value := range updates.Dimensions {
						// Skip _data prefixed fields (already handled)
						if strings.HasPrefix(dimName, "_data.") {
							continue
						}

						// Find dimension config
						dim, found := s.dimensionSet.Get(dimName)
						var dimConfig *DimensionConfig
						if found {
							dimConfig = &DimensionConfig{
								Name:         dim.Name,
								Type:         dim.Type,
								Values:       dim.Values,
								Prefixes:     dim.Prefixes,
								DefaultValue: dim.DefaultValue,
								RefField:     dim.RefField,
							}
						} else {
							// Try by RefField for hierarchical dimensions
							for _, dc := range s.dimensionSet.Hierarchical() {
								if dc.RefField == dimName {
									dimConfig = &DimensionConfig{
										Name:         dc.Name,
										Type:         dc.Type,
										Values:       dc.Values,
										Prefixes:     dc.Prefixes,
										DefaultValue: dc.DefaultValue,
										RefField:     dc.RefField,
									}
									break
								}
							}
						}

						if dimConfig == nil {
							return 0, fmt.Errorf("unknown dimension: %s", dimName)
						}

						// Validate enumerated dimension values
						if dimConfig.Type == Enumerated && value != nil {
							strVal := fmt.Sprintf("%v", value)
							if !contains(dimConfig.Values, strVal) {
								return 0, fmt.Errorf("invalid value %q for dimension %q", strVal, dimName)
							}
							doc.Dimensions[dimName] = strVal
						} else if dimConfig.Type == Hierarchical {
							// Store hierarchical dimension value
							if value != nil {
								parentID := fmt.Sprintf("%v", value)
								// Try to resolve if it's a SimpleID
								if !ids.IsValidUUID(parentID) {
									if resolvedUUID, err := s.resolveUUIDInternal(parentID); err == nil {
										parentID = resolvedUUID
									}
									// If resolution fails, store the value as-is
								}
								doc.Dimensions[dimConfig.RefField] = parentID
							} else {
								delete(doc.Dimensions, dimConfig.RefField)
							}
						}
					}
				}

				updatedCount++
			}
		}

		// Save changes if any documents were updated
		if updatedCount > 0 {
			if err := s.saveWithLock(); err != nil {
				return 0, fmt.Errorf("failed to save after update: %w", err)
			}
		}

		return updatedCount, nil
	})

	if err != nil {
		return 0, err
	}
	return result.(int), nil
}

// UpdateWhere updates documents matching a custom WHERE clause
func (s *jsonFileStore) UpdateWhere(whereClause string, updates UpdateRequest, args ...interface{}) (int, error) {
	// This method doesn't make sense for JSON store
	return 0, errors.New("UpdateWhere not supported in JSON store")
}

// Close releases any resources
func (s *jsonFileStore) Close() error {
	return s.lockManager.execute(writeOperation, func() error {
		// Don't need to save - data is saved on each operation
		// Just ensure the lock file is cleaned up
		lockPath := s.filePath + ".lock"
		_ = os.Remove(lockPath)

		return nil
	})
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

// sortDocuments sorts documents according to the order clauses
func (s *jsonFileStore) sortDocuments(docs []Document, orderBy []OrderClause) {
	sort.Slice(docs, func(i, j int) bool {
		for _, clause := range orderBy {
			// Get values for comparison
			valI := s.getDocumentValue(docs[i], clause.Column)
			valJ := s.getDocumentValue(docs[j], clause.Column)

			// Convert to comparable strings
			strI := s.valueToString(valI)
			strJ := s.valueToString(valJ)

			// Compare
			if strI < strJ {
				return !clause.Descending
			} else if strI > strJ {
				return clause.Descending
			}
			// If equal, continue to next order clause
		}
		return false // All equal
	})
}

// getDocumentValue retrieves a value from a document by field name
func (s *jsonFileStore) getDocumentValue(doc Document, column string) interface{} {
	switch column {
	case "uuid":
		return doc.UUID
	case "simple_id", "simpleid":
		return doc.SimpleID
	case "title":
		return doc.Title
	case "body":
		return doc.Body
	case "created_at":
		return doc.CreatedAt
	case "updated_at":
		return doc.UpdatedAt
	default:
		// Check if it's a dimension
		if val, exists := doc.Dimensions[column]; exists {
			return val
		}
		// Try with _data prefix for non-dimension fields (transparent ordering support)
		if val, exists := doc.Dimensions["_data."+column]; exists {
			return val
		}
		// Return empty string for non-existent fields
		return ""
	}
}
