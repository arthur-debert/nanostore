package todo_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestDebugHierarchicalIDs(t *testing.T) {
	// Create store directly with default config
	store, err := nanostore.New(":memory:", nanostore.Config{
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
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Add root document
	rootID, err := store.Add("Root", nil)
	if err != nil {
		t.Fatalf("failed to add root: %v", err)
	}
	t.Logf("Root UUID: %s", rootID)

	// Add child documents
	child1ID, err := store.Add("Child 1", map[string]interface{}{"parent_uuid": rootID})
	if err != nil {
		t.Fatalf("failed to add child 1: %v", err)
	}
	t.Logf("Child 1 UUID: %s", child1ID)

	child2ID, err := store.Add("Child 2", map[string]interface{}{"parent_uuid": rootID})
	if err != nil {
		t.Fatalf("failed to add child 2: %v", err)
	}
	t.Logf("Child 2 UUID: %s", child2ID)

	// List all documents
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	t.Logf("\nAll documents:")
	for _, doc := range docs {
		parentInfo := "nil"
		if parentUUID, hasParent := doc.Dimensions["parent_uuid"].(string); hasParent && parentUUID != "" {
			parentInfo = parentUUID
		}
		status, _ := doc.Dimensions["status"].(string)
		t.Logf("  ID: %s, Title: %s, UUID: %s, Parent: %s, Status: %s",
			doc.UserFacingID, doc.Title, doc.UUID, parentInfo, status)
	}

	// Check if we got hierarchical IDs
	foundHierarchical := false
	for _, doc := range docs {
		if doc.Title == "Child 1" && doc.UserFacingID == "1.1" {
			foundHierarchical = true
			break
		}
	}

	if !foundHierarchical {
		t.Errorf("Expected hierarchical ID '1.1' for Child 1, but didn't find it")
	}
}
