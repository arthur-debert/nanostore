package nanostore_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestDeleteWithSmartID(t *testing.T) {
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
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "high"},
				DefaultValue: "low",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create documents
	uuid1, err := store.Add("Document 1", map[string]interface{}{
		"status":   "active",
		"priority": "high",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Add("Document 2", map[string]interface{}{
		"status":   "active",
		"priority": "low",
	})
	if err != nil {
		t.Fatal(err)
	}

	// List to get simple IDs
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(docs))
	}

	// Test 1: Delete using UUID should still work
	err = store.Delete(uuid1, false)
	if err != nil {
		t.Errorf("Delete with UUID failed: %v", err)
	}

	// Verify deletion
	docs, err = store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 document after delete, got %d", len(docs))
	}

	// Re-create the document for next test
	_, err = store.Add("Document 1", map[string]interface{}{
		"status":   "active",
		"priority": "high",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get the new simple ID
	docs, err = store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	simpleID := docs[0].SimpleID

	// Test 2: Delete using SimpleID should work
	err = store.Delete(simpleID, false)
	if err != nil {
		t.Errorf("Delete with SimpleID failed: %v", err)
	}

	// Verify deletion
	docs, err = store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 document after delete, got %d", len(docs))
	}

	// Test 3: Delete with invalid ID should fail
	err = store.Delete("invalid-id", false)
	if err == nil {
		t.Error("expected error for invalid ID, got nil")
	}
}

func TestDeleteCascadeWithSmartID(t *testing.T) {
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

	// Create parent document
	parentUUID, err := store.Add("Parent", map[string]interface{}{
		"status": "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create child document
	_, err = store.Add("Child", map[string]interface{}{
		"status":    "active",
		"parent_id": parentUUID,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get simple ID of parent
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var parentSimpleID string
	for _, doc := range docs {
		if doc.UUID == parentUUID {
			parentSimpleID = doc.SimpleID
			break
		}
	}

	// Test: Delete parent with cascade using SimpleID
	err = store.Delete(parentSimpleID, true)
	if err != nil {
		t.Errorf("Delete with cascade using SimpleID failed: %v", err)
	}

	// Verify both documents are deleted
	docs, err = store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 documents after cascade delete, got %d", len(docs))
	}
}
