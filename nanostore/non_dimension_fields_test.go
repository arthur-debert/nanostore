package nanostore_test

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

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/testutil"
)

// Test struct with both dimension and non-dimension fields
type MixedFieldsItemMigrated struct {
	nanostore.Document

	// Dimension fields
	Status   string `values:"pending,active,done" default:"pending"`
	Priority string `values:"low,medium,high" default:"medium"`

	// Non-dimension fields that should be preserved
	Description string
	Count       int
	Score       float64
	IsActive    bool
	Tags        string // Would be []string in real app, but keeping simple
	CreatedBy   string
	Metadata    string
}

func TestNonDimensionFieldsPreservedMigrated(t *testing.T) {
	// Note: This test specifically tests typed store behavior that requires
	// a custom type, so we create a separate typed store rather than using
	// the fixture's direct store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create typed store
	store, err := nanostore.NewFromType[MixedFieldsItemMigrated](tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to create typed store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("CreateAndRetrieveWithNonDimensionFields", func(t *testing.T) {
		// Create item with all fields populated
		item := &MixedFieldsItemMigrated{
			Status:      "active",
			Priority:    "high",
			Description: "This is a test item with mixed fields",
			Count:       42,
			Score:       98.5,
			IsActive:    true,
			Tags:        "test,important,verified",
			CreatedBy:   "test-user",
			Metadata:    "extra-info",
		}

		id, err := store.Create("Test with all fields", item)
		if err != nil {
			t.Fatalf("failed to create: %v", err)
		}

		// Retrieve and verify all fields are preserved
		retrieved, err := store.Get(id)
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}

		// Check dimension fields
		if retrieved.Status != item.Status {
			t.Errorf("Status: expected %s, got %s", item.Status, retrieved.Status)
		}
		if retrieved.Priority != item.Priority {
			t.Errorf("Priority: expected %s, got %s", item.Priority, retrieved.Priority)
		}

		// Check non-dimension fields
		if retrieved.Description != item.Description {
			t.Errorf("Description: expected %s, got %s", item.Description, retrieved.Description)
		}
		if retrieved.Count != item.Count {
			t.Errorf("Count: expected %d, got %d", item.Count, retrieved.Count)
		}
		if retrieved.Score != item.Score {
			t.Errorf("Score: expected %f, got %f", item.Score, retrieved.Score)
		}
		if retrieved.IsActive != item.IsActive {
			t.Errorf("IsActive: expected %v, got %v", item.IsActive, retrieved.IsActive)
		}
		if retrieved.Tags != item.Tags {
			t.Errorf("Tags: expected %s, got %s", item.Tags, retrieved.Tags)
		}
		if retrieved.CreatedBy != item.CreatedBy {
			t.Errorf("CreatedBy: expected %s, got %s", item.CreatedBy, retrieved.CreatedBy)
		}
		if retrieved.Metadata != item.Metadata {
			t.Errorf("Metadata: expected %s, got %s", item.Metadata, retrieved.Metadata)
		}
	})

	t.Run("UpdateNonDimensionFields", func(t *testing.T) {
		// Create initial item
		item := &MixedFieldsItemMigrated{
			Status:      "pending",
			Description: "Initial description",
			Count:       10,
		}

		id, err := store.Create("Update test", item)
		if err != nil {
			t.Fatalf("failed to create: %v", err)
		}

		// Update with new values
		update := &MixedFieldsItemMigrated{
			Status:      "active", // dimension field
			Description: "Updated description",
			Count:       20,
			Score:       75.5,
			IsActive:    true,
		}

		err = store.Update(id, update)
		if err != nil {
			t.Fatalf("failed to update: %v", err)
		}

		// Retrieve and verify updates
		retrieved, err := store.Get(id)
		if err != nil {
			t.Fatalf("failed to get after update: %v", err)
		}

		if retrieved.Status != "active" {
			t.Errorf("Status not updated: expected active, got %s", retrieved.Status)
		}
		if retrieved.Description != "Updated description" {
			t.Errorf("Description not updated: expected 'Updated description', got %s", retrieved.Description)
		}
		if retrieved.Count != 20 {
			t.Errorf("Count not updated: expected 20, got %d", retrieved.Count)
		}
		if retrieved.Score != 75.5 {
			t.Errorf("Score not updated: expected 75.5, got %f", retrieved.Score)
		}
		if !retrieved.IsActive {
			t.Error("IsActive not updated: expected true, got false")
		}
	})

	t.Run("QueryIgnoresNonDimensionFields", func(t *testing.T) {
		// Create items with same non-dimension values but different dimensions
		item1 := &MixedFieldsItemMigrated{
			Status:      "active",
			Description: "Same description",
			Count:       100,
		}
		item2 := &MixedFieldsItemMigrated{
			Status:      "pending",
			Description: "Same description",
			Count:       100,
		}

		_, err := store.Create("Item 1", item1)
		if err != nil {
			t.Fatalf("failed to create item 1: %v", err)
		}
		_, err = store.Create("Item 2", item2)
		if err != nil {
			t.Fatalf("failed to create item 2: %v", err)
		}

		// Query by dimension should work
		activeItems, err := store.Query().Status("active").Find()
		if err != nil {
			t.Fatalf("failed to query by status: %v", err)
		}

		// Should find only the active item
		found := false
		for _, item := range activeItems {
			if item.Title == "Item 1" && item.Status == "active" {
				found = true
			}
		}
		if !found {
			t.Error("failed to find active item through query")
		}
	})

	t.Run("ZeroValuesNotStored", func(t *testing.T) {
		// Create item with only some fields set
		item := &MixedFieldsItemMigrated{
			Status:      "active",
			Description: "Only description is set",
			// All numeric fields left as zero
		}

		id, err := store.Create("Sparse item", item)
		if err != nil {
			t.Fatalf("failed to create: %v", err)
		}

		retrieved, err := store.Get(id)
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}

		// Non-zero fields should be preserved
		if retrieved.Description != "Only description is set" {
			t.Errorf("Description not preserved: got %s", retrieved.Description)
		}

		// Zero values should remain zero (not stored, so default on unmarshal)
		if retrieved.Count != 0 {
			t.Errorf("Count should be 0, got %d", retrieved.Count)
		}
		if retrieved.Score != 0.0 {
			t.Errorf("Score should be 0.0, got %f", retrieved.Score)
		}
		if retrieved.IsActive != false {
			t.Errorf("IsActive should be false, got %v", retrieved.IsActive)
		}
	})
}

