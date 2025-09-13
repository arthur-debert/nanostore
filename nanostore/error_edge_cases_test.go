package nanostore_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestNewWithInvalidPath(t *testing.T) {
	// Try to create store with invalid database path
	_, err := nanostore.NewTestStore("/root/nonexistent/invalid.db")
	if err == nil {
		t.Error("expected error when creating store with invalid path")
	}
}

func TestNewWithReadOnlyDirectory(t *testing.T) {
	// Create a temporary directory and make it read-only
	tmpDir := t.TempDir()

	// Make the directory read-only (this might not work on all systems)
	oldMode := os.FileMode(0755)
	if err := os.Chmod(tmpDir, 0444); err != nil {
		t.Skip("cannot make directory read-only on this system")
	}

	// Restore permissions after test
	defer func() { _ = os.Chmod(tmpDir, oldMode) }()

	// Try to create database in read-only directory
	dbPath := tmpDir + "/test.db"
	_, err := nanostore.NewTestStore(dbPath)
	if err == nil {
		t.Error("expected error when creating store in read-only directory")
	}
}

func TestSetStatusTransactionFailure(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a document
	docID, err := store.Add("Test Document", nil, nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Close the store to make subsequent operations fail
	_ = store.Close()

	// Try to set status on closed store (should fail)
	err = store.SetStatus(docID, nanostore.StatusCompleted)
	if err == nil {
		t.Error("expected error when setting status on closed store")
	}
}

func TestListWithComplexFilterCombinations(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test documents
	parentID, _ := store.Add("Parent", nil, nil)
	childID, _ := store.Add("Child with searchable content", &parentID, nil)
	_ = store.SetStatus(childID, nanostore.StatusCompleted)

	// Test complex filter combinations that might expose edge cases
	testCases := []struct {
		name string
		opts nanostore.ListOptions
	}{
		{
			name: "empty parent filter with status",
			opts: nanostore.ListOptions{
				FilterByParent: func() *string { s := ""; return &s }(),
				FilterByStatus: []nanostore.Status{nanostore.StatusPending},
			},
		},
		{
			name: "all filters combined",
			opts: nanostore.ListOptions{
				FilterByStatus: []nanostore.Status{nanostore.StatusCompleted},
				FilterBySearch: "searchable",
				FilterByParent: &parentID,
			},
		},
		{
			name: "search with special characters",
			opts: nanostore.ListOptions{
				FilterBySearch: "content with % and _ wildcards",
			},
		},
		{
			name: "multiple statuses with search",
			opts: nanostore.ListOptions{
				FilterByStatus: []nanostore.Status{
					nanostore.StatusPending,
					nanostore.StatusCompleted,
				},
				FilterBySearch: "searchable",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			docs, err := store.List(tc.opts)
			if err != nil {
				t.Errorf("complex filter failed: %v", err)
			}
			// Don't check results, just ensure no errors
			_ = docs
		})
	}
}

func TestAddWithExtremelyLongParentChain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long parent chain test in short mode")
	}

	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a very long parent chain to test potential stack overflow
	// or performance issues in recursive queries
	var parentID *string
	const chainLength = 1000

	for i := 0; i < chainLength; i++ {
		id, err := store.Add("Deep", parentID, nil)
		if err != nil {
			t.Fatalf("failed to add document %d: %v", i, err)
		}
		parentID = &id

		// Test every 100 documents to ensure operations still work
		if i%100 == 0 {
			docs, err := store.List(nanostore.ListOptions{})
			if err != nil {
				t.Fatalf("failed to list at depth %d: %v", i, err)
			}
			if len(docs) != i+1 {
				t.Errorf("expected %d documents at depth %d, got %d", i+1, i, len(docs))
			}
		}
	}
}

func TestUpdateWithNonExistentParent(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a document
	docID, err := store.Add("Test Document", nil, nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Try to set a non-existent parent (should fail due to foreign key constraint)
	nonExistentParent := "00000000-0000-0000-0000-000000000000"
	err = store.Update(docID, nanostore.UpdateRequest{
		ParentID: &nonExistentParent,
	})
	if err == nil {
		t.Error("expected error when setting non-existent parent")
	}
}

func TestResolveUUIDWithMalformedInput(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Test various malformed inputs that might cause panics or unexpected behavior
	malformedIDs := []string{
		"",
		"0",
		"c0",
		"-1",
		"c-1",
		"1.",
		".1",
		"1..2",
		"1.2.",
		".1.2",
		"1.2.3.4.5.6.7.8.9.10",  // Very deep
		"999999999999999999999", // Very large number
		"c999999999999999999",   // Very large completed number
	}

	for _, id := range malformedIDs {
		t.Run("malformed_"+id, func(t *testing.T) {
			_, err := store.ResolveUUID(id)
			if err == nil {
				t.Errorf("expected error for malformed ID: %s", id)
			}
		})
	}
}

func TestConcurrentCircularReferenceCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}

	tmpFile := t.TempDir() + "/concurrent.db"
	store, err := nanostore.NewTestStore(tmpFile)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create a simple hierarchy: A -> B -> C
	aID, _ := store.Add("A", nil, nil)
	bID, _ := store.Add("B", &aID, nil)
	cID, _ := store.Add("C", &bID, nil)

	_ = store.Close()

	// Try concurrent updates that could create circular references
	// This tests the robustness of the circular reference detection
	errChan := make(chan error, 2)

	// Goroutine 1: Try to make A child of C
	go func() {
		s, err := nanostore.NewTestStore(tmpFile)
		if err != nil {
			errChan <- err
			return
		}
		defer func() { _ = s.Close() }()

		err = s.Update(aID, nanostore.UpdateRequest{
			ParentID: &cID,
		})
		errChan <- err
	}()

	// Goroutine 2: Try to make B child of C (also creates circle)
	go func() {
		s, err := nanostore.NewTestStore(tmpFile)
		if err != nil {
			errChan <- err
			return
		}
		defer func() { _ = s.Close() }()

		err = s.Update(bID, nanostore.UpdateRequest{
			ParentID: &cID,
		})
		errChan <- err
	}()

	// Both should fail due to circular reference detection
	for i := 0; i < 2; i++ {
		err := <-errChan
		if err == nil {
			t.Error("expected circular reference error in concurrent test")
		}
	}
}
