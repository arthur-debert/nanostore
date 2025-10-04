package api_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// Test structs for export functionality
type SimpleTestStruct struct {
	nanostore.Document
	Status   string `values:"pending,active,done" default:"pending" prefix:"done=d"`
	Priority string `values:"low,medium,high" default:"medium"`
}

type ComplexTestStruct struct {
	nanostore.Document
	Status   string `values:"pending,active,done" default:"pending" prefix:"done=d,active=a"`
	Priority string `values:"low,medium,high" prefix:"high=h" default:"medium"`
	Activity string `values:"active,archived,deleted" default:"active"`
	ParentID string `dimension:"parent_id,ref"`
	Pinned   bool   `default:"false"`

	// Data fields
	Description string
	AssignedTo  string
	Tags        string
	DueDate     time.Time
	CreatedBy   *string
	Score       *int
}

type PointerFieldsStruct struct {
	nanostore.Document
	Status     *string `values:"active,inactive" default:"active"`
	Priority   *int    `default:"1"`
	LastUpdate *time.Time

	// Data fields
	Description *string
	Score       *float64
}

func TestExportConfigFromType(t *testing.T) {
	t.Run("Simple struct export", func(t *testing.T) {
		jsonBytes, err := api.ExportConfigFromType[SimpleTestStruct]()
		if err != nil {
			t.Fatalf("Export failed: %v", err)
		}

		// Parse the JSON to validate structure
		var config api.JSONStoreConfig
		if err := json.Unmarshal(jsonBytes, &config); err != nil {
			t.Fatalf("Failed to parse exported JSON: %v", err)
		}

		// Validate basic structure
		if config.StoreName != "simpleteststruct" {
			t.Errorf("Expected store_name 'simpleteststruct', got '%s'", config.StoreName)
		}

		if config.Version != "1.0" {
			t.Errorf("Expected version '1.0', got '%s'", config.Version)
		}

		// Validate dimensions
		if len(config.Dimensions) != 2 {
			t.Errorf("Expected 2 dimensions, got %d", len(config.Dimensions))
		}

		// Check status dimension
		statusDim, exists := config.Dimensions["status"]
		if !exists {
			t.Error("Status dimension not found")
		} else {
			if statusDim.Type != "enumerated" {
				t.Errorf("Expected status type 'enumerated', got '%s'", statusDim.Type)
			}
			if statusDim.FieldType != "string" {
				t.Errorf("Expected status field_type 'string', got '%s'", statusDim.FieldType)
			}
			if len(statusDim.Values) != 3 {
				t.Errorf("Expected 3 status values, got %d", len(statusDim.Values))
			}
			if statusDim.Default != "pending" {
				t.Errorf("Expected default 'pending', got '%v'", statusDim.Default)
			}
			if statusDim.Prefixes["done"] != "d" {
				t.Errorf("Expected prefix for 'done' to be 'd', got '%s'", statusDim.Prefixes["done"])
			}
		}

		// Check priority dimension
		priorityDim, exists := config.Dimensions["priority"]
		if !exists {
			t.Error("Priority dimension not found")
		} else {
			if priorityDim.Default != "medium" {
				t.Errorf("Expected priority default 'medium', got '%v'", priorityDim.Default)
			}
		}

		// Should have no data fields for this simple struct
		if len(config.DataFields) != 0 {
			t.Errorf("Expected 0 data fields, got %d", len(config.DataFields))
		}
	})

	t.Run("Complex struct export", func(t *testing.T) {
		jsonBytes, err := api.ExportConfigFromType[ComplexTestStruct]()
		if err != nil {
			t.Fatalf("Export failed: %v", err)
		}

		var config api.JSONStoreConfig
		if err := json.Unmarshal(jsonBytes, &config); err != nil {
			t.Fatalf("Failed to parse exported JSON: %v", err)
		}

		// Validate dimensions count (status, priority, activity, parent_id_hierarchy)
		// Note: pinned is a data field, not a dimension, since it only has default tag
		expectedDimensions := 4
		if len(config.Dimensions) != expectedDimensions {
			t.Errorf("Expected %d dimensions, got %d", expectedDimensions, len(config.Dimensions))
		}

		// Check hierarchical dimension
		hierarchicalFound := false
		for name, dim := range config.Dimensions {
			if dim.Type == "hierarchical" {
				hierarchicalFound = true
				if dim.RefField != "parent_id" {
					t.Errorf("Expected hierarchical dimension ref_field 'parent_id', got '%s'", dim.RefField)
				}
				t.Logf("Found hierarchical dimension: %s", name)
				break
			}
		}
		if !hierarchicalFound {
			t.Error("No hierarchical dimension found")
		}

		// Check that pinned is correctly classified as a data field, not dimension
		_, pinnedInDimensions := config.Dimensions["pinned"]
		if pinnedInDimensions {
			t.Error("Pinned should be a data field, not a dimension")
		}

		pinnedField, pinnedInData := config.DataFields["pinned"]
		if !pinnedInData {
			t.Error("Pinned field not found in data fields")
		} else {
			if pinnedField.FieldType != "bool" {
				t.Errorf("Expected pinned field_type 'bool', got '%s'", pinnedField.FieldType)
			}
		}

		// Validate data fields
		expectedDataFields := 7 // description, assigned_to, tags, due_date, created_by, score, pinned
		if len(config.DataFields) != expectedDataFields {
			t.Errorf("Expected %d data fields, got %d", expectedDataFields, len(config.DataFields))
		}

		// Check specific data fields
		dataFieldTests := map[string]struct {
			expectedType string
			nullable     bool
		}{
			"description": {"string", false},
			"assignedto":  {"string", false},
			"tags":        {"string", false},
			"duedate":     {"time.Time", false},
			"createdby":   {"string", true},
			"score":       {"int", true},
		}

		for fieldName, expected := range dataFieldTests {
			field, exists := config.DataFields[fieldName]
			if !exists {
				t.Errorf("Data field '%s' not found", fieldName)
				continue
			}
			if field.FieldType != expected.expectedType {
				t.Errorf("Field '%s': expected type '%s', got '%s'", fieldName, expected.expectedType, field.FieldType)
			}
			if field.Nullable != expected.nullable {
				t.Errorf("Field '%s': expected nullable %v, got %v", fieldName, expected.nullable, field.Nullable)
			}
		}
	})

	t.Run("Pointer fields export", func(t *testing.T) {
		jsonBytes, err := api.ExportConfigFromType[PointerFieldsStruct]()
		if err != nil {
			t.Fatalf("Export failed: %v", err)
		}

		var config api.JSONStoreConfig
		if err := json.Unmarshal(jsonBytes, &config); err != nil {
			t.Fatalf("Failed to parse exported JSON: %v", err)
		}

		// Check pointer dimension fields are marked as nullable
		statusDim, exists := config.Dimensions["status"]
		if !exists {
			t.Error("Status dimension not found")
		} else {
			if !statusDim.Nullable {
				t.Error("Expected pointer status field to be nullable")
			}
		}

		// Priority should be a data field since it only has default tag, not values tag
		priorityField, exists := config.DataFields["priority"]
		if !exists {
			t.Error("Priority data field not found")
		} else {
			if priorityField.FieldType != "int" {
				t.Errorf("Expected priority field_type 'int', got '%s'", priorityField.FieldType)
			}
			if !priorityField.Nullable {
				t.Error("Expected pointer priority field to be nullable")
			}
		}

		// Check pointer data fields
		descField, exists := config.DataFields["description"]
		if !exists {
			t.Error("Description data field not found")
		} else {
			if !descField.Nullable {
				t.Error("Expected pointer description field to be nullable")
			}
		}
	})

	t.Run("JSON output is valid and readable", func(t *testing.T) {
		jsonBytes, err := api.ExportConfigFromType[ComplexTestStruct]()
		if err != nil {
			t.Fatalf("Export failed: %v", err)
		}

		// Validate JSON is properly formatted
		var prettyJSON interface{}
		if err := json.Unmarshal(jsonBytes, &prettyJSON); err != nil {
			t.Fatalf("Generated JSON is not valid: %v", err)
		}

		// Check that it's pretty-printed (contains newlines and indentation)
		jsonStr := string(jsonBytes)
		if !containsIndentation(jsonStr) {
			t.Error("Generated JSON is not pretty-printed")
		}

		t.Logf("Generated JSON:\n%s", jsonStr)
	})
}

func TestExportErrorHandling(t *testing.T) {
	// Test with invalid struct (no Document embedding)
	type InvalidStruct struct {
		Status string `values:"pending,done"`
	}

	_, err := api.ExportConfigFromType[InvalidStruct]()
	if err == nil {
		t.Error("Expected error for struct without Document embedding")
	}
}

// Helper function to check if JSON contains proper indentation
func containsIndentation(jsonStr string) bool {
	return len(jsonStr) > 100 && // Reasonable size check
		(jsonStr[0] == '{' || jsonStr[0] == '[') && // Starts with JSON
		(jsonStr[1] == '\n' || jsonStr[2] == '\n') // Has early newline (pretty printed)
}
