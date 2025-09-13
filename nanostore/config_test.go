package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/types"
)

func TestDefaultConfig(t *testing.T) {
	config := nanostore.DefaultConfig()

	// Should have exactly 2 dimensions
	if len(config.Dimensions) != 2 {
		t.Errorf("expected 2 dimensions, got %d", len(config.Dimensions))
	}

	// First dimension should be status
	statusDim := config.Dimensions[0]
	if statusDim.Name != "status" {
		t.Errorf("expected first dimension to be 'status', got '%s'", statusDim.Name)
	}
	if statusDim.Type != types.Enumerated {
		t.Errorf("expected status dimension to be Enumerated, got %d", statusDim.Type)
	}
	if len(statusDim.Values) != 2 {
		t.Errorf("expected status dimension to have 2 values, got %d", len(statusDim.Values))
	}

	// Second dimension should be parent
	parentDim := config.Dimensions[1]
	if parentDim.Name != "parent" {
		t.Errorf("expected second dimension to be 'parent', got '%s'", parentDim.Name)
	}
	if parentDim.Type != types.Hierarchical {
		t.Errorf("expected parent dimension to be Hierarchical, got %d", parentDim.Type)
	}
	if parentDim.RefField != "parent_uuid" {
		t.Errorf("expected parent dimension RefField to be 'parent_uuid', got '%s'", parentDim.RefField)
	}

	// Config should be valid
	if err := nanostore.ValidateConfig(config); err != nil {
		t.Errorf("default config should be valid: %v", err)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    types.Config
		shouldErr bool
		errorMsg  string
	}{
		{
			name: "valid config",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:         "status",
						Type:         types.Enumerated,
						Values:       []string{"pending", "completed"},
						Prefixes:     map[string]string{"completed": "c"},
						DefaultValue: "pending",
					},
				},
			},
			shouldErr: false,
		},
		{
			name: "empty dimensions",
			config: types.Config{
				Dimensions: []types.DimensionConfig{},
			},
			shouldErr: true,
			errorMsg:  "at least one dimension must be configured",
		},
		{
			name: "empty dimension name",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name: "",
						Type: types.Enumerated,
					},
				},
			},
			shouldErr: true,
			errorMsg:  "name cannot be empty",
		},
		{
			name: "reserved column name",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name: "uuid",
						Type: types.Enumerated,
					},
				},
			},
			shouldErr: true,
			errorMsg:  "reserved column name",
		},
		{
			name: "duplicate dimension names",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:   "status",
						Type:   types.Enumerated,
						Values: []string{"pending"},
					},
					{
						Name:   "status",
						Type:   types.Enumerated,
						Values: []string{"active"},
					},
				},
			},
			shouldErr: true,
			errorMsg:  "duplicate dimension name",
		},
		{
			name: "enumerated dimension without values",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:   "status",
						Type:   types.Enumerated,
						Values: []string{},
					},
				},
			},
			shouldErr: true,
			errorMsg:  "must have at least one value",
		},
		{
			name: "enumerated dimension with empty value",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:   "status",
						Type:   types.Enumerated,
						Values: []string{"pending", ""},
					},
				},
			},
			shouldErr: true,
			errorMsg:  "values cannot be empty",
		},
		{
			name: "enumerated dimension with duplicate values",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:   "status",
						Type:   types.Enumerated,
						Values: []string{"pending", "pending"},
					},
				},
			},
			shouldErr: true,
			errorMsg:  "duplicate value",
		},
		{
			name: "invalid default value",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:         "status",
						Type:         types.Enumerated,
						Values:       []string{"pending", "completed"},
						DefaultValue: "active",
					},
				},
			},
			shouldErr: true,
			errorMsg:  "default value 'active' is not in values list",
		},
		{
			name: "prefix for unknown value",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:     "status",
						Type:     types.Enumerated,
						Values:   []string{"pending", "completed"},
						Prefixes: map[string]string{"active": "a"},
					},
				},
			},
			shouldErr: true,
			errorMsg:  "prefix defined for unknown value",
		},
		{
			name: "empty prefix",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:     "status",
						Type:     types.Enumerated,
						Values:   []string{"pending", "completed"},
						Prefixes: map[string]string{"completed": ""},
					},
				},
			},
			shouldErr: true,
			errorMsg:  "prefix for value 'completed' cannot be empty",
		},
		{
			name: "invalid prefix characters",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:     "status",
						Type:     types.Enumerated,
						Values:   []string{"pending", "completed"},
						Prefixes: map[string]string{"completed": "C1"},
					},
				},
			},
			shouldErr: true,
			errorMsg:  "contains invalid characters",
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
						Name:     "priority",
						Type:     types.Enumerated,
						Values:   []string{"low", "critical"},
						Prefixes: map[string]string{"critical": "c"},
					},
				},
			},
			shouldErr: true,
			errorMsg:  "prefix 'c' conflicts",
		},
		{
			name: "enumerated with RefField",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:     "status",
						Type:     types.Enumerated,
						Values:   []string{"pending"},
						RefField: "parent_uuid",
					},
				},
			},
			shouldErr: true,
			errorMsg:  "RefField should not be set for enumerated dimensions",
		},
		{
			name: "hierarchical without RefField",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name: "parent",
						Type: types.Hierarchical,
					},
				},
			},
			shouldErr: true,
			errorMsg:  "hierarchical dimensions must specify RefField",
		},
		{
			name: "hierarchical with reserved RefField",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:     "parent",
						Type:     types.Hierarchical,
						RefField: "uuid",
					},
				},
			},
			shouldErr: true,
			errorMsg:  "RefField 'uuid' is a reserved column name",
		},
		{
			name: "hierarchical with Values",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:     "parent",
						Type:     types.Hierarchical,
						Values:   []string{"root"},
						RefField: "parent_uuid",
					},
				},
			},
			shouldErr: true,
			errorMsg:  "Values should not be set for hierarchical dimensions",
		},
		{
			name: "hierarchical with Prefixes",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:     "parent",
						Type:     types.Hierarchical,
						Prefixes: map[string]string{"root": "r"},
						RefField: "parent_uuid",
					},
				},
			},
			shouldErr: true,
			errorMsg:  "Prefixes should not be set for hierarchical dimensions",
		},
		{
			name: "hierarchical with DefaultValue",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:         "parent",
						Type:         types.Hierarchical,
						DefaultValue: "root",
						RefField:     "parent_uuid",
					},
				},
			},
			shouldErr: true,
			errorMsg:  "DefaultValue should not be set for hierarchical dimensions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := nanostore.ValidateConfig(tt.config)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorMsg)
				} else if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestConfigHelperMethods(t *testing.T) {
	config := types.Config{
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
			{
				Name:     "parent",
				Type:     types.Hierarchical,
				RefField: "parent_uuid",
			},
			{
				Name:     "project",
				Type:     types.Hierarchical,
				RefField: "project_uuid",
			},
		},
	}

	// Test GetEnumeratedDimensions
	enumerated := nanostore.GetEnumeratedDimensions(config)
	if len(enumerated) != 2 {
		t.Errorf("expected 2 enumerated dimensions, got %d", len(enumerated))
	}
	if enumerated[0].Name != "status" || enumerated[1].Name != "priority" {
		t.Errorf("unexpected enumerated dimensions: %v", enumerated)
	}

	// Test GetHierarchicalDimensions
	hierarchical := nanostore.GetHierarchicalDimensions(config)
	if len(hierarchical) != 2 {
		t.Errorf("expected 2 hierarchical dimensions, got %d", len(hierarchical))
	}
	if hierarchical[0].Name != "parent" || hierarchical[1].Name != "project" {
		t.Errorf("unexpected hierarchical dimensions: %v", hierarchical)
	}

	// Test GetDimension
	statusDim, found := nanostore.GetDimension(config, "status")
	if !found {
		t.Errorf("expected to find status dimension")
	}
	if statusDim.Name != "status" {
		t.Errorf("expected status dimension, got %v", statusDim)
	}

	_, found = nanostore.GetDimension(config, "nonexistent")
	if found {
		t.Errorf("expected not to find nonexistent dimension")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) &&
		(s == substr || s[len(s)-len(substr):] == substr ||
			s[:len(substr)] == substr ||
			len(s) > len(substr) &&
				func() bool {
					for i := 1; i <= len(s)-len(substr); i++ {
						if s[i:i+len(substr)] == substr {
							return true
						}
					}
					return false
				}())
}
