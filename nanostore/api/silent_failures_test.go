package api_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// TestSilentFailuresItem represents a test item for silent failures testing
type TestSilentFailuresItem struct {
	nanostore.Document
	Status   string `values:"pending,active,done" default:"pending"`
	Priority string `values:"low,medium,high" default:"medium"`

	// Data fields - these are the known valid data fields for this type
	Assignee    string
	Description string
	Tags        string
	Estimate    int
}

func TestSilentFailuresInQueries(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test_silent_failures*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TestSilentFailuresItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create test data
	_, err = store.Create("Task 1", &TestSilentFailuresItem{
		Status:      "active",
		Priority:    "high",
		Assignee:    "alice",
		Description: "Important task",
		Tags:        "urgent,backend",
		Estimate:    8,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Data query with invalid field name should return error", func(t *testing.T) {
		// This test should FAIL initially - demonstrates the silent failure bug
		// Currently this succeeds and returns empty results instead of an error

		_, err := store.Query().Data("nonexistent_field", "value").Find()
		if err == nil {
			t.Error("Expected error when querying with invalid data field name, but got none")
		}

		if err != nil && !containsString(err.Error(), "nonexistent_field") {
			t.Errorf("Expected error message to mention invalid field name 'nonexistent_field', got: %v", err)
		}
	})

	t.Run("Data query with typo in field name should return error", func(t *testing.T) {
		// Common typo: "assigne" instead of "assignee"
		_, err := store.Query().Data("assigne", "alice").Find()
		if err == nil {
			t.Error("Expected error when querying with typo in field name 'assigne', but got none")
		}

		if err != nil && !containsString(err.Error(), "assigne") {
			t.Errorf("Expected error message to mention invalid field name 'assigne', got: %v", err)
		}
	})

	t.Run("OrderByData with invalid field name should return error", func(t *testing.T) {
		// This should fail but currently succeeds with undefined ordering
		_, err := store.Query().OrderByData("invalid_order_field").Find()
		if err == nil {
			t.Error("Expected error when ordering by invalid data field name, but got none")
		}

		if err != nil && !containsString(err.Error(), "invalid_order_field") {
			t.Errorf("Expected error message to mention invalid field name 'invalid_order_field', got: %v", err)
		}
	})

	t.Run("OrderByData with case mismatch should work (case-insensitive)", func(t *testing.T) {
		// Case insensitive behavior: "assignee" should work even though field is "Assignee"
		// This is user-friendly behavior - auto-correct case instead of erroring
		results, err := store.Query().OrderByData("assignee").Find()
		if err != nil {
			t.Errorf("Case-insensitive ordering should work, got error: %v", err)
		}

		// Should return results successfully (case corrected internally)
		if len(results) == 0 {
			t.Error("Expected to find results with case-insensitive field ordering")
		}
	})

	t.Run("DataIn with invalid field name should return error", func(t *testing.T) {
		_, err := store.Query().DataIn("bad_field", "value1", "value2").Find()
		if err == nil {
			t.Error("Expected error when using DataIn with invalid field name, but got none")
		}

		if err != nil && !containsString(err.Error(), "bad_field") {
			t.Errorf("Expected error message to mention invalid field name 'bad_field', got: %v", err)
		}
	})

	t.Run("DataNot with invalid field name should return error", func(t *testing.T) {
		_, err := store.Query().DataNot("wrong_field", "value").Find()
		if err == nil {
			t.Error("Expected error when using DataNot with invalid field name, but got none")
		}

		if err != nil && !containsString(err.Error(), "wrong_field") {
			t.Errorf("Expected error message to mention invalid field name 'wrong_field', got: %v", err)
		}
	})

	t.Run("DataNotIn with invalid field name should return error", func(t *testing.T) {
		_, err := store.Query().DataNotIn("missing_field", "value1", "value2").Find()
		if err == nil {
			t.Error("Expected error when using DataNotIn with invalid field name, but got none")
		}

		if err != nil && !containsString(err.Error(), "missing_field") {
			t.Errorf("Expected error message to mention invalid field name 'missing_field', got: %v", err)
		}
	})

	t.Run("OrderByDataDesc with invalid field name should return error", func(t *testing.T) {
		_, err := store.Query().OrderByDataDesc("unknown_field").Find()
		if err == nil {
			t.Error("Expected error when using OrderByDataDesc with invalid field name, but got none")
		}

		if err != nil && !containsString(err.Error(), "unknown_field") {
			t.Errorf("Expected error message to mention invalid field name 'unknown_field', got: %v", err)
		}
	})

	t.Run("Chained queries with invalid field should return error", func(t *testing.T) {
		// Even in complex queries, invalid fields should be caught
		_, err := store.Query().
			Status("active").
			Data("valid_assignee", "alice"). // This field doesn't exist
			Priority("high").
			OrderByData("another_bad_field"). // This field doesn't exist
			Find()

		if err == nil {
			t.Error("Expected error when chaining queries with invalid field names, but got none")
		}

		// Should mention at least one of the invalid field names
		if err != nil {
			errStr := err.Error()
			if !containsString(errStr, "valid_assignee") && !containsString(errStr, "another_bad_field") {
				t.Errorf("Expected error message to mention one of the invalid field names, got: %v", err)
			}
		}
	})

	t.Run("Valid data fields should work correctly", func(t *testing.T) {
		// Ensure we don't break valid functionality with our fixes
		// These should all work without errors

		// Test that valid field operations still work correctly with exact case
		results, err := store.Query().Data("Assignee", "alice").Find()
		if err != nil {
			t.Errorf("Valid data field query failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("Expected to find results with valid data field query")
		}

		// Test valid ordering with exact case
		results, err = store.Query().OrderByData("Assignee").Find()
		if err != nil {
			t.Errorf("Valid data field ordering failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("Expected to find results with valid data field ordering")
		}

		// Test valid DataIn with exact case
		_, err = store.Query().DataIn("Assignee", "alice", "bob").Find()
		if err != nil {
			t.Errorf("Valid DataIn query failed: %v", err)
		}

		// Test case-insensitive data field queries (should also work)
		results, err = store.Query().Data("assignee", "alice").Find()
		if err != nil {
			t.Errorf("Case-insensitive data field query failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("Expected to find results with case-insensitive data field query")
		}

		// Test case-insensitive DataIn
		_, err = store.Query().DataIn("assignee", "alice", "bob").Find()
		if err != nil {
			t.Errorf("Case-insensitive DataIn query failed: %v", err)
		}
	})
}

func TestSilentFailuresInCustomOrderBy(t *testing.T) {
	// Test generic OrderBy with invalid column names

	tmpfile, err := os.CreateTemp("", "test_orderby_failures*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TestSilentFailuresItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create test data
	_, err = store.Create("Task 1", &TestSilentFailuresItem{
		Status:   "active",
		Priority: "high",
		Assignee: "alice",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("OrderBy with invalid column name should return error", func(t *testing.T) {
		// Test the most common case - invalid _data columns
		_, err := store.Query().OrderByData("completely_unknown_field").Find()
		if err == nil {
			t.Error("Expected error when ordering by completely unknown field, but got none")
		}
	})

	t.Run("Mixed valid and invalid fields should return error", func(t *testing.T) {
		// Start with valid query, then add invalid field
		_, err := store.Query().
			Status("active").          // Valid
			Data("assignee", "alice"). // Valid
			OrderByData("bad_field").  // Invalid - should cause error
			Find()

		if err == nil {
			t.Error("Expected error when mixing valid and invalid field names, but got none")
		}
	})
}

func TestFieldValidationErrorMessages(t *testing.T) {
	// Test that error messages are helpful and include suggestions

	tmpfile, err := os.CreateTemp("", "test_error_messages*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TestSilentFailuresItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("Error message should be helpful", func(t *testing.T) {
		_, err := store.Query().Data("assigneee", "alice").Find() // typo: extra 'e'
		if err == nil {
			t.Error("Expected error for typo in field name")
			return
		}

		errMsg := err.Error()

		// Should mention the invalid field name
		if !containsString(errMsg, "assigneee") {
			t.Errorf("Error message should mention the invalid field name 'assigneee', got: %v", err)
		}

		// Should ideally suggest the correct field name (this is aspirational)
		if !containsString(errMsg, "assignee") && !containsString(errMsg, "did you mean") {
			t.Logf("Error message could be more helpful by suggesting similar field names. Got: %v", err)
		}

		// Should list valid field names
		if !containsString(errMsg, "valid fields") && !containsString(errMsg, "available") {
			t.Logf("Error message could be more helpful by listing valid field names. Got: %v", err)
		}
	})
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsString(str, substr string) bool {
	return len(str) >= len(substr) &&
		(str == substr ||
			len(str) > len(substr) &&
				(hasSubstring(str, substr)))
}

// Simple substring check helper
func hasSubstring(str, substr string) bool {
	if len(substr) > len(str) {
		return false
	}
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
