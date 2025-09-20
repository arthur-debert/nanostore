//go:build integration
// +build integration

package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arthur-debert/nanostore/types"
)

// TestJSONStoreIntegration verifies that the real file system operations work correctly
// These tests use actual files and should be kept minimal
func TestJSONStoreIntegration(t *testing.T) {
	t.Run("real file system operations", func(t *testing.T) {
		// Create a temporary directory for test files
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "integration_test.json")

		config := &testConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"pending", "done"}, DefaultValue: "pending"},
			},
		}

		// Create store with default file system (OSFileSystem)
		store, err := New(testFile, config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}

		// Add documents
		id1, err := store.Add("First Document", map[string]interface{}{"status": "pending"})
		if err != nil {
			t.Fatalf("failed to add first document: %v", err)
		}

		id2, err := store.Add("Second Document", map[string]interface{}{"status": "done"})
		if err != nil {
			t.Fatalf("failed to add second document: %v", err)
		}

		// Close the store
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close store: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Fatal("expected file to exist after save")
		}

		// Open store again to verify persistence
		store2, err := New(testFile, config)
		if err != nil {
			t.Fatalf("failed to reopen store: %v", err)
		}
		defer store2.Close()

		// List all documents
		docs, err := store2.List(types.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		if len(docs) != 2 {
			t.Fatalf("expected 2 documents, got %d", len(docs))
		}

		// Verify documents by UUID
		foundFirst, foundSecond := false, false
		for _, doc := range docs {
			if doc.UUID == id1 && doc.Title == "First Document" {
				foundFirst = true
			}
			if doc.UUID == id2 && doc.Title == "Second Document" {
				foundSecond = true
			}
		}

		if !foundFirst {
			t.Error("first document not found after reload")
		}
		if !foundSecond {
			t.Error("second document not found after reload")
		}

		// Verify lock file is cleaned up after close
		if err := store2.Close(); err != nil {
			t.Fatalf("failed to close store: %v", err)
		}

		lockFile := testFile + ".lock"
		if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
			t.Error("lock file should be removed after close")
		}
	})

	t.Run("file permissions and errors", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("skipping permission test when running as root")
		}

		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "readonly", "test.json")

		// Create directory
		if err := os.MkdirAll(filepath.Dir(testFile), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		config := &testConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"pending", "done"}, DefaultValue: "pending"},
			},
		}

		// Create store and add a document
		store, err := New(testFile, config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}

		_, err = store.Add("Test Document", map[string]interface{}{"status": "pending"})
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		store.Close()

		// Make directory read-only
		if err := os.Chmod(filepath.Dir(testFile), 0555); err != nil {
			t.Fatalf("failed to change permissions: %v", err)
		}
		defer os.Chmod(filepath.Dir(testFile), 0755) // Restore permissions

		// Try to reopen and modify - should handle permission error gracefully
		store2, err := New(testFile, config)
		if err != nil {
			// This is OK - might fail to create lock file
			return
		}
		defer store2.Close()

		// Try to add a document - should fail due to permissions
		_, err = store2.Add("Another Document", map[string]interface{}{"status": "done"})
		if err == nil {
			t.Error("expected error when writing to read-only directory")
		}
	})
}
