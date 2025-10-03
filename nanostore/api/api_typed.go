package api

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
)

// MarshalDimensions converts a struct with dimension tags into dimensions and data maps
func MarshalDimensions(v interface{}) (dimensions map[string]interface{}, data map[string]interface{}, err error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("expected struct, got %s", val.Kind())
	}

	typ := val.Type()
	dimensions = make(map[string]interface{})
	data = make(map[string]interface{})

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !fieldVal.CanInterface() {
			continue
		}

		// Skip embedded Document field
		if field.Anonymous && field.Type == reflect.TypeOf(nanostore.Document{}) {
			continue
		}

		// Get the value, handling pointer types
		var value interface{}
		if fieldVal.Kind() == reflect.Ptr {
			if fieldVal.IsNil() {
				// nil pointer - store as nil (will be handled appropriately later)
				value = nil
			} else {
				// non-nil pointer - get the pointed-to value
				value = fieldVal.Elem().Interface()
			}
		} else {
			// non-pointer value
			value = fieldVal.Interface()
		}

		// Convert time.Time values to RFC3339 string format for storage
		if value != nil {
			if t, ok := value.(time.Time); ok {
				value = t.Format(time.RFC3339)
			}
		}

		// Check for dimension tag
		dimTag := field.Tag.Get("dimension")
		isDimension := false

		if dimTag != "" {
			isDimension = true
		} else if field.Tag.Get("values") != "" {
			// Also check for values tag (declarative API style)
			dimTag = strings.ToLower(field.Name)
			isDimension = true
		}

		if isDimension {
			// Handle dimension:"name,options" format
			parts := strings.Split(dimTag, ",")
			dimName := parts[0]

			// Skip if dimension name is "-"
			if dimName == "-" {
				continue
			}

			// Handle default values for enumerated dimensions with zero values
			if isZeroValue(fieldVal) && fieldVal.Kind() != reflect.Bool {
				valuesTag := field.Tag.Get("values")
				defaultTag := field.Tag.Get("default")

				// If this is an enumerated dimension with a default value, use the default
				if valuesTag != "" && defaultTag != "" {
					value = defaultTag
				} else if valuesTag != "" {
					// Enumerated dimension without default - validate the empty value (will be rejected)
					// Don't skip, let validation handle it
				} else {
					// Skip non-enumerated zero values
					continue
				}
			}

			// Validate that the dimension value is a simple type
			if err = ValidateSimpleType(value, dimName); err != nil {
				return nil, nil, err
			}

			// For values tag style, use lowercase field name as dimension name
			if field.Tag.Get("values") != "" && dimTag == strings.ToLower(field.Name) {
				dimName = strings.ToLower(field.Name)
			}

			// Validate enumerated dimension values against their allowed values
			if valuesTag := field.Tag.Get("values"); valuesTag != "" {
				if err = validateEnumeratedValue(value, valuesTag, field.Name); err != nil {
					return nil, nil, err
				}
			}

			dimensions[dimName] = value
		} else {
			// Non-dimension field - store in data map
			// Skip zero values to avoid storing empty data
			if !isZeroValue(fieldVal) {
				// Store all non-dimension fields in data map
				data[field.Name] = value
			}
		}
	}

	return dimensions, data, nil
}

