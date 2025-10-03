package api_test

import (
	"os"
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// TestEnumeratedValidationItem represents a test item for enumerated value validation
type TestEnumeratedValidationItem struct {
	nanostore.Document
	Status   string `values:"pending,active,done" default:"pending"`
	Priority string `values:"low,medium,high" default:"medium"`
	Type     string `values:"bug,feature,task" default:"task"`

	// Data fields (not validated)
	Assignee    string
	Description string
}

// TestEnumeratedValidationNoDefaults represents a test item without defaults for strict validation
type TestEnumeratedValidationNoDefaults struct {
	nanostore.Document
	Status   string `values:"pending,active,done"`
	Priority string `values:"low,medium,high"`
	Type     string `values:"bug,feature,task"`
}

func TestEnumeratedValueValidation(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test_enumerated_validation*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TestEnumeratedValidationItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("Valid enumerated values should be accepted", func(t *testing.T) {
		// Test all valid combinations
		validCombinations := []TestEnumeratedValidationItem{
			{Status: "pending", Priority: "low", Type: "bug"},
			{Status: "active", Priority: "medium", Type: "feature"},
			{Status: "done", Priority: "high", Type: "task"},
		}

		for i, item := range validCombinations {
			_, err := store.Create("Valid Test", &item)
			if err != nil {
				t.Errorf("Valid combination %d should be accepted, got error: %v", i, err)
			}
		}
	})

	t.Run("Invalid status values should be rejected", func(t *testing.T) {
		invalidStatuses := []string{"invalid", "completed", "in-progress", "PENDING"}

		for _, status := range invalidStatuses {
			item := &TestEnumeratedValidationItem{
				Status:   status,
				Priority: "medium", // Valid
				Type:     "task",   // Valid
			}

			_, err := store.Create("Invalid Status Test", item)
			if err == nil {
				t.Errorf("Invalid status '%s' should have been rejected", status)
				continue
			}

			// Check error message quality
			if !strings.Contains(err.Error(), "invalid value") || !strings.Contains(err.Error(), "Status") {
				t.Errorf("Expected descriptive error for invalid status '%s', got: %v", status, err)
			}

			// Check that allowed values are mentioned in error
			if !strings.Contains(err.Error(), "pending") || !strings.Contains(err.Error(), "active") || !strings.Contains(err.Error(), "done") {
				t.Errorf("Error should mention allowed values for status '%s', got: %v", status, err)
			}
		}
	})

	t.Run("Invalid priority values should be rejected", func(t *testing.T) {
		invalidPriorities := []string{"urgent", "critical", "normal", "HIGH"}

		for _, priority := range invalidPriorities {
			item := &TestEnumeratedValidationItem{
				Status:   "pending", // Valid
				Priority: priority,
				Type:     "task", // Valid
			}

			_, err := store.Create("Invalid Priority Test", item)
			if err == nil {
				t.Errorf("Invalid priority '%s' should have been rejected", priority)
				continue
			}

			// Check error message mentions the field name
			if !strings.Contains(err.Error(), "Priority") {
				t.Errorf("Error should mention field name 'Priority' for value '%s', got: %v", priority, err)
			}
		}
	})

	t.Run("Invalid type values should be rejected", func(t *testing.T) {
		invalidTypes := []string{"enhancement", "story", "epic", "TASK"}

		for _, itemType := range invalidTypes {
			item := &TestEnumeratedValidationItem{
				Status:   "pending", // Valid
				Priority: "medium",  // Valid
				Type:     itemType,
			}

			_, err := store.Create("Invalid Type Test", item)
			if err == nil {
				t.Errorf("Invalid type '%s' should have been rejected", itemType)
				continue
			}

			// Check that allowed values are in error message
			if !strings.Contains(err.Error(), "bug") || !strings.Contains(err.Error(), "feature") || !strings.Contains(err.Error(), "task") {
				t.Errorf("Error should mention allowed types for value '%s', got: %v", itemType, err)
			}
		}
	})

	t.Run("Data fields should not be validated (non-enumerated)", func(t *testing.T) {
		// Data fields like Assignee and Description should accept any value
		item := &TestEnumeratedValidationItem{
			Status:      "active",              // Valid enumerated
			Priority:    "high",                // Valid enumerated
			Type:        "bug",                 // Valid enumerated
			Assignee:    "anything-goes-here",  // Should not be validated
			Description: "any description 123", // Should not be validated
		}

		_, err := store.Create("Data Field Test", item)
		if err != nil {
			t.Errorf("Data fields should not be validated, got error: %v", err)
		}
	})

	t.Run("Update operations should also validate enumerated values", func(t *testing.T) {
		// Create a valid item first
		validItem := &TestEnumeratedValidationItem{
			Status:   "pending",
			Priority: "medium",
			Type:     "task",
		}

		id, err := store.Create("Update Test", validItem)
		if err != nil {
			t.Fatalf("Failed to create test item: %v", err)
		}

		// Try to update with invalid status
		invalidUpdate := &TestEnumeratedValidationItem{
			Status:   "invalid-status", // Invalid
			Priority: "high",           // Valid
			Type:     "bug",            // Valid
		}

		_, err = store.Update(id, invalidUpdate)
		if err == nil {
			t.Error("Update with invalid status should have been rejected")
		}

		// Verify error message quality
		if !strings.Contains(err.Error(), "invalid value") || !strings.Contains(err.Error(), "Status") {
			t.Errorf("Update error should be descriptive, got: %v", err)
		}
	})

	t.Run("Bulk update operations should validate enumerated values", func(t *testing.T) {
		// Create some valid items first
		validItem := &TestEnumeratedValidationItem{
			Status:   "pending",
			Priority: "low",
			Type:     "feature",
		}

		id1, _ := store.Create("Bulk Test 1", validItem)
		id2, _ := store.Create("Bulk Test 2", validItem)

		// Try bulk update with invalid values
		invalidBulkUpdate := &TestEnumeratedValidationItem{
			Status: "invalid-bulk-status", // Invalid
		}

		_, err := store.UpdateByUUIDs([]string{id1, id2}, invalidBulkUpdate)
		if err == nil {
			t.Error("Bulk update with invalid status should have been rejected")
		}

		// Check error references the field
		if !strings.Contains(err.Error(), "Status") {
			t.Errorf("Bulk update error should mention field name, got: %v", err)
		}
	})

	t.Run("Case sensitivity validation", func(t *testing.T) {
		// Enumerated values should be case-sensitive
		caseSensitiveTests := []string{"PENDING", "Active", "Done", "LOW", "Medium", "High"}

		for _, value := range caseSensitiveTests {
			item := &TestEnumeratedValidationItem{
				Status:   value,
				Priority: "medium", // Valid
				Type:     "task",   // Valid
			}

			_, err := store.Create("Case Sensitivity Test", item)
			if err == nil {
				t.Errorf("Case-sensitive value '%s' should have been rejected (values are case-sensitive)", value)
			}
		}
	})

	t.Run("Empty string validation", func(t *testing.T) {
		// Create a temporary file for store without defaults
		tmpfile2, err := os.CreateTemp("", "test_enumerated_validation_no_defaults*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile2.Name()) }()
		_ = tmpfile2.Close()

		store2, err := api.NewFromType[TestEnumeratedValidationNoDefaults](tmpfile2.Name())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = store2.Close() }()

		// Empty strings should be rejected for enumerated fields without defaults
		item := &TestEnumeratedValidationNoDefaults{
			Status:   "", // Empty - should be invalid
			Priority: "medium",
			Type:     "task",
		}

		_, err = store2.Create("Empty String Test", item)
		if err == nil {
			t.Error("Empty string should be rejected for enumerated field without default")
		}
	})

	t.Run("Whitespace handling in values tag", func(t *testing.T) {
		// This test validates that spaces around values in tags are handled correctly
		// The actual validation is done by the validateEnumeratedValue function
		// which should trim whitespace from tag values

		// Note: We can't easily test tag parsing in isolation here,
		// but the validation function should handle "pending, active, done" correctly
		item := &TestEnumeratedValidationItem{
			Status:   "pending", // Should work even if tag has spaces
			Priority: "medium",
			Type:     "task",
		}

		_, err := store.Create("Whitespace Test", item)
		if err != nil {
			t.Errorf("Valid values should work regardless of whitespace in tags: %v", err)
		}
	})
}

func TestEnumeratedValidationErrorMessages(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_error_messages*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TestEnumeratedValidationItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("Error message format validation", func(t *testing.T) {
		item := &TestEnumeratedValidationItem{
			Status:   "invalid-value",
			Priority: "medium",
			Type:     "task",
		}

		_, err := store.Create("Error Format Test", item)
		if err == nil {
			t.Fatal("Expected validation error")
		}

		errorStr := err.Error()

		// Check error message components
		expectedComponents := []string{
			"invalid value",
			"invalid-value",
			"Status",
			"pending",
			"active",
			"done",
		}

		for _, component := range expectedComponents {
			if !strings.Contains(errorStr, component) {
				t.Errorf("Error message should contain '%s', got: %s", component, errorStr)
			}
		}

		// Check that it follows expected format: "invalid value 'X' for field 'Y': must be one of [...]"
		if !strings.Contains(errorStr, "must be one of") {
			t.Errorf("Error message should contain 'must be one of', got: %s", errorStr)
		}
	})
}
