package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestListEmpty(t *testing.T) {
	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("expected 0 documents, got %d", len(docs))
	}
}

func TestListWithIDs(t *testing.T) {
	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add some pending documents
	id1, err := store.Add("First task", nil)
	if err != nil {
		t.Fatalf("failed to add first document: %v", err)
	}

	id2, err := store.Add("Second task", nil)
	if err != nil {
		t.Fatalf("failed to add second document: %v", err)
	}

	// Add a completed document
	id3, err := store.Add("Completed task", nil)
	if err != nil {
		t.Fatalf("failed to add third document: %v", err)
	}

	err = store.SetStatus(id3, nanostore.StatusCompleted)
	if err != nil {
		t.Fatalf("failed to set status: %v", err)
	}

	// List all documents
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) != 3 {
		t.Errorf("expected 3 documents, got %d", len(docs))
	}

	// Check IDs are generated correctly
	expectedIDs := map[string]string{
		id1: "1",  // First pending
		id2: "2",  // Second pending
		id3: "c1", // First completed
	}

	for _, doc := range docs {
		expectedID, ok := expectedIDs[doc.UUID]
		if !ok {
			t.Errorf("unexpected document UUID: %s", doc.UUID)
			continue
		}

		if doc.UserFacingID != expectedID {
			t.Errorf("expected ID %s for document %s, got %s", expectedID, doc.UUID, doc.UserFacingID)
		}
	}
}

func TestListHierarchical(t *testing.T) {
	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a parent document
	parentID, err := store.Add("Parent task", nil)
	if err != nil {
		t.Fatalf("failed to add parent: %v", err)
	}

	// Add child documents
	child1ID, err := store.Add("Child 1", &parentID)
	if err != nil {
		t.Fatalf("failed to add child 1: %v", err)
	}

	child2ID, err := store.Add("Child 2", &parentID)
	if err != nil {
		t.Fatalf("failed to add child 2: %v", err)
	}

	// Add a completed child
	child3ID, err := store.Add("Completed child", &parentID)
	if err != nil {
		t.Fatalf("failed to add child 3: %v", err)
	}
	err = store.SetStatus(child3ID, nanostore.StatusCompleted)
	if err != nil {
		t.Fatalf("failed to set status: %v", err)
	}

	// List all documents
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) != 4 {
		t.Errorf("expected 4 documents, got %d", len(docs))
	}

	// Check hierarchical IDs
	expectedIDs := map[string]string{
		parentID: "1",    // Parent
		child1ID: "1.1",  // First child
		child2ID: "1.2",  // Second child
		child3ID: "1.c1", // First completed child
	}

	for _, doc := range docs {
		expectedID, ok := expectedIDs[doc.UUID]
		if !ok {
			t.Errorf("unexpected document UUID: %s", doc.UUID)
			continue
		}

		if doc.UserFacingID != expectedID {
			t.Errorf("expected ID %s for document %s, got %s", expectedID, doc.UUID, doc.UserFacingID)
		}
	}
}

