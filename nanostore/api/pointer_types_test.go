package api

import (
	"os"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
)

// TestPointerType represents a comprehensive test struct with all supported pointer types
type TestPointerType struct {
	nanostore.Document

	// Enumerated dimension
	Status string `values:"pending,active,done" default:"pending"`

	// Pointer type dimensions (limit to 6 to stay under the 7 dimension limit)
	DeletedAt  *time.Time `dimension:"deleted_at"`
	Score      *float64   `dimension:"score"`
	Priority   *int       `dimension:"priority"`
	IsArchived *bool      `dimension:"is_archived"`
	Notes      *string    `dimension:"notes"`

	// Data fields (non-dimensions)
	Description string
	Metadata    map[string]string
}

func TestPointerTypeSupport(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_pointer_types*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := NewFromType[TestPointerType](tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	now := time.Now()
	score := 85.5
	priority := 3
	isArchived := true
	notes := "Important notes"

	t.Run("CreateWithPointerValues", func(t *testing.T) {
		item := &TestPointerType{
			Status:      "active",
			DeletedAt:   &now,        // non-nil time.Time
			Score:       &score,      // non-nil float64
			Priority:    &priority,   // non-nil int
			IsArchived:  &isArchived, // non-nil bool
			Notes:       &notes,      // non-nil string
			Description: "Test description",
		}

		id, err := store.Create("Test Pointer Item", item)
		if err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}

		// Retrieve the item
		retrieved, err := store.Get(id)
		if err != nil {
			t.Fatalf("Failed to retrieve item: %v", err)
		}

		// Verify non-nil values
		if retrieved.Status != "active" {
			t.Errorf("Expected Status 'active', got '%s'", retrieved.Status)
		}

		if retrieved.DeletedAt == nil {
			t.Error("Expected DeletedAt to be non-nil")
		} else {
			// Compare times with truncated precision due to RFC3339 formatting
			expectedTime := now.Truncate(time.Second)
			actualTime := retrieved.DeletedAt.Truncate(time.Second)
			if !expectedTime.Equal(actualTime) {
				t.Errorf("Expected DeletedAt %v, got %v", expectedTime, actualTime)
			}
		}

		if retrieved.Score == nil {
			t.Error("Expected Score to be non-nil")
		} else if *retrieved.Score != score {
			t.Errorf("Expected Score %v, got %v", score, *retrieved.Score)
		}

		if retrieved.Priority == nil {
			t.Error("Expected Priority to be non-nil")
		} else if *retrieved.Priority != priority {
			t.Errorf("Expected Priority %v, got %v", priority, *retrieved.Priority)
		}

		if retrieved.IsArchived == nil {
			t.Error("Expected IsArchived to be non-nil")
		} else if *retrieved.IsArchived != isArchived {
			t.Errorf("Expected IsArchived %v, got %v", isArchived, *retrieved.IsArchived)
		}

		if retrieved.Notes == nil {
			t.Error("Expected Notes to be non-nil")
		} else if *retrieved.Notes != notes {
			t.Errorf("Expected Notes '%s', got '%s'", notes, *retrieved.Notes)
		}

		// Note: All pointer fields in this test have non-nil values

		// Verify data field
		if retrieved.Description != "Test description" {
			t.Errorf("Expected Description 'Test description', got '%s'", retrieved.Description)
		}
	})

	t.Run("CreateWithAllNilPointers", func(t *testing.T) {
		item := &TestPointerType{
			Status:      "pending", // Use default
			DeletedAt:   nil,
			Score:       nil,
			Priority:    nil,
			IsArchived:  nil,
			Notes:       nil,
			Description: "All nil test",
		}

		id, err := store.Create("Test All Nil", item)
		if err != nil {
			t.Fatalf("Failed to create item with nil pointers: %v", err)
		}

		// Retrieve the item
		retrieved, err := store.Get(id)
		if err != nil {
			t.Fatalf("Failed to retrieve item: %v", err)
		}

		// All pointer fields should be nil
		if retrieved.DeletedAt != nil {
			t.Errorf("Expected DeletedAt to be nil, got %v", *retrieved.DeletedAt)
		}
		if retrieved.Score != nil {
			t.Errorf("Expected Score to be nil, got %v", *retrieved.Score)
		}
		if retrieved.Priority != nil {
			t.Errorf("Expected Priority to be nil, got %v", *retrieved.Priority)
		}
		if retrieved.IsArchived != nil {
			t.Errorf("Expected IsArchived to be nil, got %v", *retrieved.IsArchived)
		}
		if retrieved.Notes != nil {
			t.Errorf("Expected Notes to be nil, got '%s'", *retrieved.Notes)
		}
	})

	t.Run("UpdatePointerValues", func(t *testing.T) {
		item := &TestPointerType{
			Status: "active",
		}

		id, err := store.Create("Test Update", item)
		if err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}

		// Update with pointer values
		newScore := 95.0
		newPriority := 1
		newNotes := "Updated notes"

		updates := &TestPointerType{
			Score:    &newScore,
			Priority: &newPriority,
			Notes:    &newNotes,
		}

		_, err = store.Update(id, updates)
		if err != nil {
			t.Fatalf("Failed to update item: %v", err)
		}

		// Retrieve and verify
		retrieved, err := store.Get(id)
		if err != nil {
			t.Fatalf("Failed to retrieve updated item: %v", err)
		}

		if retrieved.Score == nil || *retrieved.Score != newScore {
			t.Errorf("Expected Score %v, got %v", newScore, retrieved.Score)
		}
		if retrieved.Priority == nil || *retrieved.Priority != newPriority {
			t.Errorf("Expected Priority %v, got %v", newPriority, retrieved.Priority)
		}
		if retrieved.Notes == nil || *retrieved.Notes != newNotes {
			t.Errorf("Expected Notes '%s', got %v", newNotes, retrieved.Notes)
		}
	})

	t.Run("QueryByPointerDimensions", func(t *testing.T) {
		// Create a fresh store for this test to avoid interference
		tmpfile2, err := os.CreateTemp("", "test_pointer_query*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile2.Name()) }()
		_ = tmpfile2.Close()

		queryStore, err := NewFromType[TestPointerType](tmpfile2.Name())
		if err != nil {
			t.Fatalf("Failed to create query store: %v", err)
		}
		defer func() { _ = queryStore.Close() }()

		// Create items with different pointer values
		highScore := 90.0
		lowScore := 60.0
		archived := true
		active := false

		item1 := &TestPointerType{
			Status:     "active",
			Score:      &highScore,
			IsArchived: &active,
		}
		item2 := &TestPointerType{
			Status:     "done",
			Score:      &lowScore,
			IsArchived: &archived,
		}

		_, err = queryStore.Create("High Score Item", item1)
		if err != nil {
			t.Fatalf("Failed to create item1: %v", err)
		}

		_, err = queryStore.Create("Low Score Item", item2)
		if err != nil {
			t.Fatalf("Failed to create item2: %v", err)
		}

		// Query by pointer dimension
		results, err := queryStore.Query().
			Where("score = ?", "90").
			Find()
		if err != nil {
			t.Fatalf("Failed to query by score: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result for score=90, got %d", len(results))
		} else if results[0].Title != "High Score Item" {
			t.Errorf("Expected 'High Score Item', got '%s'", results[0].Title)
		}

		// Query by boolean pointer dimension
		results, err = queryStore.Query().
			Where("is_archived = ?", "true").
			Find()
		if err != nil {
			t.Fatalf("Failed to query by is_archived: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result for is_archived=true, got %d", len(results))
		} else if results[0].Title != "Low Score Item" {
			t.Errorf("Expected 'Low Score Item', got '%s'", results[0].Title)
		}
	})
}

func TestPointerTypeMarshalingEdgeCases(t *testing.T) {
	t.Run("TimeFormatting", func(t *testing.T) {
		// Test specific time formatting
		specificTime := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)

		item := &TestPointerType{
			Status:    "active",
			DeletedAt: &specificTime,
		}

		dims, _, err := MarshalDimensions(item)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		expectedTimeStr := "2023-12-25T15:30:45Z"
		if dims["deleted_at"] != expectedTimeStr {
			t.Errorf("Expected time string '%s', got '%s'", expectedTimeStr, dims["deleted_at"])
		}
	})

	t.Run("NilPointerMarshal", func(t *testing.T) {
		item := &TestPointerType{
			Status:    "pending",
			DeletedAt: nil,
			Score:     nil,
		}

		dims, _, err := MarshalDimensions(item)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		// Only status should be present (nil pointers should be skipped)
		if len(dims) != 1 {
			t.Errorf("Expected 1 dimension, got %d: %v", len(dims), dims)
		}

		if dims["status"] != "pending" {
			t.Errorf("Expected status 'pending', got '%v'", dims["status"])
		}
	})
}
