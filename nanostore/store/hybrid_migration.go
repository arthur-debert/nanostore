package store

import (
	"encoding/json"
	"fmt"

	"github.com/arthur-debert/nanostore/nanostore/storage"
)

// MigrationOptions configures how to migrate storage formats
type MigrationOptions struct {
	// TargetBodyStorage specifies how bodies should be stored after migration
	TargetBodyStorage BodyStorageType

	// EmbedSizeLimit is the maximum size for embedded bodies when migrating to hybrid
	// If 0, uses the store's default
	EmbedSizeLimit int64

	// ForceEmbed forces all bodies to be embedded regardless of size
	ForceEmbed bool

	// CleanupOrphaned removes orphaned body files during migration
	CleanupOrphaned bool

	// ProgressCallback is called for each document migrated
	ProgressCallback func(current, total int, docID string)
}

// MigrateStorage migrates between storage formats
type MigrateStorage interface {
	// MigrateToHybrid converts a standard store to hybrid storage
	MigrateToHybrid(opts MigrationOptions) error

	// MigrateBodyStorage changes how bodies are stored (embedded vs file)
	MigrateBodyStorage(opts MigrationOptions) error

	// CleanupOrphaned removes body files not referenced by any document
	CleanupOrphaned() ([]string, error)
}

// MigrateToHybrid implements storage migration for hybrid store
func (s *hybridJSONFileStore) MigrateToHybrid(opts MigrationOptions) error {
	// This store is already hybrid, so this is a no-op
	return nil
}

// MigrateBodyStorage changes how bodies are stored
func (s *hybridJSONFileStore) MigrateBodyStorage(opts MigrationOptions) error {
	return s.lockManager.Execute(storage.WriteOperation, func() error {
		total := len(s.hybridData.Documents)
		migrated := 0

		for i := range s.hybridData.Documents {
			doc := &s.hybridData.Documents[i]

			// Skip documents without bodies
			if doc.BodyMeta == nil {
				continue
			}

			// Determine if migration is needed
			currentType := doc.BodyMeta.Type
			targetType := opts.TargetBodyStorage

			// Handle auto-detection based on size
			if targetType == "" {
				if opts.ForceEmbed {
					targetType = BodyStorageEmbedded
				} else {
					// Determine based on size
					limit := opts.EmbedSizeLimit
					if limit == 0 {
						limit = s.hybridData.Metadata.BodyStorageConfig.EmbedSizeLimit
					}
					if doc.BodyMeta.Size <= limit {
						targetType = BodyStorageEmbedded
					} else {
						targetType = BodyStorageFile
					}
				}
			}

			// Call progress callback for all documents
			if opts.ProgressCallback != nil {
				opts.ProgressCallback(i+1, total, doc.UUID)
			}

			// Skip if already in target format
			if currentType == targetType {
				continue
			}

			// Perform migration
			newMeta, newEmbedded, err := s.bodyStorage.MigrateBody(*doc.BodyMeta, doc.Body, targetType, doc.UUID)
			if err != nil {
				return fmt.Errorf("failed to migrate body for document %s: %w", doc.UUID, err)
			}

			// Update document
			doc.BodyMeta = &newMeta
			doc.Body = newEmbedded
			migrated++
		}

		// Update metadata
		s.hybridData.Metadata.UpdatedAt = s.timeFunc()

		// Save changes
		if err := s.saveWithLock(); err != nil {
			return fmt.Errorf("failed to save after migration: %w", err)
		}

		// Cleanup orphaned files if requested
		if opts.CleanupOrphaned {
			if _, err := s.CleanupOrphaned(); err != nil {
				// Log but don't fail
				fmt.Printf("Warning: Failed to cleanup orphaned files: %v\n", err)
			}
		}

		return nil
	})
}

// CleanupOrphaned removes body files not referenced by any document
func (s *hybridJSONFileStore) CleanupOrphaned() ([]string, error) {
	var cleaned []string

	err := s.lockManager.Execute(storage.WriteOperation, func() error {
		// Build list of body metadata from documents
		metas := make([]BodyMetadata, 0, len(s.hybridData.Documents))
		for _, doc := range s.hybridData.Documents {
			if doc.BodyMeta != nil {
				metas = append(metas, *doc.BodyMeta)
			}
		}

		// Find orphaned files
		orphaned, err := s.bodyStorage.ListOrphanedFiles(metas)
		if err != nil {
			return fmt.Errorf("failed to list orphaned files: %w", err)
		}

		// Delete each orphaned file
		for _, filename := range orphaned {
			meta := BodyMetadata{
				Type:     BodyStorageFile,
				Filename: filename,
			}
			if err := s.bodyStorage.DeleteBody(meta); err != nil {
				fmt.Printf("Warning: Failed to delete orphaned file %s: %v\n", filename, err)
			} else {
				cleaned = append(cleaned, filename)
			}
		}

		return nil
	})

	return cleaned, err
}

// ConvertLegacyStore creates a new hybrid store from a legacy store file
func ConvertLegacyStore(legacyPath, hybridPath string, config Config, embedLimit int64) error {
	// Create file systems
	fs := &OSFileSystemExt{}

	// Read legacy file
	data, err := fs.ReadFile(legacyPath)
	if err != nil {
		return fmt.Errorf("failed to read legacy store: %w", err)
	}

	// Parse legacy format
	var legacyData storage.StoreData
	if err := json.Unmarshal(data, &legacyData); err != nil {
		return fmt.Errorf("failed to parse legacy store: %w", err)
	}

	// Create new hybrid store
	store, err := NewHybridWithOptions(hybridPath, config,
		WithFileSystemExt(fs),
		WithEmbedSizeLimit(embedLimit),
	)
	if err != nil {
		return fmt.Errorf("failed to create hybrid store: %w", err)
	}
	defer func() { _ = store.Close() }()

	// Get the underlying hybrid store
	hybridStore, ok := store.(*hybridJSONFileStore)
	if !ok {
		return fmt.Errorf("unexpected store type")
	}

	// Convert and save
	hybridStore.hybridData = hybridStore.convertLegacyToHybrid(&legacyData)

	// Migrate large bodies to files
	opts := MigrationOptions{
		TargetBodyStorage: "", // Auto-detect based on size
		EmbedSizeLimit:    embedLimit,
	}

	if err := hybridStore.MigrateBodyStorage(opts); err != nil {
		return fmt.Errorf("failed to migrate body storage: %w", err)
	}

	return nil
}
