package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/internal/validation"
	"github.com/arthur-debert/nanostore/nanostore/ids"
	"github.com/arthur-debert/nanostore/nanostore/query"
	"github.com/arthur-debert/nanostore/nanostore/storage"
	"github.com/arthur-debert/nanostore/types"
	"github.com/google/uuid"
)

// hybridJSONFileStore implements the Store interface with hybrid body storage
type hybridJSONFileStore struct {
	filePath      string
	config        Config
	dimensionSet  *types.DimensionSet
	canonicalView *types.CanonicalView
	idGenerator   *ids.IDGenerator
	preprocessor  *hybridCommandPreprocessor
	queryProc     query.Processor
	lockManager   *storage.LockManager

	// File system abstractions
	fs          FileSystemExt
	lockFactory FileLockFactory
	fileLock    FileLock

	// Body storage handler
	bodyStorage BodyStorage

	// Hybrid data storage
	hybridData *HybridStoreData

	// timeFunc is used to get the current time
	timeFunc func() time.Time
}

// newHybridJSONFileStore creates a new hybrid JSON file store
func newHybridJSONFileStore(filePath string, config Config, opts ...HybridJSONFileStoreOption) (*hybridJSONFileStore, error) {
	// Create canonical view from config
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

	store := &hybridJSONFileStore{
		filePath:      filePath,
		config:        config,
		dimensionSet:  config.GetDimensionSet(),
		canonicalView: canonicalView,
		idGenerator:   idGen,
		queryProc:     query.NewProcessor(config.GetDimensionSet(), idGen),
		lockManager:   storage.NewLockManager(),
		timeFunc:      time.Now, // Default to time.Now
		hybridData: &HybridStoreData{
			Documents: []HybridDocument{},
			Metadata: HybridMetadata{
				Version:        "1.0",
				StorageVersion: "hybrid_v1",
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
		},
	}

	// Apply options (includes file system setup)
	for _, opt := range opts {
		opt(store)
	}

	// Set defaults for dependencies not provided via options
	if store.fs == nil {
		store.fs = &OSFileSystemExt{}
	}
	if store.lockFactory == nil {
		store.lockFactory = &FlockFactory{}
	}

	// Create file lock using the factory
	lockPath := filePath + ".lock"
	store.fileLock = store.lockFactory.New(lockPath)

	// Initialize body storage with the configured embed limit
	basePath := filepath.Dir(filePath)
	embedLimit := store.hybridData.Metadata.BodyStorageConfig.EmbedSizeLimit
	if embedLimit == 0 {
		embedLimit = int64(1024) // 1KB default
		store.hybridData.Metadata.BodyStorageConfig.EmbedSizeLimit = embedLimit
	}
	store.bodyStorage = NewHybridBodyStorage(store.fs, basePath, embedLimit)

	// Update metadata for bodies directory if not set
	if store.hybridData.Metadata.BodyStorageConfig.BodiesDir == "" {
		store.hybridData.Metadata.BodyStorageConfig.BodiesDir = "bodies"
	}

	// Initialize preprocessor
	store.preprocessor = newHybridCommandPreprocessor(store)

	// Try to load existing data with lock
	if err := store.loadWithLock(); err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	return store, nil
}

// loadWithLock loads the data file with proper locking
func (s *hybridJSONFileStore) loadWithLock() error {
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

// load reads the JSON file into memory and validates body files
func (s *hybridJSONFileStore) load() error {
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

	// Try to parse as hybrid format first
	var hybridData HybridStoreData
	if err := json.Unmarshal(data, &hybridData); err == nil && hybridData.Metadata.StorageVersion == "hybrid_v1" {
		// It's a hybrid format file
		s.hybridData = &hybridData

		// Validate body files
		for i, doc := range s.hybridData.Documents {
			if doc.BodyMeta != nil {
				if err := s.bodyStorage.ValidateBody(*doc.BodyMeta); err != nil {
					// Log warning but don't fail load
					// In production, you might want to handle this differently
					fmt.Printf("Warning: Document %s has invalid body storage: %v\n", doc.UUID, err)
					// Clear the body metadata to force re-save
					s.hybridData.Documents[i].BodyMeta = nil
				}
			}
		}

		return nil
	}

	// Try to parse as legacy format
	var legacyData storage.StoreData
	if err := json.Unmarshal(data, &legacyData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert legacy format to hybrid format
	s.hybridData = s.convertLegacyToHybrid(&legacyData)

	// Mark for save to persist in new format
	// This will happen on the next write operation

	return nil
}

// convertLegacyToHybrid converts legacy storage format to hybrid format
func (s *hybridJSONFileStore) convertLegacyToHybrid(legacy *storage.StoreData) *HybridStoreData {
	hybrid := &HybridStoreData{
		Documents: make([]HybridDocument, len(legacy.Documents)),
		Metadata: HybridMetadata{
			Version:        legacy.Metadata.Version,
			StorageVersion: "hybrid_v1",
			CreatedAt:      legacy.Metadata.CreatedAt,
			UpdatedAt:      legacy.Metadata.UpdatedAt,
		},
	}

	// Set body storage config
	hybrid.Metadata.BodyStorageConfig.EmbedSizeLimit = 1024
	hybrid.Metadata.BodyStorageConfig.BodiesDir = "bodies"

	// Convert documents
	for i, doc := range legacy.Documents {
		// Determine body storage based on size
		bodySize := int64(len(doc.Body))
		var bodyMeta *BodyMetadata
		var embeddedBody string

		if bodySize > 0 {
			if bodySize <= hybrid.Metadata.BodyStorageConfig.EmbedSizeLimit {
				// Small enough to embed
				bodyMeta = &BodyMetadata{
					Type:   BodyStorageEmbedded,
					Format: BodyFormatText, // Assume text for legacy
					Size:   bodySize,
				}
				embeddedBody = doc.Body
			} else {
				// Too large, will need to store in file on next save
				bodyMeta = &BodyMetadata{
					Type:   BodyStorageEmbedded, // Keep embedded for now
					Format: BodyFormatText,
					Size:   bodySize,
				}
				embeddedBody = doc.Body
			}
		}

		hybrid.Documents[i] = *FromStandardDocument(doc, bodyMeta, embeddedBody)
	}

	return hybrid
}

// saveWithLock saves the data with proper locking
func (s *hybridJSONFileStore) saveWithLock() error {
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
func (s *hybridJSONFileStore) save() error {
	// Update metadata
	s.hybridData.Metadata.UpdatedAt = s.timeFunc()

	// Marshal to JSON with pretty printing
	data, err := json.MarshalIndent(s.hybridData, "", "  ")
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

// acquireLock acquires the file lock with retries
func (s *hybridJSONFileStore) acquireLock(ctx context.Context) error {
	locked, err := s.fileLock.TryLockContext(ctx, 50*time.Millisecond)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("could not acquire lock: timeout")
	}
	return nil
}

// releaseLock releases the file lock
func (s *hybridJSONFileStore) releaseLock() error {
	return s.fileLock.Unlock()
}

// List returns documents based on the provided options
func (s *hybridJSONFileStore) List(opts types.ListOptions) ([]types.Document, error) {
	var result []types.Document
	err := s.lockManager.Execute(storage.ReadOperation, func() error {
		// Convert hybrid documents to standard documents
		standardDocs := make([]types.Document, len(s.hybridData.Documents))
		for i, hdoc := range s.hybridData.Documents {
			// Load body content if needed
			if hdoc.BodyMeta != nil {
				body, err := s.bodyStorage.ReadBody(*hdoc.BodyMeta, hdoc.Body)
				if err != nil {
					// Log error but continue with empty body
					fmt.Printf("Warning: Failed to read body for document %s: %v\n", hdoc.UUID, err)
					body = ""
				}
				hdoc.Body = body
			}
			standardDocs[i] = hdoc.ToStandardDocument()
		}

		// Use query processor to execute the query
		docs, err := s.queryProc.Execute(standardDocs, opts)
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
func (s *hybridJSONFileStore) Add(title string, dimensions map[string]interface{}) (string, error) {
	// Extract body from dimensions if present
	body := ""
	if bodyVal, ok := dimensions["_body"]; ok {
		if bodyStr, ok := bodyVal.(string); ok {
			body = bodyStr
		}
		delete(dimensions, "_body") // Remove from dimensions
	}

	cmd := AddCommand{
		Title:      title,
		Body:       body,
		Dimensions: dimensions,
	}

	// Preprocess the command
	if err := s.preprocessor.preprocessCommand(&cmd); err != nil {
		return "", err
	}

	var newID string
	err := s.lockManager.Execute(storage.WriteOperation, func() error {
		// Create new document
		doc := HybridDocument{
			UUID:       uuid.New().String(),
			Title:      cmd.Title,
			Dimensions: make(map[string]interface{}),
			CreatedAt:  s.timeFunc(),
			UpdatedAt:  s.timeFunc(),
		}

		// Handle body if provided
		if cmd.Body != "" {
			format := BodyFormatText // Default format
			if formatStr, ok := cmd.Dimensions["_body.format"].(string); ok {
				if parsed, err := ParseBodyFormat(formatStr); err == nil {
					format = parsed
				}
				// Remove format from dimensions as it's metadata
				delete(cmd.Dimensions, "_body.format")
			}

			// Determine if we should force embed
			forceEmbed := false
			if force, ok := cmd.Dimensions["_body.embed"].(bool); ok {
				forceEmbed = force
				delete(cmd.Dimensions, "_body.embed")
			}

			// Write body
			bodyMeta, embeddedBody, err := s.bodyStorage.WriteBody(doc.UUID, cmd.Body, format, forceEmbed)
			if err != nil {
				return fmt.Errorf("failed to write body: %w", err)
			}
			doc.BodyMeta = &bodyMeta
			doc.Body = embeddedBody
		}

		// Validate dimensions
		for name, value := range cmd.Dimensions {
			// Skip validation for _data fields - they can be any type
			if strings.HasPrefix(name, "_data.") {
				continue
			}
			if err := validation.ValidateSimpleType(value, name); err != nil {
				return err
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
						return fmt.Errorf("invalid value %q for dimension %q", strVal, dimConfig.Name)
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
		s.hybridData.Documents = append(s.hybridData.Documents, doc)
		newID = doc.UUID

		// Save to file
		if err := s.saveWithLock(); err != nil {
			// Remove the document we just added
			s.hybridData.Documents = s.hybridData.Documents[:len(s.hybridData.Documents)-1]
			// Clean up body file if created
			if doc.BodyMeta != nil {
				_ = s.bodyStorage.DeleteBody(*doc.BodyMeta)
			}
			return fmt.Errorf("failed to save: %w", err)
		}

		return nil
	})

	if err != nil {
		return "", err
	}
	return newID, nil
}

// Update modifies an existing document
func (s *hybridJSONFileStore) Update(id string, updates types.UpdateRequest) error {
	// Extract body from Dimensions if present
	if updates.Dimensions != nil {
		if bodyVal, ok := updates.Dimensions["_body"]; ok {
			if bodyStr, ok := bodyVal.(string); ok {
				updates.Body = &bodyStr
			}
			delete(updates.Dimensions, "_body") // Remove from dimensions
		}
	}

	cmd := UpdateCommand{
		ID:      id,
		Request: updates,
	}

	// Preprocess the command
	if err := s.preprocessor.preprocessCommand(&cmd); err != nil {
		return err
	}
	updates = cmd.Request

	return s.lockManager.Execute(storage.WriteOperation, func() error {
		// Find the document
		var found bool
		var docIndex int
		for i, doc := range s.hybridData.Documents {
			if doc.UUID == cmd.ID {
				found = true
				docIndex = i
				break
			}
		}

		if !found {
			return fmt.Errorf("document not found: %s", cmd.ID)
		}

		// Update the document
		doc := &s.hybridData.Documents[docIndex]
		doc.UpdatedAt = s.timeFunc()

		// Update title if provided
		if updates.Title != nil && *updates.Title != "" {
			doc.Title = *updates.Title
		}

		// Update body if provided
		if updates.Body != nil {
			newBody := *updates.Body

			// Determine format
			format := BodyFormatText
			if doc.BodyMeta != nil {
				format = doc.BodyMeta.Format
			}
			if formatStr, ok := updates.Dimensions["_body.format"].(string); ok {
				if parsed, err := ParseBodyFormat(formatStr); err == nil {
					format = parsed
				}
				delete(updates.Dimensions, "_body.format")
			}

			// Determine if we should force embed
			forceEmbed := false
			if force, ok := updates.Dimensions["_body.embed"].(bool); ok {
				forceEmbed = force
				delete(updates.Dimensions, "_body.embed")
			}

			// Delete old body if it was in a file
			if doc.BodyMeta != nil && doc.BodyMeta.Type == BodyStorageFile {
				_ = s.bodyStorage.DeleteBody(*doc.BodyMeta)
			}

			// Write new body
			bodyMeta, embeddedBody, err := s.bodyStorage.WriteBody(doc.UUID, newBody, format, forceEmbed)
			if err != nil {
				return fmt.Errorf("failed to write body: %w", err)
			}
			doc.BodyMeta = &bodyMeta
			doc.Body = embeddedBody
		}

		// Apply dimension updates
		if updates.Dimensions != nil {
			// Validate updates
			for name, value := range updates.Dimensions {
				if !strings.HasPrefix(name, "_data.") {
					if err := validation.ValidateSimpleType(value, name); err != nil {
						return err
					}
				}
			}

			// Apply dimension updates
			for key, value := range updates.Dimensions {
				if value == nil {
					// nil value means delete the dimension
					delete(doc.Dimensions, key)
				} else {
					doc.Dimensions[key] = value
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

// Delete removes a document and optionally its children
func (s *hybridJSONFileStore) Delete(id string, cascade bool) error {
	return s.deleteMultiple([]string{id})
}

// deleteMultiple removes multiple documents
func (s *hybridJSONFileStore) deleteMultiple(ids []string) error {
	var deletedCount int
	err := s.lockManager.Execute(storage.WriteOperation, func() error {
		// Track which documents to keep
		var keepDocs []HybridDocument
		bodiesToDelete := []BodyMetadata{}

		for _, doc := range s.hybridData.Documents {
			shouldDelete := false
			for _, id := range ids {
				if doc.UUID == id {
					shouldDelete = true
					// Track body file for deletion
					if doc.BodyMeta != nil && doc.BodyMeta.Type == BodyStorageFile {
						bodiesToDelete = append(bodiesToDelete, *doc.BodyMeta)
					}
					deletedCount++
					break
				}
			}
			if !shouldDelete {
				keepDocs = append(keepDocs, doc)
			}
		}

		// Update documents
		s.hybridData.Documents = keepDocs

		// Save to file first
		if err := s.saveWithLock(); err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}

		// Then delete body files (after successful save)
		for _, bodyMeta := range bodiesToDelete {
			if err := s.bodyStorage.DeleteBody(bodyMeta); err != nil {
				// Log error but don't fail the delete operation
				fmt.Printf("Warning: Failed to delete body file: %v\n", err)
			}
		}

		return nil
	})

	return err
}

// GetByID retrieves a single document by ID
func (s *hybridJSONFileStore) GetByID(id string) (*types.Document, error) {
	var result *types.Document
	err := s.lockManager.Execute(storage.ReadOperation, func() error {
		// Find the document
		for _, hdoc := range s.hybridData.Documents {
			if hdoc.UUID == id {
				// Load body content
				if hdoc.BodyMeta != nil {
					body, err := s.bodyStorage.ReadBody(*hdoc.BodyMeta, hdoc.Body)
					if err != nil {
						return fmt.Errorf("failed to read body: %w", err)
					}
					hdoc.Body = body
				}
				doc := hdoc.ToStandardDocument()
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

// ResolveID implements the IDResolver interface
func (s *hybridJSONFileStore) ResolveID(simpleID string) (string, error) {
	// Convert to standard documents for processing
	standardDocs := make([]types.Document, len(s.hybridData.Documents))
	for i, hdoc := range s.hybridData.Documents {
		standardDocs[i] = hdoc.ToStandardDocument()
	}

	// Use the ID generator to resolve
	return s.idGenerator.ResolveID(simpleID, standardDocs)
}

// SetTimeFunc sets a custom time function for testing
func (s *hybridJSONFileStore) SetTimeFunc(fn func() time.Time) {
	s.timeFunc = fn
}

// Close releases any resources
func (s *hybridJSONFileStore) Close() error {
	return s.lockManager.Execute(storage.WriteOperation, func() error {
		// Don't need to save - data is saved on each operation
		// Just ensure the lock file is cleaned up
		lockPath := s.filePath + ".lock"
		_ = s.fs.Remove(lockPath)

		return nil
	})
}

// ResolveUUID converts a simple ID to UUID (delegated to query processor)
func (s *hybridJSONFileStore) ResolveUUID(simpleID string) (string, error) {
	// Convert to standard documents for processing
	standardDocs := make([]types.Document, len(s.hybridData.Documents))
	for i, hdoc := range s.hybridData.Documents {
		standardDocs[i] = hdoc.ToStandardDocument()
	}

	// Use the ID generator to resolve
	return s.idGenerator.ResolveID(simpleID, standardDocs)
}

// DeleteByDimension removes all documents matching filters
func (s *hybridJSONFileStore) DeleteByDimension(filters map[string]interface{}) (int, error) {
	return 0, errors.New("DeleteByDimension not implemented in hybrid store")
}

// UpdateByDimension updates all documents matching filters
func (s *hybridJSONFileStore) UpdateByDimension(filters map[string]interface{}, updates types.UpdateRequest) (int, error) {
	return 0, errors.New("UpdateByDimension not implemented in hybrid store")
}

// DeleteWhere removes documents matching a WHERE clause
func (s *hybridJSONFileStore) DeleteWhere(whereClause string, args ...interface{}) (int, error) {
	return 0, errors.New("DeleteWhere not supported in hybrid JSON store")
}

// UpdateWhere is not supported in the hybrid JSON store
func (s *hybridJSONFileStore) UpdateWhere(whereClause string, updates types.UpdateRequest, args ...interface{}) (int, error) {
	return 0, errors.New("UpdateWhere not supported in hybrid JSON store")
}
