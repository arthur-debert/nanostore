package nanostore_test

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestBulkAdd(t *testing.T) {
	store, err := nanostore.New(":memory:", nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "completed"},
				Prefixes:     map[string]string{"completed": "c"},
				DefaultValue: "pending",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Test adding many documents rapidly
	count := 1000
	start := time.Now()

	ids := make([]string, count)
	for i := 0; i < count; i++ {
		id, err := store.Add(fmt.Sprintf("Bulk Document %d", i), nil)
		if err != nil {
			t.Fatalf("failed to add document %d: %v", i, err)
		}
		ids[i] = id
	}

	elapsed := time.Since(start)
	t.Logf("Added %d documents in %v (%.2f docs/sec)",
		count, elapsed, float64(count)/elapsed.Seconds())

	// Verify all documents exist
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(docs) != count {
		t.Errorf("expected %d documents, got %d", count, len(docs))
	}

	// Verify sequential IDs
	for i, doc := range docs {
		expectedID := fmt.Sprintf("%d", i+1)
		if doc.UserFacingID != expectedID {
			t.Errorf("document %d has ID %s, expected %s",
				i, doc.UserFacingID, expectedID)
		}
	}
}

func TestBulkUpdate(t *testing.T) {
	store, err := nanostore.New(":memory:", nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "completed"},
				Prefixes:     map[string]string{"completed": "c"},
				DefaultValue: "pending",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create documents
	count := 500
	ids := make([]string, count)
	for i := 0; i < count; i++ {
		id, err := store.Add(fmt.Sprintf("Original %d", i), nil)
		if err != nil {
			t.Fatalf("failed to add document %d: %v", i, err)
		}
		ids[i] = id
	}

	// Bulk update
	start := time.Now()
	for i, id := range ids {
		newTitle := fmt.Sprintf("Updated %d", i)
		newBody := fmt.Sprintf("Body content for document %d", i)
		err := store.Update(id, nanostore.UpdateRequest{
			Title: &newTitle,
			Body:  &newBody,
		})
		if err != nil {
			t.Fatalf("failed to update document %d: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	t.Logf("Updated %d documents in %v (%.2f updates/sec)",
		count, elapsed, float64(count)/elapsed.Seconds())

	// Verify updates
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	for i, doc := range docs {
		expectedTitle := fmt.Sprintf("Updated %d", i)
		expectedBody := fmt.Sprintf("Body content for document %d", i)

		if doc.Title != expectedTitle {
			t.Errorf("document %d title mismatch: got %q, want %q",
				i, doc.Title, expectedTitle)
		}
		if doc.Body != expectedBody {
			t.Errorf("document %d body mismatch", i)
		}
	}
}

func TestBulkStatusChange(t *testing.T) {
	store, err := nanostore.New(":memory:", nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "completed"},
				Prefixes:     map[string]string{"completed": "c"},
				DefaultValue: "pending",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create documents
	count := 500
	ids := make([]string, count)
	for i := 0; i < count; i++ {
		id, err := store.Add(fmt.Sprintf("Task %d", i), nil)
		if err != nil {
			t.Fatalf("failed to add document %d: %v", i, err)
		}
		ids[i] = id
	}

	// Bulk status change - mark even numbers as completed
	start := time.Now()
	statusChanges := 0
	for i, id := range ids {
		if i%2 == 0 {
			err := store.Update(id, nanostore.UpdateRequest{
				Dimensions: map[string]string{"status": "completed"},
			})
			if err != nil {
				t.Fatalf("failed to set status for document %d: %v", i, err)
			}
			statusChanges++
		}
	}
	elapsed := time.Since(start)
	t.Logf("Changed %d statuses in %v (%.2f changes/sec)",
		statusChanges, elapsed, float64(statusChanges)/elapsed.Seconds())

	// Verify status distribution
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	pendingCount := 0
	completedCount := 0
	for _, doc := range docs {
		status, _ := doc.Dimensions["status"].(string)
		switch status {
		case "pending":
			pendingCount++
		case "completed":
			completedCount++
		}
	}

	if pendingCount != count/2 {
		t.Errorf("expected %d pending documents, got %d", count/2, pendingCount)
	}
	if completedCount != count/2 {
		t.Errorf("expected %d completed documents, got %d", count/2, completedCount)
	}
}

func TestBulkHierarchicalCreation(t *testing.T) {
	store, err := nanostore.New(":memory:", nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "completed"},
				Prefixes:     map[string]string{"completed": "c"},
				DefaultValue: "pending",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a wide hierarchy
	rootCount := 10
	childrenPerRoot := 50

	start := time.Now()
	totalDocs := 0

	for i := 0; i < rootCount; i++ {
		rootID, err := store.Add(fmt.Sprintf("Project %d", i), nil)
		if err != nil {
			t.Fatalf("failed to add root %d: %v", i, err)
		}
		totalDocs++

		for j := 0; j < childrenPerRoot; j++ {
			_, err := store.Add(fmt.Sprintf("Task %d.%d", i, j), map[string]interface{}{"parent_uuid": rootID})
			if err != nil {
				t.Fatalf("failed to add child %d.%d: %v", i, j, err)
			}
			totalDocs++
		}
	}

	elapsed := time.Since(start)
	t.Logf("Created %d hierarchical documents in %v (%.2f docs/sec)",
		totalDocs, elapsed, float64(totalDocs)/elapsed.Seconds())

	// Verify hierarchy
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(docs) != totalDocs {
		t.Errorf("expected %d documents, got %d", totalDocs, len(docs))
	}

	// Count roots and children
	roots := 0
	children := 0
	for _, doc := range docs {
		if doc.GetParentUUID() == nil {
			roots++
		} else {
			children++
		}
	}

	if roots != rootCount {
		t.Errorf("expected %d roots, got %d", rootCount, roots)
	}
	if children != rootCount*childrenPerRoot {
		t.Errorf("expected %d children, got %d", rootCount*childrenPerRoot, children)
	}
}

func TestConcurrentBulkOperations(t *testing.T) {
	// Note: SQLite serializes writes, but we can test that concurrent
	// bulk operations don't corrupt data
	// Use file-based database for proper concurrent access
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "concurrent_bulk.db")

	store, err := nanostore.New(dbPath, nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "completed"},
				Prefixes:     map[string]string{"completed": "c"},
				DefaultValue: "pending",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create initial documents
	baseCount := 100
	for i := 0; i < baseCount; i++ {
		_, err := store.Add(fmt.Sprintf("Base %d", i), nil)
		if err != nil {
			t.Fatalf("failed to add base document %d: %v", i, err)
		}
	}

	// Run concurrent operations
	var wg sync.WaitGroup
	errors := make(chan error, 300)

	// Goroutine 1: Add more documents
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_, err := store.Add(fmt.Sprintf("Concurrent Add %d", i), nil)
			if err != nil {
				errors <- fmt.Errorf("add error: %v", err)
			}
		}
	}()

	// Goroutine 2: List operations
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			docs, err := store.List(nanostore.ListOptions{})
			if err != nil {
				errors <- fmt.Errorf("list error: %v", err)
			}
			if len(docs) < baseCount {
				errors <- fmt.Errorf("missing documents during concurrent ops")
			}
		}
	}()

	// Goroutine 3: Resolve operations
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 100; i++ {
			_, err := store.ResolveUUID(fmt.Sprintf("%d", i))
			if err != nil && i <= baseCount {
				errors <- fmt.Errorf("resolve error for ID %d: %v", i, err)
			}
		}
	}()

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("concurrent operation error: %v", err)
		errorCount++
		if errorCount > 10 {
			t.Fatal("too many concurrent errors")
		}
	}

	// Final verification
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list after concurrent ops: %v", err)
	}

	if len(docs) != 200 { // 100 base + 100 concurrent
		t.Errorf("expected 200 documents after concurrent ops, got %d", len(docs))
	}
}

func TestBulkOperationMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	store, err := nanostore.New(":memory:", nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "completed"},
				Prefixes:     map[string]string{"completed": "c"},
				DefaultValue: "pending",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add a large number of documents and verify memory doesn't explode
	// This is a basic test - real memory profiling would use pprof
	count := 10000

	for i := 0; i < count; i++ {
		_, err := store.Add(fmt.Sprintf("Memory Test Document %d with some content to make it non-trivial", i), nil)
		if err != nil {
			t.Fatalf("failed at document %d: %v", i, err)
		}

		// Periodically list to ensure we can handle large result sets
		if i%1000 == 999 {
			docs, err := store.List(nanostore.ListOptions{})
			if err != nil {
				t.Fatalf("failed to list at %d documents: %v", i+1, err)
			}
			if len(docs) != i+1 {
				t.Errorf("expected %d documents, got %d", i+1, len(docs))
			}
			t.Logf("Successfully handling %d documents", i+1)
		}
	}

	// Final verification
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list all: %v", err)
	}

	if len(docs) != count {
		t.Errorf("expected %d documents, got %d", count, len(docs))
	}
}
