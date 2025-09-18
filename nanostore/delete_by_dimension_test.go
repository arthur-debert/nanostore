package nanostore_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestDeleteByDimension(t *testing.T) {
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
				Values:       []string{"active", "inactive", "deleted"},
				DefaultValue: "active",
			},
			{
				Name:         "category",
				Type:         nanostore.Enumerated,
				Values:       []string{"work", "personal", "archive"},
				DefaultValue: "work",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test documents
	_, err = store.Add("Doc 1", map[string]interface{}{
		"status":   "active",
		"category": "work",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Add("Doc 2", map[string]interface{}{
		"status":   "inactive",
		"category": "work",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Add("Doc 3", map[string]interface{}{
		"status":   "active",
		"category": "personal",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Add("Doc 4", map[string]interface{}{
		"status":   "deleted",
		"category": "archive",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Test 1: Delete by single dimension
	count, err := store.DeleteByDimension(map[string]interface{}{
		"status": "deleted",
	})
	if err != nil {
		t.Fatalf("DeleteByDimension failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected to delete 1 document, deleted %d", count)
	}

	// Verify deletion
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 3 {
		t.Errorf("expected 3 documents after deletion, got %d", len(docs))
	}

	// Test 2: Delete by multiple dimensions
	count, err = store.DeleteByDimension(map[string]interface{}{
		"status":   "inactive",
		"category": "work",
	})
	if err != nil {
		t.Fatalf("DeleteByDimension failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected to delete 1 document, deleted %d", count)
	}

	// Verify deletion
	docs, err = store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 documents after deletion, got %d", len(docs))
	}

	// Test 3: Delete with no matches
	count, err = store.DeleteByDimension(map[string]interface{}{
		"status": "nonexistent",
	})
	if err != nil {
		t.Fatalf("DeleteByDimension failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected to delete 0 documents, deleted %d", count)
	}

	// Test 4: Delete all remaining with category filter
	count, err = store.DeleteByDimension(map[string]interface{}{
		"category": "work",
	})
	if err != nil {
		t.Fatalf("DeleteByDimension failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected to delete 1 document, deleted %d", count)
	}

	// Verify only one document remains
	docs, err = store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 document after deletion, got %d", len(docs))
	}
	if docs[0].Dimensions["category"] != "personal" {
		t.Errorf("wrong document remains: %v", docs[0])
	}
}

func TestDeleteByDimensionWithNonDimensionFields(t *testing.T) {
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
				Values:       []string{"active", "inactive"},
				DefaultValue: "active",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create documents with non-dimension fields
	_, err = store.Add("Doc 1", map[string]interface{}{
		"status":        "active",
		"_data.version": "1.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Add("Doc 2", map[string]interface{}{
		"status":        "active",
		"_data.version": "2.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Add("Doc 3", map[string]interface{}{
		"status":        "inactive",
		"_data.version": "1.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Test: Delete by non-dimension field
	count, err := store.DeleteByDimension(map[string]interface{}{
		"_data.version": "1.0",
	})
	if err != nil {
		t.Fatalf("DeleteByDimension failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected to delete 2 documents, deleted %d", count)
	}

	// Verify deletion
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 document after deletion, got %d", len(docs))
	}
	if docs[0].Dimensions["_data.version"] != "2.0" {
		t.Errorf("wrong document remains: %v", docs[0])
	}
}
