package nanostore_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestRefFieldSmartIDResolution(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"active", "archived"},
				DefaultValue: "active",
			},
			{
				Name:     "location",
				Type:     nanostore.Hierarchical,
				RefField: "parent_id",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Test 1: Add with SimpleID as parent_id
	// First create a parent
	parentUUID, err := store.Add("Parent", map[string]interface{}{
		"status": "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get the parent's simple ID
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}
	parentSimpleID := docs[0].SimpleID

	// Create a child using parent's SimpleID
	childUUID, err := store.Add("Child", map[string]interface{}{
		"status":    "active",
		"parent_id": parentSimpleID, // Using SimpleID instead of UUID
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify the child has the correct parent UUID stored
	docs, err = store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"uuid": childUUID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}
	if docs[0].Dimensions["parent_id"] != parentUUID {
		t.Errorf("expected parent_id to be resolved to UUID %q, got %q", parentUUID, docs[0].Dimensions["parent_id"])
	}

	// Test 2: Update with SimpleID as parent_id
	// Create another parent
	parent2UUID, err := store.Add("Parent 2", map[string]interface{}{
		"status": "archived",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get parent 2's simple ID
	docs, err = store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"uuid": parent2UUID},
	})
	if err != nil {
		t.Fatal(err)
	}
	parent2SimpleID := docs[0].SimpleID

	// Update child to point to parent 2 using SimpleID
	err = store.Update(childUUID, nanostore.UpdateRequest{
		Dimensions: map[string]interface{}{
			"parent_id": parent2SimpleID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify the update
	docs, err = store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"uuid": childUUID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if docs[0].Dimensions["parent_id"] != parent2UUID {
		t.Errorf("expected parent_id to be resolved to UUID %q, got %q", parent2UUID, docs[0].Dimensions["parent_id"])
	}

	// Test 3: UpdateByDimension with SimpleID as parent_id
	// Create more children
	_, err = store.Add("Child 2", map[string]interface{}{
		"status":    "active",
		"parent_id": parentUUID,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Add("Child 3", map[string]interface{}{
		"status":    "active",
		"parent_id": parentUUID,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Update all active children to point to parent2 using SimpleID
	count, err := store.UpdateByDimension(
		map[string]interface{}{
			"status":    "active",
			"parent_id": parentUUID,
		},
		nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{
				"parent_id": parent2SimpleID,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected to update 2 documents, updated %d", count)
	}

	// Verify all children now point to parent2
	docs, err = store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"parent_id": parent2UUID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 3 {
		t.Errorf("expected 3 children with parent2, got %d", len(docs))
	}
}

func TestRefFieldWithInvalidSimpleID(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"active", "archived"},
				DefaultValue: "active",
			},
			{
				Name:     "location",
				Type:     nanostore.Hierarchical,
				RefField: "parent_id",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Test: Add with invalid SimpleID as parent_id (should store as-is)
	childUUID, err := store.Add("Child", map[string]interface{}{
		"status":    "active",
		"parent_id": "invalid-simple-id",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify the invalid ID is stored as-is
	docs, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"uuid": childUUID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if docs[0].Dimensions["parent_id"] != "invalid-simple-id" {
		t.Errorf("expected parent_id to be stored as-is 'invalid-simple-id', got %q", docs[0].Dimensions["parent_id"])
	}
}
