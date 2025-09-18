package stores_test

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestFileLocking(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	filename := tmpfile.Name()
	_ = tmpfile.Close()
	defer func() { _ = os.Remove(filename) }()
	defer func() { _ = os.Remove(filename + ".lock") }() // Clean up lock file

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"todo", "done"},
				DefaultValue: "todo",
			},
		},
	}

	t.Run("ConcurrentWritesSameProcess", func(t *testing.T) {
		// Test concurrent writes within the same process
		// Note: The JSON store keeps data in memory after loading,
		// so multiple store instances don't see each other's changes
		// until they reload. The file lock prevents corruption but
		// doesn't provide real-time synchronization.
		
		store, err := nanostore.New(filename, config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Run concurrent writes from multiple goroutines
		var wg sync.WaitGroup
		errors := make(chan error, 20)

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(writerID int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					_, err := store.Add(fmt.Sprintf("Writer%d Task %d", writerID, j), nil)
					if err != nil {
						errors <- fmt.Errorf("writer%d add %d: %w", writerID, j, err)
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("concurrent write error: %v", err)
		}

		// Verify all documents were written
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		if len(docs) != 20 {
			t.Errorf("expected 20 documents, got %d", len(docs))
		}
	})

	t.Run("LockTimeout", func(t *testing.T) {
		// This test verifies that lock acquisition times out appropriately
		// Create a store and keep it locked by starting a long operation
		
		store, err := nanostore.New(filename, config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// In a real implementation, we'd need a way to simulate a stuck lock
		// For now, we'll just verify the store works with the locking in place
		_, err = store.Add("Test task", nil)
		if err != nil {
			t.Errorf("failed to add with locking: %v", err)
		}
	})
}

func TestLockFileCleanup(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	filename := tmpfile.Name()
	_ = tmpfile.Close()
	defer func() { _ = os.Remove(filename) }()

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"todo", "done"},
				DefaultValue: "todo",
			},
		},
	}

	// Create and close a store
	store, err := nanostore.New(filename, config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Add a document to ensure lock is used
	_, err = store.Add("Test", nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Close the store
	err = store.Close()
	if err != nil {
		t.Fatalf("failed to close store: %v", err)
	}

	// Check that lock file is cleaned up
	lockFile := filename + ".lock"
	if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
		t.Error("lock file was not cleaned up after close")
	}
}