func TestNonDimensionFieldTypesMigrated(t *testing.T) {
	// Test various field types are handled correctly
	type ComplexFieldsItemMigrated struct {
		nanostore.Document

		// Dimension
		Type string `values:"A,B,C" default:"A"`

		// Various non-dimension types
		IntField    int
		Int64Field  int64
		Float32     float32
		Float64     float64
		BoolField   bool
		TimeField   time.Time
		StringField string
	}

	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := nanostore.NewFromType[ComplexFieldsItemMigrated](tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	now := time.Now().Round(time.Second) // Round to avoid nanosecond precision issues
	item := &ComplexFieldsItemMigrated{
		Type:        "B",
		IntField:    -42,
		Int64Field:  9223372036854775807, // max int64
		Float32:     3.14159,
		Float64:     2.71828,
		BoolField:   true,
		TimeField:   now,
		StringField: "test string",
	}

	id, err := store.Create("Complex types test", item)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	retrieved, err := store.Get(id)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	// Verify all types are preserved correctly
	if retrieved.IntField != item.IntField {
		t.Errorf("IntField: expected %d, got %d", item.IntField, retrieved.IntField)
	}
	if retrieved.Int64Field != item.Int64Field {
		t.Errorf("Int64Field: expected %d, got %d", item.Int64Field, retrieved.Int64Field)
	}
	if retrieved.Float32 != item.Float32 {
		t.Errorf("Float32: expected %f, got %f", item.Float32, retrieved.Float32)
	}
	if retrieved.Float64 != item.Float64 {
		t.Errorf("Float64: expected %f, got %f", item.Float64, retrieved.Float64)
	}
	if retrieved.BoolField != item.BoolField {
		t.Errorf("BoolField: expected %v, got %v", item.BoolField, retrieved.BoolField)
	}
	if !retrieved.TimeField.Equal(item.TimeField) {
		t.Errorf("TimeField: expected %v, got %v", item.TimeField, retrieved.TimeField)
	}
	if retrieved.StringField != item.StringField {
		t.Errorf("StringField: expected %s, got %s", item.StringField, retrieved.StringField)
	}
}

// TestNonDimensionFieldsWithFixtureMigrated tests interaction with fixture data
func TestNonDimensionFieldsWithFixtureMigrated(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("VerifyFixtureNonDimensionData", func(t *testing.T) {
		// The fixture documents have body content which is non-dimension data
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.TeamMeeting.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		if len(docs) != 1 {
			t.Fatalf("expected 1 document, got %d", len(docs))
		}

		doc := docs[0]
		if doc.Body == "" {
			t.Error("expected TeamMeeting to have body content")
		}
	})

	t.Run("UpdateNonDimensionDataInFixture", func(t *testing.T) {
		// Update body content for a fixture document
		newBody := "Updated meeting notes with important decisions"
		err := store.Update(universe.TeamMeeting.UUID, nanostore.UpdateRequest{
			Body: &newBody,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Verify update
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.TeamMeeting.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		if len(docs) != 1 {
			t.Fatalf("expected 1 document, got %d", len(docs))
		}

		if docs[0].Body != newBody {
			t.Errorf("expected body to be updated, got %q", docs[0].Body)
		}
	})
}
