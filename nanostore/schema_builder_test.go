package nanostore

import (
	"testing"
)

func TestValidateSchemaCompatibility(t *testing.T) {
	tests := []struct {
		name            string
		config          Config
		existingColumns map[string]string
		wantErr         bool
		errContains     string
	}{
		{
			name: "compatible schema - all columns are TEXT",
			config: Config{
				Dimensions: []DimensionConfig{
					{Name: "status", Type: Enumerated},
					{Name: "priority", Type: Enumerated},
					{Name: "parent", Type: Hierarchical, RefField: "parent_uuid"},
				},
			},
			existingColumns: map[string]string{
				"status":      "TEXT",
				"priority":    "TEXT",
				"parent_uuid": "TEXT",
			},
			wantErr: false,
		},
		{
			name: "incompatible enumerated dimension - not TEXT",
			config: Config{
				Dimensions: []DimensionConfig{
					{Name: "status", Type: Enumerated},
				},
			},
			existingColumns: map[string]string{
				"status": "INTEGER",
			},
			wantErr:     true,
			errContains: "dimension 'status' exists with incompatible type 'INTEGER', expected TEXT",
		},
		{
			name: "incompatible hierarchical dimension - not TEXT",
			config: Config{
				Dimensions: []DimensionConfig{
					{Name: "parent", Type: Hierarchical, RefField: "parent_id"},
				},
			},
			existingColumns: map[string]string{
				"parent_id": "BLOB",
			},
			wantErr:     true,
			errContains: "hierarchical dimension 'parent' field 'parent_id' exists with incompatible type 'BLOB', expected TEXT",
		},
		{
			name: "schema with missing columns - should be valid",
			config: Config{
				Dimensions: []DimensionConfig{
					{Name: "status", Type: Enumerated},
					{Name: "priority", Type: Enumerated},
				},
			},
			existingColumns: map[string]string{
				"status": "TEXT",
				// priority column missing - this is OK
			},
			wantErr: false,
		},
		{
			name: "schema with extra columns - should be valid",
			config: Config{
				Dimensions: []DimensionConfig{
					{Name: "status", Type: Enumerated},
				},
			},
			existingColumns: map[string]string{
				"status":      "TEXT",
				"extra_col":   "TEXT",
				"another_col": "INTEGER",
			},
			wantErr: false,
		},
		{
			name: "empty config with existing columns - should be valid",
			config: Config{
				Dimensions: []DimensionConfig{},
			},
			existingColumns: map[string]string{
				"some_column": "TEXT",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := &schemaBuilder{config: tt.config}
			err := sb.ValidateSchemaCompatibility(tt.existingColumns)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateSchemaCompatibility() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && err.Error() != tt.errContains {
					t.Errorf("ValidateSchemaCompatibility() error = %v, want %v", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Errorf("ValidateSchemaCompatibility() unexpected error = %v", err)
			}
		})
	}
}

func TestGetExpectedColumns(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected map[string]string
	}{
		{
			name: "no dimensions - base columns only",
			config: Config{
				Dimensions: []DimensionConfig{},
			},
			expected: map[string]string{
				"uuid":       "TEXT",
				"title":      "TEXT",
				"body":       "TEXT",
				"created_at": "INTEGER",
				"updated_at": "INTEGER",
			},
		},
		{
			name: "with enumerated dimensions only",
			config: Config{
				Dimensions: []DimensionConfig{
					{Name: "status", Type: Enumerated},
					{Name: "priority", Type: Enumerated},
					{Name: "category", Type: Enumerated},
				},
			},
			expected: map[string]string{
				"uuid":       "TEXT",
				"title":      "TEXT",
				"body":       "TEXT",
				"created_at": "INTEGER",
				"updated_at": "INTEGER",
				"status":     "TEXT",
				"priority":   "TEXT",
				"category":   "TEXT",
			},
		},
		{
			name: "with hierarchical dimensions only",
			config: Config{
				Dimensions: []DimensionConfig{
					{Name: "parent", Type: Hierarchical, RefField: "parent_uuid"},
					{Name: "folder", Type: Hierarchical, RefField: "folder_id"},
				},
			},
			expected: map[string]string{
				"uuid":        "TEXT",
				"title":       "TEXT",
				"body":        "TEXT",
				"created_at":  "INTEGER",
				"updated_at":  "INTEGER",
				"parent_uuid": "TEXT",
				"folder_id":   "TEXT",
			},
		},
		{
			name: "with mixed dimension types",
			config: Config{
				Dimensions: []DimensionConfig{
					{Name: "status", Type: Enumerated},
					{Name: "priority", Type: Enumerated},
					{Name: "parent", Type: Hierarchical, RefField: "parent_uuid"},
					{Name: "category", Type: Enumerated},
					{Name: "project", Type: Hierarchical, RefField: "project_id"},
				},
			},
			expected: map[string]string{
				"uuid":        "TEXT",
				"title":       "TEXT",
				"body":        "TEXT",
				"created_at":  "INTEGER",
				"updated_at":  "INTEGER",
				"status":      "TEXT",
				"priority":    "TEXT",
				"parent_uuid": "TEXT",
				"category":    "TEXT",
				"project_id":  "TEXT",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := &schemaBuilder{config: tt.config}
			got := sb.GetExpectedColumns()

			// Check that we have the right number of columns
			if len(got) != len(tt.expected) {
				t.Errorf("GetExpectedColumns() returned %d columns, expected %d", len(got), len(tt.expected))
			}

			// Check each expected column
			for col, expectedType := range tt.expected {
				gotType, exists := got[col]
				if !exists {
					t.Errorf("GetExpectedColumns() missing column %s", col)
					continue
				}
				if gotType != expectedType {
					t.Errorf("GetExpectedColumns() column %s has type %s, expected %s", col, gotType, expectedType)
				}
			}

			// Check for unexpected columns
			for col := range got {
				if _, expected := tt.expected[col]; !expected {
					t.Errorf("GetExpectedColumns() has unexpected column %s", col)
				}
			}
		})
	}
}
