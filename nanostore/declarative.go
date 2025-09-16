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

// helper functions for case conversion
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
	hasDimensions := false

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
		hasDimensions = true
	}

	// Validate that at least one dimension is defined
	if !hasDimensions {
		return nil, fmt.Errorf("at least one dimension must be defined")
	}

	return config, nil
}

// TypedStore provides type-safe operations for a specific struct type
type TypedStore[T any] struct {
	store        Store
	config       *Config
	structType   reflect.Type
	fieldIndices map[string]int // maps dimension name to struct field index
}

// NewFromType creates a new TypedStore from a struct type definition
// The struct's tags define the store configuration
func NewFromType[T any](dbPath string) (*TypedStore[T], error) {
	var zero T
	structType := reflect.TypeOf(zero)

	// Validate that T embeds Document
	if !hasEmbeddedDocument(structType) {
		return nil, fmt.Errorf("type %s must embed nanostore.Document", structType.Name())
	}

	// Parse struct tags to get field metadata
	metas, err := parseStructTags(structType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse struct tags: %w", err)
	}

	// Build config from metadata
	config, err := buildConfigFromMeta(metas)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	// Create the underlying store
	store, err := New(dbPath, *config)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Build field index map for efficient access
	fieldIndices := make(map[string]int)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.IsExported() && !field.Anonymous {
			// Find corresponding meta
			for _, meta := range metas {
				if meta.fieldName == field.Name && !meta.skipDimension {
					fieldIndices[meta.dimensionName] = i
					break
				}
			}
		}
	}

	return &TypedStore[T]{
		store:        store,
		config:       config,
		structType:   structType,
		fieldIndices: fieldIndices,
	}, nil
}

// hasEmbeddedDocument checks if a type embeds Document
func hasEmbeddedDocument(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}

	docType := reflect.TypeOf(Document{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous && field.Type == docType {
			return true
		}
	}
	return false
}

// Close closes the underlying store
func (ts *TypedStore[T]) Close() error {
	return ts.store.Close()
}

// Create adds a new document to the store
func (ts *TypedStore[T]) Create(title string, item *T) (string, error) {
	// Extract dimensions from the struct
	dimensions, err := ts.extractDimensions(item)
	if err != nil {
		return "", fmt.Errorf("failed to extract dimensions: %w", err)
	}

	// Create the document
	uuid, err := ts.store.Add(title, dimensions)
	if err != nil {
		return "", err
	}

	// Update the item with the created document info
	if err := ts.populateDocument(item, uuid); err != nil {
		return "", fmt.Errorf("failed to populate document: %w", err)
	}

	return uuid, nil
}

// Update modifies an existing document
func (ts *TypedStore[T]) Update(id string, item *T) error {
	// Extract dimensions from the struct
	dimensions, err := ts.extractDimensions(item)
	if err != nil {
		return fmt.Errorf("failed to extract dimensions: %w", err)
	}

	// Get the document value for title and body
	docValue := reflect.ValueOf(item).Elem()
	var title, body *string

	// Find Document embedded field
	for i := 0; i < docValue.NumField(); i++ {
		field := docValue.Field(i)
		if field.Type() == reflect.TypeOf(Document{}) {
			doc := field.Interface().(Document)
			title = &doc.Title
			body = &doc.Body
			break
		}
	}

	// Create update request
	req := UpdateRequest{
		Title:      title,
		Body:       body,
		Dimensions: dimensions,
	}

	return ts.store.Update(id, req)
}

// Delete removes a document from the store
func (ts *TypedStore[T]) Delete(id string, cascade bool) error {
	return ts.store.Delete(id, cascade)
}

// Get retrieves a document by ID
func (ts *TypedStore[T]) Get(id string) (*T, error) {
	// First resolve the ID to UUID if necessary
	uuid, err := ts.store.ResolveUUID(id)
	if err != nil {
		// If resolution fails, assume it's already a UUID
		uuid = id
	}

	// Use filtered List to get only the document with matching UUID
	docs, err := ts.store.List(ListOptions{
		Filters: map[string]interface{}{
			"uuid": uuid,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("document not found")
	}

	doc := &docs[0]

	// Create new instance of T
	item := new(T)

	// Set Document embedded field
	itemValue := reflect.ValueOf(item).Elem()
	for i := 0; i < itemValue.NumField(); i++ {
		field := itemValue.Field(i)
		if field.Type() == reflect.TypeOf(Document{}) {
			field.Set(reflect.ValueOf(*doc))
			break
		}
	}

	// Populate dimension fields
	if err := ts.populateDimensions(item, doc.Dimensions); err != nil {
		return nil, fmt.Errorf("failed to populate dimensions: %w", err)
	}

	return item, nil
}

// extractDimensions extracts dimension values from struct fields
func (ts *TypedStore[T]) extractDimensions(item *T) (map[string]interface{}, error) {
	dimensions := make(map[string]interface{})
	itemValue := reflect.ValueOf(item).Elem()

	if ts.structType.Kind() == reflect.Ptr {
		ts.structType = ts.structType.Elem()
	}

	for dimName, fieldIdx := range ts.fieldIndices {
		field := itemValue.Field(fieldIdx)
		value := field.String()
		if value != "" {
			dimensions[dimName] = value
		}
	}

	return dimensions, nil
}

// populateDimensions sets struct fields from dimension values
func (ts *TypedStore[T]) populateDimensions(item *T, dimensions map[string]interface{}) error {
	itemValue := reflect.ValueOf(item).Elem()

	for dimName, fieldIdx := range ts.fieldIndices {
		if value, exists := dimensions[dimName]; exists {
			field := itemValue.Field(fieldIdx)
			if strValue, ok := value.(string); ok {
				field.SetString(strValue)
			}
		}
	}

	return nil
}

// populateDocument updates the document UUID in the struct
func (ts *TypedStore[T]) populateDocument(item *T, uuid string) error {
	itemValue := reflect.ValueOf(item).Elem()

	// Find and update Document embedded field
	for i := 0; i < itemValue.NumField(); i++ {
		field := itemValue.Field(i)
		if field.Type() == reflect.TypeOf(Document{}) {
			if field.CanSet() {
				doc := field.Interface().(Document)
				doc.UUID = uuid
				field.Set(reflect.ValueOf(doc))
				return nil
			}
		}
	}

	return fmt.Errorf("Document field not found or not settable")
}
