package migration

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/types"
)

func TestTransformField(t *testing.T) {
	t.Run("transform string to lowercase", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID:  "doc1",
				Title: "Test Doc 1",
				Dimensions: map[string]interface{}{
					"status": "ACTIVE",
					"name":   "UPPERCASE",
				},
			},
			{
				UUID:  "doc2",
				Title: "Test Doc 2",
				Dimensions: map[string]interface{}{
					"status": "PENDING",
					"name":   "MixedCase",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &TransformField{
			FieldName:       "status",
			TransformerName: "toLowerCase",
		}

		result := cmd.Execute(ctx)

		// Check result
		if !result.Success {
			t.Errorf("expected success, got failure: %v", result.Messages)
		}
		if result.Stats.ModifiedDocs != 2 {
			t.Errorf("expected 2 modified docs, got %d", result.Stats.ModifiedDocs)
		}

		// Verify transformation
		if ctx.Documents[0].Dimensions["status"] != "active" {
			t.Errorf("expected 'active', got %v", ctx.Documents[0].Dimensions["status"])
		}
		if ctx.Documents[1].Dimensions["status"] != "pending" {
			t.Errorf("expected 'pending', got %v", ctx.Documents[1].Dimensions["status"])
		}
		// Other field should be unchanged
		if ctx.Documents[0].Dimensions["name"] != "UPPERCASE" {
			t.Error("unrelated field was modified")
		}
	})

	t.Run("transform string to int", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "doc1",
				Dimensions: map[string]interface{}{
					"count": "42",
				},
			},
			{
				UUID: "doc2",
				Dimensions: map[string]interface{}{
					"count": "123",
				},
			},
			{
				UUID: "doc3",
				Dimensions: map[string]interface{}{
					"count": 999, // Already an int
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &TransformField{
			FieldName:       "count",
			TransformerName: "toInt",
		}

		result := cmd.Execute(ctx)

		if !result.Success {
			t.Errorf("expected success, got failure: %v", result.Messages)
		}

		// Verify transformation
		if ctx.Documents[0].Dimensions["count"] != 42 {
			t.Errorf("expected 42, got %v (%T)", ctx.Documents[0].Dimensions["count"], ctx.Documents[0].Dimensions["count"])
		}
		if ctx.Documents[1].Dimensions["count"] != 123 {
			t.Errorf("expected 123, got %v (%T)", ctx.Documents[1].Dimensions["count"], ctx.Documents[1].Dimensions["count"])
		}
		if ctx.Documents[2].Dimensions["count"] != 999 {
			t.Errorf("expected 999, got %v", ctx.Documents[2].Dimensions["count"])
		}
	})

	t.Run("transform data field", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "doc1",
				Dimensions: map[string]interface{}{
					"status":       "active",
					"_data.config": "  needs trim  ",
				},
			},
			{
				UUID: "doc2",
				Dimensions: map[string]interface{}{
					"status":       "pending",
					"_data.config": "\ttabs and spaces\n",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &TransformField{
			FieldName:       "config",
			TransformerName: "trim",
		}

		result := cmd.Execute(ctx)

		if !result.Success {
			t.Errorf("expected success, got failure: %v", result.Messages)
		}

		// Verify data field transformation
		if ctx.Documents[0].Dimensions["_data.config"] != "needs trim" {
			t.Errorf("expected 'needs trim', got %q", ctx.Documents[0].Dimensions["_data.config"])
		}
		if ctx.Documents[1].Dimensions["_data.config"] != "tabs and spaces" {
			t.Errorf("expected 'tabs and spaces', got %q", ctx.Documents[1].Dimensions["_data.config"])
		}
	})

	t.Run("transform with errors", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "doc1",
				Dimensions: map[string]interface{}{
					"value": "123", // Can convert
				},
			},
			{
				UUID: "doc2",
				Dimensions: map[string]interface{}{
					"value": "not a number", // Cannot convert
				},
			},
			{
				UUID: "doc3",
				Dimensions: map[string]interface{}{
					"value": "456", // Can convert
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &TransformField{
			FieldName:       "value",
			TransformerName: "toInt",
		}

		result := cmd.Execute(ctx)

		// Should fail with partial failure
		if result.Success {
			t.Error("expected failure due to transformation errors")
		}
		if result.Code != CodePartialFailure {
			t.Errorf("expected partial failure code, got %d", result.Code)
		}

		// Check error details
		hasErrorMsg := false
		for _, msg := range result.Messages {
			if msg.Level == LevelError && strings.Contains(msg.Text, "Failed to transform") {
				hasErrorMsg = true
				if msg.Details != nil {
					errors, ok := msg.Details["errors"]
					if !ok {
						t.Error("expected error details")
					} else if len(errors.([]map[string]interface{})) == 0 {
						t.Error("expected at least one error detail")
					}
				}
			}
		}
		if !hasErrorMsg {
			t.Error("expected error message about transformation failure")
		}
	})

	t.Run("unknown transformer", func(t *testing.T) {
		ctx := &MigrationContext{
			Documents: []types.Document{{UUID: "doc1", Dimensions: map[string]interface{}{"field": "value"}}},
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &TransformField{
			FieldName:       "field",
			TransformerName: "unknownTransformer",
		}

		result := cmd.Execute(ctx)

		if result.Success {
			t.Error("expected failure for unknown transformer")
		}
		if result.Code != CodeValidationError {
			t.Errorf("expected validation error, got code %d", result.Code)
		}
	})

	t.Run("dry run mode", func(t *testing.T) {
		docs := []types.Document{
			{
				UUID: "doc1",
				Dimensions: map[string]interface{}{
					"text": "UPPERCASE",
				},
			},
		}

		ctx := &MigrationContext{
			Documents: docs,
			Config:    types.Config{},
			DryRun:    true,
		}

		cmd := &TransformField{
			FieldName:       "text",
			TransformerName: "toLowerCase",
		}

		result := cmd.Execute(ctx)

		if !result.Success {
			t.Errorf("expected success, got failure")
		}

		// Verify no changes in dry run
		if ctx.Documents[0].Dimensions["text"] != "UPPERCASE" {
			t.Error("value was modified in dry run mode")
		}
	})

	t.Run("field not found", func(t *testing.T) {
		ctx := &MigrationContext{
			Documents: []types.Document{{UUID: "doc1", Dimensions: map[string]interface{}{"other": "value"}}},
			Config:    types.Config{},
			DryRun:    false,
		}

		cmd := &TransformField{
			FieldName:       "nonexistent",
			TransformerName: "toString",
		}

		result := cmd.Execute(ctx)

		// Should succeed with warning
		if !result.Success {
			t.Error("expected success with warning")
		}
		if result.Stats.ModifiedDocs != 0 {
			t.Errorf("expected 0 modified docs, got %d", result.Stats.ModifiedDocs)
		}

		// Check for warning
		hasWarning := false
		for _, msg := range result.Messages {
			if msg.Level == LevelWarning && strings.Contains(msg.Text, "not found") {
				hasWarning = true
			}
		}
		if !hasWarning {
			t.Error("expected warning about field not found")
		}
	})
}

func TestTransformers(t *testing.T) {
	t.Run("toString", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected string
		}{
			{nil, ""},
			{"hello", "hello"},
			{123, "123"},
			{3.14, "3.14"},
			{true, "true"},
			{false, "false"},
		}

		for _, tc := range tests {
			result, err := ToString(tc.input)
			if err != nil {
				t.Errorf("unexpected error for %v: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("ToString(%v) = %v, want %v", tc.input, result, tc.expected)
			}
		}
	})

	t.Run("toInt", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected int
			wantErr  bool
		}{
			{nil, 0, false},
			{42, 42, false},
			{int64(123), 123, false},
			{3.14, 3, false},
			{"123", 123, false},
			{" 456 ", 456, false},
			{true, 1, false},
			{false, 0, false},
			{"not a number", 0, true},
			{[]int{1, 2}, 0, true},
		}

		for _, tc := range tests {
			result, err := ToInt(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for %v", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %v: %v", tc.input, err)
				}
				if result != tc.expected {
					t.Errorf("ToInt(%v) = %v, want %v", tc.input, result, tc.expected)
				}
			}
		}
	})

	t.Run("toBool", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected bool
			wantErr  bool
		}{
			{nil, false, false},
			{true, true, false},
			{false, false, false},
			{"true", true, false},
			{"TRUE", true, false},
			{"yes", true, false},
			{"1", true, false},
			{"on", true, false},
			{"false", false, false},
			{"no", false, false},
			{"0", false, false},
			{"off", false, false},
			{"", false, false},
			{1, true, false},
			{0, false, false},
			{3.14, true, false},
			{0.0, false, false},
			{"invalid", false, true},
			{[]int{}, false, true},
		}

		for _, tc := range tests {
			result, err := ToBool(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for %v", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %v: %v", tc.input, err)
				}
				if result != tc.expected {
					t.Errorf("ToBool(%v) = %v, want %v", tc.input, result, tc.expected)
				}
			}
		}
	})

	t.Run("case transformers", func(t *testing.T) {
		// toLowerCase
		lower, _ := ToLowerCase("HELLO World")
		if lower != "hello world" {
			t.Errorf("ToLowerCase failed: got %v", lower)
		}

		// toUpperCase
		upper, _ := ToUpperCase("hello WORLD")
		if upper != "HELLO WORLD" {
			t.Errorf("ToUpperCase failed: got %v", upper)
		}

		// Works with non-strings
		lower, _ = ToLowerCase(123)
		if lower != "123" {
			t.Errorf("ToLowerCase(123) failed: got %v", lower)
		}
	})

	t.Run("trim", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected string
		}{
			{"  hello  ", "hello"},
			{"\t\nworld\t\n", "world"},
			{"no trim", "no trim"},
			{123, "123"},
			{nil, ""},
		}

		for _, tc := range tests {
			result, err := Trim(tc.input)
			if err != nil {
				t.Errorf("unexpected error for %v: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("Trim(%v) = %q, want %q", tc.input, result, tc.expected)
			}
		}
	})
}
