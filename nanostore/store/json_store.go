package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/internal/validation"
	"github.com/arthur-debert/nanostore/nanostore/ids"
	"github.com/arthur-debert/nanostore/nanostore/query"
	"github.com/arthur-debert/nanostore/nanostore/storage"
	"github.com/arthur-debert/nanostore/types"
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
	queryProc     query.Processor
	lockManager   *storage.LockManager
	// File system abstractions
	fs          FileSystem
	lockFactory FileLockFactory
	fileLock    FileLock // Cross-process file locking

	data *storage.StoreData
	// timeFunc is used to get the current time, defaults to time.Now
	// Can be overridden for testing
	timeFunc func() time.Time
}

// newJSONFileStore creates a new JSON file store
func newJSONFileStore(filePath string, config Config, opts ...JSONFileStoreOption) (*jsonFileStore, error) {

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

	idGen := ids.NewIDGenerator(config.GetDimensionSet(), canonicalView)

	store := &jsonFileStore{
		filePath:      filePath,
		config:        config,
		dimensionSet:  config.GetDimensionSet(),
		canonicalView: canonicalView,
		idGenerator:   idGen,
		queryProc:     query.NewProcessor(config.GetDimensionSet(), idGen),
		lockManager:   storage.NewLockManager(),
		timeFunc:      time.Now, // Default to time.Now
		data: &storage.StoreData{
			Documents: []types.Document{},
			Metadata: storage.Metadata{
				Version:   "1.0",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(store)
	}

	// Set defaults for dependencies not provided via options
	if store.fs == nil {
		store.fs = &OSFileSystem{}
	}
	if store.lockFactory == nil {
		store.lockFactory = &FlockFactory{}
	}

	// Create file lock using the factory
	lockPath := filePath + ".lock"
	store.fileLock = store.lockFactory.New(lockPath)

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
	_ = s.lockManager.Execute(storage.WriteOperation, func() error {
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
	if _, err := s.fs.Stat(s.filePath); errors.Is(err, os.ErrNotExist) {
		// File doesn't exist yet, that's OK
		return nil
	}

	// Read the file
	data, err := s.fs.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Empty file is OK
	if len(data) == 0 {
		return nil
	}

	// Parse JSON
	var storeData storage.StoreData
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
	if err := s.fs.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Rename temp file to actual file (atomic on most filesystems)
	if err := s.fs.Rename(tmpFile, s.filePath); err != nil {
		_ = s.fs.Remove(tmpFile) // Clean up temp file
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// List returns documents based on the provided options
func (s *jsonFileStore) List(opts types.ListOptions) ([]types.Document, error) {
	var result []types.Document
	err := s.lockManager.Execute(storage.ReadOperation, func() error {
		// Use query processor to execute the query
		docs, err := s.queryProc.Execute(s.data.Documents, opts)
		if err != nil {
			return err
		}
		result = docs
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
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

	result, err := s.lockManager.ExecuteWithResult(storage.WriteOperation, func() (interface{}, error) {

		// Generate UUID
		docUUID := uuid.New().String()

		// Create document
		now := s.timeFunc()
		doc := types.Document{
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
			if err := validation.ValidateSimpleType(value, name); err != nil {
				return "", err
			}
		}

		// Apply dimension values
		for _, dimConfig := range s.dimensionSet.All() {
			switch dimConfig.Type {
			case types.Enumerated:
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
			case types.Hierarchical:
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
func (s *jsonFileStore) Update(id string, updates types.UpdateRequest) error {
	// Preprocess command to resolve IDs
	cmd := &UpdateCommand{
		ID:      id,
		Request: updates,
	}
	if err := s.preprocessor.preprocessCommand(cmd); err != nil {
		return fmt.Errorf("preprocessing failed: %w", err)
	}

	return s.lockManager.Execute(storage.WriteOperation, func() error {
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
					if err := validation.ValidateSimpleType(value, name); err != nil {
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
				var dimConfig *types.DimensionConfig
				if found {
					dimConfig = &types.DimensionConfig{
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
							dimConfig = &types.DimensionConfig{
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
				if dimConfig.Type == types.Enumerated && value != nil {
					strVal := fmt.Sprintf("%v", value)
					if !contains(dimConfig.Values, strVal) {
						return fmt.Errorf("invalid value %q for dimension %q", strVal, dimName)
					}
					doc.Dimensions[dimName] = strVal
				} else if dimConfig.Type == types.Hierarchical {
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
	result, err := s.lockManager.ExecuteWithResult(storage.ReadOperation, func() (interface{}, error) {
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
	allDocs := make([]types.Document, len(s.data.Documents))
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

	return s.lockManager.Execute(storage.WriteOperation, func() error {
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
			hierDim := &types.DimensionConfig{
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
			hierDim := &types.DimensionConfig{
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
	result, err := s.lockManager.ExecuteWithResult(storage.WriteOperation, func() (interface{}, error) {
		// Find all documents matching the filters
		toDelete := []int{}
		for i, doc := range s.data.Documents {
			if s.queryProc.MatchesFilters(doc, filters) {
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
	if strings.TrimSpace(whereClause) == "" {
		return 0, errors.New("WHERE clause cannot be empty")
	}

	evaluator := NewWhereEvaluator(whereClause, args...)

	result, err := s.lockManager.ExecuteWithResult(storage.WriteOperation, func() (interface{}, error) {
		// Load current data
		err := s.load()
		if err != nil {
			return 0, fmt.Errorf("failed to load data: %w", err)
		}

		var matchingUUIDs []string

		// Find documents that match the WHERE clause
		for _, doc := range s.data.Documents {
			matches, err := evaluator.EvaluateDocument(&doc)
			if err != nil {
				return 0, fmt.Errorf("failed to evaluate WHERE clause for document %s: %w", doc.UUID, err)
			}
			if matches {
				matchingUUIDs = append(matchingUUIDs, doc.UUID)
			}
		}

		if len(matchingUUIDs) == 0 {
			return 0, nil // No matching documents
		}

		// Delete matching documents
		deletedCount := 0
		filteredDocs := make([]types.Document, 0, len(s.data.Documents)-len(matchingUUIDs))

		for _, doc := range s.data.Documents {
			found := false
			for _, uuid := range matchingUUIDs {
				if doc.UUID == uuid {
					found = true
					deletedCount++
					break
				}
			}
			if !found {
				filteredDocs = append(filteredDocs, doc)
			}
		}

		s.data.Documents = filteredDocs

		// Save the updated data
		err = s.save()
		if err != nil {
			return 0, fmt.Errorf("failed to save data after deletion: %w", err)
		}

		return deletedCount, nil
	})

	if err != nil {
		return 0, err
	}

	return result.(int), nil
}

// DeleteByUUIDs deletes multiple documents by their UUIDs in a single operation
func (s *jsonFileStore) DeleteByUUIDs(uuids []string) (int, error) {
	if len(uuids) == 0 {
		return 0, nil
	}

	result, err := s.lockManager.ExecuteWithResult(storage.WriteOperation, func() (interface{}, error) {
		// Create a map of UUIDs for faster lookup
		uuidMap := make(map[string]bool, len(uuids))
		for _, uuid := range uuids {
			uuidMap[uuid] = true
		}

		// Find all documents to delete (collect indices in reverse order)
		toDelete := []int{}
		for i, doc := range s.data.Documents {
			if uuidMap[doc.UUID] {
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

// UpdateByDimension updates documents matching dimension filters
func (s *jsonFileStore) UpdateByDimension(filters map[string]interface{}, updates types.UpdateRequest) (int, error) {
	result, err := s.lockManager.ExecuteWithResult(storage.WriteOperation, func() (interface{}, error) {

		// Validate update dimensions if provided
		if updates.Dimensions != nil {
			for name, value := range updates.Dimensions {
				// Skip validation for _data fields - they can be any type
				if strings.HasPrefix(name, "_data.") {
					continue
				}
				if value != nil {
					if err := validation.ValidateSimpleType(value, name); err != nil {
						return 0, err
					}
				}
			}
		}

		// Find and update all matching documents
		updatedCount := 0
		now := s.timeFunc()

		for i := range s.data.Documents {
			if s.queryProc.MatchesFilters(s.data.Documents[i], filters) {
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
						var dimConfig *types.DimensionConfig
						if found {
							dimConfig = &types.DimensionConfig{
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
									dimConfig = &types.DimensionConfig{
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
						if dimConfig.Type == types.Enumerated && value != nil {
							strVal := fmt.Sprintf("%v", value)
							if !contains(dimConfig.Values, strVal) {
								return 0, fmt.Errorf("invalid value %q for dimension %q", strVal, dimName)
							}
							doc.Dimensions[dimName] = strVal
						} else if dimConfig.Type == types.Hierarchical {
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
func (s *jsonFileStore) UpdateWhere(whereClause string, updates types.UpdateRequest, args ...interface{}) (int, error) {
	if strings.TrimSpace(whereClause) == "" {
		return 0, errors.New("WHERE clause cannot be empty")
	}

	evaluator := NewWhereEvaluator(whereClause, args...)

	result, err := s.lockManager.ExecuteWithResult(storage.WriteOperation, func() (interface{}, error) {
		// Load current data
		err := s.load()
		if err != nil {
			return 0, fmt.Errorf("failed to load data: %w", err)
		}

		// Validate update dimensions if provided
		if updates.Dimensions != nil {
			for name, value := range updates.Dimensions {
				// Skip validation for _data fields - they can be any type
				if strings.HasPrefix(name, "_data.") {
					continue
				}
				if value != nil {
					if err := validation.ValidateSimpleType(value, name); err != nil {
						return 0, err
					}
				}
			}
		}

		updatedCount := 0

		// Update documents that match the WHERE clause
		for i, doc := range s.data.Documents {
			matches, err := evaluator.EvaluateDocument(&doc)
			if err != nil {
				return 0, fmt.Errorf("failed to evaluate WHERE clause for document %s: %w", doc.UUID, err)
			}

			if matches {
				// Apply updates to this document
				if updates.Title != nil {
					s.data.Documents[i].Title = *updates.Title
				}
				if updates.Body != nil {
					s.data.Documents[i].Body = *updates.Body
				}
				if updates.Dimensions != nil {
					// Update dimensions
					for key, value := range updates.Dimensions {
						if s.data.Documents[i].Dimensions == nil {
							s.data.Documents[i].Dimensions = make(map[string]interface{})
						}
						s.data.Documents[i].Dimensions[key] = value
					}
				}
				// Update timestamp
				s.data.Documents[i].UpdatedAt = s.timeFunc()
				updatedCount++
			}
		}

		if updatedCount > 0 {
			// Save the updated data
			err = s.save()
			if err != nil {
				return 0, fmt.Errorf("failed to save data after update: %w", err)
			}
		}

		return updatedCount, nil
	})

	if err != nil {
		return 0, err
	}

	return result.(int), nil
}

// UpdateByUUIDs updates multiple documents by their UUIDs in a single operation
func (s *jsonFileStore) UpdateByUUIDs(uuids []string, updates types.UpdateRequest) (int, error) {
	if len(uuids) == 0 {
		return 0, nil
	}

	result, err := s.lockManager.ExecuteWithResult(storage.WriteOperation, func() (interface{}, error) {
		// Validate update dimensions if provided
		if updates.Dimensions != nil {
			for name, value := range updates.Dimensions {
				// Skip validation for _data fields - they can be any type
				if strings.HasPrefix(name, "_data.") {
					continue
				}
				if value != nil {
					if err := validation.ValidateSimpleType(value, name); err != nil {
						return 0, err
					}
				}
			}
		}

		// Create a map of UUIDs for faster lookup
		uuidMap := make(map[string]bool, len(uuids))
		for _, uuid := range uuids {
			uuidMap[uuid] = true
		}

		// Find and update all matching documents
		updatedCount := 0
		now := s.timeFunc()

		for i := range s.data.Documents {
			doc := &s.data.Documents[i]
			if uuidMap[doc.UUID] {
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
						var dimConfig *types.DimensionConfig
						if found {
							dimConfig = &types.DimensionConfig{
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
									dimConfig = &types.DimensionConfig{
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

						// Update the dimension value
						if dimConfig != nil && dimConfig.Type == types.Enumerated {
							// Store enumerated dimension value
							if value != nil {
								doc.Dimensions[dimName] = value
							} else {
								delete(doc.Dimensions, dimName)
							}
						} else if dimConfig != nil && dimConfig.Type == types.Hierarchical {
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

// GetByID retrieves a single document by ID
func (s *jsonFileStore) GetByID(id string) (*types.Document, error) {
	var result *types.Document
	err := s.lockManager.Execute(storage.ReadOperation, func() error {
		// Find the document
		for _, doc := range s.data.Documents {
			if doc.UUID == id {
				result = &doc
				return nil
			}
		}
		return nil // Not found
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// Close releases any resources
func (s *jsonFileStore) Close() error {
	return s.lockManager.Execute(storage.WriteOperation, func() error {
		// Don't need to save - data is saved on each operation
		// Just ensure the lock file is cleaned up
		lockPath := s.filePath + ".lock"
		_ = s.fs.Remove(lockPath)

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