// MarshalDimensionsForUpdate converts a struct with dimension tags into dimensions and data maps
// for update operations. Unlike MarshalDimensions, this version preserves zero values to allow
// field clearing in update operations.
//
// Key differences from MarshalDimensions:
// - Zero values in data fields (non-dimension fields) are preserved and will clear existing values
// - Zero values in enumerated dimension fields (with "values" tag) are skipped to avoid validation errors
// - Zero values in non-enumerated dimension fields (like refs) are preserved
//
// This enables the ability to clear fields in bulk update operations like UpdateByUUIDs,
// UpdateByDimension, and UpdateWhere, which was previously impossible due to zero value skipping.
//
// BREAKING CHANGE: This changes the behavior of all Update methods. Previously, zero values
// were ignored. Now zero values will clear the corresponding fields in the document.
//
// Example:
//
//	// This will clear the Assignee field and set Priority to "low"
//	updates := &Task{
//		Assignee: "",    // Zero value - WILL clear the field
//		Priority: "low", // Non-zero value - will update the field
//		// Note: Other data fields like Description will also be cleared if not set
//	}
//	store.UpdateByUUIDs(uuids, updates)
func MarshalDimensionsForUpdate(v interface{}) (dimensions map[string]interface{}, data map[string]interface{}, err error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("expected struct, got %s", val.Kind())
	}

	typ := val.Type()
	dimensions = make(map[string]interface{})
	data = make(map[string]interface{})

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !fieldVal.CanInterface() {
			continue
		}

		// Skip embedded Document field
		if field.Anonymous && field.Type == reflect.TypeOf(nanostore.Document{}) {
			continue
		}

		// Get the value, handling pointer types
		var value interface{}
		if fieldVal.Kind() == reflect.Ptr {
			if fieldVal.IsNil() {
				// nil pointer - store as nil (will be handled appropriately later)
				value = nil
			} else {
				// non-nil pointer - get the pointed-to value
				value = fieldVal.Elem().Interface()
			}
		} else {
			// non-pointer value
			value = fieldVal.Interface()
		}

		// Convert time.Time values to RFC3339 string format for storage
		if value != nil {
			if t, ok := value.(time.Time); ok {
				value = t.Format(time.RFC3339)
			}
		}

		// Check for dimension tag
		dimTag := field.Tag.Get("dimension")
		isDimension := false

		if dimTag != "" {
			isDimension = true
		} else if field.Tag.Get("values") != "" {
			// Also check for values tag (declarative API style)
			dimTag = strings.ToLower(field.Name)
			isDimension = true
		}

		if isDimension {
			// Handle dimension:"name,options" format
			parts := strings.Split(dimTag, ",")
			dimName := parts[0]

			// Skip if dimension name is "-"
			if dimName == "-" {
				continue
			}

			// For dimension fields, we need to be careful about zero values
			// Skip zero values for enumerated dimensions to avoid validation errors
			// But allow zero values for non-enumerated dimensions (like refs)
			if isZeroValue(fieldVal) && field.Tag.Get("values") != "" {
				// Skip zero values for enumerated dimensions (they would fail validation)
				continue
			}

			// Validate that the dimension value is a simple type
			if err = ValidateSimpleType(value, dimName); err != nil {
				return nil, nil, err
			}

			// For values tag style, use lowercase field name as dimension name
			if field.Tag.Get("values") != "" && dimTag == strings.ToLower(field.Name) {
				dimName = strings.ToLower(field.Name)
			}

			// Validate enumerated dimension values against their allowed values
			if valuesTag := field.Tag.Get("values"); valuesTag != "" {
				if err = validateEnumeratedValue(value, valuesTag, field.Name); err != nil {
					return nil, nil, err
				}
			}

			dimensions[dimName] = value
		} else {
			// Non-dimension field - store in data map
			// For updates, preserve zero values to allow field clearing
			data[field.Name] = value
		}
	}

	return dimensions, data, nil
}

// UnmarshalDimensions populates a struct from a Document, mapping dimensions to tagged fields
func UnmarshalDimensions(doc nanostore.Document, v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("expected pointer to struct, got %s", val.Kind())
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected pointer to struct, got pointer to %s", val.Kind())
	}

	typ := val.Type()

	// Extract _data prefixed values to a separate map
	dataMap := make(map[string]interface{})
	for key, value := range doc.Dimensions {
		if strings.HasPrefix(key, "_data.") {
			fieldName := strings.TrimPrefix(key, "_data.")
			dataMap[fieldName] = value
		}
	}

	// First, populate the embedded Document fields if present
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous && field.Type == reflect.TypeOf(nanostore.Document{}) {
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
		if field.Anonymous && field.Type == reflect.TypeOf(nanostore.Document{}) {
			continue
		}

		// Check for dimension tag
		dimTag := field.Tag.Get("dimension")
		var dimName string
		var defaultValue string
		isDimension := false

		if dimTag != "" {
			isDimension = true
		} else if field.Tag.Get("values") != "" {
			// Check for values tag (declarative API style)
			dimName = strings.ToLower(field.Name)
			defaultValue = field.Tag.Get("default")
			isDimension = true
		}

		if isDimension && dimTag != "" {
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

		if isDimension {
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
		} else {
			// Non-dimension field - check our extracted data map
			if dataValue, exists := dataMap[field.Name]; exists {
				if err := setFieldFromInterface(fieldVal, dataValue); err != nil {
					return fmt.Errorf("failed to set data field %s: %w", field.Name, err)
				}
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
	case reflect.Struct:
		// Handle time.Time specially
		if field.Type() == reflect.TypeOf(time.Time{}) {
			t, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return fmt.Errorf("failed to parse time value '%s': %w", value, err)
			}
			field.Set(reflect.ValueOf(t))
			return nil
		}
		return fmt.Errorf("unsupported struct type: %s", field.Type())
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}

// setFieldFromInterface sets a field value from an interface{}
func setFieldFromInterface(field reflect.Value, value interface{}) error {
	if value == nil {
		// For pointer types, set to nil. For non-pointer types, leave as zero value.
		if field.Kind() == reflect.Ptr {
			field.Set(reflect.Zero(field.Type()))
		}
		return nil
	}

	// Handle pointer types
	if field.Kind() == reflect.Ptr {
		// Create a new instance of the pointed-to type
		elemType := field.Type().Elem()
		newPtr := reflect.New(elemType)

		// Set the value on the pointed-to element
		if err := setFieldFromInterface(newPtr.Elem(), value); err != nil {
			return err
		}

		// Set the pointer
		field.Set(newPtr)
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

// extractDocumentFields extracts Document fields from an embedded nanostore.Document
func extractDocumentFields(v interface{}) (title string, body string, found bool) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return "", "", false
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Look for embedded Document field
		if field.Anonymous && field.Type == reflect.TypeOf(nanostore.Document{}) {
			// Extract the Document value
			doc := fieldVal.Interface().(nanostore.Document)
			return doc.Title, doc.Body, true
		}
	}

	return "", "", false
}

// getValidDataFields extracts the names of non-dimension fields (data fields) from a struct type
// These are fields that don't have dimension tags and would be stored with "_data." prefix
func getValidDataFields(v interface{}) ([]string, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", val.Kind())
	}

	typ := val.Type()
	var dataFields []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip embedded Document field
		if field.Anonymous && field.Type == reflect.TypeOf(nanostore.Document{}) {
			continue
		}

		// Check if this is a dimension field
		dimTag := field.Tag.Get("dimension")
		isEnumeratedDimension := field.Tag.Get("values") != ""

		// If it's not a dimension field, it's a data field
		if dimTag == "" && !isEnumeratedDimension {
			dataFields = append(dataFields, field.Name)
		}
	}

	return dataFields, nil
}

