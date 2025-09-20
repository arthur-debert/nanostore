package migration

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/types"
)

func TestAddField(t *testing.T) {
	t.Run("add dimension field", func(t *testing.T) {
		// Setup test documents
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"status":   "active",
					"priority": "high",
				},
			},
			{
				UUID:  "doc2",
				Title: "Test Doc 2",
				Dimensions: map[string]interface{}{
					"status":   "pending",
					"priority": "low",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &AddField{
			FieldName:    "category",
			DefaultValue: "general",
			IsDataField:  false,
		}

		result := cmd.Execute(ctx)

		// Check result
		if !result.Success {
			t.Errorf("expected success, got failure: %v", result.Messages)
		}
		if result.Code != CodeSuccess {
			t.Errorf("expected code %d, got %d", CodeSuccess, result.Code)
		}
		if result.Stats.ModifiedDocs != 2 {
			t.Errorf("expected 2 modified docs, got %d", result.Stats.ModifiedDocs)
		}

		// Verify the field was added
		if ctx.Documents[0].Dimensions["category"] != "general" {
			t.Errorf("expected category='general' in doc1, got %v", ctx.Documents[0].Dimensions["category"])
		}
		if ctx.Documents[1].Dimensions["category"] != "general" {
			t.Errorf("expected category='general' in doc2, got %v", ctx.Documents[1].Dimensions["category"])
		}
	})

	t.Run("add data field", func(t *testing.T) {
		// Setup test documents
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"status": "active",
				},
			},
			{
				UUID:  "doc2",
				Title: "Test Doc 2",
				Dimensions: map[string]interface{}{
					"status": "pending",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &AddField{
			FieldName:    "version",
			DefaultValue: "1.0",
			IsDataField:  true,
		}

		result := cmd.Execute(ctx)

		// Check result
		if !result.Success {
			t.Errorf("expected success, got failure: %v", result.Messages)
		}

		// Verify the data field was added
		if ctx.Documents[0].Dimensions["_data.version"] != "1.0" {
			t.Errorf("expected _data.version='1.0' in doc1, got %v", ctx.Documents[0].Dimensions["_data.version"])
		}
		if ctx.Documents[1].Dimensions["_data.version"] != "1.0" {
			t.Errorf("expected _data.version='1.0' in doc2, got %v", ctx.Documents[1].Dimensions["_data.version"])
		}
	})

	t.Run("add field with various types", func(t *testing.T) {
		testCases := []struct {
			name         string
			defaultValue interface{}
			isDataField  bool
			expectError  bool
		}{
			{"string dimension", "test", false, false},
			{"int dimension", 42, false, false},
			{"float dimension", 3.14, false, false},
			{"bool dimension", true, false, false},
			{"nil dimension", nil, false, false},
			{"array dimension", []string{"a", "b"}, false, true},         // Should fail for dimension
			{"map dimension", map[string]string{"a": "b"}, false, true},  // Should fail for dimension
			{"array data field", []string{"a", "b"}, true, false},        // OK for data field
			{"map data field", map[string]string{"a": "b"}, true, false}, // OK for data field
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				docs := []types.Document{
					{
						UUID:       "doc1",
						Title:      "Test Doc",
						Dimensions: map[string]interface{}{},
					},
				}

				ctx := &MigrationContext{
					Documents: docs,
					Config:    types.Config{},
					DryRun:    false,
				}

				cmd := &AddField{
					FieldName:    "test_field",
					DefaultValue: tc.defaultValue,
					IsDataField:  tc.isDataField,
				}

				result := cmd.Execute(ctx)

				if tc.expectError {
					if result.Success {
						t.Error("expected failure for invalid type")
					}
					if result.Code != CodeValidationError {
						t.Errorf("expected validation error code, got %d", result.Code)
					}
				} else {
					if !result.Success {
						t.Errorf("expected success, got failure: %v", result.Messages)
					}
				}
			})
		}
	})

	t.Run("field already exists", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"existing": "value",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &AddField{
			FieldName:    "existing",
			DefaultValue: "new value",
			IsDataField:  false,
		}

		result := cmd.Execute(ctx)

		// Should fail validation
		if result.Success {
			t.Error("expected failure when field already exists")
		}
		if result.Code != CodeValidationError {
			t.Errorf("expected validation error code, got %d", result.Code)
		}

		// Original value should be unchanged
		if ctx.Documents[0].Dimensions["existing"] != "value" {
			t.Error("existing field value was modified")
		}
	})

	t.Run("dry run mode", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"status": "active",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    true,
		}

		cmd := &AddField{
			FieldName:    "new_field",
			DefaultValue: "value",
			IsDataField:  false,
		}

		result := cmd.Execute(ctx)

		// Check result
		if !result.Success {
			t.Errorf("expected success, got failure")
		}

		// Verify no changes were made
		if _, exists := ctx.Documents[0].Dimensions["new_field"]; exists {
			t.Error("field was added in dry run mode")
		}
	})

	t.Run("empty field name", func(t *testing.T) {
		ctx := &MigrationContext{
			Documents: []types.Document{},
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &AddField{
			FieldName:    "",
			DefaultValue: "value",
			IsDataField:  false,
		}

		result := cmd.Execute(ctx)

		// Should fail validation
		if result.Success {
			t.Error("expected failure for empty field name")
		}
		if result.Code != CodeValidationError {
			t.Errorf("expected validation error code, got %d", result.Code)
		}
	})

	t.Run("field name with data prefix", func(t *testing.T) {
		ctx := &MigrationContext{
			Documents: []types.Document{
				{UUID: "doc1", Dimensions: map[string]interface{}{}},
			},
			Config: types.Config{},
			DryRun: false,
		}

		cmd := &AddField{
			FieldName:    "_data.field",
			DefaultValue: "value",
			IsDataField:  false,
		}

		result := cmd.Execute(ctx)

		// Should fail validation
		if result.Success {
			t.Error("expected failure for field name starting with _data.")
		}

		hasError := false
		for _, msg := range result.Messages {
			if msg.Level == LevelError && strings.Contains(msg.Text, "_data.") {
				hasError = true
			}
		}
		if !hasError {
			t.Error("expected error about _data. prefix")
		}
	})
}

func TestAddFieldWithExistingFields(t *testing.T) {
	t.Run("skip existing fields", func(t *testing.T) {
		// Mix of documents with and without the field
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Has field",
				Dimensions: map[string]interface{}{
					"field": "existing",
				},
			},
			{
				UUID:  "doc2",
				Title: "No field",
				Dimensions: map[string]interface{}{
					"other": "value",
				},
			},
			{
				UUID:  "doc3",
				Title: "No field",
				Dimensions: map[string]interface{}{
					"other": "value",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &AddField{
			FieldName:    "field",
			DefaultValue: "default",
			IsDataField:  false,
		}

		// This should fail during validation
		result := cmd.Execute(ctx)

		if result.Success {
			t.Error("expected failure when field exists in some documents")
		}
		if result.Code != CodeValidationError {
			t.Errorf("expected validation error, got code %d", result.Code)
		}
	})
}
