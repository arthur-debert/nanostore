package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/types"
)

func TestHybridBodyStorage(t *testing.T) {
	t.Run("embedded storage for small bodies", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()
		storage := NewHybridBodyStorage(mockFS, "/test", 1024) // 1KB limit

		// Small body should be embedded
		meta, embedded, err := storage.WriteBody("doc1", "Small content", BodyFormatText, false)
		if err != nil {
			t.Fatalf("failed to write small body: %v", err)
		}

		if meta.Type != BodyStorageEmbedded {
			t.Errorf("expected embedded storage, got %s", meta.Type)
		}
		if embedded != "Small content" {
			t.Errorf("expected embedded content, got %q", embedded)
		}
		if meta.Size != int64(len("Small content")) {
			t.Errorf("expected size %d, got %d", len("Small content"), meta.Size)
		}

		// Should not create any files
		entries, _ := mockFS.ReadDir("/test/bodies")
		if len(entries) > 0 {
			t.Error("expected no files for embedded storage")
		}
	})

	t.Run("file storage for large bodies", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()
		storage := NewHybridBodyStorage(mockFS, "/test", 10) // 10 byte limit

		// Large body should go to file
		largeContent := strings.Repeat("Large content ", 10)
		meta, embedded, err := storage.WriteBody("doc2", largeContent, BodyFormatMarkdown, false)
		if err != nil {
			t.Fatalf("failed to write large body: %v", err)
		}

		if meta.Type != BodyStorageFile {
			t.Errorf("expected file storage, got %s", meta.Type)
		}
		if embedded != "" {
			t.Errorf("expected empty embedded content, got %q", embedded)
		}
		if meta.Filename != "doc2.md" {
			t.Errorf("expected filename doc2.md, got %s", meta.Filename)
		}

		// Verify file was created
		content, ok := mockFS.GetFileContent("/test/bodies/doc2.md")
		if !ok {
			t.Fatal("body file not created")
		}
		if string(content) != largeContent {
			t.Errorf("file content mismatch")
		}
	})

	t.Run("force embed large bodies", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()
		storage := NewHybridBodyStorage(mockFS, "/test", 10) // 10 byte limit

		// Force embed even for large content
		largeContent := strings.Repeat("Large ", 10)
		meta, embedded, err := storage.WriteBody("doc3", largeContent, BodyFormatText, true)
		if err != nil {
			t.Fatalf("failed to write with force embed: %v", err)
		}

		if meta.Type != BodyStorageEmbedded {
			t.Errorf("expected embedded storage with force flag, got %s", meta.Type)
		}
		if embedded != largeContent {
			t.Errorf("expected embedded content with force flag")
		}
	})

	t.Run("read body from different storage types", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()
		storage := NewHybridBodyStorage(mockFS, "/test", 100)

		// Test embedded read
		embeddedMeta := BodyMetadata{
			Type:   BodyStorageEmbedded,
			Format: BodyFormatText,
			Size:   14,
		}
		content, err := storage.ReadBody(embeddedMeta, "Embedded text")
		if err != nil {
			t.Fatalf("failed to read embedded body: %v", err)
		}
		if content != "Embedded text" {
			t.Errorf("embedded content mismatch: got %q", content)
		}

		// Test file read
		fileContent := "File content here"
		_ = mockFS.WriteFile("/test/bodies/doc4.txt", []byte(fileContent), 0644)

		fileMeta := BodyMetadata{
			Type:     BodyStorageFile,
			Format:   BodyFormatText,
			Filename: "doc4.txt",
			Size:     int64(len(fileContent)),
		}
		content, err = storage.ReadBody(fileMeta, "")
		if err != nil {
			t.Fatalf("failed to read file body: %v", err)
		}
		if content != fileContent {
			t.Errorf("file content mismatch: got %q", content)
		}
	})

	t.Run("delete body files", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()
		storage := NewHybridBodyStorage(mockFS, "/test", 10)

		// Create a file
		_ = mockFS.MkdirAll("/test/bodies", 0755)
		_ = mockFS.WriteFile("/test/bodies/doc5.html", []byte("<p>HTML</p>"), 0644)

		// Delete file storage
		fileMeta := BodyMetadata{
			Type:     BodyStorageFile,
			Filename: "doc5.html",
		}
		err := storage.DeleteBody(fileMeta)
		if err != nil {
			t.Fatalf("failed to delete body file: %v", err)
		}

		// Verify file is gone
		if mockFS.FileExists("/test/bodies/doc5.html") {
			t.Error("body file should be deleted")
		}

		// Delete embedded storage (should be no-op)
		embeddedMeta := BodyMetadata{
			Type: BodyStorageEmbedded,
		}
		err = storage.DeleteBody(embeddedMeta)
		if err != nil {
			t.Fatalf("unexpected error deleting embedded body: %v", err)
		}
	})

	t.Run("validate body files", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()
		storage := NewHybridBodyStorage(mockFS, "/test", 100)

		// Validate embedded (always valid)
		embeddedMeta := BodyMetadata{Type: BodyStorageEmbedded}
		if err := storage.ValidateBody(embeddedMeta); err != nil {
			t.Errorf("embedded body should always be valid: %v", err)
		}

		// Validate existing file
		_ = mockFS.MkdirAll("/test/bodies", 0755)
		_ = mockFS.WriteFile("/test/bodies/exists.txt", []byte("content"), 0644)

		existsMeta := BodyMetadata{
			Type:     BodyStorageFile,
			Filename: "exists.txt",
		}
		if err := storage.ValidateBody(existsMeta); err != nil {
			t.Errorf("existing file should be valid: %v", err)
		}

		// Validate missing file
		missingMeta := BodyMetadata{
			Type:     BodyStorageFile,
			Filename: "missing.txt",
		}
		if err := storage.ValidateBody(missingMeta); err == nil {
			t.Error("missing file should be invalid")
		}
	})

	t.Run("list orphaned files", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()
		storage := NewHybridBodyStorage(mockFS, "/test", 100)

		// Create some files
		_ = mockFS.MkdirAll("/test/bodies", 0755)
		_ = mockFS.WriteFile("/test/bodies/referenced.txt", []byte("ref"), 0644)
		_ = mockFS.WriteFile("/test/bodies/orphaned.txt", []byte("orphan"), 0644)
		_ = mockFS.WriteFile("/test/bodies/.gitkeep", []byte(""), 0644)

		// Document metadata only references one file
		metas := []BodyMetadata{
			{Type: BodyStorageFile, Filename: "referenced.txt"},
			{Type: BodyStorageEmbedded}, // This doesn't reference a file
		}

		orphaned, err := storage.ListOrphanedFiles(metas)
		if err != nil {
			t.Fatalf("failed to list orphaned files: %v", err)
		}

		if len(orphaned) != 1 {
			t.Fatalf("expected 1 orphaned file, got %d", len(orphaned))
		}
		if orphaned[0] != "orphaned.txt" {
			t.Errorf("expected orphaned.txt, got %s", orphaned[0])
		}
	})

	t.Run("migrate body storage", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()
		storage := NewHybridBodyStorage(mockFS, "/test", 100)

		// Start with embedded
		embeddedMeta := BodyMetadata{
			Type:   BodyStorageEmbedded,
			Format: BodyFormatText,
			Size:   20,
		}
		embeddedContent := "This is embedded"

		// Migrate to file
		testUUID := "test-uuid-123"
		newMeta, newEmbedded, err := storage.MigrateBody(embeddedMeta, embeddedContent, BodyStorageFile, testUUID)
		if err != nil {
			t.Fatalf("failed to migrate to file: %v", err)
		}

		if newMeta.Type != BodyStorageFile {
			t.Errorf("expected file storage after migration, got %s", newMeta.Type)
		}
		if newEmbedded != "" {
			t.Errorf("expected empty embedded content after migration to file")
		}
		expectedFilename := testUUID + ".txt"
		if newMeta.Filename != expectedFilename {
			t.Errorf("expected filename %s, got %s", expectedFilename, newMeta.Filename)
		}

		// Verify file was created
		if !mockFS.FileExists("/test/bodies/" + newMeta.Filename) {
			t.Error("body file should be created during migration")
		}

		// Migrate back to embedded
		finalMeta, finalEmbedded, err := storage.MigrateBody(newMeta, "", BodyStorageEmbedded, "")
		if err != nil {
			t.Fatalf("failed to migrate back to embedded: %v", err)
		}

		if finalMeta.Type != BodyStorageEmbedded {
			t.Errorf("expected embedded storage after migration, got %s", finalMeta.Type)
		}
		if finalEmbedded != embeddedContent {
			t.Errorf("content lost during migration: got %q", finalEmbedded)
		}

		// Old file should be deleted
		if mockFS.FileExists("/test/bodies/" + newMeta.Filename) {
			t.Error("old body file should be deleted after migration")
		}
	})
}

