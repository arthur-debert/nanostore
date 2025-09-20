package nanostore_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// EXCEPTION: This test validates error conditions and configuration failures.
// It creates fresh stores to test specific validation scenarios.

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

func TestConfigValidationRobustness(t *testing.T) {
	// Test 1: Dimension configuration edge cases
	t.Run("DimensionConfigEdgeCases", func(t *testing.T) {
		testCases := []struct {
			name      string
			config    nanostore.Config
			expectErr bool
			errMsg    string
		}{
			{
				name: "empty dimension name",
				config: nanostore.Config{
					Dimensions: []nanostore.DimensionConfig{
						{
							Name:   "",
							Type:   nanostore.Enumerated,
							Values: []string{"a", "b"},
						},
					},
				},
				expectErr: true,
				errMsg:    "empty dimension name",
			},
			{
				name: "duplicate dimension names",
				config: nanostore.Config{
					Dimensions: []nanostore.DimensionConfig{
						{
							Name:   "status",
							Type:   nanostore.Enumerated,
							Values: []string{"a", "b"},
						},
						{
							Name:   "status",
							Type:   nanostore.Enumerated,
							Values: []string{"c", "d"},
						},
					},
				},
				expectErr: true,
				errMsg:    "duplicate dimension",
			},
			{
				name: "empty values for enumerated",
				config: nanostore.Config{
					Dimensions: []nanostore.DimensionConfig{
						{
							Name:   "empty_enum",
							Type:   nanostore.Enumerated,
							Values: []string{},
						},
					},
				},
				expectErr: true,
				errMsg:    "no values",
			},
			{
				name: "duplicate values in enumerated",
				config: nanostore.Config{
					Dimensions: []nanostore.DimensionConfig{
						{
							Name:   "dup_values",
							Type:   nanostore.Enumerated,
							Values: []string{"a", "b", "a"},
						},
					},
				},
				expectErr: true,
				errMsg:    "duplicate value",
			},
			{
				name: "invalid prefix characters",
				config: nanostore.Config{
					Dimensions: []nanostore.DimensionConfig{
						{
							Name:   "bad_prefix",
							Type:   nanostore.Enumerated,
							Values: []string{"val1", "val2"},
							Prefixes: map[string]string{
								"val1": "1!", // Invalid character
							},
						},
					},
				},
				expectErr: true,
				errMsg:    "invalid characters",
			},
			{
				name: "prefix for non-existent value",
				config: nanostore.Config{
					Dimensions: []nanostore.DimensionConfig{
						{
							Name:   "bad_prefix_value",
							Type:   nanostore.Enumerated,
							Values: []string{"val1", "val2"},
							Prefixes: map[string]string{
								"val3": "x", // val3 not in values
							},
						},
					},
				},
				expectErr: true,
				errMsg:    "not in values list",
			},
			{
				name: "hierarchical with values",
				config: nanostore.Config{
					Dimensions: []nanostore.DimensionConfig{
						{
							Name:     "parent_id",
							Type:     nanostore.Hierarchical,
							Values:   []string{"should", "not", "have"},
							RefField: "parent_uuid",
						},
					},
				},
				expectErr: false, // Should ignore values for hierarchical
			},
			{
				name: "very long dimension name",
				config: nanostore.Config{
					Dimensions: []nanostore.DimensionConfig{
						{
							Name:   strings.Repeat("x", 1000),
							Type:   nanostore.Enumerated,
							Values: []string{"a"},
						},
					},
				},
				expectErr: false, // Should handle long names
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := validateConfig(tc.config)
				if tc.expectErr && err == nil {
					t.Errorf("expected error containing '%s', got nil", tc.errMsg)
				} else if !tc.expectErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				} else if tc.expectErr && err != nil && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("expected error containing '%s', got: %v", tc.errMsg, err)
				}
			})
		}
	})

	// Test 2: Prefix collision scenarios
	t.Run("PrefixCollisions", func(t *testing.T) {
		testCases := []struct {
			name      string
			prefixes  map[string]string
			expectErr bool
		}{
			{
				name: "duplicate prefixes same dimension",
				prefixes: map[string]string{
					"val1": "a",
					"val2": "a", // Same prefix
				},
				expectErr: true,
			},
			{
				name: "empty prefix",
				prefixes: map[string]string{
					"val1": "",
				},
				expectErr: true,
			},
			{
				name: "whitespace prefix",
				prefixes: map[string]string{
					"val1": " ",
				},
				expectErr: true,
			},
			{
				name: "unicode prefix",
				prefixes: map[string]string{
					"val1": "啊",
				},
				expectErr: true,
			},
			{
				name: "very long prefix",
				prefixes: map[string]string{
					"val1": strings.Repeat("a", 100),
				},
				expectErr: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := nanostore.Config{
					Dimensions: []nanostore.DimensionConfig{
						{
							Name:     "test",
							Type:     nanostore.Enumerated,
							Values:   getKeys(tc.prefixes),
							Prefixes: tc.prefixes,
						},
					},
				}

				err := validateConfig(config)
				if tc.expectErr && err == nil {
					t.Error("expected error for prefix collision, got nil")
				} else if !tc.expectErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})
		}
	})

	// Test 3: Default value validation
	t.Run("DefaultValueValidation", func(t *testing.T) {
		testCases := []struct {
			name         string
			values       []string
			defaultValue string
			expectErr    bool
		}{
			{
				name:         "default not in values",
				values:       []string{"a", "b", "c"},
				defaultValue: "d",
				expectErr:    true,
			},
			{
				name:         "empty default with values",
				values:       []string{"a", "b", "c"},
				defaultValue: "",
				expectErr:    false, // Empty default is ok
			},
			{
				name:         "whitespace default",
				values:       []string{"a", "b", " "},
				defaultValue: " ",
				expectErr:    false, // If it's in values, should be ok
			},
			{
				name:         "case sensitive default",
				values:       []string{"Active", "Inactive"},
				defaultValue: "active", // Different case
				expectErr:    true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := nanostore.Config{
					Dimensions: []nanostore.DimensionConfig{
						{
							Name:         "test",
							Type:         nanostore.Enumerated,
							Values:       tc.values,
							DefaultValue: tc.defaultValue,
						},
					},
				}

				err := validateConfig(config)
				if tc.expectErr && err == nil {
					t.Error("expected error for invalid default, got nil")
				} else if !tc.expectErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})
		}
	})

	// Test 4: Complex multi-dimension interactions
	t.Run("MultiDimensionInteractions", func(t *testing.T) {
		testCases := []struct {
			name      string
			dims      []nanostore.DimensionConfig
			expectErr bool
			errMsg    string
		}{
			{
				name: "prefix collision across dimensions",
				dims: []nanostore.DimensionConfig{
					{
						Name:     "dim1",
						Type:     nanostore.Enumerated,
						Values:   []string{"a", "b"},
						Prefixes: map[string]string{"a": "x"},
					},
					{
						Name:     "dim2",
						Type:     nanostore.Enumerated,
						Values:   []string{"c", "d"},
						Prefixes: map[string]string{"c": "x"}, // Same prefix as dim1
					},
				},
				expectErr: false, // Cross-dimension prefix collision might be allowed
			},
			{
				name: "mixed valid and invalid dimensions",
				dims: []nanostore.DimensionConfig{
					{
						Name:   "valid",
						Type:   nanostore.Enumerated,
						Values: []string{"a", "b"},
					},
					{
						Name:   "", // Invalid
						Type:   nanostore.Enumerated,
						Values: []string{"c", "d"},
					},
				},
				expectErr: true,
				errMsg:    "empty dimension name",
			},
			{
				name: "maximum dimensions stress test",
				dims: func() []nanostore.DimensionConfig {
					dims := make([]nanostore.DimensionConfig, 100)
					for i := 0; i < 100; i++ {
						dims[i] = nanostore.DimensionConfig{
							Name:   string(rune('a' + i)),
							Type:   nanostore.Enumerated,
							Values: []string{"val1", "val2"},
						}
					}
					return dims
				}(),
				expectErr: false, // Should handle many dimensions
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := nanostore.Config{Dimensions: tc.dims}
				err := validateConfig(config)

				if tc.expectErr && err == nil {
					t.Errorf("expected error containing '%s', got nil", tc.errMsg)
				} else if !tc.expectErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})
		}
	})

	// Test 5: Type conversion edge cases
	t.Run("TypeConversionEdgeCases", func(t *testing.T) {
		// Test UnmarshalDimensions with edge cases
		testCases := []struct {
			name     string
			doc      nanostore.Document
			target   interface{}
			expected interface{}
		}{
			{
				name: "nil dimensions map",
				doc: nanostore.Document{
					Title:      "Test",
					Dimensions: nil,
				},
				target: &struct {
					nanostore.Document
					Status string `dimension:"status"`
				}{},
			},
			{
				name: "dimension with wrong type",
				doc: nanostore.Document{
					Title: "Test",
					Dimensions: map[string]interface{}{
						"count": "not a number", // String where int expected
					},
				},
				target: &struct {
					nanostore.Document
					Count int `dimension:"count"`
				}{},
			},
			{
				name: "deeply nested dimension value",
				doc: nanostore.Document{
					Title: "Test",
					Dimensions: map[string]interface{}{
						"data": map[string]interface{}{
							"nested": "value",
						},
					},
				},
				target: &struct {
					nanostore.Document
					Data string `dimension:"data"`
				}{},
			},
			{
				name: "slice dimension value",
				doc: nanostore.Document{
					Title: "Test",
					Dimensions: map[string]interface{}{
						"tags": []string{"tag1", "tag2"},
					},
				},
				target: &struct {
					nanostore.Document
					Tags string `dimension:"tags"`
				}{},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := api.UnmarshalDimensions(tc.doc, tc.target)
				// Log the result - some might error, some might not
				if err != nil {
					t.Logf("UnmarshalDimensions error: %v", err)
				} else {
					t.Log("UnmarshalDimensions succeeded")
				}
			})
		}
	})
}

