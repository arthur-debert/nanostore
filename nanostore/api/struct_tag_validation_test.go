package api_test

import (
	"os"
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// Valid struct for testing
type ValidTestStruct struct {
	nanostore.Document
	Status   string `values:"pending,active,done" default:"pending" prefix:"done=d"`
	Priority string `values:"low,medium,high" default:"medium"`
}

// Test structs with various malformed tags

// Empty values tag
type EmptyValuesTag struct {
	nanostore.Document
	Status string `values:""`
}

// Trailing comma in values
type TrailingCommaValues struct {
	nanostore.Document
	Status string `values:"pending,active,done,"`
}

// Empty value in middle of values list
type EmptyValueInList struct {
	nanostore.Document
	Status string `values:"pending,,done"`
}

// Invalid default (not in values list)
type InvalidDefault struct {
	nanostore.Document
	Status string `values:"pending,active,done" default:"invalid"`
}

// Empty default tag
type EmptyDefault struct {
	nanostore.Document
	Status string `values:"pending,active,done" default:""`
}

// Default with comma (multiple values)
type DefaultWithComma struct {
	nanostore.Document
	Status string `values:"pending,active,done" default:"pending,active"`
}

// Malformed prefix tag
type MalformedPrefix struct {
	nanostore.Document
	Status string `values:"pending,active,done" prefix:"done"`
}

// Empty prefix mapping
type EmptyPrefixMapping struct {
	nanostore.Document
	Status string `values:"pending,active,done" prefix:"done="`
}

// Empty prefix tag
type EmptyPrefixTag struct {
	nanostore.Document
	Status string `values:"pending,active,done" prefix:""`
}

// Prefix for unknown value
type PrefixForUnknownValue struct {
	nanostore.Document
	Status string `values:"pending,active,done" prefix:"unknown=u"`
}

// Duplicate values
type DuplicateValues struct {
	nanostore.Document
	Status string `values:"pending,active,done,active"`
}

// Duplicate dimension names (same field name lowercased)
type DuplicateDimensionNames struct {
	nanostore.Document
	Status        string `values:"pending,active"`
	AnotherStatus string `dimension:"status"` // This would create duplicate dimension name "status"
}

// Conflicting prefixes across dimensions
type ConflictingPrefixes struct {
	nanostore.Document
	Status   string `values:"pending,active,done" prefix:"done=d"`
	Priority string `values:"low,medium,high" prefix:"high=d"` // Conflict: both use "d"
}

// Value contains equals sign (common mistake)
type ValueWithEquals struct {
	nanostore.Document
	Status string `values:"pending,active=yes,done"`
}

func TestStructTagValidation(t *testing.T) {
	t.Run("Valid struct should create store successfully", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_valid*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[ValidTestStruct](tmpfile.Name())
		if err != nil {
			t.Errorf("Valid struct should not produce error: %v", err)
		}
	})

	t.Run("Empty values tag should be rejected", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_empty_values*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[EmptyValuesTag](tmpfile.Name())
		if err == nil {
			t.Error("Empty values tag should be rejected")
		}

		if !strings.Contains(err.Error(), "values tag cannot be empty") && !strings.Contains(err.Error(), "at least one dimension must be configured") {
			t.Errorf("Error should mention empty values tag: %v", err)
		}
	})

	t.Run("Trailing comma in values should be rejected", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_trailing_comma*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[TrailingCommaValues](tmpfile.Name())
		if err == nil {
			t.Error("Trailing comma in values should be rejected")
		}

		if !strings.Contains(err.Error(), "empty value") {
			t.Errorf("Error should mention empty value: %v", err)
		}
	})

	t.Run("Empty value in middle of list should be rejected", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_empty_middle*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[EmptyValueInList](tmpfile.Name())
		if err == nil {
			t.Error("Empty value in middle should be rejected")
		}

		if !strings.Contains(err.Error(), "empty value at position") {
			t.Errorf("Error should mention empty value position: %v", err)
		}
	})

	t.Run("Invalid default value should be rejected", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_invalid_default*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[InvalidDefault](tmpfile.Name())
		if err == nil {
			t.Error("Invalid default should be rejected")
		}

		if !strings.Contains(err.Error(), "default value") && !strings.Contains(err.Error(), "not in values list") {
			t.Errorf("Error should mention invalid default: %v", err)
		}
	})

	t.Run("Empty default tag should be rejected", func(t *testing.T) {
		t.Skip("TODO: Implement proper empty default tag validation")
		tmpfile, err := os.CreateTemp("", "test_empty_default*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[EmptyDefault](tmpfile.Name())
		if err == nil {
			t.Error("Empty default tag should be rejected")
			return
		}

		if !strings.Contains(err.Error(), "default tag cannot be empty") {
			t.Errorf("Error should mention empty default tag: %v", err)
		}
	})

	t.Run("Default with comma should be rejected", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_default_comma*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[DefaultWithComma](tmpfile.Name())
		if err == nil {
			t.Error("Default with comma should be rejected")
		}

		if !strings.Contains(err.Error(), "contains comma") {
			t.Errorf("Error should mention comma in default: %v", err)
		}
	})

	t.Run("Malformed prefix tag should be rejected", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_malformed_prefix*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[MalformedPrefix](tmpfile.Name())
		if err == nil {
			t.Error("Malformed prefix should be rejected")
		}

		if !strings.Contains(err.Error(), "format should be 'value=prefix'") {
			t.Errorf("Error should mention prefix format: %v", err)
		}
	})

	t.Run("Empty prefix mapping should be rejected", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_empty_prefix*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[EmptyPrefixMapping](tmpfile.Name())
		if err == nil {
			t.Error("Empty prefix mapping should be rejected")
		}

		if !strings.Contains(err.Error(), "empty prefix") {
			t.Errorf("Error should mention empty prefix: %v", err)
		}
	})

	t.Run("Empty prefix tag should be rejected", func(t *testing.T) {
		t.Skip("TODO: Implement proper empty prefix tag validation")
		tmpfile, err := os.CreateTemp("", "test_empty_prefix_tag*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[EmptyPrefixTag](tmpfile.Name())
		if err == nil {
			t.Error("Empty prefix tag should be rejected")
		}

		if !strings.Contains(err.Error(), "prefix tag cannot be empty") {
			t.Errorf("Error should mention empty prefix tag: %v", err)
		}
	})

	t.Run("Prefix for unknown value should be rejected", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_unknown_prefix*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[PrefixForUnknownValue](tmpfile.Name())
		if err == nil {
			t.Error("Prefix for unknown value should be rejected")
		}

		if !strings.Contains(err.Error(), "prefix mapping for unknown value") {
			t.Errorf("Error should mention unknown value: %v", err)
		}
	})

	t.Run("Duplicate values should be rejected", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_duplicate_values*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[DuplicateValues](tmpfile.Name())
		if err == nil {
			t.Error("Duplicate values should be rejected")
		}

		if !strings.Contains(err.Error(), "duplicate value") {
			t.Errorf("Error should mention duplicate value: %v", err)
		}
	})

	t.Run("Conflicting prefixes should be rejected", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_conflicting_prefixes*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[ConflictingPrefixes](tmpfile.Name())
		if err == nil {
			t.Error("Conflicting prefixes should be rejected")
		}

		if !strings.Contains(err.Error(), "prefix") && !strings.Contains(err.Error(), "conflicts") {
			t.Errorf("Error should mention prefix conflict: %v", err)
		}
	})

	t.Run("Value containing equals sign should be rejected", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_value_equals*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[ValueWithEquals](tmpfile.Name())
		if err == nil {
			t.Error("Value with equals should be rejected")
		}

		if !strings.Contains(err.Error(), "contains '='") {
			t.Errorf("Error should mention equals sign in value: %v", err)
		}
	})
}