func TestHybridJSONStore(t *testing.T) {
	t.Run("create and load hybrid store", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()
		mockLockFactory := NewMockFileLockFactory()

		config := &testConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"todo", "done"}},
			},
		}

		store, err := NewHybridWithOptions("/test/store.json", config,
			WithFileSystemExt(mockFS),
			WithHybridFileLockFactory(mockLockFactory),
			WithEmbedSizeLimit(50),
		)
		if err != nil {
			t.Fatalf("failed to create hybrid store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Add document with small body
		id1, err := store.Add("Task 1", map[string]interface{}{
			"status": "todo",
			"_body":  "Small body",
		})
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Add document with large body
		largeBody := strings.Repeat("Large body content. ", 10)
		id2, err := store.Add("Task 2", map[string]interface{}{
			"status":       "done",
			"_body":        largeBody,
			"_body.format": "md",
		})
		if err != nil {
			t.Fatalf("failed to add document with large body: %v", err)
		}

		// Verify store data
		content, ok := mockFS.GetFileContent("/test/store.json")
		if !ok {
			t.Fatal("store file not created")
		}

		var hybridData HybridStoreData
		if err := json.Unmarshal(content, &hybridData); err != nil {
			t.Fatalf("failed to parse hybrid store data: %v", err)
		}

		if hybridData.Metadata.StorageVersion != "hybrid_v1" {
			t.Errorf("expected hybrid_v1, got %s", hybridData.Metadata.StorageVersion)
		}

		// Check first document (embedded)
		doc1 := findHybridDocByID(hybridData.Documents, id1)
		if doc1 == nil {
			t.Fatal("document 1 not found")
		}
		if doc1.BodyMeta == nil || doc1.BodyMeta.Type != BodyStorageEmbedded {
			t.Error("small body should be embedded")
		}
		if doc1.Body != "Small body" {
			t.Errorf("embedded body mismatch: got %q", doc1.Body)
		}

		// Check second document (file)
		doc2 := findHybridDocByID(hybridData.Documents, id2)
		if doc2 == nil {
			t.Fatal("document 2 not found")
		}
		if doc2.BodyMeta == nil || doc2.BodyMeta.Type != BodyStorageFile {
			t.Error("large body should be in file")
		}
		if doc2.Body != "" {
			t.Error("file-stored body should not be embedded")
		}

		// Verify body file exists
		bodyFile := fmt.Sprintf("/test/bodies/%s.md", id2)
		if !mockFS.FileExists(bodyFile) {
			t.Error("body file should exist")
		}

		// Test retrieval
		retrieved, err := store.GetByID(id2)
		if err != nil {
			t.Fatalf("failed to retrieve document: %v", err)
		}
		if retrieved.Body != largeBody {
			t.Error("body content should be loaded from file")
		}
	})

	t.Run("update document body", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()
		mockLockFactory := NewMockFileLockFactory()

		config := &testConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"todo", "done"}},
			},
		}

		store, err := NewHybridWithOptions("/test/store.json", config,
			WithFileSystemExt(mockFS),
			WithHybridFileLockFactory(mockLockFactory),
			WithEmbedSizeLimit(20),
		)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Add document with embedded body
		id, err := store.Add("Doc", map[string]interface{}{
			"status": "todo",
			"_body":  "Short",
		})
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Update with large body
		longBody := strings.Repeat("Long content ", 10)
		err = store.Update(id, types.UpdateRequest{
			Body: &longBody,
			Dimensions: map[string]interface{}{
				"_body.format": "html",
			},
		})
		if err != nil {
			t.Fatalf("failed to update body: %v", err)
		}

		// Verify body is now in file
		doc, _ := store.GetByID(id)
		if doc.Body != longBody {
			t.Error("body content mismatch after update")
		}

		bodyFile := fmt.Sprintf("/test/bodies/%s.html", id)
		if !mockFS.FileExists(bodyFile) {
			t.Error("body file should exist after update")
		}
	})

	t.Run("delete documents with body files", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()
		mockLockFactory := NewMockFileLockFactory()

		config := &testConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"todo", "done"}},
			},
		}

		store, err := NewHybridWithOptions("/test/store.json", config,
			WithFileSystemExt(mockFS),
			WithHybridFileLockFactory(mockLockFactory),
			WithEmbedSizeLimit(10),
		)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Add documents
		id1, _ := store.Add("Small", map[string]interface{}{
			"_body": "tiny",
		})

		id2, _ := store.Add("Large", map[string]interface{}{
			"_body": strings.Repeat("large ", 20),
		})

		// Verify body file exists
		bodyFile := fmt.Sprintf("/test/bodies/%s.txt", id2)
		if !mockFS.FileExists(bodyFile) {
			t.Fatal("body file should exist before delete")
		}

		// Delete both documents
		err = store.Delete(id1, false)
		if err != nil {
			t.Fatalf("failed to delete: %v", err)
		}
		err = store.Delete(id2, false)
		if err != nil {
			t.Fatalf("failed to delete: %v", err)
		}

		// Verify body file is deleted
		if mockFS.FileExists(bodyFile) {
			t.Error("body file should be deleted with document")
		}

		// Verify documents are gone
		doc1, _ := store.GetByID(id1)
		doc2, _ := store.GetByID(id2)
		if doc1 != nil || doc2 != nil {
			t.Error("documents should be deleted")
		}
	})

	t.Run("load legacy format", func(t *testing.T) {
		mockFS := NewMockFileSystemExt()

		// Create a legacy format file
		legacyData := map[string]interface{}{
			"documents": []map[string]interface{}{
				{
					"uuid":       "legacy-1",
					"title":      "Legacy Doc",
					"body":       "Legacy body content",
					"dimensions": map[string]interface{}{"status": "todo"},
					"created_at": "2023-01-01T00:00:00Z",
					"updated_at": "2023-01-01T00:00:00Z",
				},
			},
			"metadata": map[string]interface{}{
				"version":    "1.0",
				"created_at": "2023-01-01T00:00:00Z",
				"updated_at": "2023-01-01T00:00:00Z",
			},
		}

		legacyJSON, _ := json.MarshalIndent(legacyData, "", "  ")
		_ = mockFS.WriteFile("/test/legacy.json", legacyJSON, 0644)

		// Load as hybrid store
		config := &testConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"todo", "done"}},
			},
		}

		store, err := NewHybridWithOptions("/test/legacy.json", config,
			WithFileSystemExt(mockFS),
			WithHybridFileLockFactory(NewMockFileLockFactory()),
		)
		if err != nil {
			t.Fatalf("failed to load legacy store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Verify document was converted
		docs, err := store.List(types.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		if len(docs) != 1 {
			t.Fatalf("expected 1 document, got %d", len(docs))
		}

		if docs[0].UUID != "legacy-1" {
			t.Errorf("document UUID mismatch")
		}
		if docs[0].Body != "Legacy body content" {
			t.Errorf("body content lost in conversion")
		}

		// Add a new document to trigger save in hybrid format
		_, err = store.Add("New Doc", map[string]interface{}{"status": "done"})
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Verify file is now in hybrid format
		content, _ := mockFS.GetFileContent("/test/legacy.json")
		var hybridData HybridStoreData
		if err := json.Unmarshal(content, &hybridData); err != nil {
			t.Fatalf("failed to parse as hybrid: %v", err)
		}

		if hybridData.Metadata.StorageVersion != "hybrid_v1" {
			t.Error("file should be saved in hybrid format")
		}
	})
}

// Helper function to find document in hybrid format
func findHybridDocByID(docs []HybridDocument, id string) *HybridDocument {
	for i := range docs {
		if docs[i].UUID == id {
			return &docs[i]
		}
	}
	return nil
}