func TestListFilteredIDs(t *testing.T) {
	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a mix of pending and completed documents
	pending1, _ := store.Add("Pending 1", nil)
	pending2, _ := store.Add("Pending 2", nil)
	pending3, _ := store.Add("Pending 3", nil)

	completed1, _ := store.Add("Completed 1", nil)
	_ = store.SetStatus(completed1, nanostore.StatusCompleted)

	completed2, _ := store.Add("Completed 2", nil)
	_ = store.SetStatus(completed2, nanostore.StatusCompleted)

	// Add more pending after completed
	pending4, _ := store.Add("Pending 4", nil)

	// Test 1: Filter for pending documents only
	pendingDocs, err := store.List(nanostore.ListOptions{
		FilterByStatus: []nanostore.Status{nanostore.StatusPending},
	})
	if err != nil {
		t.Fatalf("failed to list pending: %v", err)
	}

	if len(pendingDocs) != 4 {
		t.Errorf("expected 4 pending documents, got %d", len(pendingDocs))
	}

	// Verify IDs are contiguous for pending
	expectedPendingIDs := map[string]string{
		pending1: "1",
		pending2: "2",
		pending3: "3",
		pending4: "4",
	}

	for _, doc := range pendingDocs {
		expectedID, ok := expectedPendingIDs[doc.UUID]
		if !ok {
			t.Errorf("unexpected pending document: %s", doc.UUID)
			continue
		}
		if doc.UserFacingID != expectedID {
			t.Errorf("expected ID %s for pending doc %s, got %s", expectedID, doc.UUID, doc.UserFacingID)
		}
	}

	// Test 2: Filter for completed documents only
	completedDocs, err := store.List(nanostore.ListOptions{
		FilterByStatus: []nanostore.Status{nanostore.StatusCompleted},
	})
	if err != nil {
		t.Fatalf("failed to list completed: %v", err)
	}

	if len(completedDocs) != 2 {
		t.Errorf("expected 2 completed documents, got %d", len(completedDocs))
	}

	// Verify IDs are contiguous for completed
	expectedCompletedIDs := map[string]string{
		completed1: "c1",
		completed2: "c2",
	}

	for _, doc := range completedDocs {
		expectedID, ok := expectedCompletedIDs[doc.UUID]
		if !ok {
			t.Errorf("unexpected completed document: %s", doc.UUID)
			continue
		}
		if doc.UserFacingID != expectedID {
			t.Errorf("expected ID %s for completed doc %s, got %s", expectedID, doc.UUID, doc.UserFacingID)
		}
	}
}

func TestListFilteredHierarchicalIDs(t *testing.T) {
	// When filtering by status/search, hierarchical IDs are not preserved
	// because the tree structure is broken. Documents get simple sequential IDs.

	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create hierarchical structure with mixed statuses
	root1, _ := store.Add("Root 1", nil)
	root2, _ := store.Add("Root 2", nil)

	// Children of root1
	child1_1, _ := store.Add("Child 1.1", &root1)
	child1_2, _ := store.Add("Child 1.2", &root1)
	_ = store.SetStatus(child1_2, nanostore.StatusCompleted)
	child1_3, _ := store.Add("Child 1.3", &root1)

	// Children of root2 (all completed)
	child2_1, _ := store.Add("Child 2.1", &root2)
	_ = store.SetStatus(child2_1, nanostore.StatusCompleted)
	child2_2, _ := store.Add("Child 2.2", &root2)
	_ = store.SetStatus(child2_2, nanostore.StatusCompleted)

	// Grandchildren
	grandchild, _ := store.Add("Grandchild", &child1_1)

	// Test: Filter for pending documents only
	pendingDocs, err := store.List(nanostore.ListOptions{
		FilterByStatus: []nanostore.Status{nanostore.StatusPending},
	})
	if err != nil {
		t.Fatalf("failed to list pending: %v", err)
	}

	// When filtering by status, hierarchical IDs are not preserved
	// Documents are numbered sequentially: 1, 2, 3, 4, 5
	// Order is by created_at: root1, root2, child1_1, child1_3, grandchild
	expectedPendingIDs := map[string]string{
		root1:      "1",
		root2:      "2",
		child1_1:   "3",
		child1_3:   "4",
		grandchild: "5",
	}

	if len(pendingDocs) != len(expectedPendingIDs) {
		t.Errorf("expected %d pending documents, got %d", len(expectedPendingIDs), len(pendingDocs))
	}

	for _, doc := range pendingDocs {
		expectedID, ok := expectedPendingIDs[doc.UUID]
		if !ok {
			t.Errorf("unexpected pending document: %s", doc.UUID)
			continue
		}
		if doc.UserFacingID != expectedID {
			t.Errorf("expected ID %s for pending doc %s, got %s", expectedID, doc.UUID, doc.UserFacingID)
		}
	}
}

