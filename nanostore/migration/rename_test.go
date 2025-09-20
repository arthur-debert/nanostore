package migration

import (
	"testing"

	"github.com/arthur-debert/nanostore/types"
)

func TestRenameField(t *testing.T) {
	t.Run("rename dimension field", func(t *testing.T) {
		// Setup test documents
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"old_status": "active",
					"priority":   "high",
				},
			},
			{
				UUID:  "doc2",
				Title: "Test Doc 2",
				Dimensions: map[string]interface{}{
					"old_status": "pending",
					"priority":   "low",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &RenameField{
			OldName: "old_status",
			NewName: "status",
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

		// Verify the rename happened
		if ctx.Documents[0].Dimensions["status"] != "active" {
			t.Errorf("expected doc1 status to be 'active', got %v", ctx.Documents[0].Dimensions["status"])
		}
		if ctx.Documents[1].Dimensions["status"] != "pending" {
			t.Errorf("expected doc2 status to be 'pending', got %v", ctx.Documents[1].Dimensions["status"])
		}

		// Verify old field is gone
		if _, exists := ctx.Documents[0].Dimensions["old_status"]; exists {
			t.Error("old_status still exists in doc1")
		}
		if _, exists := ctx.Documents[1].Dimensions["old_status"]; exists {
			t.Error("old_status still exists in doc2")
		}
	})

	t.Run("rename data field", func(t *testing.T) {
		// Setup test documents with data fields
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"status":         "active",
					"_data.old_prop": "value1",
				},
			},
			{
				UUID:  "doc2",
				Title: "Test Doc 2",
				Dimensions: map[string]interface{}{
					"status":         "pending",
					"_data.old_prop": "value2",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &RenameField{
			OldName: "old_prop",
			NewName: "new_prop",
		}

		result := cmd.Execute(ctx)

		// Check result
		if !result.Success {
			t.Errorf("expected success, got failure: %v", result.Messages)
		}

		// Verify the rename happened for data fields
		if ctx.Documents[0].Dimensions["_data.new_prop"] != "value1" {
			t.Errorf("expected doc1 _data.new_prop to be 'value1', got %v", ctx.Documents[0].Dimensions["_data.new_prop"])
		}
		if ctx.Documents[1].Dimensions["_data.new_prop"] != "value2" {
			t.Errorf("expected doc2 _data.new_prop to be 'value2', got %v", ctx.Documents[1].Dimensions["_data.new_prop"])
		}

		// Verify old field is gone
		if _, exists := ctx.Documents[0].Dimensions["_data.old_prop"]; exists {
			t.Error("_data.old_prop still exists in doc1")
		}
	})

	t.Run("dry run mode", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"old_field": "value",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    true,
		}

		cmd := &RenameField{
			OldName: "old_field",
			NewName: "new_field",
		}

		result := cmd.Execute(ctx)

		// Check result
		if !result.Success {
			t.Errorf("expected success, got failure")
		}

		// Verify no changes were made
		if _, exists := ctx.Documents[0].Dimensions["new_field"]; exists {
			t.Error("new_field exists in dry run mode")
		}
		if _, exists := ctx.Documents[0].Dimensions["old_field"]; !exists {
			t.Error("old_field was removed in dry run mode")
		}
	})

	t.Run("field not found", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID:       "doc1",
				Title:      "Test Doc 1",
				Dimensions: map[string]interface{}{},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &RenameField{
			OldName: "nonexistent",
			NewName: "new_field",
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
			if msg.Level == LevelWarning {
				hasWarning = true
			}
		}
		if !hasWarning {
			t.Error("expected warning message for nonexistent field")
		}
	})

	t.Run("new field already exists", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"old_field": "value1",
					"new_field": "value2",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &RenameField{
			OldName: "old_field",
			NewName: "new_field",
		}

		result := cmd.Execute(ctx)

		// Should fail validation
		if result.Success {
			t.Error("expected failure when new field already exists")
		}
		if result.Code != CodeValidationError {
			t.Errorf("expected validation error code, got %d", result.Code)
		}

		// Original data should be unchanged
		if ctx.Documents[0].Dimensions["old_field"] != "value1" {
			t.Error("old_field was modified despite validation error")
		}
	})

	t.Run("same field name", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"field": "value",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &RenameField{
			OldName: "field",
			NewName: "field",
		}

		result := cmd.Execute(ctx)

		// Should fail validation
		if result.Success {
			t.Error("expected failure when old and new names are the same")
		}
		if result.Code != CodeValidationError {
			t.Errorf("expected validation error code, got %d", result.Code)
		}
	})
}

func TestRenameFieldValidation(t *testing.T) {
	t.Run("empty field names", func(t *testing.T) {
		ctx := &MigrationContext{
			Documents: []types.Document{},
			Config:    types.Config{},
		}

		testCases := []struct {
			name    string
			oldName string
			newName string
			wantErr bool
		}{
			{"empty old name", "", "new", true},
			{"empty new name", "old", "", true},
			{"whitespace old name", "  ", "new", true},
			{"whitespace new name", "old", "  ", true},
			{"valid names", "old", "new", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cmd := &RenameField{
					OldName: tc.oldName,
					NewName: tc.newName,
				}

				messages := cmd.Validate(ctx)
				hasError := false
				for _, msg := range messages {
					if msg.Level == LevelError {
						hasError = true
					}
				}

				if hasError != tc.wantErr {
					t.Errorf("expected error=%v, got error=%v", tc.wantErr, hasError)
				}
			})
		}
	})
}

func TestRenameFieldStats(t *testing.T) {
	// Create documents with varying field presence
	docs := []types.Document{
		{
			UUID:  "doc1",
			Title: "Has field",
			Dimensions: map[string]interface{}{
				"target_field": "value1",
			},
		},
		{
			UUID:  "doc2",
			Title: "No field",
			Dimensions: map[string]interface{}{
				"other_field": "value2",
			},
		},
		{
			UUID:  "doc3",
			Title: "Has field",
			Dimensions: map[string]interface{}{
				"target_field": "value3",
			},
		},
	}

	ctx := &MigrationContext{
		Documents: docs,
		Config:    types.Config{},
		DryRun:    false,
	}

	cmd := &RenameField{
		OldName: "target_field",
		NewName: "renamed_field",
	}

	result := cmd.Execute(ctx)

	// Check statistics
	if result.Stats.TotalDocs != 3 {
		t.Errorf("expected TotalDocs=3, got %d", result.Stats.TotalDocs)
	}
	if result.Stats.ModifiedDocs != 2 {
		t.Errorf("expected ModifiedDocs=2, got %d", result.Stats.ModifiedDocs)
	}
	if result.Stats.Duration <= 0 {
		t.Error("expected positive duration")
	}

	// Check modified doc list
	if len(result.ModifiedDocs) != 2 {
		t.Errorf("expected 2 modified doc IDs, got %d", len(result.ModifiedDocs))
	}
	expectedModified := map[string]bool{"doc1": true, "doc3": true}
	for _, id := range result.ModifiedDocs {
		if !expectedModified[id] {
			t.Errorf("unexpected modified doc ID: %s", id)
		}
	}
}