func TestStructTagValidationErrorMessages(t *testing.T) {
	t.Run("Error messages should be descriptive and helpful", func(t *testing.T) {
		testCases := []struct {
			name          string
			structType    interface{}
			expectedError string
		}{
			{
				name:          "Empty values",
				structType:    (*EmptyValuesTag)(nil),
				expectedError: "values tag cannot be empty",
			},
			{
				name:          "Invalid default",
				structType:    (*InvalidDefault)(nil),
				expectedError: "default value 'invalid' not in values list",
			},
			{
				name:          "Malformed prefix",
				structType:    (*MalformedPrefix)(nil),
				expectedError: "format should be 'value=prefix'",
			},
			{
				name:          "Value with equals",
				structType:    (*ValueWithEquals)(nil),
				expectedError: "contains '=' - did you mean to use the prefix tag?",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tmpfile, err := os.CreateTemp("", "test_error_msg*.json")
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = os.Remove(tmpfile.Name()) }()
				_ = tmpfile.Close()

				var err2 error
				switch tc.structType.(type) {
				case *EmptyValuesTag:
					_, err2 = api.NewFromType[EmptyValuesTag](tmpfile.Name())
				case *InvalidDefault:
					_, err2 = api.NewFromType[InvalidDefault](tmpfile.Name())
				case *MalformedPrefix:
					_, err2 = api.NewFromType[MalformedPrefix](tmpfile.Name())
				case *ValueWithEquals:
					_, err2 = api.NewFromType[ValueWithEquals](tmpfile.Name())
				}

				if err2 == nil {
					t.Errorf("Expected error for %s", tc.name)
					return
				}

				if !strings.Contains(err2.Error(), tc.expectedError) {
					t.Errorf("Error message should contain '%s', got: %v", tc.expectedError, err2)
				}

				// All error messages should include field/dimension name for context
				if !strings.Contains(err2.Error(), "field") && !strings.Contains(err2.Error(), "Status") && !strings.Contains(err2.Error(), "dimension") {
					t.Errorf("Error message should include field context: %v", err2)
				}
			})
		}
	})

	t.Run("Error messages should use consistent formatting", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_formatting*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[InvalidDefault](tmpfile.Name())
		if err != nil {
			errorStr := err.Error()

			// Should use single quotes for values
			if !strings.Contains(errorStr, "'invalid'") {
				t.Errorf("Error should use single quotes for values: %v", err)
			}

			// Should follow hierarchical error structure
			if !strings.Contains(errorStr, "struct tag validation failed") {
				t.Errorf("Error should mention struct tag validation: %v", err)
			}
		}
	})
}
