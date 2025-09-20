package nanostore_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)


import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/testutil"
)

func TestRefFieldSmartIDResolutionMigrated(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("add with SimpleID as parent_id", func(t *testing.T) {
		// Get the parent's simple ID
		parentDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.PersonalRoot.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(parentDocs) != 1 {
			t.Fatalf("expected 1 document, got %d", len(parentDocs))
		}
		parentSimpleID := parentDocs[0].SimpleID

		// Create a child using parent's SimpleID
		childUUID, err := store.Add("New Child Task", map[string]interface{}{
			"status":    "pending",
			"priority":  "medium",
			"parent_id": parentSimpleID, // Using SimpleID instead of UUID
		})
		if err != nil {
			t.Fatalf("failed to add child with parent SimpleID: %v", err)
		}

		// Verify the child has the correct parent UUID stored
		childDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"uuid": childUUID},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(childDocs) != 1 {
			t.Fatalf("expected 1 child document, got %d", len(childDocs))
		}
		if childDocs[0].Dimensions["parent_id"] != universe.PersonalRoot.UUID {
			t.Errorf("expected parent_id to be resolved to UUID %q, got %q",
				universe.PersonalRoot.UUID, childDocs[0].Dimensions["parent_id"])
		}
	})

	// Reload to ensure clean state
	store, universe = testutil.LoadUniverse(t)

	t.Run("update with SimpleID as parent_id", func(t *testing.T) {
		// Get a document that currently has no parent
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.EmptyTitle.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 {
			t.Fatal("document not found")
		}

		// Get WorkRoot's SimpleID
		workDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.WorkRoot.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(workDocs) != 1 {
			t.Fatal("work root not found")
		}
		workSimpleID := workDocs[0].SimpleID

		// Update the document to have WorkRoot as parent using SimpleID
		err = store.Update(universe.EmptyTitle.UUID, nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{
				"parent_id": workSimpleID,
			},
		})
		if err != nil {
			t.Fatalf("failed to update with parent SimpleID: %v", err)
		}

		// Verify the update
		updatedDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"uuid": universe.EmptyTitle.UUID},
		})
		if err != nil {
			t.Fatal(err)
		}
		if updatedDocs[0].Dimensions["parent_id"] != universe.WorkRoot.UUID {
			t.Errorf("expected parent_id to be resolved to UUID %q, got %q",
				universe.WorkRoot.UUID, updatedDocs[0].Dimensions["parent_id"])
		}
	})

	// Reload for next test
	store, universe = testutil.LoadUniverse(t)

	t.Run("UpdateByDimension with SimpleID as parent_id", func(t *testing.T) {
		// Get WorkRoot's SimpleID to move documents there
		workDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.WorkRoot.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(workDocs) != 1 {
			t.Fatal("work root not found")
		}
		workSimpleID := workDocs[0].SimpleID

		// Update all pending tasks under PersonalRoot to be done and under WorkRoot
		count, err := store.UpdateByDimension(
			map[string]interface{}{
				"status":    "pending",
				"parent_id": universe.PersonalRoot.UUID,
			},
			nanostore.UpdateRequest{
				Dimensions: map[string]interface{}{
					"parent_id": workSimpleID,
					"status":    "done",
				},
			},
		)
		if err != nil {
			t.Fatalf("UpdateByDimension failed: %v", err)
		}
		if count == 0 {
			t.Error("expected to update at least 1 document, updated 0")
		}

		// Verify some documents were moved
		movedDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"parent_id": universe.WorkRoot.UUID,
				"status":    "done",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(movedDocs) < count {
			t.Errorf("expected at least %d documents to be moved, found %d", count, len(movedDocs))
		}
	})

	t.Run("add with non-existent SimpleID as parent_id", func(t *testing.T) {
		// Try to add a document with a non-existent SimpleID
		// This should store the value as-is since it can't be resolved
		childUUID, err := store.Add("Orphan Child", map[string]interface{}{
			"status":    "pending",
			"priority":  "low",
			"parent_id": "999", // Non-existent SimpleID
		})
		if err != nil {
			t.Fatalf("failed to add child: %v", err)
		}

		// Verify the invalid ID is stored as-is
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"uuid": childUUID},
		})
		if err != nil {
			t.Fatal(err)
		}
		if docs[0].Dimensions["parent_id"] != "999" {
			t.Errorf("expected parent_id to be stored as-is '999', got %q",
				docs[0].Dimensions["parent_id"])
		}
	})

	t.Run("mixed UUID and SimpleID updates", func(t *testing.T) {
		// Create a new parent
		parentUUID, err := store.Add("Mixed Test Parent", map[string]interface{}{
			"status":   "active",
			"priority": "high",
		})
		if err != nil {
			t.Fatal(err)
		}

		// Get its SimpleID
		parentDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": parentUUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		parentSimpleID := parentDocs[0].SimpleID

		// Create two children - one with UUID, one with SimpleID
		child1UUID, err := store.Add("Child with UUID ref", map[string]interface{}{
			"status":    "pending",
			"parent_id": parentUUID, // Using UUID
		})
		if err != nil {
			t.Fatal(err)
		}

		child2UUID, err := store.Add("Child with SimpleID ref", map[string]interface{}{
			"status":    "pending",
			"parent_id": parentSimpleID, // Using SimpleID
		})
		if err != nil {
			t.Fatal(err)
		}

		// Verify both children have the same parent UUID stored
		children, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"parent_id": parentUUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		foundChild1 := false
		foundChild2 := false
		for _, child := range children {
			if child.UUID == child1UUID {
				foundChild1 = true
			}
			if child.UUID == child2UUID {
				foundChild2 = true
			}
		}

		if !foundChild1 {
			t.Error("child created with UUID ref not found under parent")
		}
		if !foundChild2 {
			t.Error("child created with SimpleID ref not found under parent")
		}
	})
}
