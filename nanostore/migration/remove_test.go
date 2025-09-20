package migration

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/types"
)

func TestRemoveField(t *testing.T) {
	t.Run("remove dimension field", func(t *testing.T) {
		// Setup test documents
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"status":    "active",
					"priority":  "high",
					"to_remove": "value1",
				},
			},
			{
				UUID:  "doc2",
				Title: "Test Doc 2",
				Dimensions: map[string]interface{}{
					"status":    "pending",
					"priority":  "low",
					"to_remove": "value2",
				},
			},
			{
				UUID:  "doc3",
				Title: "Test Doc 3",
				Dimensions: map[string]interface{}{
					"status":   "done",
					"priority": "medium",
					// No to_remove field
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &RemoveField{
			FieldName: "to_remove",
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

		// Verify the removal happened
		if _, exists := ctx.Documents[0].Dimensions["to_remove"]; exists {
			t.Error("to_remove still exists in doc1")
		}
		if _, exists := ctx.Documents[1].Dimensions["to_remove"]; exists {
			t.Error("to_remove still exists in doc2")
		}

		// Verify other fields remain
		if ctx.Documents[0].Dimensions["status"] != "active" {
			t.Error("status field was affected")
		}
		if ctx.Documents[1].Dimensions["priority"] != "low" {
			t.Error("priority field was affected")
		}
	})

	t.Run("remove data field", func(t *testing.T) {
		// Setup test documents with data fields
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"status":         "active",
					"_data.metadata": "some metadata",
				},
			},
			{
				UUID:  "doc2",
				Title: "Test Doc 2",
				Dimensions: map[string]interface{}{
					"status":         "pending",
					"_data.metadata": "other metadata",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &RemoveField{
			FieldName: "metadata",
		}

		result := cmd.Execute(ctx)

		// Check result
		if !result.Success {
			t.Errorf("expected success, got failure: %v", result.Messages)
		}

		// Verify the removal happened for data fields
		if _, exists := ctx.Documents[0].Dimensions["_data.metadata"]; exists {
			t.Error("_data.metadata still exists in doc1")
		}
		if _, exists := ctx.Documents[1].Dimensions["_data.metadata"]; exists {
			t.Error("_data.metadata still exists in doc2")
		}
	})

	t.Run("remove mixed field", func(t *testing.T) {
		// Setup documents where same field name exists as both dimension and data
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"field": "dimension value",
				},
			},
			{
				UUID:  "doc2",
				Title: "Test Doc 2",
				Dimensions: map[string]interface{}{
					"_data.field": "data value",
				},
			},
			{
				UUID:  "doc3",
				Title: "Test Doc 3",
				Dimensions: map[string]interface{}{
					"field":       "dimension value",
					"_data.field": "data value",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &RemoveField{
			FieldName: "field",
		}

		result := cmd.Execute(ctx)

		// Should remove both dimension and data fields
		if result.Stats.ModifiedDocs != 3 {
			t.Errorf("expected 3 modified docs, got %d", result.Stats.ModifiedDocs)
		}

		// Verify all occurrences removed
		for i, doc := range ctx.Documents {
			if _, exists := doc.Dimensions["field"]; exists {
				t.Errorf("field still exists in doc%d", i+1)
			}
			if _, exists := doc.Dimensions["_data.field"]; exists {
				t.Errorf("_data.field still exists in doc%d", i+1)
			}
		}
	})

	t.Run("dry run mode", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"remove_me": "value",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    true,
		}

		cmd := &RemoveField{
			FieldName: "remove_me",
		}

		result := cmd.Execute(ctx)

		// Check result
		if !result.Success {
			t.Errorf("expected success, got failure")
		}

		// Verify no changes were made
		if _, exists := ctx.Documents[0].Dimensions["remove_me"]; !exists {
			t.Error("field was removed in dry run mode")
		}
	})

	t.Run("field not found", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID:       "doc1",
				Title:      "Test Doc 1",
				Dimensions: map[string]interface{}{"other": "value"},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &RemoveField{
			FieldName: "nonexistent",
		}

		result := cmd.Execute(ctx)

		// Should succeed with warning
		if !result.Success {
			t.Errorf("expected success with warning, got failure")
		}
		if result.Stats.ModifiedDocs != 0 {
			t.Errorf("expected 0 modified docs, got %d", result.Stats.ModifiedDocs)
		}

		// Check for warning message
		hasWarning := false
		for _, msg := range result.Messages {
			if msg.Level == LevelWarning && strings.Contains(msg.Text, "not found") {
				hasWarning = true
			}
		}
		if !hasWarning {
			t.Error("expected warning message for nonexistent field")
		}
	})

	t.Run("empty field name", func(t *testing.T) {
		ctx := &MigrationContext{
			Documents: []types.Document{},
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &RemoveField{
			FieldName: "",
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
}

func TestRemoveFieldValidation(t *testing.T) {
	t.Run("dimension field warning", func(t *testing.T) {
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
		}

		cmd := &RemoveField{
			FieldName: "status",
		}

		messages := cmd.Validate(ctx)

		// Should have warning about dimension field
		hasWarning := false
		for _, msg := range messages {
			if msg.Level == LevelWarning && strings.Contains(msg.Text, "dimension field") {
				hasWarning = true
			}
		}
		if !hasWarning {
			t.Error("expected warning about removing dimension field")
		}
	})

	t.Run("field statistics", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "doc1",
				Dimensions: map[string]interface{}{
					"field": "value1",
				},
			},
			{
				UUID: "doc2",
				Dimensions: map[string]interface{}{
					"_data.field": "value2",
				},
			},
			{
				UUID: "doc3",
				Dimensions: map[string]interface{}{
					"field":       "value3",
					"_data.field": "value4",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
		}

		cmd := &RemoveField{
			FieldName: "field",
		}

		messages := cmd.Validate(ctx)

		// Should report mixed field type and counts
		hasInfo := false
		for _, msg := range messages {
			if msg.Level == LevelInfo {
				hasInfo = true
				if msg.Details != nil {
					dimCount, _ := msg.Details["dimension_occurrences"].(int)
					dataCount, _ := msg.Details["data_occurrences"].(int)
					if dimCount != 2 {
						t.Errorf("expected 2 dimension occurrences, got %d", dimCount)
					}
					if dataCount != 2 {
						t.Errorf("expected 2 data occurrences, got %d", dataCount)
					}
				}
			}
		}
		if !hasInfo {
			t.Error("expected info message with statistics")
		}
	})
}
