package nanostore_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestUpdateByDimension(t *testing.T) {
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
				Values:       []string{"draft", "review", "published"},
				DefaultValue: "draft",
			},
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "medium", "high"},
				DefaultValue: "medium",
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
		"status":   "draft",
		"priority": "low",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Add("Doc 2", map[string]interface{}{
		"status":   "draft",
		"priority": "high",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Add("Doc 3", map[string]interface{}{
		"status":   "review",
		"priority": "medium",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Add("Doc 4", map[string]interface{}{
		"status":   "published",
		"priority": "low",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Test 1: Update by single dimension
	newTitle := "Updated Title"
	count, err := store.UpdateByDimension(
		map[string]interface{}{"status": "draft"},
		nanostore.UpdateRequest{
			Title: &newTitle,
			Dimensions: map[string]interface{}{
				"status": "review",
			},
		},
	)
	if err != nil {
		t.Fatalf("UpdateByDimension failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected to update 2 documents, updated %d", count)
	}

	// Verify update
	docs, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"status": "review"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 3 { // 1 original + 2 updated
		t.Errorf("expected 3 documents with status=review, got %d", len(docs))
	}
	for _, doc := range docs {
		if doc.Title == newTitle && (doc.Dimensions["priority"] != "low" && doc.Dimensions["priority"] != "high") {
			t.Errorf("document was not updated correctly: %v", doc)
		}
	}

	// Test 2: Update multiple fields
	newBody := "Updated Body"
	count, err = store.UpdateByDimension(
		map[string]interface{}{
			"status":   "review",
			"priority": "medium",
		},
		nanostore.UpdateRequest{
			Body: &newBody,
			Dimensions: map[string]interface{}{
				"priority": "high",
			},
		},
	)
	if err != nil {
		t.Fatalf("UpdateByDimension failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected to update 1 document, updated %d", count)
	}

	// Test 3: Update with no matches
	count, err = store.UpdateByDimension(
		map[string]interface{}{"status": "nonexistent"},
		nanostore.UpdateRequest{
			Title: &newTitle,
		},
	)
	if err != nil {
		t.Fatalf("UpdateByDimension failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected to update 0 documents, updated %d", count)
	}

	// Test 4: Invalid dimension value in update
	_, err = store.UpdateByDimension(
		map[string]interface{}{"status": "review"},
		nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{
				"status": "invalid",
			},
		},
	)
	if err == nil {
		t.Error("expected error for invalid dimension value, got nil")
	}
}

func TestUpdateByDimensionWithNonDimensionFields(t *testing.T) {
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
				Name:         "type",
				Type:         nanostore.Enumerated,
				Values:       []string{"A", "B", "C"},
				DefaultValue: "A",
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
		"type":          "A",
		"_data.version": 1.0,
		"_data.author":  "alice",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Add("Doc 2", map[string]interface{}{
		"type":          "A",
		"_data.version": 2.0,
		"_data.author":  "bob",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Add("Doc 3", map[string]interface{}{
		"type":          "B",
		"_data.version": 1.0,
		"_data.author":  "alice",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Test 1: Update by non-dimension field
	count, err := store.UpdateByDimension(
		map[string]interface{}{"_data.author": "alice"},
		nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{
				"_data.version": 2.5,
				"_data.status":  "updated",
			},
		},
	)
	if err != nil {
		t.Fatalf("UpdateByDimension failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected to update 2 documents, updated %d", count)
	}

	// Verify updates
	docs, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"_data.author": "alice"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, doc := range docs {
		if doc.Dimensions["_data.version"] != 2.5 {
			t.Errorf("expected version 2.5, got %v", doc.Dimensions["_data.version"])
		}
		if doc.Dimensions["_data.status"] != "updated" {
			t.Errorf("expected status 'updated', got %v", doc.Dimensions["_data.status"])
		}
	}

	// Test 2: Mixed dimension and non-dimension update
	newTitle := "Mixed Update"
	count, err = store.UpdateByDimension(
		map[string]interface{}{
			"type":          "A",
			"_data.version": 2.5,
		},
		nanostore.UpdateRequest{
			Title: &newTitle,
			Dimensions: map[string]interface{}{
				"type":           "C",
				"_data.category": "processed",
			},
		},
	)
	if err != nil {
		t.Fatalf("UpdateByDimension failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected to update 1 document, updated %d", count)
	}

	// Verify the mixed update
	docs, err = store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"type": "C"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 document with type=C, got %d", len(docs))
	}
	if docs[0].Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, docs[0].Title)
	}
	if docs[0].Dimensions["_data.category"] != "processed" {
		t.Errorf("expected category 'processed', got %v", docs[0].Dimensions["_data.category"])
	}
}
