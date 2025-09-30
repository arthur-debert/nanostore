package formats

import (
	"testing"
)

func TestRegister(t *testing.T) {
	// Save original registry
	originalRegistry := registry
	defer func() { registry = originalRegistry }()

	// Clear registry for testing
	registry = make(map[string]*DocumentFormat)

	tests := []struct {
		name      string
		format    *DocumentFormat
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid format",
			format: &DocumentFormat{
				Name:        "test-format",
				Extension:   ".test",
				Serialize:   func(t, c string, m map[string]interface{}) string { return t + c },
				Deserialize: func(d string) (string, string, map[string]interface{}, error) { return "", d, nil, nil },
			},
			wantError: false,
		},
		{
			name: "invalid name with uppercase",
			format: &DocumentFormat{
				Name:      "TestFormat",
				Extension: ".test",
			},
			wantError: true,
			errorMsg:  "invalid format name",
		},
		{
			name: "invalid name with special chars",
			format: &DocumentFormat{
				Name:      "test@format",
				Extension: ".test",
			},
			wantError: true,
			errorMsg:  "invalid format name",
		},
		{
			name: "empty name",
			format: &DocumentFormat{
				Name:      "",
				Extension: ".test",
			},
			wantError: true,
			errorMsg:  "invalid format name",
		},
		{
			name: "extension without dot",
			format: &DocumentFormat{
				Name:      "test-format-2",
				Extension: "test",
				Serialize: func(t, c string, m map[string]interface{}) string { return t + c },
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Register(tt.format)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// Check extension normalization
				if tt.format.Extension != "" && tt.format.Extension[0] != '.' {
					t.Errorf("extension not normalized: %q", tt.format.Extension)
				}
			}
		})
	}

	// Test duplicate registration
	t.Run("duplicate format", func(t *testing.T) {
		format := &DocumentFormat{
			Name:      "duplicate",
			Extension: ".dup",
		}

		err := Register(format)
		if err != nil {
			t.Fatalf("first registration failed: %v", err)
		}

		err = Register(format)
		if err == nil {
			t.Error("expected error for duplicate registration")
		} else if !contains(err.Error(), "already registered") {
			t.Errorf("expected 'already registered' error, got %q", err.Error())
		}
	})
}

func TestGet(t *testing.T) {
	// Save original registry
	originalRegistry := registry
	defer func() { registry = originalRegistry }()

	// Set up test registry
	registry = make(map[string]*DocumentFormat)
	testFormat := &DocumentFormat{
		Name:      "test",
		Extension: ".test",
	}
	registry["test"] = testFormat

	tests := []struct {
		name       string
		formatName string
		wantError  bool
	}{
		{
			name:       "existing format",
			formatName: "test",
			wantError:  false,
		},
		{
			name:       "non-existent format",
			formatName: "nonexistent",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := Get(tt.formatName)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if format != testFormat {
					t.Error("returned wrong format")
				}
			}
		})
	}
}

func TestList(t *testing.T) {
	// Save original registry
	originalRegistry := registry
	defer func() { registry = originalRegistry }()

	// Set up test registry
	registry = make(map[string]*DocumentFormat)
	registry["format1"] = &DocumentFormat{Name: "format1"}
	registry["format2"] = &DocumentFormat{Name: "format2"}

	names := List()
	if len(names) != 2 {
		t.Errorf("expected 2 formats, got %d", len(names))
	}

	// Check that both formats are in the list
	hasFormat1, hasFormat2 := false, false
	for _, name := range names {
		if name == "format1" {
			hasFormat1 = true
		}
		if name == "format2" {
			hasFormat2 = true
		}
	}

	if !hasFormat1 || !hasFormat2 {
		t.Errorf("List() missing formats: got %v", names)
	}
}

func TestIsValidFormatName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"lowercase letters", "test", true},
		{"with numbers", "test123", true},
		{"with dashes", "test-format", true},
		{"with underscores", "test_format", true},
		{"all valid chars", "test-format_123", true},
		{"uppercase letters", "Test", false},
		{"special chars", "test@format", false},
		{"spaces", "test format", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidFormatName(tt.input)
			if got != tt.want {
				t.Errorf("isValidFormatName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && containsHelper(s[1:], substr)
}

func containsHelper(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s[:len(substr)] == substr {
		return true
	}
	return containsHelper(s[1:], substr)
}
