package nanostore_test

import (
	"os"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestDateTimeFiltering(t *testing.T) {
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
				Values:       []string{"todo", "done"},
				DefaultValue: "todo",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Get test store interface
	testStore := nanostore.AsTestStore(store)
	if testStore == nil {
		t.Fatal("store doesn't support testing features")
	}

	// Use deterministic timestamps
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	currentTime := baseTime

	// Set time function that increments by 1 minute each call
	testStore.SetTimeFunc(func() time.Time {
		t := currentTime
		currentTime = currentTime.Add(1 * time.Minute)
		return t
	})

	// Add documents with different timestamps
	doc1ID, _ := store.Add("First task", nil)  // 10:00
	doc2ID, _ := store.Add("Second task", nil) // 10:01  
	doc3ID, _ := store.Add("Third task", nil)  // 10:02

	// Get all documents to check their timestamps
	allDocs, _ := store.List(nanostore.ListOptions{})
	
	var doc1, doc2, doc3 nanostore.Document
	for _, doc := range allDocs {
		switch doc.UUID {
		case doc1ID:
			doc1 = doc
		case doc2ID:
			doc2 = doc
		case doc3ID:
			doc3 = doc
		}
	}

	t.Run("FilterByCreatedAt", func(t *testing.T) {
		// Filter by exact created_at timestamp
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"created_at": doc2.CreatedAt,
			},
		})
		if err != nil {
			t.Fatalf("failed to filter by created_at: %v", err)
		}

		if len(docs) != 1 {
			t.Errorf("expected 1 document, got %d", len(docs))
		}
		if len(docs) > 0 && docs[0].UUID != doc2ID {
			t.Errorf("expected doc2, got %s", docs[0].UUID)
		}
	})

	t.Run("FilterByCreatedAtString", func(t *testing.T) {
		// Filter using string representation of time
		timeStr := doc1.CreatedAt.Format(time.RFC3339Nano)
		
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"created_at": timeStr,
			},
		})
		if err != nil {
			t.Fatalf("failed to filter by created_at string: %v", err)
		}

		if len(docs) != 1 {
			t.Errorf("expected 1 document, got %d", len(docs))
		}
		if len(docs) > 0 && docs[0].UUID != doc1ID {
			t.Errorf("expected doc1, got %s", docs[0].UUID)
		}
	})

	t.Run("FilterByUpdatedAt", func(t *testing.T) {
		// Update a document to change its updated_at
		newTitle := "Updated second task"
		err := store.Update(doc2ID, nanostore.UpdateRequest{
			Title: &newTitle,
		})
		if err != nil {
			t.Fatalf("failed to update document: %v", err)
		}

		// Get the updated document
		docs, _ := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": doc2ID,
			},
		})
		updatedDoc := docs[0]

		// Filter by updated_at
		docs, err = store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"updated_at": updatedDoc.UpdatedAt,
			},
		})
		if err != nil {
			t.Fatalf("failed to filter by updated_at: %v", err)
		}

		if len(docs) != 1 {
			t.Errorf("expected 1 document, got %d", len(docs))
		}
		if len(docs) > 0 && docs[0].UUID != doc2ID {
			t.Errorf("expected doc2, got %s", docs[0].UUID)
		}
	})

	t.Run("FilterByDateTimeInSlice", func(t *testing.T) {
		// Filter using time values in a slice
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"created_at": []interface{}{doc1.CreatedAt, doc3.CreatedAt},
			},
		})
		if err != nil {
			t.Fatalf("failed to filter by datetime slice: %v", err)
		}

		if len(docs) != 2 {
			t.Errorf("expected 2 documents, got %d", len(docs))
		}

		// Verify we got the right documents
		foundDoc1 := false
		foundDoc3 := false
		for _, doc := range docs {
			if doc.UUID == doc1ID {
				foundDoc1 = true
			}
			if doc.UUID == doc3ID {
				foundDoc3 = true
			}
		}

		if !foundDoc1 || !foundDoc3 {
			t.Error("didn't find expected documents")
		}
	})

	t.Run("FilterByVariousDateFormats", func(t *testing.T) {
		// Test that various date string formats work
		baseTime := doc1.CreatedAt

		formats := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05Z",
		}

		for _, format := range formats {
			timeStr := baseTime.Format(format)
			
			// For date-only format, we need to match the date part
			if format == "2006-01-02" {
				// Skip date-only format in this exact match test
				continue
			}

			_, err := store.List(nanostore.ListOptions{
				Filters: map[string]interface{}{
					"created_at": timeStr,
				},
			})
			
			// Some formats lose precision, so we might not get exact matches
			if err != nil {
				t.Errorf("failed to filter with format %s: %v", format, err)
			}
		}
	})
}

func TestDateTimeConsistency(t *testing.T) {
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

	// Create store and add a document
	store1, err := nanostore.New(filename, config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	docID, _ := store1.Add("Test task", nil)
	
	// Get the document to check its timestamps
	docs, _ := store1.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"uuid": docID},
	})
	originalDoc := docs[0]
	
	_ = store1.Close()

	// Reopen the store
	store2, err := nanostore.New(filename, config)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	defer func() { _ = store2.Close() }()

	// Filter by the original created_at time
	docs, err = store2.List(nanostore.ListOptions{
		Filters: map[string]interface{}{
			"created_at": originalDoc.CreatedAt,
		},
	})
	if err != nil {
		t.Fatalf("failed to filter by created_at after reload: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("expected 1 document after reload, got %d", len(docs))
	}
	if len(docs) > 0 && docs[0].UUID != docID {
		t.Errorf("expected same document after reload")
	}
}