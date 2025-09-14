package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestResolveUUID(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test documents
	// Root documents
	id1, _ := store.Add("First", nil)
	id2, _ := store.Add("Second", nil)
	id3, _ := store.Add("Third", nil)

	// Mark one as completed
	_ = nanostore.SetStatus(store, id3, "completed")

	// Test cases
	tests := []struct {
		userFacingID string
		expectedUUID string
	}{
		{"1", id1},
		{"2", id2},
		{"c1", id3},
	}

	for _, tc := range tests {
		uuid, err := store.ResolveUUID(tc.userFacingID)
		if err != nil {
			t.Errorf("failed to resolve ID %s: %v", tc.userFacingID, err)
			continue
		}

		if uuid != tc.expectedUUID {
			t.Errorf("for ID %s, expected UUID %s, got %s", tc.userFacingID, tc.expectedUUID, uuid)
		}
	}
}

func TestResolveHierarchicalUUID(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create hierarchical structure
	parentID, _ := store.Add("Parent", nil)
	child1ID, _ := store.Add("Child 1", map[string]interface{}{"parent_uuid": parentID})
	child2ID, _ := store.Add("Child 2", map[string]interface{}{"parent_uuid": parentID})
	child3ID, _ := store.Add("Child 3", map[string]interface{}{"parent_uuid": parentID})

	// Mark one child as completed
	_ = nanostore.SetStatus(store, child3ID, "completed")

	// Nested child
	grandchildID, _ := store.Add("Grandchild", map[string]interface{}{"parent_uuid": child1ID})

	// Test cases
	tests := []struct {
		userFacingID string
		expectedUUID string
	}{
		{"1", parentID},
		{"1.1", child1ID},
		{"1.2", child2ID},
		{"1.c1", child3ID},
		{"1.1.1", grandchildID},
	}

	for _, tc := range tests {
		uuid, err := store.ResolveUUID(tc.userFacingID)
		if err != nil {
			t.Errorf("failed to resolve ID %s: %v", tc.userFacingID, err)
			continue
		}

		if uuid != tc.expectedUUID {
			t.Errorf("for ID %s, expected UUID %s, got %s", tc.userFacingID, tc.expectedUUID, uuid)
		}
	}
}

func TestResolveInvalidID(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Test invalid IDs
	invalidIDs := []string{
		"999",   // Non-existent
		"abc",   // Invalid format
		"1.999", // Non-existent child
		"",      // Empty
		"c",     // Missing number
	}

	for _, id := range invalidIDs {
		_, err := store.ResolveUUID(id)
		if err == nil {
			t.Errorf("expected error for invalid ID %q, but got none", id)
		}
	}
}
