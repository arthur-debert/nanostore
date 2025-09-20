package api

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)

import (
	"reflect"
	"testing"
)

func TestIsZeroValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		// Bool values (never zero according to function)
		{"bool true", true, false},
		{"bool false", false, false},

		// Integer types
		{"int zero", int(0), true},
		{"int non-zero", int(42), false},
		{"int64 zero", int64(0), true},
		{"int64 non-zero", int64(-100), false},

		// Unsigned types
		{"uint zero", uint(0), true},
		{"uint non-zero", uint(10), false},
		{"uint64 zero", uint64(0), true},
		{"uint64 non-zero", uint64(100), false},

		// Float types
		{"float32 zero", float32(0), true},
		{"float32 non-zero", float32(3.14), false},
		{"float64 zero", float64(0), true},
		{"float64 non-zero", float64(-2.5), false},

		// String
		{"string empty", "", true},
		{"string non-empty", "hello", false},

		// Pointers
		{"nil pointer", (*int)(nil), true},
		{"non-nil pointer", new(int), false},

		// Slices
		{"nil slice", ([]int)(nil), true},
		{"empty slice", []int{}, false}, // Empty but not nil
		{"non-empty slice", []int{1, 2}, false},

		// Maps
		{"nil map", (map[string]int)(nil), true},
		{"empty map", map[string]int{}, false}, // Empty but not nil
		{"non-empty map", map[string]int{"a": 1}, false},

		// Note: Can't test nil interface directly as reflect.ValueOf(nil) creates invalid Value
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)
			result := isZeroValue(v)
			if result != tt.expected {
				t.Errorf("isZeroValue(%v) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}

	// Special test for nil interface
	t.Run("nil interface special", func(t *testing.T) {
		var i interface{}
		v := reflect.ValueOf(&i).Elem()
		if !isZeroValue(v) {
			t.Error("expected nil interface to be zero value")
		}

		// Non-nil interface
		i = 42
		v = reflect.ValueOf(&i).Elem()
		if isZeroValue(v) {
			t.Error("expected non-nil interface to not be zero value")
		}
	})
}

func TestSetFieldValue(t *testing.T) {
	tests := []struct {
		name      string
		fieldType reflect.Type
		input     string
		expected  interface{}
		wantErr   bool
	}{
		// String
		{"string", reflect.TypeOf(""), "hello", "hello", false},
		{"empty string", reflect.TypeOf(""), "", "", false},

		// Bool
		{"bool true", reflect.TypeOf(true), "true", true, false},
		{"bool false", reflect.TypeOf(false), "false", false, false},
		{"bool 1", reflect.TypeOf(true), "1", true, false},
		{"bool 0", reflect.TypeOf(false), "0", false, false},
		{"bool invalid", reflect.TypeOf(false), "invalid", false, true},

		// Integers
		{"int positive", reflect.TypeOf(int(0)), "42", int(42), false},
		{"int negative", reflect.TypeOf(int(0)), "-42", int(-42), false},
		{"int invalid", reflect.TypeOf(int(0)), "abc", int(0), true},
		{"int64", reflect.TypeOf(int64(0)), "9223372036854775807", int64(9223372036854775807), false},

		// Unsigned
		{"uint", reflect.TypeOf(uint(0)), "42", uint(42), false},
		{"uint invalid negative", reflect.TypeOf(uint(0)), "-1", uint(0), true},
		{"uint64", reflect.TypeOf(uint64(0)), "18446744073709551615", uint64(18446744073709551615), false},

		// Floats
		{"float32", reflect.TypeOf(float32(0)), "3.14", float32(3.14), false},
		{"float64", reflect.TypeOf(float64(0)), "-2.5e10", float64(-2.5e10), false},
		{"float invalid", reflect.TypeOf(float64(0)), "not a number", float64(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a pointer to the type so we can set it
			ptr := reflect.New(tt.fieldType)
			field := ptr.Elem()

			err := setFieldValue(field, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("setFieldValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && field.Interface() != tt.expected {
				t.Errorf("setFieldValue() set value = %v, expected %v", field.Interface(), tt.expected)
			}
		})
	}

	// Test unsupported types
	t.Run("unsupported type", func(t *testing.T) {
		type customType struct{}
		ptr := reflect.New(reflect.TypeOf(customType{}))
		err := setFieldValue(ptr.Elem(), "value")
		if err == nil {
			t.Error("expected error for unsupported type, got nil")
		}
	})
}

func TestSetFieldFromInterface(t *testing.T) {
	tests := []struct {
		name      string
		fieldType reflect.Type
		input     interface{}
		expected  interface{}
		wantErr   bool
	}{
		// Nil value
		{"nil value", reflect.TypeOf(""), nil, "", false},

		// Same type assignments
		{"string to string", reflect.TypeOf(""), "hello", "hello", false},
		{"int to int", reflect.TypeOf(int(0)), 42, 42, false},
		{"bool to bool", reflect.TypeOf(false), true, true, false},

		// String to other types
		{"string to int", reflect.TypeOf(int(0)), "123", 123, false},
		{"string to bool", reflect.TypeOf(false), "true", true, false},
		{"string to float", reflect.TypeOf(float64(0)), "3.14", 3.14, false},

		// Cross-type numeric conversions
		{"int to float64", reflect.TypeOf(float64(0)), 42, float64(42), false},
		{"float64 to int", reflect.TypeOf(int(0)), 3.14, 0, true}, // Will fail because "3.14" can't parse as int

		// Invalid conversions
		{"invalid string to int", reflect.TypeOf(int(0)), "abc", 0, true},
		{"struct to string", reflect.TypeOf(""), struct{}{}, "", true}, // Now properly rejects struct types
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a pointer to the type so we can set it
			ptr := reflect.New(tt.fieldType)
			field := ptr.Elem()

			err := setFieldFromInterface(field, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("setFieldFromInterface() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && field.Interface() != tt.expected {
				t.Errorf("setFieldFromInterface() set value = %v, expected %v", field.Interface(), tt.expected)
			}
		})
	}
}
