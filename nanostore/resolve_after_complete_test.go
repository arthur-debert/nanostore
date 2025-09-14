package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestResolveAfterComplete(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create three todos
	id1, _ := store.Add("First", nil, nil)
	id2, _ := store.Add("Second", nil, nil)
	id3, _ := store.Add("Third", nil, nil)

	// Initial state - all should resolve
	for i, id := range []string{"1", "2", "3"} {
		uuid, err := store.ResolveUUID(id)
		if err != nil {
			t.Errorf("failed to resolve ID %s initially: %v", id, err)
		}
		expectedUUIDs := []string{id1, id2, id3}
		if uuid != expectedUUIDs[i] {
			t.Errorf("ID %s resolved to wrong UUID", id)
		}
	}

	// Complete the first one
	err = nanostore.SetStatus(store, id1, "completed")
	if err != nil {
		t.Fatalf("failed to complete first todo: %v", err)
	}

	// Now IDs should have shifted:
	// - "1" should resolve to id2 (Second)
	// - "2" should resolve to id3 (Third)
	// - "c1" should resolve to id1 (First, completed)
	// - "3" should NOT resolve (doesn't exist anymore)

	testCases := []struct {
		userFacingID string
		expectedUUID string
		shouldFail   bool
	}{
		{"1", id2, false},
		{"2", id3, false},
		{"c1", id1, false},
		{"3", "", true},
	}

	for _, tc := range testCases {
		uuid, err := store.ResolveUUID(tc.userFacingID)
		if tc.shouldFail {
			if err == nil {
				t.Errorf("expected ID %s to fail resolution but it succeeded with UUID %s", tc.userFacingID, uuid)
			}
		} else {
			if err != nil {
				t.Errorf("failed to resolve ID %s: %v", tc.userFacingID, err)
			} else if uuid != tc.expectedUUID {
				t.Errorf("ID %s resolved to %s, expected %s", tc.userFacingID, uuid, tc.expectedUUID)
			}
		}
	}

	// List to verify the actual IDs
	docs, _ := store.List(nanostore.ListOptions{})
	t.Logf("After completing first todo:")
	for _, doc := range docs {
		t.Logf("  %s: %s (UUID: %s, Status: %s)", doc.UserFacingID, doc.Title, doc.UUID, doc.GetStatus())
	}
}

func TestCompleteMultiple(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create three todos
	id1, _ := store.Add("First", nil, nil)
	_, _ = store.Add("Second", nil, nil) // was id2
	_, _ = store.Add("Third", nil, nil)  // was id3

	// Complete first one
	err = nanostore.SetStatus(store, id1, "completed")
	if err != nil {
		t.Fatalf("failed to complete first todo: %v", err)
	}

	// At this point:
	// - "1" is Second (was id2)
	// - "2" is Third (was id3)
	// - "c1" is First (id1)

	// Try to complete "1" and "2" (which should be Second and Third)
	// This simulates the command: too complete 1 2
	uuids := []string{}
	for _, userID := range []string{"1", "2"} {
		uuid, err := store.ResolveUUID(userID)
		if err != nil {
			t.Errorf("failed to resolve ID %s: %v", userID, err)
			continue
		}
		uuids = append(uuids, uuid)
	}

	// Complete them
	for _, uuid := range uuids {
		err = nanostore.SetStatus(store, uuid, "completed")
		if err != nil {
			t.Errorf("failed to complete UUID %s: %v", uuid, err)
		}
	}

	// List all to see final state
	docs, _ := store.List(nanostore.ListOptions{})
	t.Logf("After completing all:")
	for _, doc := range docs {
		t.Logf("  %s: %s (UUID: %s, Status: %s)", doc.UserFacingID, doc.Title, doc.UUID, doc.GetStatus())
	}

	// All should be completed
	if len(docs) != 3 {
		t.Errorf("expected 3 todos, got %d", len(docs))
	}
	for _, doc := range docs {
		if doc.GetStatus() != "completed" {
			t.Errorf("expected %s to be completed, but status is %s", doc.Title, doc.GetStatus())
		}
	}
}
