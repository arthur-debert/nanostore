package api_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)

import (
	"os"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore/api"
)

func TestSetTimeFunc(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.New[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("SetFixedTime", func(t *testing.T) {
		// Set a fixed time for deterministic testing
		fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		err := store.SetTimeFunc(func() time.Time { return fixedTime })
		if err != nil {
			t.Fatalf("failed to set time function: %v", err)
		}

		// Create a document and verify it has the fixed timestamp
		id, err := store.Create("Fixed Time Test", &TodoItem{
			Status:   "pending",
			Priority: "medium",
			Activity: "active",
		})
		if err != nil {
			t.Fatalf("failed to create document: %v", err)
		}

		// Get the document and check its timestamp
		doc, err := store.GetRaw(id)
		if err != nil {
			t.Fatalf("failed to get document: %v", err)
		}

		if !doc.CreatedAt.Equal(fixedTime) {
			t.Errorf("expected CreatedAt to be %v, got %v", fixedTime, doc.CreatedAt)
		}
		if !doc.UpdatedAt.Equal(fixedTime) {
			t.Errorf("expected UpdatedAt to be %v, got %v", fixedTime, doc.UpdatedAt)
		}

		t.Logf("Document created with fixed time: %v", doc.CreatedAt)
	})

	t.Run("SetSequentialTimes", func(t *testing.T) {
		// Use fixed sequential times for each document
		baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		// Create multiple documents with different fixed times
		var docIDs []string
		var expectedTimes []time.Time

		for i := 0; i < 3; i++ {
			// Set a specific time for this document
			documentTime := baseTime.Add(time.Duration(i+1) * time.Hour)
			err := store.SetTimeFunc(func() time.Time { return documentTime })
			if err != nil {
				t.Fatalf("failed to set time function for doc %d: %v", i, err)
			}

			id, err := store.Create("Sequential Test", &TodoItem{
				Status:   "pending",
				Priority: "medium",
				Activity: "active",
			})
			if err != nil {
				t.Fatalf("failed to create document %d: %v", i, err)
			}

			docIDs = append(docIDs, id)
			expectedTimes = append(expectedTimes, documentTime)
		}

		// Verify each document has the expected timestamp
		for i, id := range docIDs {
			doc, err := store.GetRaw(id)
			if err != nil {
				t.Fatalf("failed to get document %d: %v", i, err)
			}

			if !doc.CreatedAt.Equal(expectedTimes[i]) {
				t.Errorf("document %d: expected CreatedAt %v, got %v",
					i, expectedTimes[i], doc.CreatedAt)
			}

			t.Logf("Document %d created at: %v", i, doc.CreatedAt)
		}
	})

	t.Run("ResetToSystemTime", func(t *testing.T) {
		// Set to time.Now to revert to system time
		err := store.SetTimeFunc(time.Now)
		if err != nil {
			t.Fatalf("failed to reset time function: %v", err)
		}

		// Record time before creating document
		beforeCreate := time.Now()

		// Create document with system time
		id, err := store.Create("System Time Test", &TodoItem{
			Status:   "pending",
			Priority: "medium",
			Activity: "active",
		})
		if err != nil {
			t.Fatalf("failed to create document: %v", err)
		}

		// Record time after creating document
		afterCreate := time.Now()

		// Get document and verify timestamp is within expected range
		doc, err := store.GetRaw(id)
		if err != nil {
			t.Fatalf("failed to get document: %v", err)
		}

		if doc.CreatedAt.Before(beforeCreate) || doc.CreatedAt.After(afterCreate) {
			t.Errorf("document timestamp %v not between %v and %v",
				doc.CreatedAt, beforeCreate, afterCreate)
		}

		t.Logf("Document created with system time: %v (range: %v to %v)",
			doc.CreatedAt, beforeCreate, afterCreate)
	})

	t.Run("TimeAffectsUpdates", func(t *testing.T) {
		// Create a document first
		id, err := store.Create("Update Time Test", &TodoItem{
			Status:   "pending",
			Priority: "medium",
			Activity: "active",
		})
		if err != nil {
			t.Fatalf("failed to create document: %v", err)
		}

		// Set a specific time for the update
		updateTime := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
		err = store.SetTimeFunc(func() time.Time { return updateTime })
		if err != nil {
			t.Fatalf("failed to set time function: %v", err)
		}

		// Update the document
		_, err = store.Update(id, &TodoItem{
			Status:   "active",
			Priority: "high",
			Activity: "active",
		})
		if err != nil {
			t.Fatalf("failed to update document: %v", err)
		}

		// Verify the UpdatedAt timestamp
		doc, err := store.GetRaw(id)
		if err != nil {
			t.Fatalf("failed to get document: %v", err)
		}

		if !doc.UpdatedAt.Equal(updateTime) {
			t.Errorf("expected UpdatedAt to be %v, got %v", updateTime, doc.UpdatedAt)
		}

		t.Logf("Document updated at: %v", doc.UpdatedAt)
	})

	t.Run("TimeAffectsOrdering", func(t *testing.T) {
		// Create documents with specific times to test ordering
		times := []time.Time{
			time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), // Third
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // First
			time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), // Second
		}

		var docIDs []string
		for i, fixedTime := range times {
			err := store.SetTimeFunc(func() time.Time { return fixedTime })
			if err != nil {
				t.Fatalf("failed to set time function for doc %d: %v", i, err)
			}

			id, err := store.Create("Ordering Test", &TodoItem{
				Status:   "pending",
				Priority: "medium",
				Activity: "active",
			})
			if err != nil {
				t.Fatalf("failed to create document %d: %v", i, err)
			}

			docIDs = append(docIDs, id)
		}

		// Get the specific documents we created and verify their timestamps
		var actualTimes []time.Time
		for _, id := range docIDs {
			doc, err := store.GetRaw(id)
			if err != nil {
				t.Fatalf("failed to get document: %v", err)
			}
			actualTimes = append(actualTimes, doc.CreatedAt)
		}

		// Verify the times match what we set
		expectedOrder := []time.Time{
			time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), // Third
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // First
			time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), // Second
		}

		for i, actualTime := range actualTimes {
			if !actualTime.Equal(expectedOrder[i]) {
				t.Errorf("document %d: expected time %v, got %v",
					i, expectedOrder[i], actualTime)
			}
		}

		// Now test that Query().OrderBy works correctly by creating fresh documents
		// in a separate test store to avoid interference
		tmpfile2, err := os.CreateTemp("", "test*.json")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer func() { _ = os.Remove(tmpfile2.Name()) }()
		_ = tmpfile2.Close()

		orderStore, err := api.New[TodoItem](tmpfile2.Name())
		if err != nil {
			t.Fatalf("failed to create order test store: %v", err)
		}
		defer func() { _ = orderStore.Close() }()

		// Create documents in non-chronological order but with chronological times
		orderTimes := []time.Time{
			time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), // First chronologically
			time.Date(2024, 2, 3, 0, 0, 0, 0, time.UTC), // Third chronologically
			time.Date(2024, 2, 2, 0, 0, 0, 0, time.UTC), // Second chronologically
		}

		for i, orderTime := range orderTimes {
			err := orderStore.SetTimeFunc(func() time.Time { return orderTime })
			if err != nil {
				t.Fatalf("failed to set time for order test %d: %v", i, err)
			}

			_, err = orderStore.Create("Order Test", &TodoItem{
				Status:   "pending",
				Priority: "medium",
				Activity: "active",
			})
			if err != nil {
				t.Fatalf("failed to create order document %d: %v", i, err)
			}
		}

		// Query with ordering
		orderedDocs, err := orderStore.Query().OrderBy("created_at").Find()
		if err != nil {
			t.Fatalf("failed to query ordered documents: %v", err)
		}

		// Verify chronological order
		expectedChronOrder := []time.Time{
			time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), // First
			time.Date(2024, 2, 2, 0, 0, 0, 0, time.UTC), // Second
			time.Date(2024, 2, 3, 0, 0, 0, 0, time.UTC), // Third
		}

		for i, doc := range orderedDocs {
			if !doc.CreatedAt.Equal(expectedChronOrder[i]) {
				t.Errorf("ordered doc %d: expected time %v, got %v",
					i, expectedChronOrder[i], doc.CreatedAt)
			}
		}

		t.Logf("Documents correctly ordered by creation time")
	})
}