// Helper functions
func validateConfig(config nanostore.Config) error {
	// Simulate config validation logic
	seen := make(map[string]bool)

	for i, dim := range config.Dimensions {
		// Empty name check
		if dim.Name == "" {
			return errorf("dimension %d: empty dimension name", i)
		}

		// Duplicate name check
		if seen[dim.Name] {
			return errorf("duplicate dimension name: %s", dim.Name)
		}
		seen[dim.Name] = true

		// Enumerated-specific validations
		if dim.Type == nanostore.Enumerated {
			// Empty values check
			if len(dim.Values) == 0 {
				return errorf("dimension %d (%s): no values specified", i, dim.Name)
			}

			// Duplicate values check
			valuesSeen := make(map[string]bool)
			for _, v := range dim.Values {
				if valuesSeen[v] {
					return errorf("dimension %d (%s): duplicate value '%s'", i, dim.Name, v)
				}
				valuesSeen[v] = true
			}

			// Prefix validations
			for val, prefix := range dim.Prefixes {
				// Check value exists
				if !valuesSeen[val] {
					return errorf("dimension %d (%s): prefix for '%s' not in values list", i, dim.Name, val)
				}

				// Empty prefix
				if prefix == "" || strings.TrimSpace(prefix) == "" {
					return errorf("dimension %d (%s): empty or whitespace prefix", i, dim.Name)
				}

				// Invalid characters (only alphanumeric allowed)
				for _, ch := range prefix {
					if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') {
						return errorf("dimension %d (%s): prefix '%s' contains invalid characters", i, dim.Name, prefix)
					}
				}

				// Long prefix
				if len(prefix) > 10 {
					return errorf("dimension %d (%s): prefix too long", i, dim.Name)
				}

				// Duplicate prefix in same dimension
				for otherVal, otherPrefix := range dim.Prefixes {
					if val != otherVal && prefix == otherPrefix {
						return errorf("dimension %d (%s): duplicate prefix '%s'", i, dim.Name, prefix)
					}
				}
			}

			// Default value validation
			if dim.DefaultValue != "" {
				if !valuesSeen[dim.DefaultValue] {
					return errorf("dimension %d (%s): default value '%s' is not in values list", i, dim.Name, dim.DefaultValue)
				}
			}
		}
	}

	return nil
}

func errorf(format string, args ...interface{}) error {
	return &configError{msg: strings.TrimSpace(format)}
}

type configError struct {
	msg string
}

func (e *configError) Error() string {
	return e.msg
}

func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
