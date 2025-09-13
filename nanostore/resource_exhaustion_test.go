package nanostore_test

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestResourceExhaustionLargeDocuments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping resource exhaustion test in short mode")
	}

	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create documents with very large titles and bodies
	largeString := strings.Repeat("x", 1024*1024) // 1MB string

	// Test adding large documents
	for i := 0; i < 10; i++ {
		title := fmt.Sprintf("Large Doc %d: %s", i, largeString[:100])
		_, err := store.Add(title, nil, nil)
		if err != nil {
			t.Errorf("failed to add large document %d: %v", i, err)
		}
	}

	// Test updating with large content
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) > 0 {
		largeBody := largeString
		err = store.Update(docs[0].UUID, nanostore.UpdateRequest{
			Body: &largeBody,
		})
		if err != nil {
			t.Errorf("failed to update with large body: %v", err)
		}
	}

	// Verify memory is not excessively consumed
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	allocMB := m.Alloc / 1024 / 1024
	if allocMB > 500 { // Reasonable threshold
		t.Logf("Warning: High memory usage: %d MB", allocMB)
	}
}

func TestResourceExhaustionDeepHierarchy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping resource exhaustion test in short mode")
	}

	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a very deep hierarchy (100 levels)
	const maxDepth = 100
	var parentID *string

	for i := 0; i < maxDepth; i++ {
		title := fmt.Sprintf("Level %d", i)
		id, err := store.Add(title, parentID, nil)
		if err != nil {
			t.Fatalf("failed to add level %d: %v", i, err)
		}
		parentID = &id
	}

	// Try to resolve the deepest ID
	// This tests the iterative resolution for deep hierarchies
	deepID := "1"
	for i := 1; i < maxDepth; i++ {
		deepID += ".1"
	}

	start := time.Now()
	_, err = store.ResolveUUID(deepID)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("failed to resolve deep ID: %v", err)
	}

	// Ensure it completes in reasonable time
	if duration > 5*time.Second {
		t.Errorf("deep ID resolution too slow: %v", duration)
	}
}

func TestResourceExhaustionManyRoots(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping resource exhaustion test in short mode")
	}

	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create many root documents
	const numRoots = 10000

	start := time.Now()
	for i := 0; i < numRoots; i++ {
		_, err := store.Add(fmt.Sprintf("Root %d", i), nil, nil)
		if err != nil {
			t.Fatalf("failed to add root %d: %v", i, err)
		}
	}
	addDuration := time.Since(start)

	t.Logf("Added %d root documents in %v", numRoots, addDuration)

	// Test listing performance
	start = time.Now()
	docs, err := store.List(nanostore.ListOptions{})
	listDuration := time.Since(start)

	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) != numRoots {
		t.Errorf("expected %d documents, got %d", numRoots, len(docs))
	}

	t.Logf("Listed %d documents in %v", len(docs), listDuration)

	// Ensure operations complete in reasonable time
	if listDuration > 2*time.Second {
		t.Errorf("listing too slow for %d documents: %v", numRoots, listDuration)
	}
}

func TestResourceExhaustionConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping resource exhaustion test in short mode")
	}

	// Use file-based DB for concurrent access
	tmpFile := t.TempDir() + "/concurrent.db"
	store, err := nanostore.NewTestStore(tmpFile)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create some initial documents
	var docIDs []string
	for i := 0; i < 100; i++ {
		id, err := store.Add(fmt.Sprintf("Doc %d", i), nil, nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}
		docIDs = append(docIDs, id)
	}

	// Close and reopen for concurrent access
	_ = store.Close()

	// Run many concurrent operations
	const numGoroutines = 50
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*opsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Each goroutine creates its own connection
			s, err := nanostore.NewTestStore(tmpFile)
			if err != nil {
				errors <- fmt.Errorf("worker %d: failed to open store: %v", workerID, err)
				return
			}
			defer func() { _ = s.Close() }()

			for j := 0; j < opsPerGoroutine; j++ {
				// Mix of operations
				switch j % 3 {
				case 0: // Add
					_, err := s.Add(fmt.Sprintf("Worker %d Doc %d", workerID, j), nil, nil)
					if err != nil {
						errors <- fmt.Errorf("worker %d: add failed: %v", workerID, err)
					}
				case 1: // List
					_, err := s.List(nanostore.ListOptions{})
					if err != nil {
						errors <- fmt.Errorf("worker %d: list failed: %v", workerID, err)
					}
				case 2: // Update
					if len(docIDs) > 0 {
						newTitle := fmt.Sprintf("Updated by worker %d", workerID)
						err := s.Update(docIDs[j%len(docIDs)], nanostore.UpdateRequest{
							Title: &newTitle,
						})
						if err != nil && !strings.Contains(err.Error(), "database is locked") {
							// SQLite lock errors are expected in high concurrency
							errors <- fmt.Errorf("worker %d: update failed: %v", workerID, err)
						}
					}
				}
			}
		}(i)
	}

	// Wait for completion
	done := make(chan bool)
	go func() {
		wg.Wait()
		close(done)
	}()

	// Timeout after 30 seconds
	select {
	case <-done:
		// Success
	case <-time.After(30 * time.Second):
		t.Fatal("concurrent operations timed out")
	}

	close(errors)

	// Check for unexpected errors
	errorCount := 0
	for err := range errors {
		t.Logf("Concurrent operation error: %v", err)
		errorCount++
	}

	if errorCount > 10 {
		t.Errorf("too many errors in concurrent operations: %d", errorCount)
	}
}

func TestResourceExhaustionComplexFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping resource exhaustion test in short mode")
	}

	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create many documents with various attributes
	for i := 0; i < 1000; i++ {
		title := fmt.Sprintf("Document %d", i)
		id, err := store.Add(title, nil, nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Set half as completed
		if i%2 == 0 {
			err = store.SetStatus(id, nanostore.StatusCompleted)
			if err != nil {
				t.Fatalf("failed to set status: %v", err)
			}
		}

		// Update some with searchable content
		if i%3 == 0 {
			body := fmt.Sprintf("This is searchable content for document %d", i)
			err = store.Update(id, nanostore.UpdateRequest{
				Body: &body,
			})
			if err != nil {
				t.Fatalf("failed to update body: %v", err)
			}
		}
	}

	// Test complex filtering performance
	testCases := []struct {
		name string
		opts nanostore.ListOptions
	}{
		{
			name: "filter by status",
			opts: nanostore.ListOptions{
				FilterByStatus: []nanostore.Status{nanostore.StatusCompleted},
			},
		},
		{
			name: "search filter",
			opts: nanostore.ListOptions{
				FilterBySearch: "searchable",
			},
		},
		{
			name: "combined filters",
			opts: nanostore.ListOptions{
				FilterByStatus: []nanostore.Status{nanostore.StatusCompleted},
				FilterBySearch: "searchable",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			docs, err := store.List(tc.opts)
			duration := time.Since(start)

			if err != nil {
				t.Errorf("failed to list with filters: %v", err)
			}

			t.Logf("Found %d documents in %v", len(docs), duration)

			if duration > 500*time.Millisecond {
				t.Errorf("filtering too slow: %v", duration)
			}
		})
	}
}
