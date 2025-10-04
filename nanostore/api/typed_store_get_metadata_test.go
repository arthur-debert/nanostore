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
	"time"

	"github.com/arthur-debert/nanostore/nanostore/api"
	"github.com/arthur-debert/nanostore/types"
)

func TestTypedStoreGetMetadata(t *testing.T) {
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

	// Record time before creation for timestamp validation
	beforeCreate := time.Now()

	// Create test data
	uuid, err := store.Create("Test Task", &TodoItem{
		Status:   "active",
		Priority: "high",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Record time after creation
	afterCreate := time.Now()

	// Get SimpleID for testing
	allTasks, err := store.List(types.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var simpleID string
	for _, task := range allTasks {
		if task.UUID == uuid {
			simpleID = task.SimpleID
			break
		}
	}

	if simpleID == "" {
		t.Fatal("failed to find SimpleID for test task")
	}

	t.Run("GetMetadataByUUID", func(t *testing.T) {
		metadata, err := store.GetMetadata(uuid)
		if err != nil {
			t.Fatalf("failed to get metadata by UUID: %v", err)
		}

		if metadata == nil {
			t.Fatal("expected metadata, got nil")
		}

		// Verify basic fields
		if metadata.UUID != uuid {
			t.Errorf("expected UUID %q, got %q", uuid, metadata.UUID)
		}
		if metadata.SimpleID != simpleID {
			t.Errorf("expected SimpleID %q, got %q", simpleID, metadata.SimpleID)
		}
		if metadata.Title != "Test Task" {
			t.Errorf("expected title 'Test Task', got %q", metadata.Title)
		}

		// Verify timestamps are reasonable
		if metadata.CreatedAt.Before(beforeCreate) || metadata.CreatedAt.After(afterCreate) {
			t.Errorf("CreatedAt %v should be between %v and %v", metadata.CreatedAt, beforeCreate, afterCreate)
		}
		if metadata.UpdatedAt.Before(beforeCreate) || metadata.UpdatedAt.After(afterCreate) {
			t.Errorf("UpdatedAt %v should be between %v and %v", metadata.UpdatedAt, beforeCreate, afterCreate)
		}

		// CreatedAt and UpdatedAt should be equal for new documents
		if !metadata.CreatedAt.Equal(metadata.UpdatedAt) {
			t.Errorf("for new document, CreatedAt %v should equal UpdatedAt %v", metadata.CreatedAt, metadata.UpdatedAt)
		}
	})

	t.Run("GetMetadataBySimpleID", func(t *testing.T) {
		metadata, err := store.GetMetadata(simpleID)
		if err != nil {
			t.Fatalf("failed to get metadata by SimpleID: %v", err)
		}

		if metadata == nil {
			t.Fatal("expected metadata, got nil")
		}

		// Should be the same as UUID access
		if metadata.UUID != uuid {
			t.Errorf("expected UUID %q, got %q", uuid, metadata.UUID)
		}
		if metadata.SimpleID != simpleID {
			t.Errorf("expected SimpleID %q, got %q", simpleID, metadata.SimpleID)
		}
		if metadata.Title != "Test Task" {
			t.Errorf("expected title 'Test Task', got %q", metadata.Title)
		}
	})

	t.Run("GetMetadataNonExistentDocument", func(t *testing.T) {
		_, err := store.GetMetadata("non-existent-uuid")
		if err == nil {
			t.Error("expected error for non-existent document")
		}
		if err.Error() != "document with ID 'non-existent-uuid' not found" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("GetMetadataEmptyID", func(t *testing.T) {
		_, err := store.GetMetadata("")
		if err == nil {
			t.Error("expected error for empty ID")
		}
	})

	t.Run("GetMetadataInvalidSimpleID", func(t *testing.T) {
		_, err := store.GetMetadata("999")
		if err == nil {
			t.Error("expected error for invalid SimpleID")
		}
	})

	t.Run("CompareMetadataWithRawDocument", func(t *testing.T) {
		// Get metadata
		metadata, err := store.GetMetadata(uuid)
		if err != nil {
			t.Fatalf("failed to get metadata: %v", err)
		}

		// Get raw document
		rawDoc, err := store.GetRaw(uuid)
		if err != nil {
			t.Fatalf("failed to get raw document: %v", err)
		}

		// Verify metadata matches raw document
		if metadata.UUID != rawDoc.UUID {
			t.Errorf("UUID mismatch: metadata %q vs raw %q", metadata.UUID, rawDoc.UUID)
		}
		if metadata.SimpleID != rawDoc.SimpleID {
			t.Errorf("SimpleID mismatch: metadata %q vs raw %q", metadata.SimpleID, rawDoc.SimpleID)
		}
		if metadata.Title != rawDoc.Title {
			t.Errorf("Title mismatch: metadata %q vs raw %q", metadata.Title, rawDoc.Title)
		}
		if !metadata.CreatedAt.Equal(rawDoc.CreatedAt) {
			t.Errorf("CreatedAt mismatch: metadata %v vs raw %v", metadata.CreatedAt, rawDoc.CreatedAt)
		}
		if !metadata.UpdatedAt.Equal(rawDoc.UpdatedAt) {
			t.Errorf("UpdatedAt mismatch: metadata %v vs raw %v", metadata.UpdatedAt, rawDoc.UpdatedAt)
		}
	})

	t.Run("CompareMetadataWithTypedDocument", func(t *testing.T) {
		// Get metadata
		metadata, err := store.GetMetadata(uuid)
		if err != nil {
			t.Fatalf("failed to get metadata: %v", err)
		}

		// Get typed document
		typedDoc, err := store.Get(uuid)
		if err != nil {
			t.Fatalf("failed to get typed document: %v", err)
		}

		// Verify metadata matches typed document
		if metadata.UUID != typedDoc.UUID {
			t.Errorf("UUID mismatch: metadata %q vs typed %q", metadata.UUID, typedDoc.UUID)
		}
		if metadata.SimpleID != typedDoc.SimpleID {
			t.Errorf("SimpleID mismatch: metadata %q vs typed %q", metadata.SimpleID, typedDoc.SimpleID)
		}
		if metadata.Title != typedDoc.Title {
			t.Errorf("Title mismatch: metadata %q vs typed %q", metadata.Title, typedDoc.Title)
		}
		if !metadata.CreatedAt.Equal(typedDoc.CreatedAt) {
			t.Errorf("CreatedAt mismatch: metadata %v vs typed %v", metadata.CreatedAt, typedDoc.CreatedAt)
		}
		if !metadata.UpdatedAt.Equal(typedDoc.UpdatedAt) {
			t.Errorf("UpdatedAt mismatch: metadata %v vs typed %v", metadata.UpdatedAt, typedDoc.UpdatedAt)
		}
	})
}

func TestTypedStoreGetMetadataWithHierarchy(t *testing.T) {
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

	t.Run("GetMetadataWithHierarchicalSimpleID", func(t *testing.T) {
		metadata, err := store.GetMetadata(childSimpleID)
		if err != nil {
			t.Fatalf("failed to get metadata by hierarchical SimpleID: %v", err)
		}

		// Verify it's the correct child document
		if metadata.UUID != childUUID {
			t.Errorf("expected UUID %q, got %q", childUUID, metadata.UUID)
		}
		if metadata.Title != "Child" {
			t.Errorf("expected title 'Child', got %q", metadata.Title)
		}

		// Verify SimpleID shows hierarchical structure (implementation dependent)
		t.Logf("Child SimpleID: %q (hierarchical structure)", metadata.SimpleID)
	})

	t.Run("GetMetadataForAllHierarchyLevels", func(t *testing.T) {
		// Get metadata for all levels
		grandparentMeta, err := store.GetMetadata(grandparentUUID)
		if err != nil {
			t.Fatalf("failed to get grandparent metadata: %v", err)
		}

		parentMeta, err := store.GetMetadata(parentUUID)
		if err != nil {
			t.Fatalf("failed to get parent metadata: %v", err)
		}

		childMeta, err := store.GetMetadata(childUUID)
		if err != nil {
			t.Fatalf("failed to get child metadata: %v", err)
		}

		// Verify unique UUIDs
		if grandparentMeta.UUID == parentMeta.UUID || grandparentMeta.UUID == childMeta.UUID || parentMeta.UUID == childMeta.UUID {
			t.Error("all hierarchy levels should have unique UUIDs")
		}

		// Verify unique SimpleIDs
		if grandparentMeta.SimpleID == parentMeta.SimpleID || grandparentMeta.SimpleID == childMeta.SimpleID || parentMeta.SimpleID == childMeta.SimpleID {
			t.Error("all hierarchy levels should have unique SimpleIDs")
		}

		// Verify titles
		if grandparentMeta.Title != "Grandparent" {
			t.Errorf("expected grandparent title 'Grandparent', got %q", grandparentMeta.Title)
		}
		if parentMeta.Title != "Parent" {
			t.Errorf("expected parent title 'Parent', got %q", parentMeta.Title)
		}
		if childMeta.Title != "Child" {
			t.Errorf("expected child title 'Child', got %q", childMeta.Title)
		}
	})
}

func TestTypedStoreGetMetadataWithUpdates(t *testing.T) {
	// Test that UpdatedAt changes after document updates
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
		Status:   "pending",
		Priority: "medium",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get initial metadata
	initialMeta, err := store.GetMetadata(uuid)
	if err != nil {
		t.Fatalf("failed to get initial metadata: %v", err)
	}

	// Wait a small amount to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Update the document
	_, err = store.Update(uuid, &TodoItem{
		Status:   "active",
		Priority: "high",
		Activity: "active",
	})
	if err != nil {
		t.Fatalf("failed to update document: %v", err)
	}

	t.Run("MetadataAfterUpdate", func(t *testing.T) {
		// Get metadata after update
		updatedMeta, err := store.GetMetadata(uuid)
		if err != nil {
			t.Fatalf("failed to get updated metadata: %v", err)
		}

		// UUID should remain the same
		if updatedMeta.UUID != initialMeta.UUID {
			t.Errorf("UUID should not change after update: initial %q vs updated %q", initialMeta.UUID, updatedMeta.UUID)
		}

		// SimpleID might change if dimensions change (e.g., prefix updates)
		t.Logf("SimpleID: initial %q vs updated %q", initialMeta.SimpleID, updatedMeta.SimpleID)

		// CreatedAt should remain the same
		if !updatedMeta.CreatedAt.Equal(initialMeta.CreatedAt) {
			t.Errorf("CreatedAt should not change after update: initial %v vs updated %v", initialMeta.CreatedAt, updatedMeta.CreatedAt)
		}

		// UpdatedAt should be newer than the initial value
		if !updatedMeta.UpdatedAt.After(initialMeta.UpdatedAt) {
			t.Errorf("UpdatedAt should be newer after update: initial %v vs updated %v", initialMeta.UpdatedAt, updatedMeta.UpdatedAt)
		}

		// Title should remain the same (we didn't update it)
		if updatedMeta.Title != initialMeta.Title {
			t.Errorf("Title should not change when not updated: initial %q vs updated %q", initialMeta.Title, updatedMeta.Title)
		}
	})
}
