package nanostore_test

import (
	"fmt"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestDeepNesting(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a deep hierarchy: root -> level1 -> level2 -> level3 -> level4
	rootID, err := store.Add("Root", nil)
	if err != nil {
		t.Fatalf("failed to add root: %v", err)
	}

	currentParent := rootID
	expectedIDs := []string{"1", "1.1", "1.1.1", "1.1.1.1", "1.1.1.1.1"}
	ids := []string{rootID}

	for i := 1; i < 5; i++ {
		id, err := store.Add(fmt.Sprintf("Level %d", i), map[string]interface{}{"parent_uuid": currentParent})
		if err != nil {
			t.Fatalf("failed to add level %d: %v", i, err)
		}
		ids = append(ids, id)
		currentParent = id
	}

	// Verify the deep hierarchy
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(docs) != 5 {
		t.Fatalf("expected 5 documents, got %d", len(docs))
	}

	// Check IDs
	for i, doc := range docs {
		if doc.UserFacingID != expectedIDs[i] {
			t.Errorf("document %d: expected ID %s, got %s", i, expectedIDs[i], doc.UserFacingID)
		}
		if doc.UUID != ids[i] {
			t.Errorf("document %d: UUID mismatch", i)
		}
	}

	// Test resolving deep IDs
	for i, expectedID := range expectedIDs {
		resolvedID, err := store.ResolveUUID(expectedID)
		if err != nil {
			t.Errorf("failed to resolve ID %s: %v", expectedID, err)
		}
		if resolvedID != ids[i] {
			t.Errorf("resolved ID mismatch for %s", expectedID)
		}
	}
}

func TestMixedStatusHierarchy(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create mixed hierarchy:
	// 1 (pending)
	//   1.1 (pending)
	//   1.2 (completed -> 1.c1)
	//   1.3 (pending)
	//   1.4 (completed -> 1.c2)
	// 2 (completed -> c1)
	//   c1.1 (pending)
	//   c1.c1 (completed)

	root1, _ := store.Add("Root 1", nil)
	child1, _ := store.Add("Child 1.1", map[string]interface{}{"parent_uuid": root1})
	child2, _ := store.Add("Child 1.2", map[string]interface{}{"parent_uuid": root1})
	child3, _ := store.Add("Child 1.3", map[string]interface{}{"parent_uuid": root1})
	child4, _ := store.Add("Child 1.4", map[string]interface{}{"parent_uuid": root1})

	_ = nanostore.TestSetStatusUpdate(store, child2, "completed")
	_ = nanostore.TestSetStatusUpdate(store, child4, "completed")

	root2, _ := store.Add("Root 2", nil)
	_ = nanostore.TestSetStatusUpdate(store, root2, "completed")

	child5, _ := store.Add("Child c1.1", map[string]interface{}{"parent_uuid": root2})
	child6, _ := store.Add("Child c1.c1", map[string]interface{}{"parent_uuid": root2})
	_ = nanostore.TestSetStatusUpdate(store, child6, "completed")

	// Expected IDs
	expected := map[string]string{
		root1:  "1",
		child1: "1.1",
		child2: "1.c1",
		child3: "1.2", // Note: this is 1.2 because pending are numbered separately
		child4: "1.c2",
		root2:  "c1",
		child5: "c1.1",
		child6: "c1.c1",
	}

	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	// Verify all IDs
	for _, doc := range docs {
		expectedID, ok := expected[doc.UUID]
		if !ok {
			t.Errorf("unexpected document: %s", doc.UUID)
			continue
		}
		if doc.UserFacingID != expectedID {
			t.Errorf("ID mismatch for %s: expected %s, got %s", doc.Title, expectedID, doc.UserFacingID)
		}
	}
}

func TestSiblingNumbering(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create parent
	parent, _ := store.Add("Parent", nil)

	// Add 20 pending children
	pendingIDs := make([]string, 20)
	for i := 0; i < 20; i++ {
		id, err := store.Add(fmt.Sprintf("Pending Child %d", i+1), map[string]interface{}{"parent_uuid": parent})
		if err != nil {
			t.Fatalf("failed to add pending child %d: %v", i+1, err)
		}
		pendingIDs[i] = id
	}

	// Add 15 completed children
	completedIDs := make([]string, 15)
	for i := 0; i < 15; i++ {
		id, err := store.Add(fmt.Sprintf("Completed Child %d", i+1), map[string]interface{}{"parent_uuid": parent})
		if err != nil {
			t.Fatalf("failed to add completed child %d: %v", i+1, err)
		}
		_ = nanostore.TestSetStatusUpdate(store, id, "completed")
		completedIDs[i] = id
	}

	// Verify numbering
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	// Check parent
	parentFound := false
	for _, doc := range docs {
		if doc.UUID == parent {
			if doc.UserFacingID != "1" {
				t.Errorf("parent has wrong ID: %s", doc.UserFacingID)
			}
			parentFound = true
			break
		}
	}
	if !parentFound {
		t.Error("parent not found")
	}

	// Check children have correct numbering
	for i, id := range pendingIDs {
		expectedID := fmt.Sprintf("1.%d", i+1)
		found := false
		for _, doc := range docs {
			if doc.UUID == id {
				if doc.UserFacingID != expectedID {
					t.Errorf("pending child %d: expected ID %s, got %s", i+1, expectedID, doc.UserFacingID)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pending child %d not found", i+1)
		}
	}

	for i, id := range completedIDs {
		expectedID := fmt.Sprintf("1.c%d", i+1)
		found := false
		for _, doc := range docs {
			if doc.UUID == id {
				if doc.UserFacingID != expectedID {
					t.Errorf("completed child %d: expected ID %s, got %s", i+1, expectedID, doc.UserFacingID)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("completed child %d not found", i+1)
		}
	}
}

func TestDeletedParentHandling(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a hierarchy
	parentID, err := store.Add("Parent", nil)
	if err != nil {
		t.Fatalf("failed to add parent: %v", err)
	}

	childID, err := store.Add("Child", map[string]interface{}{"parent_uuid": parentID})
	if err != nil {
		t.Fatalf("failed to add child: %v", err)
	}

	grandchildID, err := store.Add("Grandchild", map[string]interface{}{"parent_uuid": childID})
	if err != nil {
		t.Fatalf("failed to add grandchild: %v", err)
	}

	// Delete the middle node (child) with cascade
	err = store.Delete(childID, true)
	if err != nil {
		t.Errorf("failed to delete child with cascade: %v", err)
	}

	// Verify parent still exists but child and grandchild are gone
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d", len(docs))
	}

	if docs[0].UUID != parentID {
		t.Error("parent not found after cascade delete")
	}

	// Verify child and grandchild are gone
	for _, doc := range docs {
		if doc.UUID == childID || doc.UUID == grandchildID {
			t.Errorf("found deleted document: %s", doc.Title)
		}
	}

	// Test that we can't resolve the deleted documents' IDs
	_, err = store.ResolveUUID("1.1")
	if err == nil {
		t.Error("expected error when resolving deleted child ID")
	}

	_, err = store.ResolveUUID("1.1.1")
	if err == nil {
		t.Error("expected error when resolving deleted grandchild ID")
	}
}

func TestResolveComplexIDs(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create complex hierarchy
	root, _ := store.Add("Root", nil)

	// Add children with mixed status
	var lastParent string
	for i := 0; i < 3; i++ {
		child, _ := store.Add(fmt.Sprintf("Child %d", i), map[string]interface{}{"parent_uuid": root})
		if i == 1 {
			_ = nanostore.TestSetStatusUpdate(store, child, "completed")
			lastParent = child
		}
	}

	// Add grandchildren to the completed child
	for i := 0; i < 2; i++ {
		gc, _ := store.Add(fmt.Sprintf("Grandchild %d", i), map[string]interface{}{"parent_uuid": lastParent})
		if i == 0 {
			_ = nanostore.TestSetStatusUpdate(store, gc, "completed")
		}
	}

	// Test resolving various IDs
	testCases := []struct {
		id         string
		shouldFail bool
	}{
		{"1", false},       // root
		{"1.1", false},     // first child
		{"1.c1", false},    // completed child
		{"1.2", false},     // third child (second pending)
		{"1.c1.c1", false}, // completed grandchild
		{"1.c1.1", false},  // pending grandchild
		{"1.99", true},     // non-existent
		{"1.c99", true},    // non-existent completed
		{"2", true},        // non-existent root
		{"1.c1.99", true},  // non-existent grandchild
	}

	for _, tc := range testCases {
		_, err := store.ResolveUUID(tc.id)
		if tc.shouldFail && err == nil {
			t.Errorf("expected error for ID %s, but got none", tc.id)
		}
		if !tc.shouldFail && err != nil {
			t.Errorf("unexpected error for ID %s: %v", tc.id, err)
		}
	}
}
