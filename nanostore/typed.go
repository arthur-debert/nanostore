package nanostore

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// MarshalDimensions converts a struct with dimension tags into a dimensions map
func MarshalDimensions(v interface{}) (map[string]interface{}, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", val.Kind())
	}

	typ := val.Type()
	dimensions := make(map[string]interface{})

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !fieldVal.CanInterface() {
			continue
		}

		// Skip embedded Document field
		if field.Anonymous && field.Type == reflect.TypeOf(Document{}) {
			continue
		}

		// Check for dimension tag
		dimTag := field.Tag.Get("dimension")
		if dimTag == "" {
			// Also check for values tag (declarative API style)
			if field.Tag.Get("values") != "" {
				dimTag = strings.ToLower(field.Name)
			} else {
				continue
			}
		}

		// Handle dimension:"name,options" format
		parts := strings.Split(dimTag, ",")
		dimName := parts[0]

		// Skip if dimension name is "-"
		if dimName == "-" {
			continue
		}

		// Get the value
		value := fieldVal.Interface()

		// Skip zero values (except false for bools)
		if isZeroValue(fieldVal) && fieldVal.Kind() != reflect.Bool {
			continue
		}

		// Validate that the value is a simple type
		if err := validateSimpleType(value, dimName); err != nil {
			return nil, err
		}

		// For values tag style, use lowercase field name as dimension name
		if field.Tag.Get("values") != "" && dimTag == strings.ToLower(field.Name) {
			dimName = strings.ToLower(field.Name)
		}

		dimensions[dimName] = value
	}

	return dimensions, nil
}

// UnmarshalDimensions populates a struct from a Document, mapping dimensions to tagged fields
func UnmarshalDimensions(doc Document, v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("expected pointer to struct, got %s", val.Kind())
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected pointer to struct, got pointer to %s", val.Kind())
	}

	typ := val.Type()

	// First, populate the embedded Document fields if present
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous && field.Type == reflect.TypeOf(Document{}) {
			val.Field(i).Set(reflect.ValueOf(doc))
			break
		}
	}

	// Then populate dimension fields
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !fieldVal.CanSet() {
			continue
		}

		// Skip embedded Document field
		if field.Anonymous && field.Type == reflect.TypeOf(Document{}) {
			continue
		}

		// Check for dimension tag
		dimTag := field.Tag.Get("dimension")
		var dimName string
		var defaultValue string

		if dimTag == "" {
			// Check for values tag (declarative API style)
			if field.Tag.Get("values") != "" {
				dimName = strings.ToLower(field.Name)
				defaultValue = field.Tag.Get("default")
			} else {
				continue
			}
		} else {
			// Parse dimension tag
			parts := strings.Split(dimTag, ",")
			dimName = parts[0]

			// Skip if dimension name is "-"
			if dimName == "-" {
				continue
			}

			// Look for default option
			for _, part := range parts[1:] {
				if strings.HasPrefix(part, "default=") {
					defaultValue = strings.TrimPrefix(part, "default=")
					break
				}
			}
		}

		// Get value from dimensions map
		dimValue, exists := doc.Dimensions[dimName]
		if !exists && defaultValue != "" {
			// Use default value
			if err := setFieldValue(fieldVal, defaultValue); err != nil {
				return fmt.Errorf("failed to set default for field %s: %w", field.Name, err)
			}
			continue
		}

		if exists {
			// Set the field value
			if err := setFieldFromInterface(fieldVal, dimValue); err != nil {
				return fmt.Errorf("failed to set field %s: %w", field.Name, err)
			}
		}
	}

	return nil
}

// Helper functions

// isZeroValue checks if a reflect.Value is a zero value
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return false // Never skip bool values
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Interface, reflect.Ptr, reflect.Slice, reflect.Map:
		return v.IsNil()
	default:
		return false
	}
}

// setFieldValue sets a field value from a string
func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(u)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(f)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}

// setFieldFromInterface sets a field value from an interface{}
func setFieldFromInterface(field reflect.Value, value interface{}) error {
	if value == nil {
		return nil
	}

	// Try direct assignment first
	valReflect := reflect.ValueOf(value)
	if valReflect.Type().AssignableTo(field.Type()) {
		field.Set(valReflect)
		return nil
	}

	// Check if the value is a complex type that we can't convert
	switch valReflect.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct:
		// Don't silently convert complex types to strings
		return fmt.Errorf("cannot convert %T to %s", value, field.Type())
	}

	// Otherwise convert through string for simple types
	strVal := fmt.Sprintf("%v", value)
	return setFieldValue(field, strVal)
}

// validateSimpleType ensures a dimension value is a simple type (string, number, bool)
func validateSimpleType(value interface{}, dimensionName string) error {
	if value == nil {
		return nil
	}

	// Check the type using reflection
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return nil
	case reflect.Slice, reflect.Array:
		return fmt.Errorf("dimension '%s' cannot be an array/slice type, got %T", dimensionName, value)
	case reflect.Map:
		return fmt.Errorf("dimension '%s' cannot be a map type, got %T", dimensionName, value)
	case reflect.Struct:
		// Allow time.Time as it's commonly used
		if _, ok := value.(time.Time); ok {
			return nil
		}
		return fmt.Errorf("dimension '%s' cannot be a struct type, got %T", dimensionName, value)
	case reflect.Ptr, reflect.Interface:
		// Dereference and check the underlying type
		if v.IsNil() {
			return nil
		}
		return validateSimpleType(v.Elem().Interface(), dimensionName)
	default:
		return fmt.Errorf("dimension '%s' must be a simple type (string, number, or bool), got %T", dimensionName, value)
	}
}

