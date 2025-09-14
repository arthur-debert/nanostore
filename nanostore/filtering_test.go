package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestFilterByStatus(t *testing.T) {
	// Filtering by status is now implemented

	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create documents with different statuses
	pendingIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		id, err := store.Add("Pending "+string(rune('A'+i)), nil)
		if err != nil {
			t.Fatalf("failed to add pending document: %v", err)
		}
		pendingIDs[i] = id
	}

	completedIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		id, err := store.Add("Completed "+string(rune('A'+i)), nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}
		err = nanostore.SetStatus(store, id, "completed")
		if err != nil {
			t.Fatalf("failed to set status: %v", err)
		}
		completedIDs[i] = id
	}

	// Test filter by pending status
	pendingDocs, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"status": "pending"},
	})
	if err != nil {
		t.Fatalf("failed to list pending: %v", err)
	}

	if len(pendingDocs) != 5 {
		t.Errorf("expected 5 pending documents, got %d", len(pendingDocs))
	}

	for _, doc := range pendingDocs {
		if doc.GetStatus() != "pending" {
			t.Errorf("expected pending status, got %s", doc.GetStatus())
		}
	}

	// Test filter by completed status
	completedDocs, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"status": "completed"},
	})
	if err != nil {
		t.Fatalf("failed to list completed: %v", err)
	}

	if len(completedDocs) != 3 {
		t.Errorf("expected 3 completed documents, got %d", len(completedDocs))
	}

	// Test filter by multiple statuses
	allDocs, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"status": []string{"pending", "completed"}},
	})
	if err != nil {
		t.Fatalf("failed to list all: %v", err)
	}

	if len(allDocs) != 8 {
		t.Errorf("expected 8 documents total, got %d", len(allDocs))
	}

	// Test empty filter (should return all)
	allDocs2, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list without filter: %v", err)
	}

	if len(allDocs2) != 8 {
		t.Errorf("expected 8 documents without filter, got %d", len(allDocs2))
	}
}

func TestFilterByParent(t *testing.T) {
	// Filtering by parent is now implemented

	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create hierarchy
	root1, err := store.Add("Root 1", nil)
	if err != nil {
		t.Fatalf("failed to add root 1: %v", err)
	}

	root2, err := store.Add("Root 2", nil)
	if err != nil {
		t.Fatalf("failed to add root 2: %v", err)
	}

	// Children of root1
	var root1Children []string
	for i := 0; i < 3; i++ {
		id, err := store.Add("Child 1."+string(rune('A'+i)), map[string]interface{}{"parent_uuid": root1})
		if err != nil {
			t.Fatalf("failed to add child: %v", err)
		}
		root1Children = append(root1Children, id)
	}

	// Children of root2
	for i := 0; i < 2; i++ {
		_, err := store.Add("Child 2."+string(rune('A'+i)), map[string]interface{}{"parent_uuid": root2})
		if err != nil {
			t.Fatalf("failed to add child: %v", err)
		}
	}

	// Grandchildren
	grandchild, err := store.Add("Grandchild", map[string]interface{}{"parent_uuid": root1Children[0]})
	if err != nil {
		t.Fatalf("failed to add grandchild: %v", err)
	}

	// Test filter by root documents only
	roots, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"parent_uuid": ""},
	})
	if err != nil {
		t.Fatalf("failed to list roots: %v", err)
	}

	if len(roots) != 2 {
		t.Errorf("expected 2 root documents, got %d", len(roots))
	}

	// Test filter by specific parent
	root1Kids, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"parent_uuid": root1},
	})
	if err != nil {
		t.Fatalf("failed to list root1 children: %v", err)
	}

	if len(root1Kids) != 3 {
		t.Errorf("expected 3 children of root1, got %d", len(root1Kids))
	}

	// Test filter by different parent
	root2Kids, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"parent_uuid": root2},
	})
	if err != nil {
		t.Fatalf("failed to list root2 children: %v", err)
	}

	if len(root2Kids) != 2 {
		t.Errorf("expected 2 children of root2, got %d", len(root2Kids))
	}

	// Test grandchildren
	grandchildren, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"parent_uuid": root1Children[0]},
	})
	if err != nil {
		t.Fatalf("failed to list grandchildren: %v", err)
	}

	if len(grandchildren) != 1 {
		t.Errorf("expected 1 grandchild, got %d", len(grandchildren))
	}

	if grandchildren[0].UUID != grandchild {
		t.Error("grandchild UUID mismatch")
	}
}

