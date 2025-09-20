package store

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/types"
)

func TestHybridMigration(t *testing.T) {
	t.Run("migrate body storage types", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()
		mockLockFactory := NewMockFileLockFactory()

		config := &testConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"todo", "done"}},
			},
		}

		// Create store with small embed limit
		store, err := NewHybridWithOptions("/test/store.json", config,
			WithFileSystemExt(mockFS),
			WithHybridFileLockFactory(mockLockFactory),
			WithEmbedSizeLimit(20), // Small limit
		)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Add documents with various body sizes
		smallID, _ := store.Add("Small", map[string]interface{}{
			"_body": "Tiny", // Will be embedded
		})

		largeID, _ := store.Add("Large", map[string]interface{}{
			"_body": strings.Repeat("Large content ", 10), // Will go to file
		})

		// Get hybrid store for migration
		hybridStore, ok := store.(*hybridJSONFileStore)
		if !ok {
			t.Fatal("expected hybrid store type")
		}

		// Test 1: Migrate all to files
		progressCalls := 0
		err = hybridStore.MigrateBodyStorage(MigrationOptions{
			TargetBodyStorage: BodyStorageFile,
			ProgressCallback: func(current, total int, docID string) {
				progressCalls++
			},
		})
		if err != nil {
			t.Fatalf("failed to migrate to files: %v", err)
		}

		if progressCalls != 2 {
			t.Errorf("expected 2 progress calls, got %d", progressCalls)
		}

		// Verify both are now in files
		doc1, _ := store.GetByID(smallID)
		doc2, _ := store.GetByID(largeID)

		if !mockFS.FileExists("/test/bodies/" + smallID + ".txt") {
			t.Error("small doc should now have body file")
		}
		if !mockFS.FileExists("/test/bodies/" + largeID + ".txt") {
			t.Error("large doc should still have body file")
		}

		// Bodies should still be readable
		if doc1.Body != "Tiny" {
			t.Error("small body content lost")
		}
		if !strings.Contains(doc2.Body, "Large content") {
			t.Error("large body content lost")
		}

		// Test 2: Migrate back to embedded (with force)
		err = hybridStore.MigrateBodyStorage(MigrationOptions{
			TargetBodyStorage: BodyStorageEmbedded,
			ForceEmbed:        true,
		})
		if err != nil {
			t.Fatalf("failed to migrate to embedded: %v", err)
		}

		// Verify files are gone
		if mockFS.FileExists("/test/bodies/" + smallID + ".txt") {
			t.Error("small doc body file should be deleted")
		}
		if mockFS.FileExists("/test/bodies/" + largeID + ".txt") {
			t.Error("large doc body file should be deleted")
		}

		// Bodies should still be readable
		doc1, _ = store.GetByID(smallID)
		doc2, _ = store.GetByID(largeID)
		if doc1.Body != "Tiny" {
			t.Error("small body content lost after embed migration")
		}
		if !strings.Contains(doc2.Body, "Large content") {
			t.Error("large body content lost after embed migration")
		}

		// Test 3: Auto-detect based on size
		err = hybridStore.MigrateBodyStorage(MigrationOptions{
			TargetBodyStorage: "", // Auto-detect
			EmbedSizeLimit:    30, // Custom limit
		})
		if err != nil {
			t.Fatalf("failed to auto-migrate: %v", err)
		}

		// Small should be embedded, large should be in file
		if mockFS.FileExists("/test/bodies/" + smallID + ".txt") {
			t.Error("small doc should not have file with auto-detect")
		}
		if !mockFS.FileExists("/test/bodies/" + largeID + ".txt") {
			t.Error("large doc should have file with auto-detect")
		}
	})

	t.Run("cleanup orphaned files", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()
		mockLockFactory := NewMockFileLockFactory()

		// Pre-create some orphaned files
		_ = mockFS.MkdirAll("/test/bodies", 0755)
		_ = mockFS.WriteFile("/test/bodies/orphan1.txt", []byte("orphan"), 0644)
		_ = mockFS.WriteFile("/test/bodies/orphan2.md", []byte("orphan"), 0644)
		_ = mockFS.WriteFile("/test/bodies/referenced.txt", []byte("ref"), 0644)

		config := &testConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"todo", "done"}},
			},
		}

		store, err := NewHybridWithOptions("/test/store.json", config,
			WithFileSystemExt(mockFS),
			WithHybridFileLockFactory(mockLockFactory),
		)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Add document that references one file
		_, _ = store.Add("Doc", map[string]interface{}{
			"_body":        "content", // Will create referenced.txt
			"_body.format": "txt",
		})

		// Force the document to use our specific file
		hybridStore := store.(*hybridJSONFileStore)
		hybridStore.hybridData.Documents[0].BodyMeta = &BodyMetadata{
			Type:     BodyStorageFile,
			Filename: "referenced.txt",
			Format:   BodyFormatText,
		}
		hybridStore.hybridData.Documents[0].Body = ""
		_ = hybridStore.saveWithLock()

		// Cleanup orphaned
		cleaned, err := hybridStore.CleanupOrphaned()
		if err != nil {
			t.Fatalf("failed to cleanup orphaned: %v", err)
		}

		if len(cleaned) != 2 {
			t.Errorf("expected 2 orphaned files cleaned, got %d", len(cleaned))
		}

		// Verify orphaned files are gone
		if mockFS.FileExists("/test/bodies/orphan1.txt") {
			t.Error("orphan1.txt should be deleted")
		}
		if mockFS.FileExists("/test/bodies/orphan2.md") {
			t.Error("orphan2.md should be deleted")
		}

		// Referenced file should remain
		if !mockFS.FileExists("/test/bodies/referenced.txt") {
			t.Error("referenced.txt should not be deleted")
		}
	})

	t.Run("convert legacy store", func(t *testing.T) {
		mockFS := &OSFileSystemExt{} // Use real FS for this test
		tempDir := t.TempDir()

		// Create legacy store data
		legacyPath := tempDir + "/legacy.json"
		legacyData := `{
			"documents": [
				{
					"uuid": "doc1",
					"title": "Small Doc",
					"body": "Small body",
					"dimensions": {"status": "todo"},
					"created_at": "2023-01-01T00:00:00Z",
					"updated_at": "2023-01-01T00:00:00Z"
				},
				{
					"uuid": "doc2",
					"title": "Large Doc",
					"body": "` + strings.Repeat("Large content ", 20) + `",
					"dimensions": {"status": "done"},
					"created_at": "2023-01-01T00:00:00Z",
					"updated_at": "2023-01-01T00:00:00Z"
				}
			],
			"metadata": {
				"version": "1.0",
				"created_at": "2023-01-01T00:00:00Z",
				"updated_at": "2023-01-01T00:00:00Z"
			}
		}`

		if err := mockFS.WriteFile(legacyPath, []byte(legacyData), 0644); err != nil {
			t.Fatalf("failed to write legacy file: %v", err)
		}

		// Convert to hybrid
		hybridPath := tempDir + "/hybrid.json"
		config := &testConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"todo", "done"}},
			},
		}

		err := ConvertLegacyStore(legacyPath, hybridPath, config, 50) // 50 byte limit
		if err != nil {
			t.Fatalf("failed to convert store: %v", err)
		}

		// Load hybrid store and verify
		hybridStore, err := NewHybrid(hybridPath, config, 50)
		if err != nil {
			t.Fatalf("failed to load converted store: %v", err)
		}
		defer func() { _ = hybridStore.Close() }()

		docs, _ := hybridStore.List(types.ListOptions{})
		if len(docs) != 2 {
			t.Fatalf("expected 2 documents, got %d", len(docs))
		}

		// Small doc should be embedded
		smallDoc := findDocByID(docs, "doc1")
		if smallDoc == nil || smallDoc.Body != "Small body" {
			t.Error("small doc not converted correctly")
		}

		// Large doc should be in file
		largeDoc := findDocByID(docs, "doc2")
		if largeDoc == nil || !strings.Contains(largeDoc.Body, "Large content") {
			t.Error("large doc not converted correctly")
		}

		// Check that body file exists for large doc
		bodyFile := tempDir + "/bodies/doc2.txt"
		if _, err := mockFS.Stat(bodyFile); err != nil {
			t.Error("large doc body file should exist")
		}
	})
}

// Helper to find document by ID
func findDocByID(docs []types.Document, id string) *types.Document {
	for i := range docs {
		if docs[i].UUID == id {
			return &docs[i]
		}
	}
	return nil
}
