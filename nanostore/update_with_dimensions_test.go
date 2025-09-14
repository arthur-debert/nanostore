package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestUpdateWithDimensions(t *testing.T) {
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "normal", "high"},
				Prefixes:     map[string]string{"high": "h", "low": "l"},
				DefaultValue: "normal",
			},
			{
				Name:         "category",
				Type:         nanostore.Enumerated,
				Values:       []string{"work", "personal", "shopping"},
				Prefixes:     map[string]string{"work": "w", "shopping": "s"},
				DefaultValue: "personal",
			},
		},
	}

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a document with default dimensions
	docID, err := store.Add("My Task", nil, nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Update with new dimension values
	err = store.Update(docID, nanostore.UpdateRequest{
		Dimensions: map[string]string{
			"priority": "high",
			"category": "work",
		},
	})
	if err != nil {
		t.Fatalf("failed to update dimensions: %v", err)
	}

	// List and verify the ID changed
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	// Should have high priority and work category prefixes
	// Prefixes are alphabetically ordered: h (high) w (work) = wh1
	if docs[0].UserFacingID != "wh1" {
		t.Errorf("expected ID 'wh1' after update, got %s", docs[0].UserFacingID)
	}

	// Test updating with invalid dimension value
	err = store.Update(docID, nanostore.UpdateRequest{
		Dimensions: map[string]string{
			"priority": "urgent", // invalid
		},
	})
	if err == nil {
		t.Error("expected error for invalid dimension value")
	}
}
