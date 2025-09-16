package nanostore

import (
	"fmt"
	"reflect"
	"strings"
)

// fieldMeta holds parsed metadata for a struct field
type fieldMeta struct {
	fieldName     string
	dimensionName string
	isRef         bool
	values        []string
	prefixes      map[string]string
	defaultValue  string
	skipDimension bool
}

// parseStructTags analyzes a struct type and extracts dimension configuration from tags
func parseStructTags(t reflect.Type) ([]fieldMeta, error) {
	var metas []fieldMeta

	// Ensure we have a struct
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct type, got %s", t.Kind())
	}

	// Process each field
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip embedded Document field
		if field.Anonymous && field.Type == reflect.TypeOf(Document{}) {
			continue
		}

		meta := fieldMeta{
			fieldName: field.Name,
			prefixes:  make(map[string]string),
		}

		// Parse dimension tag
		if dimTag := field.Tag.Get("dimension"); dimTag != "" {
			if dimTag == "-" {
				meta.skipDimension = true
				metas = append(metas, meta)
				continue
			}

			parts := strings.Split(dimTag, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "ref" {
					meta.isRef = true
				} else if part != "" {
					// Explicit dimension name
					meta.dimensionName = part
				}
			}
		}

		// If no explicit dimension name, convert field name to snake_case
		if meta.dimensionName == "" {
			meta.dimensionName = toSnakeCase(field.Name)
		}

		// Parse values tag for enumerated dimensions
		if valuesTag := field.Tag.Get("values"); valuesTag != "" {
			values := strings.Split(valuesTag, ",")
			for _, v := range values {
				v = strings.TrimSpace(v)
				if v != "" {
					meta.values = append(meta.values, v)
				}
			}
		}

		// Parse prefix tag
		if prefixTag := field.Tag.Get("prefix"); prefixTag != "" {
			prefixPairs := strings.Split(prefixTag, ",")
			for _, pair := range prefixPairs {
				parts := strings.Split(pair, "=")
				if len(parts) == 2 {
					value := strings.TrimSpace(parts[0])
					prefix := strings.TrimSpace(parts[1])
					meta.prefixes[value] = prefix
				}
			}
		}

		// Parse default tag
		if defaultTag := field.Tag.Get("default"); defaultTag != "" {
			meta.defaultValue = defaultTag
		}

		// Validate field type
		if field.Type.Kind() != reflect.String {
			// For now, only string dimensions are supported
			return nil, fmt.Errorf("field %s: only string dimensions are currently supported, got %s", field.Name, field.Type.Kind())
		}

		// Validate custom string types
		if field.Type.Kind() == reflect.String && field.Type.PkgPath() != "" {
			// Custom string type (e.g., type Status string)
			// PkgPath is empty for built-in types like string
			if len(meta.values) == 0 {
				return nil, fmt.Errorf("field %s: custom string type %s requires 'values' tag", field.Name, field.Type.Name())
			}
		}

		// Skip fields that are explicitly excluded
		if !meta.skipDimension {
			metas = append(metas, meta)
		}
	}

	return metas, nil
}

// toSnakeCase converts a CamelCase string to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	result.Grow(len(s) + 10)

	for i, r := range s {
		if i > 0 && isUpper(r) {
			// Check if previous rune is lowercase or next rune is lowercase
			prevIsLower := i > 0 && isLower(rune(s[i-1]))
			nextIsLower := i+1 < len(s) && isLower(rune(s[i+1]))

			if prevIsLower || nextIsLower {
				result.WriteRune('_')
			}
		}
		result.WriteRune(toLower(r))
	}

	return result.String()
}

// Helper functions for case conversion
func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func isLower(r rune) bool {
	return r >= 'a' && r <= 'z'
}

func toLower(r rune) rune {
	if isUpper(r) {
		return r + ('a' - 'A')
	}
	return r
}

// buildConfigFromMeta converts field metadata to a Config
func buildConfigFromMeta(metas []fieldMeta) (*Config, error) {
	config := &Config{
		Dimensions: []DimensionConfig{},
	}

	// Check for duplicate dimension names
	seen := make(map[string]bool)

	for _, meta := range metas {
		if meta.skipDimension {
			continue
		}

		if seen[meta.dimensionName] {
			return nil, fmt.Errorf("duplicate dimension name: %s", meta.dimensionName)
		}
		seen[meta.dimensionName] = true

		dimConfig := DimensionConfig{
			Name:         meta.dimensionName,
			DefaultValue: meta.defaultValue,
		}

		// Configure based on type
		if meta.isRef {
			dimConfig.Type = Hierarchical
			dimConfig.RefField = meta.dimensionName // Use dimension name as ref field
		} else if len(meta.values) > 0 {
			dimConfig.Type = Enumerated
			dimConfig.Values = meta.values
			dimConfig.Prefixes = meta.prefixes
		} else {
			// String dimension (for future filtering support)
			dimConfig.Type = Enumerated // For now, treat as enumerated without values
		}

		config.Dimensions = append(config.Dimensions, dimConfig)
	}

	return config, nil
}
