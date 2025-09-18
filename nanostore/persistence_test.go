package nanostore_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestPersistence(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	filename := tmpfile.Name()
	_ = tmpfile.Close()
	defer func() { _ = os.Remove(filename) }()

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"todo", "done"},
				DefaultValue: "todo",
			},
		},
	}

	// Create store and add documents
	store1, err := nanostore.New(filename, config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	id1, _ := store1.Add("Task 1", nil)
	id2, _ := store1.Add("Task 2", map[string]interface{}{"status": "done"})

	// Close the store
	if err := store1.Close(); err != nil {
		t.Fatalf("failed to close store: %v", err)
	}

	// Verify JSON file exists and is valid
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Verify structure
	if _, ok := jsonData["documents"]; !ok {
		t.Error("missing 'documents' field in JSON")
	}
	if _, ok := jsonData["metadata"]; !ok {
		t.Error("missing 'metadata' field in JSON")
	}

	// Create a new store with the same file
	store2, err := nanostore.New(filename, config)
	if err != nil {
		t.Fatalf("failed to create second store: %v", err)
	}
	defer func() { _ = store2.Close() }()

	// Verify documents were loaded
	docs, err := store2.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}

	// Verify document IDs match
	foundIDs := map[string]bool{id1: false, id2: false}
	for _, doc := range docs {
		if _, exists := foundIDs[doc.UUID]; exists {
			foundIDs[doc.UUID] = true
		}
	}

	for id, found := range foundIDs {
		if !found {
			t.Errorf("document %s not found after reload", id)
		}
	}
}
