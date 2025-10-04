package nanostore

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore/store"
	"github.com/arthur-debert/nanostore/types"
)

func TestExportIntegration(t *testing.T) {
	// Create a temporary store for testing
	tempFile := t.TempDir() + "/test-store.json"

	// Create configuration
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:   "status",
				Type:   types.Enumerated,
				Values: []string{"pending", "active", "completed"},
			},
			{
				Name:   "priority",
				Type:   types.Enumerated,
				Values: []string{"low", "medium", "high"},
			},
		},
	}

	// Create store
	store, err := store.New(tempFile, &config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close store: %v", err)
		}
	}()

	// Add some test documents
	doc1, err := store.Add("First Document", map[string]interface{}{
		"status":   "active",
		"priority": "high",
	})
	if err != nil {
		t.Fatalf("failed to add document 1: %v", err)
	}

	doc2, err := store.Add("Second Document", map[string]interface{}{
		"status":   "pending",
		"priority": "medium",
	})
	if err != nil {
		t.Fatalf("failed to add document 2: %v", err)
	}

	doc3, err := store.Add("Third Document", map[string]interface{}{
		"status":   "completed",
		"priority": "low",
	})
	if err != nil {
		t.Fatalf("failed to add document 3: %v", err)
	}

	// Test export metadata
	t.Run("export metadata", func(t *testing.T) {
		metadata, err := GetExportMetadata(store, ExportOptions{})
		if err != nil {
			t.Fatalf("failed to get export metadata: %v", err)
		}

		if metadata.DocumentCount != 3 {
			t.Errorf("expected 3 documents, got %d", metadata.DocumentCount)
		}

		if len(metadata.Documents) != 3 {
			t.Errorf("expected 3 document infos, got %d", len(metadata.Documents))
		}

		// Check that we have document info for each document
		foundDocs := make(map[string]bool)
		for _, docInfo := range metadata.Documents {
			foundDocs[docInfo.UUID] = true
			if docInfo.Title == "" {
				t.Errorf("document %s has empty title", docInfo.UUID)
			}
			if docInfo.Filename == "" {
				t.Errorf("document %s has empty filename", docInfo.UUID)
			}
		}

		if !foundDocs[doc1] || !foundDocs[doc2] || !foundDocs[doc3] {
			t.Error("not all documents found in metadata")
		}
	})

	// Test filtered export metadata
	t.Run("filtered export metadata", func(t *testing.T) {
		metadata, err := GetExportMetadata(store, ExportOptions{
			DimensionFilters: map[string]interface{}{
				"status": "active",
			},
		})
		if err != nil {
			t.Fatalf("failed to get filtered export metadata: %v", err)
		}

		if metadata.DocumentCount != 1 {
			t.Errorf("expected 1 document, got %d", metadata.DocumentCount)
		}
	})

	// Test full export
	t.Run("full export", func(t *testing.T) {
		archivePath, err := Export(store, ExportOptions{})
		if err != nil {
			t.Fatalf("failed to export: %v", err)
		}

		// Clean up
		defer func() {
			_ = os.RemoveAll(archivePath)
		}()

		// Verify archive exists
		if _, err := os.Stat(archivePath); os.IsNotExist(err) {
			t.Fatal("archive file was not created")
		}

		// Verify archive is not empty
		info, err := os.Stat(archivePath)
		if err != nil {
			t.Fatalf("failed to stat archive: %v", err)
		}
		if info.Size() == 0 {
			t.Fatal("archive file is empty")
		}
	})

	// Test export by IDs
	t.Run("export by IDs", func(t *testing.T) {
		// Get the simple IDs for the documents
		docs, err := store.List(types.NewListOptions())
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		if len(docs) < 2 {
			t.Fatal("not enough documents for ID test")
		}

		// Export only the first two documents
		archivePath, err := Export(store, ExportOptions{
			IDs: []string{docs[0].SimpleID, docs[1].SimpleID},
		})
		if err != nil {
			t.Fatalf("failed to export by IDs: %v", err)
		}

		// Clean up
		defer func() {
			_ = os.RemoveAll(archivePath)
		}()

		// Verify archive exists
		if _, err := os.Stat(archivePath); os.IsNotExist(err) {
			t.Fatal("archive file was not created")
		}
	})

	// Test export to specific path
	t.Run("export to path", func(t *testing.T) {
		exportPath := t.TempDir() + "/my-export.zip"

		err := ExportToPath(store, ExportOptions{}, exportPath)
		if err != nil {
			t.Fatalf("failed to export to path: %v", err)
		}

		// Verify archive exists at specified path
		if _, err := os.Stat(exportPath); os.IsNotExist(err) {
			t.Fatal("archive file was not created at specified path")
		}
	})
}
