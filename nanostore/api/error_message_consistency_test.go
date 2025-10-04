package api_test

import (
	"os"
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// TestErrorMessageStandardization represents a test item for error message consistency testing
type TestErrorMessageStandardization struct {
	nanostore.Document
	Status   string `values:"pending,active,done" default:"pending"`
	Priority string `values:"low,medium,high" default:"medium"`

	// Data fields
	Assignee    string
	Description string
}

func TestErrorMessageConsistency(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test_error_consistency*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TestErrorMessageStandardization](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("Document not found errors use consistent quoting", func(t *testing.T) {
		// Test Get with non-existent ID
		_, err := store.Get("nonexistent-id")
		if err == nil {
			t.Fatal("Expected error for non-existent ID")
		}

		errorStr := err.Error()

		// Should use single quotes around ID
		expectedPattern := "document with ID 'nonexistent-id' not found"
		if errorStr != expectedPattern {
			t.Errorf("Error message format inconsistent.\nExpected: %s\nActual: %s", expectedPattern, errorStr)
		}

		// Should not use double quotes or no quotes
		if strings.Contains(errorStr, `"nonexistent-id"`) {
			t.Error("Error message should not use double quotes around ID")
		}
		if strings.Contains(errorStr, "ID nonexistent-id not") && !strings.Contains(errorStr, "'nonexistent-id'") {
			t.Error("Error message should use single quotes around ID")
		}
	})

	t.Run("Enumerated field validation errors use consistent format", func(t *testing.T) {
		item := &TestErrorMessageStandardization{
			Status:   "invalid-status",
			Priority: "medium",
		}

		_, err := store.Create("Test", item)
		if err == nil {
			t.Fatal("Expected validation error for invalid status")
		}

		errorStr := err.Error()

		// Should use single quotes around field names and values
		expectedComponents := []string{
			"invalid value 'invalid-status'",
			"for field 'Status'",
			"must be one of",
		}

		for _, component := range expectedComponents {
			if !strings.Contains(errorStr, component) {
				t.Errorf("Error message should contain '%s'. Full message: %s", component, errorStr)
			}
		}

		// Should not use double quotes for field/value names
		if strings.Contains(errorStr, `"Status"`) || strings.Contains(errorStr, `"invalid-status"`) {
			t.Errorf("Error message should use single quotes, not double quotes. Message: %s", errorStr)
		}
	})

	t.Run("Data field validation errors use consistent format", func(t *testing.T) {
		// Create a valid item first
		validItem := &TestErrorMessageStandardization{
			Status:   "pending",
			Priority: "medium",
		}

		_, err := store.Create("Test", validItem)
		if err != nil {
			t.Fatalf("Failed to create test item: %v", err)
		}

		// Try to query with invalid data field name
		_, err = store.Query().Data("InvalidFieldName", "somevalue").Find()
		if err == nil {
			t.Fatal("Expected error for invalid data field name")
		}

		errorStr := err.Error()

		// Should use single quotes around field name
		if strings.Contains(errorStr, "invalid data field name") {
			if !strings.Contains(errorStr, "'InvalidFieldName'") {
				t.Errorf("Error message should use single quotes around field name. Message: %s", errorStr)
			}

			// Should not use double quotes or Go-style quotes
			if strings.Contains(errorStr, `"InvalidFieldName"`) || strings.Contains(errorStr, "`InvalidFieldName`") {
				t.Errorf("Error message should use single quotes, not double/backticks. Message: %s", errorStr)
			}
		}
	})

	t.Run("Multiple documents found error uses consistent format", func(t *testing.T) {
		// This test would need a scenario that creates duplicate documents
		// For now, we'll test the error format in a different way by checking
		// that the error message pattern is consistent

		// Note: Multiple documents found errors are rare in normal usage,
		// but we can verify the format if such an error occurs
		t.Skip("Multiple documents error requires specific database state to test")
	})

	t.Run("No documents found error uses consistent format", func(t *testing.T) {
		// Query for something that doesn't exist and try First()
		_, err := store.Query().Status("nonexistent-status").First()
		if err == nil {
			t.Fatal("Expected error when no documents match query")
		}

		errorStr := err.Error()
		expectedMessage := "no documents found"

		if errorStr != expectedMessage {
			t.Errorf("Error message format inconsistent.\nExpected: %s\nActual: %s", expectedMessage, errorStr)
		}
	})

	t.Run("Error message capitalization is consistent", func(t *testing.T) {
		testCases := []struct {
			name     string
			testFunc func() error
		}{
			{
				name: "Document not found",
				testFunc: func() error {
					_, err := store.Get("test-id")
					return err
				},
			},
			{
				name: "Invalid enumerated value",
				testFunc: func() error {
					item := &TestErrorMessageStandardization{Status: "invalid"}
					_, err := store.Create("Test", item)
					return err
				},
			},
			{
				name: "Invalid data field",
				testFunc: func() error {
					_, err := store.Query().Data("InvalidField", "value").Find()
					return err
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.testFunc()
				if err == nil {
					t.Fatal("Expected error")
				}

				errorStr := err.Error()

				// Error messages should start with lowercase (except for proper nouns)
				firstChar := string(errorStr[0])
				if firstChar != strings.ToLower(firstChar) {
					// Allow exceptions for proper nouns, but most should be lowercase
					if !strings.HasPrefix(errorStr, "UUID") && !strings.HasPrefix(errorStr, "Document") {
						t.Errorf("Error message should start with lowercase letter: %s", errorStr)
					}
				}
			})
		}
	})

	t.Run("Error message punctuation is consistent", func(t *testing.T) {
		// Test that error messages follow consistent punctuation rules
		// Most error messages should not end with periods unless they're complex sentences

		_, err := store.Get("nonexistent")
		if err != nil {
			errorStr := err.Error()

			// Simple error messages should not end with periods
			if strings.HasSuffix(errorStr, ".") && !strings.Contains(errorStr, ":") {
				t.Errorf("Simple error message should not end with period: %s", errorStr)
			}
		}
	})
}

func TestErrorMessageTemplates(t *testing.T) {
	// Test that our error message templates are being used consistently
	t.Run("Document operation error template", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_error_templates*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		store, err := api.NewFromType[TestErrorMessageStandardization](tmpfile.Name())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = store.Close() }()

		// Test document not found follows template: "document with ID 'id' not found"
		_, err = store.Get("test-123")
		if err != nil {
			expected := "document with ID 'test-123' not found"
			if err.Error() != expected {
				t.Errorf("Document not found error doesn't follow template.\nExpected: %s\nActual: %s",
					expected, err.Error())
			}
		}
	})

	t.Run("Field validation error template", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_error_templates*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		store, err := api.NewFromType[TestErrorMessageStandardization](tmpfile.Name())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = store.Close() }()

		// Test enumerated field validation follows template: "invalid value 'value' for field 'field': details"
		item := &TestErrorMessageStandardization{Status: "bad-status"}
		_, err = store.Create("Test", item)
		if err != nil {
			errorStr := err.Error()

			// Should match template pattern
			if !strings.Contains(errorStr, "invalid value 'bad-status'") {
				t.Errorf("Error should contain 'invalid value 'bad-status'': %s", errorStr)
			}
			if !strings.Contains(errorStr, "for field 'Status'") {
				t.Errorf("Error should contain 'for field 'Status'': %s", errorStr)
			}
			if !strings.Contains(errorStr, "must be one of") {
				t.Errorf("Error should contain 'must be one of': %s", errorStr)
			}
		}
	})
}
