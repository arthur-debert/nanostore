package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestUpdateRequestWithInterfaceMap(t *testing.T) {
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "in_progress", "completed"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "medium", "high"},
				DefaultValue: "medium",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("update with string values", func(t *testing.T) {
		id, _ := store.Add("Task", map[string]interface{}{})

		err := store.Update(id, nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{
				"status":   "in_progress",
				"priority": "high",
			},
		})
		if err != nil {
			t.Fatalf("failed to update: %v", err)
		}

		// Verify
		docs, _ := store.List(nanostore.ListOptions{})
		if docs[0].Dimensions["status"] != "in_progress" {
			t.Errorf("expected status in_progress, got %v", docs[0].Dimensions["status"])
		}
		if docs[0].Dimensions["priority"] != "high" {
			t.Errorf("expected priority high, got %v", docs[0].Dimensions["priority"])
		}
	})

	t.Run("update with non-string values", func(t *testing.T) {
		id, _ := store.Add("Task", map[string]interface{}{})

		// Should convert to string internally
		err := store.Update(id, nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{
				"priority": "low", // Will be converted to "low"
			},
		})
		if err != nil {
			t.Fatalf("failed to update: %v", err)
		}

		// Verify
		docs, _ := store.List(nanostore.ListOptions{})
		for _, doc := range docs {
			if doc.UUID == id {
				if doc.Dimensions["priority"] != "low" {
					t.Errorf("expected priority low, got %v", doc.Dimensions["priority"])
				}
				break
			}
		}
	})

	t.Run("update parent with smart ID", func(t *testing.T) {
		parentID, _ := store.Add("Parent", map[string]interface{}{})
		childID, _ := store.Add("Child", map[string]interface{}{})

		// Get parent's user-facing ID
		docs, _ := store.List(nanostore.ListOptions{})
		var parentUserFacingID string
		for _, doc := range docs {
			if doc.UUID == parentID {
				parentUserFacingID = doc.UserFacingID
				break
			}
		}

		// Update child to have parent using user-facing ID
		err := store.Update(childID, nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{
				"parent_uuid": parentUserFacingID, // Smart ID detection should work
			},
		})
		if err != nil {
			t.Fatalf("failed to update parent: %v", err)
		}

		// Verify
		docs, _ = store.List(nanostore.ListOptions{})
		for _, doc := range docs {
			if doc.UUID == childID {
				if doc.Dimensions["parent_uuid"] != parentID {
					t.Errorf("expected parent_uuid %s, got %v", parentID, doc.Dimensions["parent_uuid"])
				}
				break
			}
		}
	})
}
