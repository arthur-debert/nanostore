package api_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore/api"
	"github.com/arthur-debert/nanostore/types"
)

func TestTypedStoreGetDimensions(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create test data with standard struct
	typedUUID, err := store.Create("Standard Task", &TodoItem{
		Status:   "active",
		Priority: "high",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create test data with custom fields using AddRaw
	rawUUID, err := store.AddRaw("Task with Custom Data", map[string]interface{}{
		"status":         "pending",
		"priority":       "medium",
		"activity":       "active",
		"_data.assignee": "alice",
		"_data.tags":     "urgent,backend",
		"_data.estimate": 5,
		"_data.metadata": map[string]interface{}{
			"created_by": "system",
			"version":    "1.0",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get SimpleIDs for testing
	allTasks, err := store.List(types.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var typedSimpleID string
	for _, task := range allTasks {
		if task.UUID == typedUUID {
			typedSimpleID = task.SimpleID
		}
	}

	t.Run("GetDimensionsFromTypedDocument", func(t *testing.T) {
		dimensions, err := store.GetDimensions(typedUUID)
		if err != nil {
			t.Fatalf("failed to get dimensions by UUID: %v", err)
		}

		if dimensions == nil {
			t.Fatal("expected dimensions map, got nil")
		}

		// Verify standard dimensions
		if dimensions["status"] != "active" {
			t.Errorf("expected status 'active', got %v", dimensions["status"])
		}
		if dimensions["priority"] != "high" {
			t.Errorf("expected priority 'high', got %v", dimensions["priority"])
		}
		if dimensions["activity"] != "active" {
			t.Errorf("expected activity 'active', got %v", dimensions["activity"])
		}

		// Should not have custom data fields
		if _, exists := dimensions["_data.assignee"]; exists {
			t.Error("typed document should not have custom _data.assignee field")
		}
	})

	t.Run("GetDimensionsBySimpleID", func(t *testing.T) {
		dimensions, err := store.GetDimensions(typedSimpleID)
		if err != nil {
			t.Fatalf("failed to get dimensions by SimpleID: %v", err)
		}

		if dimensions == nil {
			t.Fatal("expected dimensions map, got nil")
		}

		// Should have the same dimensions as UUID access
		if dimensions["status"] != "active" {
			t.Errorf("expected status 'active', got %v", dimensions["status"])
		}
		if dimensions["priority"] != "high" {
			t.Errorf("expected priority 'high', got %v", dimensions["priority"])
		}
	})

	t.Run("GetDimensionsWithCustomData", func(t *testing.T) {
		dimensions, err := store.GetDimensions(rawUUID)
		if err != nil {
			t.Fatalf("failed to get dimensions for raw document: %v", err)
		}

		if dimensions == nil {
			t.Fatal("expected dimensions map, got nil")
		}

		// Verify standard dimensions
		if dimensions["status"] != "pending" {
			t.Errorf("expected status 'pending', got %v", dimensions["status"])
		}
		if dimensions["priority"] != "medium" {
			t.Errorf("expected priority 'medium', got %v", dimensions["priority"])
		}

		// Verify custom data fields are accessible
		if dimensions["_data.assignee"] != "alice" {
			t.Errorf("expected _data.assignee 'alice', got %v", dimensions["_data.assignee"])
		}
		if dimensions["_data.tags"] != "urgent,backend" {
			t.Errorf("expected _data.tags 'urgent,backend', got %v", dimensions["_data.tags"])
		}
		if dimensions["_data.estimate"] != 5 {
			t.Errorf("expected _data.estimate 5, got %v", dimensions["_data.estimate"])
		}

		// Verify nested custom data
		metadata, ok := dimensions["_data.metadata"]
		if !ok {
			t.Error("expected _data.metadata to exist")
		} else {
			metaMap, ok := metadata.(map[string]interface{})
			if !ok {
				t.Errorf("expected _data.metadata to be map, got %T", metadata)
			} else {
				if metaMap["created_by"] != "system" {
					t.Errorf("expected created_by 'system', got %v", metaMap["created_by"])
				}
				if metaMap["version"] != "1.0" {
					t.Errorf("expected version '1.0', got %v", metaMap["version"])
				}
			}
		}
	})

	t.Run("GetDimensionsNonExistentDocument", func(t *testing.T) {
		_, err := store.GetDimensions("non-existent-uuid")
		if err == nil {
			t.Error("expected error for non-existent document")
		}
		if err.Error() != "document with ID 'non-existent-uuid' not found" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("GetDimensionsEmptyID", func(t *testing.T) {
		_, err := store.GetDimensions("")
		if err == nil {
			t.Error("expected error for empty ID")
		}
	})

	t.Run("GetDimensionsInvalidSimpleID", func(t *testing.T) {
		_, err := store.GetDimensions("999")
		if err == nil {
			t.Error("expected error for invalid SimpleID")
		}
	})

	t.Run("CompareDimensionsWithTypedAccess", func(t *testing.T) {
		// Get dimensions via GetDimensions
		dimensions, err := store.GetDimensions(typedUUID)
		if err != nil {
			t.Fatalf("failed to get dimensions: %v", err)
		}

		// Get the same document via typed access
		typedDoc, err := store.Get(typedUUID)
		if err != nil {
			t.Fatalf("failed to get typed document: %v", err)
		}

		// Verify dimensions match typed fields
		if dimensions["status"] != typedDoc.Status {
			t.Errorf("status mismatch: dimensions %v vs typed %q", dimensions["status"], typedDoc.Status)
		}
		if dimensions["priority"] != typedDoc.Priority {
			t.Errorf("priority mismatch: dimensions %v vs typed %q", dimensions["priority"], typedDoc.Priority)
		}
		if dimensions["activity"] != typedDoc.Activity {
			t.Errorf("activity mismatch: dimensions %v vs typed %q", dimensions["activity"], typedDoc.Activity)
		}
	})

	t.Run("GetDimensionsWithHierarchy", func(t *testing.T) {
		// Create parent-child relationship
		parentUUID, err := store.Create("Parent Task", &TodoItem{
			Status:   "active",
			Priority: "high",
			Activity: "active",
		})
		if err != nil {
			t.Fatal(err)
		}

		childUUID, err := store.Create("Child Task", &TodoItem{
			Status:   "pending",
			Priority: "low",
			Activity: "active",
			ParentID: parentUUID,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Get child dimensions
		childDimensions, err := store.GetDimensions(childUUID)
		if err != nil {
			t.Fatalf("failed to get child dimensions: %v", err)
		}

		// Verify parent relationship is in dimensions
		if childDimensions["parent_id"] != parentUUID {
			t.Errorf("expected parent_id %q, got %v", parentUUID, childDimensions["parent_id"])
		}

		// Get parent dimensions
		parentDimensions, err := store.GetDimensions(parentUUID)
		if err != nil {
			t.Fatalf("failed to get parent dimensions: %v", err)
		}

		// Parent should not have parent_id (or it should be empty/nil)
		if parentID, exists := parentDimensions["parent_id"]; exists && parentID != "" && parentID != nil {
			t.Errorf("parent should not have parent_id, got %v", parentID)
		}
	})
}

func TestTypedStoreGetDimensionsModification(t *testing.T) {
	// Test that modifying the returned dimensions map doesn't affect the store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	uuid, err := store.Create("Test Task", &TodoItem{
		Status:   "active",
		Priority: "high",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("ModifyReturnedDimensions", func(t *testing.T) {
		// Get dimensions
		dimensions, err := store.GetDimensions(uuid)
		if err != nil {
			t.Fatalf("failed to get dimensions: %v", err)
		}

		originalStatus := dimensions["status"]

		// Modify the returned map
		dimensions["status"] = "modified"
		dimensions["new_field"] = "new_value"

		// Get dimensions again - should be unchanged
		freshDimensions, err := store.GetDimensions(uuid)
		if err != nil {
			t.Fatalf("failed to get fresh dimensions: %v", err)
		}

		if freshDimensions["status"] != originalStatus {
			t.Errorf("dimensions were modified in store: expected %v, got %v", originalStatus, freshDimensions["status"])
		}

		if _, exists := freshDimensions["new_field"]; exists {
			t.Error("new field should not exist in fresh dimensions")
		}
	})
}
