package nanostore_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestUpdateWithSmartID(t *testing.T) {
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
				Values:       []string{"draft", "published"},
				DefaultValue: "draft",
			},
			{
				Name:         "category",
				Type:         nanostore.Enumerated,
				Values:       []string{"personal", "work"},
				DefaultValue: "personal",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a document
	uuid1, err := store.Add("Document 1", map[string]interface{}{
		"status":   "draft",
		"category": "personal",
	})
	if err != nil {
		t.Fatal(err)
	}

	// List to get the simple ID
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}
	simpleID := docs[0].SimpleID

	// Test 1: Update using UUID should still work
	newTitle := "Updated via UUID"
	err = store.Update(uuid1, nanostore.UpdateRequest{
		Title: &newTitle,
	})
	if err != nil {
		t.Errorf("Update with UUID failed: %v", err)
	}

	// Verify update
	docs, err = store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if docs[0].Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, docs[0].Title)
	}

	// Test 2: Update using SimpleID should work
	newTitle2 := "Updated via SimpleID"
	err = store.Update(simpleID, nanostore.UpdateRequest{
		Title: &newTitle2,
	})
	if err != nil {
		t.Errorf("Update with SimpleID failed: %v", err)
	}

	// Verify update
	docs, err = store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if docs[0].Title != newTitle2 {
		t.Errorf("expected title %q, got %q", newTitle2, docs[0].Title)
	}

	// Test 3: Update with invalid ID should fail
	err = store.Update("invalid-id", nanostore.UpdateRequest{
		Title: &newTitle,
	})
	if err == nil {
		t.Error("expected error for invalid ID, got nil")
	}
}

func TestUpdateDimensionsWithSmartID(t *testing.T) {
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
				Values:       []string{"draft", "published"},
				DefaultValue: "draft",
			},
			{
				Name:         "category",
				Type:         nanostore.Enumerated,
				Values:       []string{"personal", "work"},
				DefaultValue: "personal",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a document
	_, err = store.Add("Test Doc", map[string]interface{}{
		"status":   "draft",
		"category": "work",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get the simple ID
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	simpleID := docs[0].SimpleID

	// Update dimensions using SimpleID
	err = store.Update(simpleID, nanostore.UpdateRequest{
		Dimensions: map[string]interface{}{
			"status":   "published",
			"category": "personal",
		},
	})
	if err != nil {
		t.Errorf("Update dimensions with SimpleID failed: %v", err)
	}

	// Verify update
	docs, err = store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if docs[0].Dimensions["status"] != "published" {
		t.Errorf("expected status 'published', got %v", docs[0].Dimensions["status"])
	}
	if docs[0].Dimensions["category"] != "personal" {
		t.Errorf("expected category 'personal', got %v", docs[0].Dimensions["category"])
	}
}
