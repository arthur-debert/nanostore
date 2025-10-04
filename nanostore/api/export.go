package api

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/types"
)

// JSONDimensionConfig represents a dimension in the JSON schema
type JSONDimensionConfig struct {
	Type      string            `json:"type"`                // "enumerated", "hierarchical", or "simple"
	FieldType string            `json:"field_type"`          // "string", "bool", "int", etc.
	Values    []string          `json:"values,omitempty"`    // For enumerated dimensions
	Default   interface{}       `json:"default,omitempty"`   // Default value
	Prefixes  map[string]string `json:"prefixes,omitempty"`  // Value to prefix mappings
	RefField  string            `json:"ref_field,omitempty"` // For hierarchical dimensions
	Nullable  bool              `json:"nullable,omitempty"`  // Whether field can be nil
}

// JSONDataFieldConfig represents a data field in the JSON schema
type JSONDataFieldConfig struct {
	FieldType string `json:"field_type"` // "string", "bool", "int", etc.
	Nullable  bool   `json:"nullable"`   // Whether field can be nil
}

// JSONStoreConfig represents the complete store configuration in JSON
type JSONStoreConfig struct {
	StoreName  string                         `json:"store_name"`
	Version    string                         `json:"version"`
	Dimensions map[string]JSONDimensionConfig `json:"dimensions"`
	DataFields map[string]JSONDataFieldConfig `json:"data_fields"`
}

// ExportConfigFromType extracts configuration from a Go struct type and converts it to JSON
func ExportConfigFromType[T any]() ([]byte, error) {
	var zero T
	typ := reflect.TypeOf(zero)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Ensure T embeds Document (same validation as New)
	if !embedsDocument(typ) {
		return nil, fmt.Errorf("type %s must embed nanostore.Document", typ.Name())
	}

	// Generate the internal config using existing logic
	internalConfig, err := generateConfigFromType(typ)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config from type: %w", err)
	}

	// Convert to JSON schema format
	jsonConfig, err := convertToJSONSchema(typ, internalConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to JSON schema: %w", err)
	}

	// Marshal to JSON with pretty printing
	return json.MarshalIndent(jsonConfig, "", "  ")
}

// convertToJSONSchema converts internal nanostore.Config to JSONStoreConfig
func convertToJSONSchema(typ reflect.Type, config nanostore.Config) (JSONStoreConfig, error) {
	jsonConfig := JSONStoreConfig{
		StoreName:  strings.ToLower(typ.Name()),
		Version:    "1.0",
		Dimensions: make(map[string]JSONDimensionConfig),
		DataFields: make(map[string]JSONDataFieldConfig),
	}

	// Track which fields are dimensions to identify data fields
	dimensionFields := make(map[string]bool)

	// Convert dimensions
	for _, dim := range config.Dimensions {
		jsonDim := JSONDimensionConfig{
			Type:      dim.Type.String(),
			FieldType: "string", // Will be updated based on actual field type
			Values:    dim.Values,
			Prefixes:  dim.Prefixes,
			RefField:  dim.RefField,
		}

		// Set default value
		if dim.DefaultValue != "" {
			jsonDim.Default = dim.DefaultValue
		}

		// Find the original field to get type information
		fieldName, fieldType, nullable := findFieldForDimension(typ, dim)
		if fieldName != "" {
			jsonDim.FieldType = fieldType
			jsonDim.Nullable = nullable
			dimensionFields[fieldName] = true
		}

		jsonConfig.Dimensions[dim.Name] = jsonDim
	}

	// Add data fields (non-dimension fields)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip embedded Document field
		if field.Anonymous && field.Type == reflect.TypeOf(nanostore.Document{}) {
			continue
		}

		// Skip dimension fields
		fieldNameLower := strings.ToLower(field.Name)
		if dimensionFields[fieldNameLower] || dimensionFields[field.Name] {
			continue
		}

		// Skip fields with dimension tags (they're already processed)
		if hasAnyDimensionTag(field) {
			continue
		}

		// This is a data field
		fieldType, nullable := getFieldTypeString(field.Type)
		jsonConfig.DataFields[fieldNameLower] = JSONDataFieldConfig{
			FieldType: fieldType,
			Nullable:  nullable,
		}
	}

	return jsonConfig, nil
}

// findFieldForDimension finds the original struct field that corresponds to a dimension
func findFieldForDimension(typ reflect.Type, dim types.DimensionConfig) (string, string, bool) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip embedded Document field
		if field.Anonymous && field.Type == reflect.TypeOf(nanostore.Document{}) {
			continue
		}

		// Check if this field matches the dimension
		fieldNameLower := strings.ToLower(field.Name)

		// For enumerated dimensions, check if field name matches dimension name
		if dim.Type == types.Enumerated && fieldNameLower == dim.Name {
			fieldType, nullable := getFieldTypeString(field.Type)
			return field.Name, fieldType, nullable
		}

		// For hierarchical dimensions, check dimension tag
		if dim.Type == types.Hierarchical {
			if dimTag := field.Tag.Get("dimension"); dimTag != "" {
				parts := strings.Split(dimTag, ",")
				if len(parts) > 0 && parts[0] == dim.RefField {
					fieldType, nullable := getFieldTypeString(field.Type)
					return field.Name, fieldType, nullable
				}
			}
		}

		// Also check for simple dimensions (fields with default but no values)
		if dim.Type == types.Enumerated && len(dim.Values) == 0 {
			// This might be a simple dimension field (like bool with default)
			if _, hasDefault := field.Tag.Lookup("default"); hasDefault && fieldNameLower == dim.Name {
				fieldType, nullable := getFieldTypeString(field.Type)
				return field.Name, fieldType, nullable
			}
		}
	}

	return "", "string", false
}

// hasAnyDimensionTag checks if a field has any dimension-related tags
func hasAnyDimensionTag(field reflect.StructField) bool {
	_, hasValues := field.Tag.Lookup("values")
	_, hasDimension := field.Tag.Lookup("dimension")
	return hasValues || hasDimension
}

// getFieldTypeString converts Go reflect.Type to JSON schema type string
func getFieldTypeString(t reflect.Type) (string, bool) {
	nullable := false

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		nullable = true
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return "string", nullable
	case reflect.Bool:
		return "bool", nullable
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int", nullable
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "uint", nullable
	case reflect.Float32, reflect.Float64:
		return "float", nullable
	case reflect.Struct:
		// Handle time.Time specifically
		if t == reflect.TypeOf(time.Time{}) {
			return "time.Time", nullable
		}
		return "struct", nullable
	case reflect.Slice:
		elemType, _ := getFieldTypeString(t.Elem())
		return "[]" + elemType, nullable
	case reflect.Map:
		keyType, _ := getFieldTypeString(t.Key())
		valueType, _ := getFieldTypeString(t.Elem())
		return "map[" + keyType + "]" + valueType, nullable
	default:
		return t.String(), nullable
	}
}