// validateDataFieldName checks if a field name is valid for data queries.
//
// This function eliminates silent failures in data field queries by validating field names
// against the actual Go struct fields. Key features:
//
// - **Case-insensitive validation**: Accepts field names in any case (e.g., "assignee", "Assignee", "ASSIGNEE")
// - **Auto-correction**: Returns the correct Go field name case for storage consistency
// - **Helpful error messages**: Provides suggestions for typos and lists valid field names
// - **No silent failures**: Invalid field names always return clear error messages
//
// Example:
//   - Input: "assignee" → Output: "Assignee" (if struct has field named "Assignee")
//   - Input: "assigne" → Error: "invalid data field name 'assigne', did you mean: [Assignee]?"
//
// Returns the correct case field name if valid, or an error with suggestions if invalid
func validateDataFieldName(fieldName string, validFields []string) (string, error) {
	// Convert field names to lowercase for case-insensitive comparison
	fieldNameLower := strings.ToLower(fieldName)

	for _, validField := range validFields {
		if strings.ToLower(validField) == fieldNameLower {
			// Field exists - return the actual Go field name (correct case) for storage consistency
			return validField, nil
		}
	}

	// Field doesn't exist - provide helpful error with suggestions
	var suggestions []string
	for _, validField := range validFields {
		// Simple similarity check: if field name is contained in valid field or vice versa
		if strings.Contains(strings.ToLower(validField), fieldNameLower) ||
			strings.Contains(fieldNameLower, strings.ToLower(validField)) {
			suggestions = append(suggestions, validField)
		}
	}

	errMsg := fmt.Sprintf("invalid data field name '%s'", fieldName)

	if len(suggestions) > 0 {
		errMsg += fmt.Sprintf(", did you mean one of: %v?", suggestions)
	} else if len(validFields) > 0 {
		errMsg += fmt.Sprintf(", valid data fields are: %v", validFields)
	} else {
		errMsg += " (no data fields available for this type)"
	}

	return "", fmt.Errorf("%s", errMsg)
}

// ValidateSimpleType ensures a dimension value is a simple type (string, number, bool)
func ValidateSimpleType(value interface{}, dimensionName string) error {
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
		return fmt.Errorf("dimension '%s' cannot be a struct type, got %T (time.Time check failed)", dimensionName, value)
	case reflect.Ptr, reflect.Interface:
		// Dereference and check the underlying type
		if v.IsNil() {
			return nil
		}
		return ValidateSimpleType(v.Elem().Interface(), dimensionName)
	default:
		return fmt.Errorf("dimension '%s' must be a simple type (string, number, or bool), got %T", dimensionName, value)
	}
}

// validateEnumeratedValue validates that a field value is one of the allowed enumerated values
// specified in the "values" struct tag. This prevents invalid enumerated dimension values
// from being stored in the system.
//
// Parameters:
//   - value: The actual field value to validate
//   - valuesTag: The "values" tag content (e.g., "pending,active,done")
//   - fieldName: The struct field name for error reporting
//
// Returns an error if the value is not in the allowed values list.
func validateEnumeratedValue(value interface{}, valuesTag, fieldName string) error {
	// Convert value to string for comparison
	valueStr := fmt.Sprintf("%v", value)

	// Parse the allowed values from the tag
	allowedValues := strings.Split(valuesTag, ",")

	// Trim whitespace from each allowed value
	for i, v := range allowedValues {
		allowedValues[i] = strings.TrimSpace(v)
	}

	// Check if the value is in the allowed list
	for _, allowed := range allowedValues {
		if valueStr == allowed {
			return nil // Value is valid
		}
	}

	// Value is not in the allowed list - return standardized error message
	return fmt.Errorf("invalid value '%s' for field '%s': must be one of [%s]",
		valueStr, fieldName, strings.Join(allowedValues, ", "))
}
