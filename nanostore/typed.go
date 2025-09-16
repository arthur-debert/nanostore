package nanostore

import (
	"fmt"
	"reflect"
	"strings"
)

// DocumentTyped provides typed access to dimensions through struct tags
type DocumentTyped interface {
	// MarshalDimensions converts struct fields to dimensions map
	MarshalDimensions() (map[string]interface{}, error)
	// UnmarshalDimensions populates struct fields from dimensions map
	UnmarshalDimensions(dimensions map[string]interface{}) error
}

// dimensionTag represents parsed dimension tag information
type dimensionTag struct {
	name         string
	defaultValue string
	isRef        bool // true for hierarchical reference fields
}

// parseDimensionTag parses a dimension tag like "status,default=pending" or "parent_id,ref"
func parseDimensionTag(tag string) (dimensionTag, error) {
	if tag == "" || tag == "-" {
		return dimensionTag{}, fmt.Errorf("empty dimension tag")
	}

	parts := strings.Split(tag, ",")
	dt := dimensionTag{name: parts[0]}

	for i := 1; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if strings.HasPrefix(part, "default=") {
			dt.defaultValue = strings.TrimPrefix(part, "default=")
		} else if part == "ref" {
			dt.isRef = true
		}
	}

	return dt, nil
}

// MarshalDimensions converts a struct with dimension tags to a dimensions map
func MarshalDimensions(v interface{}) (map[string]interface{}, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("MarshalDimensions: expected struct, got %s", rv.Kind())
	}

	rt := rv.Type()
	dimensions := make(map[string]interface{})

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fieldValue := rv.Field(i)

		// Skip embedded Document field
		if field.Type == reflect.TypeOf(Document{}) {
			continue
		}

		tagStr, ok := field.Tag.Lookup("dimension")
		if !ok || tagStr == "-" {
			continue
		}

		tag, err := parseDimensionTag(tagStr)
		if err != nil {
			return nil, fmt.Errorf("invalid dimension tag on field %s: %w", field.Name, err)
		}

		// Get the actual value
		value := fieldValue.Interface()

		// Skip zero values (empty strings, nil, false for bool, 0 for numbers)
		if isZeroValue(fieldValue) {
			// For bools, we want to include false values explicitly
			if fieldValue.Kind() != reflect.Bool {
				continue
			}
		}

		dimensions[tag.name] = value
	}

	return dimensions, nil
}

// UnmarshalDimensions populates a struct from document dimensions
func UnmarshalDimensions(doc Document, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("UnmarshalDimensions: expected non-nil pointer to struct")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("UnmarshalDimensions: expected pointer to struct")
	}

	rt := rv.Type()

	// First, handle embedded Document field
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.Type == reflect.TypeOf(Document{}) {
			rv.Field(i).Set(reflect.ValueOf(doc))
			break
		}
	}

	// Then handle dimension fields
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fieldValue := rv.Field(i)

		if field.Type == reflect.TypeOf(Document{}) {
			continue
		}

		tagStr, ok := field.Tag.Lookup("dimension")
		if !ok || tagStr == "-" {
			continue
		}

		tag, err := parseDimensionTag(tagStr)
		if err != nil {
			return fmt.Errorf("invalid dimension tag on field %s: %w", field.Name, err)
		}

		// Get dimension value
		dimValue, exists := doc.Dimensions[tag.name]
		if !exists {
			// Use default value if specified
			if tag.defaultValue != "" {
				if err := setFieldValue(fieldValue, tag.defaultValue, field.Type); err != nil {
					return fmt.Errorf("failed to set default value for field %s: %w", field.Name, err)
				}
			}
			continue
		}

		// Set the field value
		if err := setFieldValue(fieldValue, dimValue, field.Type); err != nil {
			return fmt.Errorf("failed to set value for field %s: %w", field.Name, err)
		}
	}

	return nil
}

