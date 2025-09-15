package nanostore_test

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestIsUUIDFormat(t *testing.T) {
	// Use reflection to access the unexported function for testing
	// We'll test this through the public API methods instead

	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid UUIDs
		{"valid uuid v4", "123e4567-e89b-12d3-a456-426614174000", true},
		{"valid uuid all lowercase", "abcdef12-3456-789a-bcde-f123456789ab", true},
		{"valid uuid all uppercase", "ABCDEF12-3456-789A-BCDE-F123456789AB", true},
		{"valid uuid mixed case", "AbCdEf12-3456-789a-BcDe-F123456789aB", true},
		{"valid uuid with zeros", "00000000-0000-0000-0000-000000000000", true},

		// Invalid formats - wrong length
		{"too short", "123e4567-e89b-12d3-a456-42661417400", false},
		{"too long", "123e4567-e89b-12d3-a456-4266141740000", false},
		{"empty string", "", false},

		// Invalid formats - wrong dash positions
		{"no dashes", "123e4567e89b12d3a456426614174000", false},
		{"wrong dash positions", "123e4567e-89b-12d3-a456-426614174000", false},
		{"extra dashes", "123e-4567-e89b-12d3-a456-426614174000", false},

		// Invalid characters
		{"invalid char g", "123e4567-e89b-12d3-a456-426614174g00", false},
		{"invalid char z", "z23e4567-e89b-12d3-a456-426614174000", false},
		{"with space", "123e4567-e89b-12d3-a456-42661417 000", false},
		{"with special char", "123e4567-e89b-12d3-a456-426614174@00", false},

		// User-facing IDs (should be false)
		{"simple numeric", "1", false},
		{"hierarchical", "1.2", false},
		{"with prefix", "c1", false},
		{"complex hierarchical", "1.2.c3", false},
		{"very complex", "hp1.2.c3.4", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test UUID detection indirectly through API behavior
			// We can infer the detection result based on whether methods
			// try to resolve the ID or use it directly
			store, err := nanostore.NewTestStore(":memory:")
			if err != nil {
				t.Fatalf("failed to create store: %v", err)
			}
			defer func() { _ = store.Close() }()

			// Try to update with this ID - if it's detected as UUID format,
			// it will be used directly (and fail with "document not found")
			// If detected as user-facing ID, it will try to resolve (and fail with resolution error)
			err = store.Update(tc.input, nanostore.UpdateRequest{})

			if tc.expected {
				// Should be treated as UUID - expect "document not found" error
				if err == nil || !strings.Contains(err.Error(), "document not found") {
					t.Errorf("expected UUID format to result in 'document not found' error, got: %v", err)
				}
			} else {
				// Should be treated as user-facing ID - expect resolution error
				if err == nil || !strings.Contains(err.Error(), "invalid ID") {
					t.Errorf("expected user-facing ID format to result in 'invalid ID' error, got: %v", err)
				}
			}
		})
	}
}

