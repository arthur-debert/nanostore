package nanostore_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestDatabaseErrors(t *testing.T) {
	t.Run("InvalidDatabasePath", func(t *testing.T) {
		// Try to create database in non-existent directory
		_, err := nanostore.New("/non/existent/path/db.sqlite")
		if err == nil {
			t.Fatal("expected error for invalid path")
		}
	})

	t.Run("ReadOnlyDatabase", func(t *testing.T) {
		t.Skip("Skipping read-only test - permission handling varies by OS")
		// Create temporary database
		tmpDir, err := os.MkdirTemp("", "nanostore-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		dbPath := filepath.Join(tmpDir, "readonly.db")

		// Create database first
		store, err := nanostore.New(dbPath)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		_ = store.Close()

		// Make it read-only
		err = os.Chmod(dbPath, 0444)
		if err != nil {
			t.Fatalf("failed to make database read-only: %v", err)
		}

		// Try to open read-only database
		store2, err := nanostore.New(dbPath)
		if err == nil {
			_ = store2.Close()
			t.Fatal("expected error opening read-only database")
		}
	})
}

func TestInvalidInputs(t *testing.T) {
	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("UpdateNonExistentDocument", func(t *testing.T) {
		title := "New Title"
		err := store.Update("non-existent-uuid", nanostore.UpdateRequest{
			Title: &title,
		})
		if err == nil {
			t.Fatal("expected error updating non-existent document")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("SetStatusNonExistentDocument", func(t *testing.T) {
		err := store.SetStatus("non-existent-uuid", nanostore.StatusCompleted)
		if err == nil {
			t.Fatal("expected error setting status on non-existent document")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("InvalidParentUUID", func(t *testing.T) {
		invalidParent := "invalid-uuid"
		_, err := store.Add("Child", &invalidParent)
		if err == nil {
			t.Fatal("expected error adding document with invalid parent")
		}
	})

	t.Run("ResolveInvalidFormats", func(t *testing.T) {
		invalidIDs := []string{
			"",                // empty
			"abc",             // non-numeric
			"1.abc",           // non-numeric child
			"1.",              // trailing dot
			".1",              // leading dot
			"1..2",            // double dot
			"c",               // completed without number
			"cc1",             // double 'c'
			"1c1",             // 'c' in wrong position
			"-1",              // negative
			"0",               // zero (IDs start at 1)
			"1.0",             // zero child
			"😀",               // emoji
			"1\n2",            // newline
			"1 2",             // space
			string([]byte{0}), // null byte
		}

		for _, id := range invalidIDs {
			_, err := store.ResolveUUID(id)
			if err == nil {
				t.Errorf("expected error for invalid ID format: %q", id)
			}
		}
	})
}

func TestDatabaseIntegrityErrors(t *testing.T) {
	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Note: We can't easily test constraint violations without direct SQL access
	// but we can verify that our API prevents invalid operations

	t.Run("SelfReferencingDocument", func(t *testing.T) {
		// The API doesn't allow setting a document as its own parent
		// This is more of a schema validation
		id, err := store.Add("Test", nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// We can't set a document as its own parent through the API
		// but if we could, it should fail
		_ = id // Document can't reference itself through our API
	})
}

func TestConcurrentDatabaseAccess(t *testing.T) {
	// SQLite handles concurrent access with busy timeouts
	// Our implementation should handle SQLITE_BUSY errors gracefully

	tmpDir, err := os.MkdirTemp("", "nanostore-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	dbPath := filepath.Join(tmpDir, "concurrent.db")

	// Create multiple stores accessing the same database
	stores := make([]nanostore.Store, 3)
	for i := 0; i < 3; i++ {
		store, err := nanostore.New(dbPath)
		if err != nil {
			t.Fatalf("failed to create store %d: %v", i, err)
		}
		stores[i] = store
		defer func(s nanostore.Store) { _ = s.Close() }(store)
	}

	// Each store adds documents
	for i, store := range stores {
		for j := 0; j < 10; j++ {
			_, err := store.Add(strings.Repeat("A", i*10+j), nil)
			if err != nil {
				t.Errorf("store %d failed to add document %d: %v", i, j, err)
			}
		}
	}

	// Verify all documents are visible from all stores
	for i, store := range stores {
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Errorf("store %d failed to list: %v", i, err)
		}
		if len(docs) != 30 {
			t.Errorf("store %d sees %d documents, expected 30", i, len(docs))
		}
	}
}

func TestClosedStoreErrors(t *testing.T) {
	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Close the store
	err = store.Close()
	if err != nil {
		t.Fatalf("failed to close store: %v", err)
	}

	// Try to use closed store
	t.Run("AddAfterClose", func(t *testing.T) {
		_, err := store.Add("Test", nil)
		if err == nil {
			t.Fatal("expected error using closed store")
		}
	})

	t.Run("ListAfterClose", func(t *testing.T) {
		_, err := store.List(nanostore.ListOptions{})
		if err == nil {
			t.Fatal("expected error using closed store")
		}
	})

	t.Run("UpdateAfterClose", func(t *testing.T) {
		title := "Test"
		err := store.Update("some-id", nanostore.UpdateRequest{Title: &title})
		if err == nil {
			t.Fatal("expected error using closed store")
		}
	})

	t.Run("ResolveAfterClose", func(t *testing.T) {
		_, err := store.ResolveUUID("1")
		if err == nil {
			t.Fatal("expected error using closed store")
		}
	})

	t.Run("DoubleClose", func(t *testing.T) {
		err := store.Close()
		// Double close might not error, but shouldn't panic
		_ = err
	})
}
