package migration

import (
	"testing"

	"github.com/arthur-debert/nanostore/types"
)

func TestAPIRenameField(t *testing.T) {
	// Setup test documents
	docs := []types.Document{
		{
			UUID:  "1",
			Title: "Doc 1",
			Dimensions: map[string]interface{}{
				"old_status": "active",
				"priority":   "high",
			},
		},
		{
			UUID:  "2",
			Title: "Doc 2",
			Dimensions: map[string]interface{}{
				"old_status": "pending",
				"priority":   "low",
			},
		},
	}

	api := NewAPI()

	t.Run("successful rename", func(t *testing.T) {
		// Make a copy of docs for this test
		testDocs := make([]types.Document, len(docs))
		copy(testDocs, docs)
		for i := range testDocs {
			testDocs[i].Dimensions = make(map[string]interface{})
			for k, v := range docs[i].Dimensions {
				testDocs[i].Dimensions[k] = v
			}
		}

		modifiedDocs, result := api.RenameField(testDocs, types.Config{}, "old_status", "status", Options{
			DryRun:  false,
			Verbose: false,
		})

		if !result.Success {
			t.Errorf("expected success, got failure")
		}
		if result.Code != CodeSuccess {
			t.Errorf("expected code %d, got %d", CodeSuccess, result.Code)
		}
		if result.Stats.ModifiedDocs != 2 {
			t.Errorf("expected 2 modified docs, got %d", result.Stats.ModifiedDocs)
		}

		// Verify rename in documents
		for _, doc := range modifiedDocs {
			if _, exists := doc.Dimensions["status"]; !exists {
				t.Errorf("status field not found in doc %s", doc.UUID)
			}
			if _, exists := doc.Dimensions["old_status"]; exists {
				t.Errorf("old_status field still exists in doc %s", doc.UUID)
			}
		}
	})

	t.Run("dry run", func(t *testing.T) {
		// Make a copy of docs for this test
		testDocs := make([]types.Document, len(docs))
		copy(testDocs, docs)
		for i := range testDocs {
			testDocs[i].Dimensions = make(map[string]interface{})
			for k, v := range docs[i].Dimensions {
				testDocs[i].Dimensions[k] = v
			}
		}

		modifiedDocs, result := api.RenameField(testDocs, types.Config{}, "old_status", "status", Options{
			DryRun:  true,
			Verbose: false,
		})

		if !result.Success {
			t.Errorf("expected success, got failure")
		}

		// Verify no changes in dry run
		for i, doc := range modifiedDocs {
			if _, exists := doc.Dimensions["status"]; exists {
				t.Errorf("status field found in doc %s during dry run", doc.UUID)
			}
			if _, exists := doc.Dimensions["old_status"]; !exists {
				t.Errorf("old_status field missing in doc %s during dry run", doc.UUID)
			}

			// Compare with original
			if modifiedDocs[i].Dimensions["old_status"] != docs[i].Dimensions["old_status"] {
				t.Error("document was modified during dry run")
			}
		}
	})
}

func TestAPITransformFieldPartialFailure(t *testing.T) {
	// Setup test documents with mixed values
	docs := []types.Document{
		{
			UUID: "1",
			Dimensions: map[string]interface{}{
				"count": "42", // Can convert to int
			},
		},
		{
			UUID: "2",
			Dimensions: map[string]interface{}{
				"count": "invalid", // Cannot convert to int
			},
		},
		{
			UUID: "3",
			Dimensions: map[string]interface{}{
				"count": "100", // Can convert to int
			},
		},
	}

	api := NewAPI()

	// Test partial failure - should return modified documents even on failure
	modifiedDocs, result := api.TransformField(docs, types.Config{}, "count", "toInt", Options{
		DryRun: false,
	})

	// Should be marked as partial failure
	if result.Success {
		t.Error("expected failure due to invalid conversion")
	}
	if result.Code != CodePartialFailure {
		t.Errorf("expected code %d (partial failure), got %d", CodePartialFailure, result.Code)
	}

	// Verify successful transformations were applied
	if modifiedDocs[0].Dimensions["count"] != 42 {
		t.Errorf("doc 1: expected count=42, got %v", modifiedDocs[0].Dimensions["count"])
	}
	if modifiedDocs[1].Dimensions["count"] != "invalid" {
		t.Errorf("doc 2: expected count='invalid' (unchanged), got %v", modifiedDocs[1].Dimensions["count"])
	}
	if modifiedDocs[2].Dimensions["count"] != 100 {
		t.Errorf("doc 3: expected count=100, got %v", modifiedDocs[2].Dimensions["count"])
	}

	// Original documents should be unchanged
	if docs[0].Dimensions["count"] != "42" {
		t.Error("original doc 1 was modified")
	}
	if docs[1].Dimensions["count"] != "invalid" {
		t.Error("original doc 2 was modified")
	}
	if docs[2].Dimensions["count"] != "100" {
		t.Error("original doc 3 was modified")
	}
}

