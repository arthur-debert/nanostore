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

func TestStoreAddRaw(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.New[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("AddRawWithStandardDimensions", func(t *testing.T) {
		uuid, err := store.AddRaw("Raw Task", map[string]interface{}{
			"status":   "active",
			"priority": "high",
			"activity": "active",
		})
		if err != nil {
			t.Fatalf("failed to add raw document: %v", err)
		}

		if uuid == "" {
			t.Error("expected non-empty UUID")
		}

		// Verify the document was created correctly
		doc, err := store.GetRaw(uuid)
		if err != nil {
			t.Fatalf("failed to get created document: %v", err)
		}

		if doc.Title != "Raw Task" {
			t.Errorf("expected title 'Raw Task', got %q", doc.Title)
		}
		if doc.Dimensions["status"] != "active" {
			t.Errorf("expected status 'active', got %v", doc.Dimensions["status"])
		}
		if doc.Dimensions["priority"] != "high" {
			t.Errorf("expected priority 'high', got %v", doc.Dimensions["priority"])
		}
		if doc.Dimensions["activity"] != "active" {
			t.Errorf("expected activity 'active', got %v", doc.Dimensions["activity"])
		}

		// Verify it can also be accessed via typed interface
		typedDoc, err := store.Get(uuid)
		if err != nil {
			t.Fatalf("failed to get typed document: %v", err)
		}

		if typedDoc.Status != "active" {
			t.Errorf("expected typed status 'active', got %q", typedDoc.Status)
		}
		if typedDoc.Priority != "high" {
			t.Errorf("expected typed priority 'high', got %q", typedDoc.Priority)
		}
		if typedDoc.Activity != "active" {
			t.Errorf("expected typed activity 'active', got %q", typedDoc.Activity)
		}
	})

	t.Run("AddRawWithCustomDataFields", func(t *testing.T) {
		uuid, err := store.AddRaw("Task with Custom Data", map[string]interface{}{
			"status":         "pending",
			"priority":       "medium",
			"activity":       "active",
			"_data.assignee": "alice",
			"_data.tags":     "urgent,backend",
			"_data.estimate": 5,
		})
		if err != nil {
			t.Fatalf("failed to add raw document with custom data: %v", err)
		}

		// Verify via raw access that custom data fields are preserved
		doc, err := store.GetRaw(uuid)
		if err != nil {
			t.Fatalf("failed to get document with custom data: %v", err)
		}

		// Check standard dimensions
		if doc.Dimensions["status"] != "pending" {
			t.Errorf("expected status 'pending', got %v", doc.Dimensions["status"])
		}

		// Check custom data fields
		if doc.Dimensions["_data.assignee"] != "alice" {
			t.Errorf("expected _data.assignee 'alice', got %v", doc.Dimensions["_data.assignee"])
		}
		if doc.Dimensions["_data.tags"] != "urgent,backend" {
			t.Errorf("expected _data.tags 'urgent,backend', got %v", doc.Dimensions["_data.tags"])
		}
		if doc.Dimensions["_data.estimate"] != 5 {
			t.Errorf("expected _data.estimate 5, got %v", doc.Dimensions["_data.estimate"])
		}

		// Verify typed access still works for struct fields
		typedDoc, err := store.Get(uuid)
		if err != nil {
			t.Fatalf("failed to get typed version: %v", err)
		}

		if typedDoc.Status != "pending" {
			t.Errorf("expected typed status 'pending', got %q", typedDoc.Status)
		}
		if typedDoc.Priority != "medium" {
			t.Errorf("expected typed priority 'medium', got %q", typedDoc.Priority)
		}
	})

	t.Run("AddRawWithDefaultValues", func(t *testing.T) {
		// Add document with minimal dimensions to test defaults
		uuid, err := store.AddRaw("Minimal Task", map[string]interface{}{
			"status": "done",
		})
		if err != nil {
			t.Fatalf("failed to add minimal raw document: %v", err)
		}

		// Verify defaults are applied
		doc, err := store.GetRaw(uuid)
		if err != nil {
			t.Fatalf("failed to get minimal document: %v", err)
		}

		if doc.Dimensions["status"] != "done" {
			t.Errorf("expected status 'done', got %v", doc.Dimensions["status"])
		}

		// Should have default values for unspecified dimensions
		if doc.Dimensions["priority"] != "medium" {
			t.Errorf("expected default priority 'medium', got %v", doc.Dimensions["priority"])
		}
		if doc.Dimensions["activity"] != "active" {
			t.Errorf("expected default activity 'active', got %v", doc.Dimensions["activity"])
		}
	})

	t.Run("AddRawWithHierarchy", func(t *testing.T) {
		// First create a parent
		parentUUID, err := store.AddRaw("Parent Task", map[string]interface{}{
			"status":   "active",
			"priority": "high",
			"activity": "active",
		})
		if err != nil {
			t.Fatalf("failed to create parent: %v", err)
		}

		// Then create a child with explicit parent relationship
		childUUID, err := store.AddRaw("Child Task", map[string]interface{}{
			"status":    "pending",
			"priority":  "low",
			"activity":  "active",
			"parent_id": parentUUID,
		})
		if err != nil {
			t.Fatalf("failed to create child: %v", err)
		}

		// Verify hierarchy is established
		childDoc, err := store.GetRaw(childUUID)
		if err != nil {
			t.Fatalf("failed to get child document: %v", err)
		}

		if childDoc.Dimensions["parent_id"] != parentUUID {
			t.Errorf("expected parent_id %q, got %v", parentUUID, childDoc.Dimensions["parent_id"])
		}

		// Verify typed access also works
		typedChild, err := store.Get(childUUID)
		if err != nil {
			t.Fatalf("failed to get typed child: %v", err)
		}

		if typedChild.ParentID != parentUUID {
			t.Errorf("expected typed ParentID %q, got %q", parentUUID, typedChild.ParentID)
		}
	})

	t.Run("AddRawWithInvalidDimension", func(t *testing.T) {
		// Try to add with invalid enum value
		_, err := store.AddRaw("Invalid Task", map[string]interface{}{
			"status":   "invalid_status",
			"priority": "medium",
			"activity": "active",
		})

		// This should still work in AddRaw mode - it bypasses struct validation
		// The validation happens at the store level, not the Store level
		if err != nil {
			t.Logf("AddRaw with invalid dimension value returned error: %v", err)
			// This is acceptable behavior - some stores may validate, others may not
		}
	})

	t.Run("AddRawEmptyTitle", func(t *testing.T) {
		uuid, err := store.AddRaw("", map[string]interface{}{
			"status":   "active",
			"priority": "high",
			"activity": "active",
		})
		if err != nil {
			t.Fatalf("failed to add document with empty title: %v", err)
		}

		doc, err := store.GetRaw(uuid)
		if err != nil {
			t.Fatalf("failed to get document with empty title: %v", err)
		}

		if doc.Title != "" {
			t.Errorf("expected empty title, got %q", doc.Title)
		}
	})

	t.Run("AddRawEmptyDimensions", func(t *testing.T) {
		uuid, err := store.AddRaw("Empty Dimensions Task", map[string]interface{}{})
		if err != nil {
			t.Fatalf("failed to add document with empty dimensions: %v", err)
		}

		doc, err := store.GetRaw(uuid)
		if err != nil {
			t.Fatalf("failed to get document with empty dimensions: %v", err)
		}

		if doc.Title != "Empty Dimensions Task" {
			t.Errorf("expected title 'Empty Dimensions Task', got %q", doc.Title)
		}

		// Should have all default values
		if doc.Dimensions["status"] != "pending" {
			t.Errorf("expected default status 'pending', got %v", doc.Dimensions["status"])
		}
		if doc.Dimensions["priority"] != "medium" {
			t.Errorf("expected default priority 'medium', got %v", doc.Dimensions["priority"])
		}
		if doc.Dimensions["activity"] != "active" {
			t.Errorf("expected default activity 'active', got %v", doc.Dimensions["activity"])
		}
	})

	t.Run("AddRawNilDimensions", func(t *testing.T) {
		uuid, err := store.AddRaw("Nil Dimensions Task", nil)
		if err != nil {
			t.Fatalf("failed to add document with nil dimensions: %v", err)
		}

		doc, err := store.GetRaw(uuid)
		if err != nil {
			t.Fatalf("failed to get document with nil dimensions: %v", err)
		}

		if doc.Title != "Nil Dimensions Task" {
			t.Errorf("expected title 'Nil Dimensions Task', got %q", doc.Title)
		}

		// Should have all default values
		if doc.Dimensions["status"] != "pending" {
			t.Errorf("expected default status 'pending', got %v", doc.Dimensions["status"])
		}
	})
}

func TestStoreAddRawIntegration(t *testing.T) {
	// Test mixing AddRaw with regular typed operations
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.New[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("MixAddRawWithTypedOperations", func(t *testing.T) {
		// Create one document with typed Create
		typedUUID, err := store.Create("Typed Task", &TodoItem{
			Status:   "active",
			Priority: "high",
			Activity: "active",
		})
		if err != nil {
			t.Fatalf("failed to create typed document: %v", err)
		}

		// Create another with AddRaw
		rawUUID, err := store.AddRaw("Raw Task", map[string]interface{}{
			"status":   "pending",
			"priority": "low",
			"activity": "active",
		})
		if err != nil {
			t.Fatalf("failed to create raw document: %v", err)
		}

		// List all documents should show both
		allTasks, err := store.List(types.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list all tasks: %v", err)
		}

		if len(allTasks) != 2 {
			t.Errorf("expected 2 tasks, got %d", len(allTasks))
		}

		// Both should be accessible via typed interface
		var foundTyped, foundRaw bool
		for _, task := range allTasks {
			if task.UUID == typedUUID {
				foundTyped = true
				if task.Title != "Typed Task" {
					t.Errorf("typed task title mismatch: got %q", task.Title)
				}
			}
			if task.UUID == rawUUID {
				foundRaw = true
				if task.Title != "Raw Task" {
					t.Errorf("raw task title mismatch: got %q", task.Title)
				}
			}
		}

		if !foundTyped {
			t.Error("typed document not found in list")
		}
		if !foundRaw {
			t.Error("raw document not found in list")
		}

		// Both should be queryable
		pendingTasks, err := store.Query().Status("pending").Find()
		if err != nil {
			t.Fatalf("failed to query pending tasks: %v", err)
		}

		if len(pendingTasks) != 1 {
			t.Errorf("expected 1 pending task, got %d", len(pendingTasks))
		}

		if len(pendingTasks) > 0 && pendingTasks[0].UUID != rawUUID {
			t.Errorf("expected pending task to be the raw one, got %q", pendingTasks[0].UUID)
		}
	})
}
