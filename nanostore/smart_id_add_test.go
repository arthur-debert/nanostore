package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestAddWithSmartIDSupport(t *testing.T) {
	config := nanostore.Config{
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
	}

	t.Run("add child with parent UUID", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Add parent
		parentUUID, err := store.Add("Parent Task", map[string]interface{}{})
		if err != nil {
			t.Fatalf("failed to add parent: %v", err)
		}

		// Add child using parent's UUID
		childUUID, err := store.Add("Child Task", map[string]interface{}{
			"parent_uuid": parentUUID,
		})
		if err != nil {
			t.Fatalf("failed to add child with parent UUID: %v", err)
		}

		// Verify the child was created
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		if len(docs) != 2 {
			t.Errorf("expected 2 documents, got %d", len(docs))
		}

		// Find the child document
		for _, doc := range docs {
			if doc.UUID == childUUID {
				if parent, ok := doc.Dimensions["parent_uuid"].(string); !ok || parent != parentUUID {
					t.Errorf("expected parent_uuid to be %s, got %v", parentUUID, doc.Dimensions["parent_uuid"])
				}
				break
			}
		}
	})

	t.Run("add child with parent user-facing ID", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Add parent
		parentUUID, err := store.Add("Parent Task", map[string]interface{}{})
		if err != nil {
			t.Fatalf("failed to add parent: %v", err)
		}

		// Get parent's user-facing ID
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		var parentUserFacingID string
		for _, doc := range docs {
			if doc.UUID == parentUUID {
				parentUserFacingID = doc.UserFacingID
				break
			}
		}

		if parentUserFacingID == "" {
			t.Fatal("could not find parent's user-facing ID")
		}

		// Add child using parent's user-facing ID
		childUUID, err := store.Add("Child Task", map[string]interface{}{
			"parent_uuid": parentUserFacingID,
		})
		if err != nil {
			t.Fatalf("failed to add child with parent user-facing ID: %v", err)
		}

		// Verify the child was created with correct parent
		docs, err = store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		// Find the child document
		for _, doc := range docs {
			if doc.UUID == childUUID {
				if parent, ok := doc.Dimensions["parent_uuid"].(string); !ok || parent != parentUUID {
					t.Errorf("expected parent_uuid to be %s, got %v", parentUUID, doc.Dimensions["parent_uuid"])
				}
				break
			}
		}
	})

	t.Run("add with invalid parent ID", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Try to add with non-existent parent ID
		_, err = store.Add("Child Task", map[string]interface{}{
			"parent_uuid": "invalid-id",
		})
		if err == nil {
			t.Error("expected error for invalid parent ID, got nil")
		}
	})

	t.Run("add grandchild with user-facing ID", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create hierarchy: parent -> child -> grandchild
		parentUUID, _ := store.Add("Parent", map[string]interface{}{})

		// Add child using parent UUID
		childUUID, _ := store.Add("Child", map[string]interface{}{
			"parent_uuid": parentUUID,
		})

		// Get child's user-facing ID
		docs, _ := store.List(nanostore.ListOptions{})
		var childUserFacingID string
		for _, doc := range docs {
			if doc.UUID == childUUID {
				childUserFacingID = doc.UserFacingID
				break
			}
		}

		// Add grandchild using child's user-facing ID
		grandchildUUID, err := store.Add("Grandchild", map[string]interface{}{
			"parent_uuid": childUserFacingID,
		})
		if err != nil {
			t.Fatalf("failed to add grandchild: %v", err)
		}

		// Verify grandchild has correct parent
		docs, _ = store.List(nanostore.ListOptions{})
		for _, doc := range docs {
			if doc.UUID == grandchildUUID {
				if parent, ok := doc.Dimensions["parent_uuid"].(string); !ok || parent != childUUID {
					t.Errorf("expected parent_uuid to be %s, got %v", childUUID, doc.Dimensions["parent_uuid"])
				}
				// Verify it's hierarchical ID (e.g., "1.1.1")
				if doc.UserFacingID != "1.1.1" {
					t.Errorf("expected grandchild user-facing ID to be 1.1.1, got %s", doc.UserFacingID)
				}
				break
			}
		}
	})
}

func TestResolveUUIDWithSmartIDSupport(t *testing.T) {
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "completed"},
				Prefixes:     map[string]string{"completed": "c"},
				DefaultValue: "pending",
			},
		},
	}

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add a document
	uuid, err := store.Add("Test Task", map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Get its user-facing ID
	docs, _ := store.List(nanostore.ListOptions{})
	var userFacingID string
	for _, doc := range docs {
		if doc.UUID == uuid {
			userFacingID = doc.UserFacingID
			break
		}
	}

	t.Run("resolve with UUID returns UUID", func(t *testing.T) {
		resolved, err := store.ResolveUUID(uuid)
		if err != nil {
			t.Fatalf("failed to resolve UUID: %v", err)
		}
		if resolved != uuid {
			t.Errorf("expected %s, got %s", uuid, resolved)
		}
	})

	t.Run("resolve with user-facing ID returns UUID", func(t *testing.T) {
		resolved, err := store.ResolveUUID(userFacingID)
		if err != nil {
			t.Fatalf("failed to resolve user-facing ID: %v", err)
		}
		if resolved != uuid {
			t.Errorf("expected %s, got %s", uuid, resolved)
		}
	})

	t.Run("resolve with invalid ID returns error", func(t *testing.T) {
		_, err := store.ResolveUUID("invalid-id")
		if err == nil {
			t.Error("expected error for invalid ID, got nil")
		}
	})
}