// setFieldValue sets a reflect.Value from an interface{} value with type conversion
func setFieldValue(field reflect.Value, value interface{}, fieldType reflect.Type) error {
	if !field.CanSet() {
		return fmt.Errorf("field cannot be set")
	}

	// Handle string conversions (most common case)
	if fieldType.Kind() == reflect.String {
		switch v := value.(type) {
		case string:
			field.SetString(v)
			return nil
		default:
			// Try to convert to string
			field.SetString(fmt.Sprintf("%v", v))
			return nil
		}
	}

	// Handle direct assignment for matching types
	valueType := reflect.TypeOf(value)
	if valueType.AssignableTo(fieldType) {
		field.Set(reflect.ValueOf(value))
		return nil
	}

	// Handle numeric conversions
	switch fieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case int64:
			field.SetInt(v)
		case float64:
			field.SetInt(int64(v))
		case string:
			// Try to parse string as int
			var i int64
			_, err := fmt.Sscanf(v, "%d", &i)
			if err != nil {
				return fmt.Errorf("cannot convert %q to int", v)
			}
			field.SetInt(i)
		default:
			return fmt.Errorf("cannot convert %T to %s", value, fieldType)
		}
		return nil

	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float64:
			field.SetFloat(v)
		case int64:
			field.SetFloat(float64(v))
		case string:
			// Try to parse string as float
			var f float64
			_, err := fmt.Sscanf(v, "%f", &f)
			if err != nil {
				return fmt.Errorf("cannot convert %q to float", v)
			}
			field.SetFloat(f)
		default:
			return fmt.Errorf("cannot convert %T to %s", value, fieldType)
		}
		return nil

	case reflect.Bool:
		switch v := value.(type) {
		case bool:
			field.SetBool(v)
		case string:
			// Handle common string representations
			switch strings.ToLower(v) {
			case "true", "yes", "1", "on":
				field.SetBool(true)
			case "false", "no", "0", "off":
				field.SetBool(false)
			default:
				return fmt.Errorf("cannot convert %q to bool", v)
			}
		default:
			return fmt.Errorf("cannot convert %T to bool", value)
		}
		return nil
	}

	return fmt.Errorf("unsupported field type: %s", fieldType)
}

// isZeroValue checks if a reflect.Value is the zero value for its type
func isZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map:
		return v.IsNil()
	}
	return false
}

// Typed store operations - these are standalone functions that work with the Store interface

// AddTyped creates a new document from a typed struct
func AddTyped(s Store, title string, v interface{}) (string, error) {
	dimensions, err := MarshalDimensions(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal dimensions: %w", err)
	}
	return s.Add(title, dimensions)
}

// UpdateTyped updates a document using a typed struct
func UpdateTyped(s Store, id string, v interface{}) error {
	dimensions, err := MarshalDimensions(v)
	if err != nil {
		return fmt.Errorf("failed to marshal dimensions: %w", err)
	}

	// Extract title and body if they exist in the struct
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	var title, body *string
	if titleField := rv.FieldByName("Title"); titleField.IsValid() && titleField.Kind() == reflect.String {
		t := titleField.String()
		if t != "" {
			title = &t
		}
	}
	if bodyField := rv.FieldByName("Body"); bodyField.IsValid() && bodyField.Kind() == reflect.String {
		b := bodyField.String()
		if b != "" {
			body = &b
		}
	}

	return s.Update(id, UpdateRequest{
		Title:      title,
		Body:       body,
		Dimensions: dimensions,
	})
}

// ListTyped returns typed documents
func ListTyped[T any](s Store, opts ListOptions) ([]T, error) {
	docs, err := s.List(opts)
	if err != nil {
		return nil, err
	}

	results := make([]T, len(docs))
	for i, doc := range docs {
		var item T
		if err := UnmarshalDimensions(doc, &item); err != nil {
			return nil, fmt.Errorf("failed to unmarshal document %d: %w", i, err)
		}
		results[i] = item
	}

	return results, nil
}
