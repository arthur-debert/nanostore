package migration

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/types"
)

func TestValidateSchema(t *testing.T) {
	t.Run("all documents valid", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "doc1",
				Dimensions: map[string]interface{}{
					"status":   "active",
					"priority": "high",
				},
			},
			{
				UUID: "doc2",
				Dimensions: map[string]interface{}{
					"status":   "pending",
					"priority": "low",
				},
			},
		}

		config := types.Config{
			Dimensions: []types.DimensionConfig{
				{
					Name:   "status",
					Type:   types.Enumerated,
					Values: []string{"active", "pending", "completed"},
				},
				{
					Name:   "priority",
					Type:   types.Enumerated,
					Values: []string{"high", "medium", "low"},
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    config,
			DryRun:    false,
		}

		cmd := &ValidateSchema{}
		result := cmd.Execute(ctx)

		if !result.Success {
			t.Errorf("expected success, got failure: %v", result.Messages)
		}
		if result.Code != CodeSuccess {
			t.Errorf("expected code %d, got %d", CodeSuccess, result.Code)
		}
		if result.Stats.ModifiedDocs != 2 {
			t.Errorf("expected 2 valid docs, got %d", result.Stats.ModifiedDocs)
		}
	})

	t.Run("enum validation failure", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "doc1",
				Dimensions: map[string]interface{}{
					"status": "invalid_status",
				},
			},
		}

		config := types.Config{
			Dimensions: []types.DimensionConfig{
				{
					Name:   "status",
					Type:   types.Enumerated,
					Values: []string{"active", "pending", "completed"},
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    config,
			DryRun:    false,
		}

		cmd := &ValidateSchema{}
		result := cmd.Execute(ctx)

		if result.Success {
			t.Error("expected failure for invalid enum value")
		}
		if result.Code != CodeValidationError {
			t.Errorf("expected validation error code, got %d", result.Code)
		}

		// Check error details
		hasEnumError := false
		for _, msg := range result.Messages {
			if msg.Level == LevelError && strings.Contains(msg.Text, "validation errors") {
				hasEnumError = true
				if details, ok := msg.Details["error_samples"].([]map[string]interface{}); ok && len(details) > 0 {
					errors := details[0]["errors"].([]string)
					found := false
					for _, err := range errors {
						if strings.Contains(err, "not in allowed values") {
							found = true
							break
						}
					}
					if !found {
						t.Error("expected enum validation error in details")
					}
				}
			}
		}
		if !hasEnumError {
			t.Error("expected validation error message")
		}
	})

	t.Run("required dimension missing", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "doc1",
				Dimensions: map[string]interface{}{
					"priority": "high",
				},
			},
		}

		config := types.Config{
			Dimensions: []types.DimensionConfig{
				{
					Name:   "status",
					Type:   types.Enumerated,
					Values: []string{"active", "pending"},
					// No default value, so missing will be an error
				},
				{
					Name: "priority",
					Type: types.Enumerated,
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    config,
			DryRun:    false,
		}

		cmd := &ValidateSchema{}
		result := cmd.Execute(ctx)

		if result.Success {
			t.Error("expected failure for missing required dimension")
		}

		// Check for required field error
		foundRequiredError := false
		if details, ok := result.Messages[len(result.Messages)-1].Details["error_samples"].([]map[string]interface{}); ok && len(details) > 0 {
			errors := details[0]["errors"].([]string)
			for _, err := range errors {
				if strings.Contains(err, "dimension is missing") {
					foundRequiredError = true
					break
				}
			}
		}
		if !foundRequiredError {
			t.Error("expected error about missing required dimension")
		}
	})

	t.Run("unknown dimension", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "doc1",
				Dimensions: map[string]interface{}{
					"status":  "active",
					"unknown": "value", // Not in config
				},
			},
		}

		config := types.Config{
			Dimensions: []types.DimensionConfig{
				{
					Name: "status",
					Type: types.Enumerated,
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    config,
			DryRun:    false,
		}

		cmd := &ValidateSchema{}
		result := cmd.Execute(ctx)

		if result.Success {
			t.Error("expected failure for unknown dimension")
		}

		// Check for unknown dimension error
		foundUnknownError := false
		if details, ok := result.Messages[len(result.Messages)-1].Details["error_samples"].([]map[string]interface{}); ok && len(details) > 0 {
			errors := details[0]["errors"].([]string)
			for _, err := range errors {
				if strings.Contains(err, "unknown dimension") {
					foundUnknownError = true
					break
				}
			}
		}
		if !foundUnknownError {
			t.Error("expected error about unknown dimension")
		}
	})

	t.Run("data fields are allowed", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "doc1",
				Dimensions: map[string]interface{}{
					"status":      "active",
					"_data.extra": "some data",
				},
			},
		}

		config := types.Config{
			Dimensions: []types.DimensionConfig{
				{
					Name: "status",
					Type: types.Enumerated,
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    config,
			DryRun:    false,
		}

		cmd := &ValidateSchema{}
		result := cmd.Execute(ctx)

		if !result.Success {
			t.Errorf("expected success, data fields should be allowed: %v", result.Messages)
		}
	})

	t.Run("invalid type for dimension", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "doc1",
				Dimensions: map[string]interface{}{
					"status": []string{"invalid", "type"}, // Arrays not allowed for dimensions
				},
			},
		}

		config := types.Config{
			Dimensions: []types.DimensionConfig{
				{
					Name: "status",
					Type: types.Enumerated,
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    config,
			DryRun:    false,
		}

		cmd := &ValidateSchema{}
		result := cmd.Execute(ctx)

		if result.Success {
			t.Error("expected failure for invalid dimension type")
		}
	})

	t.Run("no config dimensions", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "doc1",
				Dimensions: map[string]interface{}{
					"field1": "value1",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{}, // Empty config
			DryRun:    false,
		}

		cmd := &ValidateSchema{}
		result := cmd.Execute(ctx)

		// Should warn but not fail
		hasWarning := false
		for _, msg := range result.Messages {
			if msg.Level == LevelWarning && strings.Contains(msg.Text, "No dimensions defined") {
				hasWarning = true
			}
		}
		if !hasWarning {
			t.Error("expected warning about no dimensions defined")
		}
	})
}