func TestAPIRemoveField(t *testing.T) {
	docs := []types.Document{
		{
			UUID: "1",
			Dimensions: map[string]interface{}{
				"field1":       "value1",
				"field2":       "value2",
				"_data.field2": "data value",
			},
		},
		{
			UUID: "2",
			Dimensions: map[string]interface{}{
				"field1": "value1",
				"field2": "value2",
			},
		},
	}

	api := NewAPI()

	t.Run("remove dimension field", func(t *testing.T) {
		modifiedDocs, result := api.RemoveField(docs, types.Config{}, "field1", Options{
			DryRun:    false,
			FieldType: FieldTypeDimension,
		})

		if !result.Success {
			t.Errorf("expected success, got failure")
		}
		if result.Stats.ModifiedDocs != 2 {
			t.Errorf("expected 2 modified docs, got %d", result.Stats.ModifiedDocs)
		}

		// Verify field removed
		for _, doc := range modifiedDocs {
			if _, exists := doc.Dimensions["field1"]; exists {
				t.Errorf("field1 still exists in doc %s", doc.UUID)
			}
			// Other fields should remain
			if _, exists := doc.Dimensions["field2"]; !exists {
				t.Errorf("field2 was incorrectly removed from doc %s", doc.UUID)
			}
		}
	})

	t.Run("remove data field only", func(t *testing.T) {
		modifiedDocs, result := api.RemoveField(docs, types.Config{}, "field2", Options{
			DryRun:    false,
			FieldType: FieldTypeData,
		})

		if !result.Success {
			t.Errorf("expected success, got failure")
		}

		// Only doc 1 has _data.field2
		if result.Stats.ModifiedDocs != 1 {
			t.Errorf("expected 1 modified doc, got %d", result.Stats.ModifiedDocs)
		}

		// Verify only data field removed
		if _, exists := modifiedDocs[0].Dimensions["_data.field2"]; exists {
			t.Error("_data.field2 still exists")
		}
		if _, exists := modifiedDocs[0].Dimensions["field2"]; !exists {
			t.Error("dimension field2 was incorrectly removed")
		}
	})
}

func TestAPIAddField(t *testing.T) {
	docs := []types.Document{
		{
			UUID: "1",
			Dimensions: map[string]interface{}{
				"existing": "value",
			},
		},
		{
			UUID: "2",
			Dimensions: map[string]interface{}{
				"existing": "value",
			},
		},
	}

	api := NewAPI()

	t.Run("add dimension field", func(t *testing.T) {
		modifiedDocs, result := api.AddField(docs, types.Config{}, "newfield", "default", Options{
			DryRun:      false,
			IsDataField: false,
		})

		if !result.Success {
			t.Errorf("expected success, got failure")
		}
		if result.Stats.ModifiedDocs != 2 {
			t.Errorf("expected 2 modified docs, got %d", result.Stats.ModifiedDocs)
		}

		// Verify field added
		for _, doc := range modifiedDocs {
			if val, exists := doc.Dimensions["newfield"]; !exists {
				t.Errorf("newfield not found in doc %s", doc.UUID)
			} else if val != "default" {
				t.Errorf("expected newfield='default', got %v", val)
			}
		}
	})

	t.Run("add data field", func(t *testing.T) {
		modifiedDocs, result := api.AddField(docs, types.Config{}, "datafield", 42, Options{
			DryRun:      false,
			IsDataField: true,
		})

		if !result.Success {
			t.Errorf("expected success, got failure")
		}

		// Verify data field added
		for _, doc := range modifiedDocs {
			if val, exists := doc.Dimensions["_data.datafield"]; !exists {
				t.Errorf("_data.datafield not found in doc %s", doc.UUID)
			} else if val != 42 {
				t.Errorf("expected _data.datafield=42, got %v", val)
			}
		}
	})

	t.Run("add to existing field fails", func(t *testing.T) {
		_, result := api.AddField(docs, types.Config{}, "existing", "value", Options{
			DryRun: false,
		})

		if result.Success {
			t.Error("expected failure when adding existing field")
		}
		if result.Code != CodeValidationError {
			t.Errorf("expected validation error code, got %d", result.Code)
		}
	})
}

func TestAPIValidateSchema(t *testing.T) {
	api := NewAPI()

	t.Run("validation passes", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "1",
				Dimensions: map[string]interface{}{
					"status": "active",
				},
			},
		}

		config := types.Config{
			Dimensions: []types.DimensionConfig{
				{
					Name:   "status",
					Type:   types.Enumerated,
					Values: []string{"active", "pending"},
				},
			},
		}

		modifiedDocs, result := api.ValidateSchema(docs, config, Options{})

		if !result.Success {
			t.Error("expected validation to pass")
		}
		if len(modifiedDocs) != len(docs) {
			t.Error("documents should be returned unchanged")
		}
	})

	t.Run("validation fails", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "1",
				Dimensions: map[string]interface{}{
					"status": "invalid",
				},
			},
		}

		config := types.Config{
			Dimensions: []types.DimensionConfig{
				{
					Name:   "status",
					Type:   types.Enumerated,
					Values: []string{"active", "pending"},
				},
			},
		}

		_, result := api.ValidateSchema(docs, config, Options{})

		if result.Success {
			t.Error("expected validation to fail")
		}
		if result.Code != CodeValidationError {
			t.Errorf("expected validation error code, got %d", result.Code)
		}
	})
}
