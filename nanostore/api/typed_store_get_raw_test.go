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

func TestTypedStoreGetRaw(t *testing.T) {
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

	// Create test data
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
		Priority: "medium",
		Activity: "active",
		ParentID: parentUUID,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get SimpleIDs for testing
	allTasks, err := store.List(types.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var parentSimpleID, childSimpleID string
	for _, task := range allTasks {
		if task.UUID == parentUUID {
			parentSimpleID = task.SimpleID
		}
		if task.UUID == childUUID {
			childSimpleID = task.SimpleID
		}
	}

	if parentSimpleID == "" || childSimpleID == "" {
		t.Fatal("failed to find SimpleIDs for test tasks")
	}

	t.Run("GetRawByUUID", func(t *testing.T) {
		doc, err := store.GetRaw(parentUUID)
		if err != nil {
			t.Fatalf("failed to get raw document by UUID: %v", err)
		}

		// Verify raw document structure
		if doc.UUID != parentUUID {
			t.Errorf("expected UUID %q, got %q", parentUUID, doc.UUID)
		}
		if doc.SimpleID != parentSimpleID {
			t.Errorf("expected SimpleID %q, got %q", parentSimpleID, doc.SimpleID)
		}
		if doc.Title != "Parent Task" {
			t.Errorf("expected title 'Parent Task', got %q", doc.Title)
		}

		// Verify dimensions are accessible
		if doc.Dimensions == nil {
			t.Error("expected dimensions to be populated")
		}
		if status, ok := doc.Dimensions["status"]; !ok || status != "active" {
			t.Errorf("expected status 'active' in dimensions, got %v", status)
		}
		if priority, ok := doc.Dimensions["priority"]; !ok || priority != "high" {
			t.Errorf("expected priority 'high' in dimensions, got %v", priority)
		}
		if activity, ok := doc.Dimensions["activity"]; !ok || activity != "active" {
			t.Errorf("expected activity 'active' in dimensions, got %v", activity)
		}

		// Verify timestamps are set
		if doc.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be set")
		}
		if doc.UpdatedAt.IsZero() {
			t.Error("expected UpdatedAt to be set")
		}
	})

	t.Run("GetRawBySimpleID", func(t *testing.T) {
		doc, err := store.GetRaw(childSimpleID)
		if err != nil {
			t.Fatalf("failed to get raw document by SimpleID: %v", err)
		}

		// Verify it's the correct document
		if doc.UUID != childUUID {
			t.Errorf("expected UUID %q, got %q", childUUID, doc.UUID)
		}
		if doc.SimpleID != childSimpleID {
			t.Errorf("expected SimpleID %q, got %q", childSimpleID, doc.SimpleID)
		}
		if doc.Title != "Child Task" {
			t.Errorf("expected title 'Child Task', got %q", doc.Title)
		}

		// Verify hierarchical relationship
		if parentID, ok := doc.Dimensions["parent_id"]; !ok || parentID != parentUUID {
			t.Errorf("expected parent_id %q in dimensions, got %v", parentUUID, parentID)
		}
	})

	t.Run("GetRawWithNonExistentUUID", func(t *testing.T) {
		_, err := store.GetRaw("non-existent-uuid")
		if err == nil {
			t.Error("expected error for non-existent UUID")
		}
		if err.Error() != "document with ID non-existent-uuid not found" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("GetRawWithNonExistentSimpleID", func(t *testing.T) {
		_, err := store.GetRaw("999")
		if err == nil {
			t.Error("expected error for non-existent SimpleID")
		}
		if err.Error() != "document with ID 999 not found" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("GetRawEmptyID", func(t *testing.T) {
		_, err := store.GetRaw("")
		if err == nil {
			t.Error("expected error for empty ID")
		}
	})

	t.Run("CompareRawVsTyped", func(t *testing.T) {
		// Get the same document both ways
		typedDoc, err := store.Get(parentUUID)
		if err != nil {
			t.Fatalf("failed to get typed document: %v", err)
		}

		rawDoc, err := store.GetRaw(parentUUID)
		if err != nil {
			t.Fatalf("failed to get raw document: %v", err)
		}

		// Verify they represent the same document
		if rawDoc.UUID != typedDoc.UUID {
			t.Errorf("UUID mismatch: raw %q vs typed %q", rawDoc.UUID, typedDoc.UUID)
		}
		if rawDoc.SimpleID != typedDoc.SimpleID {
			t.Errorf("SimpleID mismatch: raw %q vs typed %q", rawDoc.SimpleID, typedDoc.SimpleID)
		}
		if rawDoc.Title != typedDoc.Title {
			t.Errorf("Title mismatch: raw %q vs typed %q", rawDoc.Title, typedDoc.Title)
		}

		// Verify dimensions match struct fields
		if rawDoc.Dimensions["status"] != typedDoc.Status {
			t.Errorf("status mismatch: raw %v vs typed %q", rawDoc.Dimensions["status"], typedDoc.Status)
		}
		if rawDoc.Dimensions["priority"] != typedDoc.Priority {
			t.Errorf("priority mismatch: raw %v vs typed %q", rawDoc.Dimensions["priority"], typedDoc.Priority)
		}
		if rawDoc.Dimensions["activity"] != typedDoc.Activity {
			t.Errorf("activity mismatch: raw %v vs typed %q", rawDoc.Dimensions["activity"], typedDoc.Activity)
		}
	})

	t.Run("AccessDataFieldsInRaw", func(t *testing.T) {
		// Update a document with a custom data field (not in struct)
		_, err := store.GetRaw(parentUUID) // First get to verify it exists
		if err != nil {
			t.Fatal(err)
		}

		// Note: We can't easily add custom _data fields through TypedStore in this test,
		// but we can verify that the raw document allows access to all dimensions
		doc, err := store.GetRaw(parentUUID)
		if err != nil {
			t.Fatal(err)
		}

		// Verify we can access all dimensions, not just struct fields
		if doc.Dimensions == nil {
			t.Error("expected dimensions map to be accessible")
		}

		// The dimensions should include all our struct fields
		expectedDimensions := []string{"status", "priority", "activity"}
		for _, dim := range expectedDimensions {
			if _, exists := doc.Dimensions[dim]; !exists {
				t.Errorf("expected dimension %q to exist in raw document", dim)
			}
		}
	})
}

func TestTypedStoreGetRawWithHierarchy(t *testing.T) {
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

	// Create hierarchical data
	grandparentUUID, err := store.Create("Grandparent", &TodoItem{
		Status:   "active",
		Priority: "high",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	parentUUID, err := store.Create("Parent", &TodoItem{
		Status:   "active",
		Priority: "medium",
		Activity: "active",
		ParentID: grandparentUUID,
	})
	if err != nil {
		t.Fatal(err)
	}

	childUUID, err := store.Create("Child", &TodoItem{
		Status:   "pending",
		Priority: "low",
		Activity: "active",
		ParentID: parentUUID,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get all documents to find SimpleIDs
	allTasks, err := store.List(types.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var childSimpleID string
	for _, task := range allTasks {
		if task.UUID == childUUID {
			childSimpleID = task.SimpleID
			break
		}
	}

	t.Run("GetRawWithHierarchicalSimpleID", func(t *testing.T) {
		doc, err := store.GetRaw(childSimpleID)
		if err != nil {
			t.Fatalf("failed to get hierarchical document by SimpleID: %v", err)
		}

		// Verify it's the correct child document
		if doc.UUID != childUUID {
			t.Errorf("expected UUID %q, got %q", childUUID, doc.UUID)
		}
		if doc.Title != "Child" {
			t.Errorf("expected title 'Child', got %q", doc.Title)
		}

		// Verify parent relationship is preserved in raw format
		if parentID, ok := doc.Dimensions["parent_id"]; !ok || parentID != parentUUID {
			t.Errorf("expected parent_id %q, got %v", parentUUID, parentID)
		}

		// Verify SimpleID shows hierarchical structure (implementation dependent)
		t.Logf("Child SimpleID: %q (hierarchical structure)", doc.SimpleID)
	})
}
