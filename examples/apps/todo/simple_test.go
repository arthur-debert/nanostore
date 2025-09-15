package todo_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestSimpleHierarchy(t *testing.T) {
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

	// Add root
	rootID, _ := store.Add("Groceries", nil)

	// Add children
	store.Add("Milk", map[string]interface{}{"parent_uuid": rootID})
	store.Add("Bread", map[string]interface{}{"parent_uuid": rootID})
	store.Add("Eggs", map[string]interface{}{"parent_uuid": rootID})

	// Add another root
	store.Add("Pack for Trip", nil)

	// List all documents
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	t.Logf("\nDocuments from nanostore:")
	for _, doc := range docs {
		parentInfo := "nil"
		if parentUUID := doc.GetParentUUID(); parentUUID != nil {
			parentInfo = *parentUUID
		}
		t.Logf("  ID: %-5s Title: %-15s Parent: %s", doc.UserFacingID, doc.Title, parentInfo)
	}
}