func TestSetTimeFuncEdgeCases(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.New[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("MultipleTimeChanges", func(t *testing.T) {
		// Change time function multiple times
		times := []time.Time{
			time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		for i, fixedTime := range times {
			err := store.SetTimeFunc(func() time.Time { return fixedTime })
			if err != nil {
				t.Fatalf("failed to set time function %d: %v", i, err)
			}

			id, err := store.Create("Multi Time Test", &TodoItem{
				Status:   "pending",
				Priority: "medium",
				Activity: "active",
			})
			if err != nil {
				t.Fatalf("failed to create document %d: %v", i, err)
			}

			doc, err := store.GetRaw(id)
			if err != nil {
				t.Fatalf("failed to get document %d: %v", i, err)
			}

			if !doc.CreatedAt.Equal(fixedTime) {
				t.Errorf("document %d: expected time %v, got %v",
					i, fixedTime, doc.CreatedAt)
			}
		}
	})

	t.Run("ZeroTime", func(t *testing.T) {
		// Test with zero time
		zeroTime := time.Time{}
		err := store.SetTimeFunc(func() time.Time { return zeroTime })
		if err != nil {
			t.Fatalf("failed to set zero time function: %v", err)
		}

		id, err := store.Create("Zero Time Test", &TodoItem{
			Status:   "pending",
			Priority: "medium",
			Activity: "active",
		})
		if err != nil {
			t.Fatalf("failed to create document: %v", err)
		}

		doc, err := store.GetRaw(id)
		if err != nil {
			t.Fatalf("failed to get document: %v", err)
		}

		if !doc.CreatedAt.Equal(zeroTime) {
			t.Errorf("expected zero time, got %v", doc.CreatedAt)
		}
	})

	t.Run("FutureTime", func(t *testing.T) {
		// Test with future time
		futureTime := time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC)
		err := store.SetTimeFunc(func() time.Time { return futureTime })
		if err != nil {
			t.Fatalf("failed to set future time function: %v", err)
		}

		id, err := store.Create("Future Time Test", &TodoItem{
			Status:   "pending",
			Priority: "medium",
			Activity: "active",
		})
		if err != nil {
			t.Fatalf("failed to create document: %v", err)
		}

		doc, err := store.GetRaw(id)
		if err != nil {
			t.Fatalf("failed to get document: %v", err)
		}

		if !doc.CreatedAt.Equal(futureTime) {
			t.Errorf("expected future time %v, got %v", futureTime, doc.CreatedAt)
		}
	})

	t.Run("TimeZoneHandling", func(t *testing.T) {
		// Test with different time zones
		locations := []*time.Location{
			time.UTC,
			time.FixedZone("EST", -5*3600),
			time.FixedZone("PST", -8*3600),
			time.FixedZone("JST", 9*3600),
		}

		baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

		for i, loc := range locations {
			zonedTime := baseTime.In(loc)
			err := store.SetTimeFunc(func() time.Time { return zonedTime })
			if err != nil {
				t.Fatalf("failed to set time function for zone %d: %v", i, err)
			}

			id, err := store.Create("Timezone Test", &TodoItem{
				Status:   "pending",
				Priority: "medium",
				Activity: "active",
			})
			if err != nil {
				t.Fatalf("failed to create document %d: %v", i, err)
			}

			doc, err := store.GetRaw(id)
			if err != nil {
				t.Fatalf("failed to get document %d: %v", i, err)
			}

			if !doc.CreatedAt.Equal(zonedTime) {
				t.Errorf("document %d: expected time %v, got %v",
					i, zonedTime, doc.CreatedAt)
			}

			t.Logf("Document %d created in %s: %v", i, loc, doc.CreatedAt)
		}
	})
}