func TestFilterBySearch(t *testing.T) {
	// Text search is now implemented

	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create documents with searchable content
	docs := []struct {
		title string
		body  string
	}{
		{"Meeting Notes", "Discussed project timeline and deliverables"},
		{"Project Plan", "Timeline for Q1 includes design phase"},
		{"Design Document", "User interface mockups and wireframes"},
		{"Test Report", "All tests passing, coverage at 95%"},
		{"Bug Report", "Issue with user authentication flow"},
	}

	for _, doc := range docs {
		id, err := store.Add(doc.title, nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		if doc.body != "" {
			err = store.Update(id, nanostore.UpdateRequest{Body: &doc.body})
			if err != nil {
				t.Fatalf("failed to update body: %v", err)
			}
		}
	}

	// Test search in title
	results, err := store.List(nanostore.ListOptions{
		FilterBySearch: "Report",
	})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 documents with 'Report' in title, got %d", len(results))
	}

	// Test search in body
	results, err = store.List(nanostore.ListOptions{
		FilterBySearch: "timeline",
	})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 documents with 'timeline', got %d", len(results))
	}

	// Test case-insensitive search
	results, err = store.List(nanostore.ListOptions{
		FilterBySearch: "PROJECT",
	})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 documents with 'PROJECT', got %d", len(results))
	}

	// Test no results
	results, err = store.List(nanostore.ListOptions{
		FilterBySearch: "nonexistent",
	})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results for 'nonexistent', got %d", len(results))
	}
}

func TestCombinedFilters(t *testing.T) {
	// Combined filtering is now implemented

	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create hierarchy with mixed statuses
	root1, _ := store.Add("Project Alpha", nil)
	root2, _ := store.Add("Project Beta", nil)

	// Add children with different statuses
	_, _ = store.Add("Design Phase", map[string]interface{}{"parent_uuid": root1})
	task2, _ := store.Add("Implementation", map[string]interface{}{"parent_uuid": root1})
	_ = nanostore.SetStatus(store, task2, "completed")

	task3, _ := store.Add("Testing Phase", map[string]interface{}{"parent_uuid": root2})
	deployTask, _ := store.Add("Deployment", map[string]interface{}{"parent_uuid": root2})
	_ = nanostore.SetStatus(store, deployTask, "completed")

	// Test: Filter by parent AND status
	results, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{
			"parent_uuid": root1,
			"status":      "completed",
		},
	})
	if err != nil {
		t.Fatalf("failed to filter: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 completed task in root1, got %d", len(results))
	}

	if results[0].UUID != task2 {
		t.Error("wrong task returned")
	}

	// Test: Filter by status AND search
	results, err = store.List(nanostore.ListOptions{
		Filters:        map[string]interface{}{"status": "pending"},
		FilterBySearch: "Phase",
	})
	if err != nil {
		t.Fatalf("failed to filter: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 pending tasks with 'Phase', got %d", len(results))
	}

	// Test: All three filters
	results, err = store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{
			"parent_uuid": root2,
			"status":      "pending",
		},
		FilterBySearch: "Test",
	})
	if err != nil {
		t.Fatalf("failed to filter: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result for combined filters, got %d", len(results))
	}

	if results[0].UUID != task3 {
		t.Error("wrong task returned for combined filters")
	}
}
