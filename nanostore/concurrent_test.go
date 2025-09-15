package nanostore_test

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestConcurrentReads(t *testing.T) {
	// Use file-based database for concurrent access
	// In-memory databases don't share data between connections
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "concurrent_test.db")

	// Create and populate store
	store, err := nanostore.NewTestStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Add test data
	for i := 0; i < 10; i++ {
		_, err := store.Add(fmt.Sprintf("Document %d", i), nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}
	}
	_ = store.Close()

	// Concurrent reads from multiple connections
	var wg sync.WaitGroup
	errors := make(chan error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			// Each goroutine opens its own connection
			s, err := nanostore.NewTestStore(dbPath)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: failed to open store: %v", n, err)
				return
			}
			defer func() { _ = s.Close() }()

			docs, err := s.List(nanostore.ListOptions{})
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: failed to list: %v", n, err)
				return
			}

			if len(docs) != 10 {
				errors <- fmt.Errorf("goroutine %d: expected 10 documents, got %d", n, len(docs))
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent read error: %v", err)
	}
}