func TestUpdateWithSmartIDDetection(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test documents
	parentUUID, err := store.Add("Parent Document", nil)
	if err != nil {
		t.Fatalf("failed to add parent: %v", err)
	}

	childUUID, err := store.Add("Child Document", map[string]interface{}{"parent_uuid": parentUUID})
	if err != nil {
		t.Fatalf("failed to add child: %v", err)
	}

	// Set status for parent to get a completed ID
	err = nanostore.SetStatus(store, parentUUID, "completed")
	if err != nil {
		t.Fatalf("failed to set parent status: %v", err)
	}

	// List to get user-facing IDs
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	var parentUserID, childUserID string
	for _, doc := range docs {
		if doc.UUID == parentUUID {
			parentUserID = doc.UserFacingID
		}
		if doc.UUID == childUUID {
			childUserID = doc.UserFacingID
		}
	}

	testCases := []struct {
		name        string
		id          string
		expectError bool
		errorMsg    string
	}{
		{
			name: "update with parent UUID",
			id:   parentUUID,
		},
		{
			name: "update with child UUID",
			id:   childUUID,
		},
		{
			name: "update with parent user-facing ID",
			id:   parentUserID,
		},
		{
			name: "update with child user-facing ID",
			id:   childUserID,
		},
		{
			name:        "update with invalid UUID",
			id:          "invalid-uuid-format-123",
			expectError: true,
			errorMsg:    "invalid ID",
		},
		{
			name:        "update with invalid user-facing ID",
			id:          "999",
			expectError: true,
			errorMsg:    "invalid ID",
		},
		{
			name:        "update with malformed UUID",
			id:          "123e4567-e89b-12d3-a456-42661417400g",
			expectError: true,
			errorMsg:    "invalid ID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newTitle := "Updated Title"
			err := store.Update(tc.id, nanostore.UpdateRequest{
				Title: &newTitle,
			})

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("expected error containing '%s', got: %v", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDeleteWithSmartIDDetection(t *testing.T) {
	testCases := []struct {
		name        string
		setupFn     func(*testing.T) (nanostore.Store, string)
		expectError bool
		errorMsg    string
	}{
		{
			name: "delete with UUID",
			setupFn: func(t *testing.T) (nanostore.Store, string) {
				store, err := nanostore.NewTestStore(":memory:")
				if err != nil {
					t.Fatalf("failed to create store: %v", err)
				}
				docUUID, err := store.Add("Document", nil)
				if err != nil {
					t.Fatalf("failed to add document: %v", err)
				}
				return store, docUUID
			},
		},
		{
			name: "delete with user-facing ID",
			setupFn: func(t *testing.T) (nanostore.Store, string) {
				store, err := nanostore.NewTestStore(":memory:")
				if err != nil {
					t.Fatalf("failed to create store: %v", err)
				}
				_, err = store.Add("Document", nil)
				if err != nil {
					t.Fatalf("failed to add document: %v", err)
				}
				docs, err := store.List(nanostore.ListOptions{})
				if err != nil {
					t.Fatalf("failed to list documents: %v", err)
				}
				return store, docs[0].UserFacingID
			},
		},
		{
			name: "delete with invalid UUID",
			setupFn: func(t *testing.T) (nanostore.Store, string) {
				store, err := nanostore.NewTestStore(":memory:")
				if err != nil {
					t.Fatalf("failed to create store: %v", err)
				}
				return store, "00000000-0000-0000-0000-000000000001"
			},
			expectError: true,
			errorMsg:    "document not found",
		},
		{
			name: "delete with invalid user-facing ID",
			setupFn: func(t *testing.T) (nanostore.Store, string) {
				store, err := nanostore.NewTestStore(":memory:")
				if err != nil {
					t.Fatalf("failed to create store: %v", err)
				}
				return store, "999"
			},
			expectError: true,
			errorMsg:    "invalid ID",
		},
		{
			name: "delete with malformed ID",
			setupFn: func(t *testing.T) (nanostore.Store, string) {
				store, err := nanostore.NewTestStore(":memory:")
				if err != nil {
					t.Fatalf("failed to create store: %v", err)
				}
				return store, "not-a-real-id"
			},
			expectError: true,
			errorMsg:    "invalid ID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store, id := tc.setupFn(t)
			defer func() { _ = store.Close() }()

			err := store.Delete(id, false)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("expected error containing '%s', got: %v", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSetStatusWithSmartIDDetection(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test document
	docUUID, err := store.Add("Test Document", nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	t.Run("set status with UUID", func(t *testing.T) {
		err := nanostore.SetStatus(store, docUUID, "completed")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// Get fresh user-facing ID after status change
	docsAfterStatusChange, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents after status change: %v", err)
	}

	completedUserID := docsAfterStatusChange[0].UserFacingID

	t.Run("set status with user-facing ID", func(t *testing.T) {
		err := nanostore.SetStatus(store, completedUserID, "pending") // Reset to pending
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("set status with invalid UUID", func(t *testing.T) {
		err := nanostore.SetStatus(store, "00000000-0000-0000-0000-000000000001", "completed")
		if err == nil {
			t.Errorf("expected error but got none")
		} else if !strings.Contains(err.Error(), "document not found") {
			t.Errorf("expected error containing 'document not found', got: %v", err)
		}
	})

	t.Run("set status with invalid user-facing ID", func(t *testing.T) {
		err := nanostore.SetStatus(store, "999", "completed")
		if err == nil {
			t.Errorf("expected error but got none")
		} else if !strings.Contains(err.Error(), "invalid ID") {
			t.Errorf("expected error containing 'invalid ID', got: %v", err)
		}
	})
}

func TestMixedIDOperations(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a hierarchy using UUIDs
	parentUUID, err := store.Add("Parent", nil)
	if err != nil {
		t.Fatalf("failed to add parent: %v", err)
	}

	child1UUID, err := store.Add("Child 1", map[string]interface{}{"parent_uuid": parentUUID})
	if err != nil {
		t.Fatalf("failed to add child 1: %v", err)
	}

	// Get user-facing IDs
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	var parentUserID, child1UserID string
	for _, doc := range docs {
		if doc.UUID == parentUUID {
			parentUserID = doc.UserFacingID
		}
		if doc.UUID == child1UUID {
			child1UserID = doc.UserFacingID
		}
	}

	// Test mixed operations: some with UUIDs, some with user-facing IDs
	t.Run("update parent with user-facing ID", func(t *testing.T) {
		newTitle := "Updated Parent"
		err := store.Update(parentUserID, nanostore.UpdateRequest{
			Title: &newTitle,
		})
		if err != nil {
			t.Errorf("failed to update parent with user-facing ID: %v", err)
		}
	})

	t.Run("update child with UUID", func(t *testing.T) {
		newTitle := "Updated Child"
		err := store.Update(child1UUID, nanostore.UpdateRequest{
			Title: &newTitle,
		})
		if err != nil {
			t.Errorf("failed to update child with UUID: %v", err)
		}
	})

	t.Run("set status with user-facing ID", func(t *testing.T) {
		err := nanostore.SetStatus(store, child1UserID, "completed")
		if err != nil {
			t.Errorf("failed to set status with user-facing ID: %v", err)
		}
	})

	t.Run("add child using parent UUID", func(t *testing.T) {
		_, err := store.Add("Child 2", map[string]interface{}{"parent_uuid": parentUUID})
		if err != nil {
			t.Errorf("failed to add child using parent UUID: %v", err)
		}
	})

	// Verify everything worked
	finalDocs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list final documents: %v", err)
	}

	if len(finalDocs) != 3 {
		t.Errorf("expected 3 documents, got %d", len(finalDocs))
	}

	// Check that updates applied correctly
	for _, doc := range finalDocs {
		switch doc.UUID {
		case parentUUID:
			if doc.Title != "Updated Parent" {
				t.Errorf("parent title not updated correctly, got: %s", doc.Title)
			}
		case child1UUID:
			if doc.Title != "Updated Child" {
				t.Errorf("child 1 title not updated correctly, got: %s", doc.Title)
			}
			if doc.GetStatus() != "completed" {
				t.Errorf("child 1 status not updated correctly, got: %s", doc.GetStatus())
			}
		}
	}
}
