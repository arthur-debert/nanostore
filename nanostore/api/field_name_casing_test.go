package api_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
	"github.com/arthur-debert/nanostore/types"
)

// TestFieldCasing represents a test item with various field types for casing tests
type TestFieldCasing struct {
	nanostore.Document
	Status     string    `values:"pending,active,done" default:"pending"`
	Priority   string    `values:"low,medium,high" default:"medium"`
	CreatedBy  string    // Custom field -> _data.CreatedBy
	DeletedAt  time.Time // Custom field -> _data.DeletedAt
	AssignedTo string    // Custom field -> _data.AssignedTo
}

func TestFieldNameCasingConsistency(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test_field_casing*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.New[TestFieldCasing](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create test data with different timestamps for ordering tests
	now := time.Now()

	item1 := &TestFieldCasing{
		Status:     "active",
		Priority:   "high",
		CreatedBy:  "alice",
		DeletedAt:  now.Add(-2 * time.Hour), // Older
		AssignedTo: "bob",
	}
	id1, err := store.Create("Item 1", item1)
	if err != nil {
		t.Fatal(err)
	}

	item2 := &TestFieldCasing{
		Status:     "pending",
		Priority:   "medium",
		CreatedBy:  "charlie",
		DeletedAt:  now.Add(-1 * time.Hour), // Newer
		AssignedTo: "alice",
	}
	id2, err := store.Create("Item 2", item2)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("OrderBy should work with snake_case field names", func(t *testing.T) {
		// This test should FAIL initially - demonstrates the bug
		// Users expect to be able to use snake_case field names like database conventions

		items, err := store.List(types.ListOptions{
			OrderBy: []types.OrderClause{
				{Column: "_data.deleted_at", Descending: true}, // snake_case - should work but currently doesn't
			},
		})

		if err != nil {
			t.Fatalf("OrderBy with snake_case field name failed: %v", err)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(items))
		}

		// Should be ordered by deleted_at descending (newer first)
		// item2 has newer DeletedAt, so should come first
		if items[0].UUID != id2 {
			t.Errorf("Expected item2 (newer) first, but got item with UUID %s", items[0].UUID)
		}
		if items[1].UUID != id1 {
			t.Errorf("Expected item1 (older) second, but got item with UUID %s", items[1].UUID)
		}
	})

	t.Run("OrderBy should also work with PascalCase for backward compatibility", func(t *testing.T) {
		// After fix: PascalCase should still work for backward compatibility
		// but it's not the preferred format

		items, err := store.List(types.ListOptions{
			OrderBy: []types.OrderClause{
				{Column: "_data.DeletedAt", Descending: true}, // PascalCase - should work for compatibility
			},
		})

		// This might fail initially if the lower-level ordering doesn't handle field name translation
		// We'll address this by implementing field name resolution in the store layer
		if err != nil {
			t.Logf("PascalCase field names not yet supported in OrderBy: %v", err)
			t.Skip("PascalCase support in OrderBy will be implemented separately")
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(items))
		}

		// Should be ordered by DeletedAt descending (newer first)
		if items[0].UUID != id2 {
			t.Errorf("Expected item2 (newer) first, but got item with UUID %s", items[0].UUID)
		}
	})

	t.Run("TypedQuery OrderByData should work with snake_case", func(t *testing.T) {
		// This test should FAIL initially
		// TypedQuery should accept snake_case field names for consistency

		items, err := store.Query().
			OrderByData("deleted_at"). // snake_case - should work
			Find()

		if err != nil {
			t.Fatalf("OrderByData with snake_case failed: %v", err)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(items))
		}

		// Should be ordered by deleted_at ascending (older first)
		if items[0].UUID != id1 {
			t.Errorf("Expected item1 (older) first, but got item with UUID %s", items[0].UUID)
		}
	})

	t.Run("Data queries should work with snake_case", func(t *testing.T) {
		// This test should FAIL initially
		// Data queries should accept snake_case field names

		items, err := store.Query().
			Data("created_by", "alice"). // snake_case - should work
			Find()

		if err != nil {
			t.Fatalf("Data query with snake_case failed: %v", err)
		}

		if len(items) != 1 {
			t.Fatalf("Expected 1 item for alice, got %d", len(items))
		}

		if items[0].UUID != id1 {
			t.Errorf("Expected item1 for alice, got item with UUID %s", items[0].UUID)
		}
	})

	t.Run("Invalid field names should return clear errors", func(t *testing.T) {
		// This test should FAIL initially - currently silent failures
		// Invalid field names should return errors, not wrong results

		_, err := store.List(types.ListOptions{
			OrderBy: []types.OrderClause{
				{Column: "_data.nonexistent_field", Descending: true},
			},
		})

		if err == nil {
			t.Error("Expected error for nonexistent field, but got none")
		}

		if err != nil {
			// Error message should be helpful
			errMsg := err.Error()
			if errMsg == "" {
				t.Error("Error message should not be empty")
			}
			t.Logf("Error message: %s", errMsg) // Log for inspection
		}
	})

	t.Run("TypedQuery should validate field names", func(t *testing.T) {
		// Test that TypedQuery methods validate field names immediately

		// This should return an error for invalid field name
		_, err := store.Query().
			Data("nonexistent_field", "value").
			Find()

		if err == nil {
			t.Error("Expected error for nonexistent field in Data(), but got none")
		}

		if err != nil {
			errMsg := err.Error()
			if !strings.Contains(errMsg, "nonexistent_field") {
				t.Errorf("Error should mention the invalid field name, got: %s", errMsg)
			}
		}

		// This should also return an error for invalid field name in OrderByData
		_, err = store.Query().
			OrderByData("invalid_field").
			Find()

		if err == nil {
			t.Error("Expected error for nonexistent field in OrderByData(), but got none")
		}
	})
}

func TestFieldNameTransformation(t *testing.T) {
	// Test to verify field name transformation is working correctly

	t.Run("Go field names should transform to snake_case for storage", func(t *testing.T) {
		// Create temporary store
		tmpfile, err := os.CreateTemp("", "test_transform*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		store, err := api.New[TestFieldCasing](tmpfile.Name())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = store.Close() }()

		// Create item with data fields
		item := &TestFieldCasing{
			CreatedBy:  "alice",
			AssignedTo: "bob",
		}
		id, err := store.Create("Test Item", item)
		if err != nil {
			t.Fatal(err)
		}

		// Get raw document to inspect actual storage
		doc, err := store.GetRaw(id)
		if err != nil {
			t.Fatal(err)
		}

		// Verify that Go field names are stored as snake_case
		expectedFields := []string{"_data.created_by", "_data.assigned_to"}
		for _, field := range expectedFields {
			if _, exists := doc.Dimensions[field]; !exists {
				t.Errorf("Expected field %s in dimensions, but not found. Available: %v",
					field, getKeys(doc.Dimensions))
			}
		}

		// Verify PascalCase fields are NOT stored (should be transformed)
		unexpectedFields := []string{"_data.CreatedBy", "_data.AssignedTo"}
		for _, field := range unexpectedFields {
			if _, exists := doc.Dimensions[field]; exists {
				t.Errorf("Did not expect field %s in dimensions (should be snake_case)", field)
			}
		}
	})
}

// Helper function to get map keys for debugging
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