func TestListFilterByParentIDs(t *testing.T) {
	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create structure
	root1, _ := store.Add("Root 1", nil)
	root2, _ := store.Add("Root 2", nil)
	_ = store.SetStatus(root2, nanostore.StatusCompleted)
	root3, _ := store.Add("Root 3", nil)

	child1, _ := store.Add("Child 1", &root1)
	child2, _ := store.Add("Child 2", &root1)
	_ = store.SetStatus(child2, nanostore.StatusCompleted)

	// Test: Get only root documents
	emptyString := ""
	rootDocs, err := store.List(nanostore.ListOptions{
		FilterByParent: &emptyString,
	})
	if err != nil {
		t.Fatalf("failed to list roots: %v", err)
	}

	if len(rootDocs) != 3 {
		t.Errorf("expected 3 root documents, got %d", len(rootDocs))
	}

	// Verify root IDs are contiguous
	expectedRootIDs := map[string]string{
		root1: "1",
		root2: "c1",
		root3: "2", // This should be 2, not 3
	}

	for _, doc := range rootDocs {
		expectedID, ok := expectedRootIDs[doc.UUID]
		if !ok {
			t.Errorf("unexpected root document: %s", doc.UUID)
			continue
		}
		if doc.UserFacingID != expectedID {
			t.Errorf("expected ID %s for root doc %s, got %s", expectedID, doc.UUID, doc.UserFacingID)
		}
	}

	// Test: Get children of root1
	root1Children, err := store.List(nanostore.ListOptions{
		FilterByParent: &root1,
	})
	if err != nil {
		t.Fatalf("failed to list children: %v", err)
	}

	if len(root1Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(root1Children))
	}

	// Verify child IDs start from 1
	expectedChildIDs := map[string]string{
		child1: "1",
		child2: "c1",
	}

	for _, doc := range root1Children {
		expectedID, ok := expectedChildIDs[doc.UUID]
		if !ok {
			t.Errorf("unexpected child document: %s", doc.UUID)
			continue
		}
		if doc.UserFacingID != expectedID {
			t.Errorf("expected ID %s for child doc %s, got %s", expectedID, doc.UUID, doc.UserFacingID)
		}
	}
}

func TestListCombinedFilters(t *testing.T) {
	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test data
	root1, _ := store.Add("Project Alpha", nil)
	root2, _ := store.Add("Project Beta", nil)
	_ = store.SetStatus(root2, nanostore.StatusCompleted)

	task1, _ := store.Add("Design mockups", &root1)
	task2, _ := store.Add("Write tests", &root1)
	_ = store.SetStatus(task2, nanostore.StatusCompleted)
	task3, _ := store.Add("Deploy to production", &root1)

	// Test: Search + Status filter
	results, err := store.List(nanostore.ListOptions{
		FilterBySearch: "Project",
		FilterByStatus: []nanostore.Status{nanostore.StatusPending},
	})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].UUID != root1 {
		t.Errorf("expected root1, got %s", results[0].UUID)
	}

	// Verify the ID is correct (should be 1, not affected by completed root2)
	if len(results) > 0 && results[0].UserFacingID != "1" {
		t.Errorf("expected ID '1', got %s", results[0].UserFacingID)
	}

	// Test: Parent + Status filter
	results, err = store.List(nanostore.ListOptions{
		FilterByParent: &root1,
		FilterByStatus: []nanostore.Status{nanostore.StatusPending},
	})
	if err != nil {
		t.Fatalf("failed to filter by parent and status: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 pending children, got %d", len(results))
	}

	// Verify IDs are contiguous for filtered children
	expectedIDs := map[string]string{
		task1: "1",
		task3: "2", // Should be 2, not 3
	}

	for _, doc := range results {
		expectedID, ok := expectedIDs[doc.UUID]
		if !ok {
			continue
		}
		if doc.UserFacingID != expectedID {
			t.Errorf("expected ID %s for doc %s, got %s", expectedID, doc.UUID, doc.UserFacingID)
		}
	}
}
