package engine

import (
	"reflect"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore/types"
)

func TestIDParser_ParseID(t *testing.T) {
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:         "status",
				Type:         types.Enumerated,
				Values:       []string{"pending", "completed", "blocked"},
				Prefixes:     map[string]string{"completed": "c", "blocked": "b"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         types.Enumerated,
				Values:       []string{"low", "normal", "high"},
				Prefixes:     map[string]string{"high": "h"},
				DefaultValue: "normal",
			},
			{
				Name:     "parent",
				Type:     types.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	parser := NewIDParser(config)

	tests := []struct {
		name     string
		input    string
		expected *ParsedID
		wantErr  bool
	}{
		{
			name:  "simple number",
			input: "1",
			expected: &ParsedID{
				Levels: []ParsedLevel{
					{
						DimensionFilters: map[string]string{
							"status":   "pending",
							"priority": "normal",
						},
						Offset: 0,
					},
				},
			},
		},
		{
			name:  "single prefix",
			input: "c2",
			expected: &ParsedID{
				Levels: []ParsedLevel{
					{
						DimensionFilters: map[string]string{
							"status":   "completed",
							"priority": "normal",
						},
						Offset: 1,
					},
				},
			},
		},
		{
			name:  "multiple prefixes",
			input: "hc3",
			expected: &ParsedID{
				Levels: []ParsedLevel{
					{
						DimensionFilters: map[string]string{
							"status":   "completed",
							"priority": "high",
						},
						Offset: 2,
					},
				},
			},
		},
		{
			name:  "hierarchical",
			input: "1.2.3",
			expected: &ParsedID{
				Levels: []ParsedLevel{
					{
						DimensionFilters: map[string]string{
							"status":   "pending",
							"priority": "normal",
						},
						Offset: 0,
					},
					{
						DimensionFilters: map[string]string{
							"status":   "pending",
							"priority": "normal",
						},
						Offset: 1,
					},
					{
						DimensionFilters: map[string]string{
							"status":   "pending",
							"priority": "normal",
						},
						Offset: 2,
					},
				},
			},
		},
		{
			name:  "hierarchical with prefixes",
			input: "h1.c2.b3",
			expected: &ParsedID{
				Levels: []ParsedLevel{
					{
						DimensionFilters: map[string]string{
							"status":   "pending",
							"priority": "high",
						},
						Offset: 0,
					},
					{
						DimensionFilters: map[string]string{
							"status":   "completed",
							"priority": "normal",
						},
						Offset: 1,
					},
					{
						DimensionFilters: map[string]string{
							"status":   "blocked",
							"priority": "normal",
						},
						Offset: 2,
					},
				},
			},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "empty segment",
			input:   "1..2",
			wantErr: true,
		},
		{
			name:    "no number",
			input:   "c",
			wantErr: true,
		},
		{
			name:    "invalid number",
			input:   "c0",
			wantErr: true,
		},
		{
			name:    "negative number",
			input:   "c-1",
			wantErr: true,
		},
		{
			name:    "unknown prefix",
			input:   "x1",
			wantErr: true,
		},
		{
			name:    "sql injection attempt",
			input:   "1'; DROP TABLE--",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.ParseID(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ParseID() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func TestIDParser_GenerateID(t *testing.T) {
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:     "priority",
				Type:     types.Enumerated,
				Values:   []string{"low", "normal", "high"},
				Prefixes: map[string]string{"high": "h"},
			},
			{
				Name:     "status",
				Type:     types.Enumerated,
				Values:   []string{"pending", "completed"},
				Prefixes: map[string]string{"completed": "c"},
			},
		},
	}

	parser := NewIDParser(config)

	tests := []struct {
		name       string
		dimensions map[string]string
		offset     int
		expected   string
	}{
		{
			name: "no prefixes",
			dimensions: map[string]string{
				"status":   "pending",
				"priority": "normal",
			},
			offset:   0,
			expected: "1",
		},
		{
			name: "single prefix",
			dimensions: map[string]string{
				"status":   "completed",
				"priority": "normal",
			},
			offset:   1,
			expected: "c2",
		},
		{
			name: "multiple prefixes alphabetical",
			dimensions: map[string]string{
				"status":   "completed",
				"priority": "high",
			},
			offset:   2,
			expected: "hc3", // priority comes after status alphabetically
		},
		{
			name: "offset conversion",
			dimensions: map[string]string{
				"status":   "pending",
				"priority": "normal",
			},
			offset:   99,
			expected: "100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.GenerateID(tt.dimensions, tt.offset)
			if got != tt.expected {
				t.Errorf("GenerateID() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestIDParser_NormalizePrefixes(t *testing.T) {
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:     "priority",
				Type:     types.Enumerated,
				Values:   []string{"low", "normal", "high"},
				Prefixes: map[string]string{"high": "h"},
			},
			{
				Name:     "status",
				Type:     types.Enumerated,
				Values:   []string{"pending", "completed"},
				Prefixes: map[string]string{"completed": "c"},
			},
		},
	}

	parser := NewIDParser(config)

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"h", "h"},
		{"c", "c"},
		{"hc", "hc"},  // Already normalized: priority comes after status alphabetically
		{"ch", "hc"},  // Normalized: priority after status
		{"chh", "hc"}, // Duplicate h ignored (can't have same dimension twice)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parser.NormalizePrefixes(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizePrefixes(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIDParser_ValidateConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		config  types.Config
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:         "status",
						Type:         types.Enumerated,
						Values:       []string{"pending", "completed"},
						Prefixes:     map[string]string{"completed": "c"},
						DefaultValue: "pending",
					},
					{
						Name:     "parent",
						Type:     types.Hierarchical,
						RefField: "parent_uuid",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "conflicting prefixes",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:     "status",
						Type:     types.Enumerated,
						Values:   []string{"pending", "completed"},
						Prefixes: map[string]string{"completed": "c"},
					},
					{
						Name:     "category",
						Type:     types.Enumerated,
						Values:   []string{"critical", "normal"},
						Prefixes: map[string]string{"critical": "c"}, // Conflict!
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no conflicts with empty prefixes",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:   "status",
						Type:   types.Enumerated,
						Values: []string{"pending", "completed"},
					},
					{
						Name:   "priority",
						Type:   types.Enumerated,
						Values: []string{"low", "high"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip config validation in this test since we're testing parser validation

			parser := NewIDParser(tt.config)
			err := parser.ValidateConfiguration()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
