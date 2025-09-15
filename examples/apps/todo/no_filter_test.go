package todo_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestNoFilterHierarchy(t *testing.T) {
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

	// Add documents
	rootID, _ := store.Add("Groceries", nil)
	store.Add("Milk", map[string]interface{}{"parent_uuid": rootID})
	store.Add("Bread", map[string]interface{}{"parent_uuid": rootID})
	store.Add("Eggs", map[string]interface{}{"parent_uuid": rootID})
	store.Add("Pack for Trip", nil)

	// List WITHOUT any filters
	t.Logf("\nListing without filters:")
	docs, _ := store.List(nanostore.ListOptions{})
	for _, doc := range docs {
		t.Logf("  ID: %-5s Title: %s", doc.UserFacingID, doc.Title)
	}
}
