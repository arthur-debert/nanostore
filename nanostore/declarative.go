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
		// Note: Only string types are supported because:
		// 1. Dimension values are used in ID generation (e.g., status="done" -> "d1", "d2")
		// 2. Prefixes only make sense for string values
		// 3. The underlying SQL schema uses TEXT columns for dimensions
		// 4. Smart ID resolution expects string-based hierarchies
		if field.Type.Kind() != reflect.String {
			return nil, fmt.Errorf("field %s: dimensions must be string types (found %s). This is required for ID generation and prefixing", field.Name, field.Type.Kind())
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
	hierarchicalCount := 0

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
			hierarchicalCount++

			// Validate hierarchical dimension
			if meta.defaultValue != "" {
				return nil, fmt.Errorf("hierarchical dimension %s cannot have a default value", meta.fieldName)
			}
			if len(meta.values) > 0 {
				return nil, fmt.Errorf("hierarchical dimension %s cannot have enumerated values", meta.fieldName)
			}
			if len(meta.prefixes) > 0 {
				return nil, fmt.Errorf("hierarchical dimension %s cannot have prefixes", meta.fieldName)
			}
		} else if len(meta.values) > 0 {
			dimConfig.Type = Enumerated
			dimConfig.Values = meta.values
			dimConfig.Prefixes = meta.prefixes

			// Validate enumerated dimension
			if meta.defaultValue != "" && !sliceContains(meta.values, meta.defaultValue) {
				return nil, fmt.Errorf("field %s: default value %q is not in the list of valid values", meta.fieldName, meta.defaultValue)
			}

			// Validate prefixes
			seenPrefixes := make(map[string]string) // prefix -> value that uses it
			for value, prefix := range meta.prefixes {
				if !sliceContains(meta.values, value) {
					return nil, fmt.Errorf("field %s: prefix defined for invalid value %q", meta.fieldName, value)
				}
				if prefix == "" {
					return nil, fmt.Errorf("field %s: empty prefix for value %q", meta.fieldName, value)
				}
				if existingValue, exists := seenPrefixes[prefix]; exists {
					return nil, fmt.Errorf("field %s: duplicate prefix %q used by both %q and %q", meta.fieldName, prefix, existingValue, value)
				}
				seenPrefixes[prefix] = value
			}
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

	// Validate maximum one hierarchical dimension
	if hierarchicalCount > 1 {
		return nil, fmt.Errorf("only one hierarchical dimension is supported, found %d", hierarchicalCount)
	}

	return config, nil
}

// sliceContains checks if a slice contains a string
func sliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
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
func NewFromType[T any](filePath string) (*TypedStore[T], error) {
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
	store, err := New(filePath, *config)
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
	// This is a best-effort operation - the document was already created successfully
	_ = ts.populateDocument(item, uuid)
	// We ignore any error here because:
	// 1. The document was already created successfully in the database
	// 2. With proper NewFromType validation, this should never fail
	// 3. Returning an error after successful creation would be confusing

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
// Supports both UUIDs and user-facing IDs (e.g., "1", "p3", "1.2.h4")
// This method is optimized: UUID filters generate simple WHERE clauses
func (ts *TypedStore[T]) Get(id string) (*T, error) {
	// First resolve the ID to UUID if necessary
	uuid, err := ts.store.ResolveUUID(id)
	if err != nil {
		// If resolution fails, assume it's already a UUID
		uuid = id
	}

	// Use filtered List to get only the document with matching UUID
	// Note: This is efficient - the query builder generates a simple WHERE uuid = ? clause
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

// Query returns a new TypedQuery for building queries
func (ts *TypedStore[T]) Query() *TypedQuery[T] {
	return &TypedQuery[T]{
		store:     ts,
		filters:   make(map[string]interface{}),
		orderBy:   []string{},
		limit:     0,
		offset:    0,
		methodMap: ts.generateQueryMethods(),
	}
}

// generateQueryMethods creates dimension-specific query methods
func (ts *TypedStore[T]) generateQueryMethods() map[string]func(*TypedQuery[T], ...interface{}) *TypedQuery[T] {
	methods := make(map[string]func(*TypedQuery[T], ...interface{}) *TypedQuery[T])

	// Generate methods for each dimension
	for _, dim := range ts.config.Dimensions {
		// Capture dimension name in closure
		dimName := dim.Name
		dimType := dim.Type

		// Basic filter method (e.g., Status(value))
		func(name string) {
			methods[name] = func(q *TypedQuery[T], args ...interface{}) *TypedQuery[T] {
				if len(args) > 0 {
					q.filters[name] = args[0]
				}
				return q
			}
		}(dimName)

		// Not filter method (e.g., StatusNot(value))
		func(name string) {
			methods[name+"Not"] = func(q *TypedQuery[T], args ...interface{}) *TypedQuery[T] {
				if len(args) > 0 {
					// Store as a special filter that the query builder can recognize
					q.filters[name+"__not"] = args[0]
				}
				return q
			}
		}(dimName)

		// In filter method (e.g., StatusIn(values...))
		func(name string) {
			methods[name+"In"] = func(q *TypedQuery[T], args ...interface{}) *TypedQuery[T] {
				if len(args) > 0 {
					// Convert to string slice for the store
					values := make([]string, len(args))
					for i, arg := range args {
						values[i] = fmt.Sprintf("%v", arg)
					}
					q.filters[name] = values
				}
				return q
			}
		}(dimName)

		// For hierarchical dimensions, add exists/not exists methods
		if dimType == Hierarchical {
			func(name string) {
				methods[name+"Exists"] = func(q *TypedQuery[T], args ...interface{}) *TypedQuery[T] {
					q.filters[name+"__exists"] = true
					return q
				}
			}(dimName)

			func(name string) {
				methods[name+"NotExists"] = func(q *TypedQuery[T], args ...interface{}) *TypedQuery[T] {
					q.filters[name+"__not_exists"] = true
					return q
				}
			}(dimName)
		}
	}

	return methods
}

// TypedQuery provides a fluent interface for querying documents
type TypedQuery[T any] struct {
	store     *TypedStore[T]
	filters   map[string]interface{}
	orderBy   []string
	limit     int
	offset    int
	methodMap map[string]func(*TypedQuery[T], ...interface{}) *TypedQuery[T]
}

// Call invokes a dynamic query method by name
func (q *TypedQuery[T]) Call(method string, args ...interface{}) *TypedQuery[T] {
	if fn, exists := q.methodMap[method]; exists {
		return fn(q, args...)
	}
	// If method doesn't exist, return q unchanged (could also panic or return error)
	return q
}

// Status filters by status dimension (convenience method)
func (q *TypedQuery[T]) Status(value string) *TypedQuery[T] {
	return q.Call("status", value)
}

// StatusNot filters by status not equal to value
func (q *TypedQuery[T]) StatusNot(value string) *TypedQuery[T] {
	return q.Call("statusNot", value)
}

// StatusIn filters by status in list of values
func (q *TypedQuery[T]) StatusIn(values ...string) *TypedQuery[T] {
	args := make([]interface{}, len(values))
	for i, v := range values {
		args[i] = v
	}
	return q.Call("statusIn", args...)
}

// Priority filters by priority dimension (convenience method)
func (q *TypedQuery[T]) Priority(value string) *TypedQuery[T] {
	return q.Call("priority", value)
}

// PriorityIn filters by priority in list of values
func (q *TypedQuery[T]) PriorityIn(values ...string) *TypedQuery[T] {
	args := make([]interface{}, len(values))
	for i, v := range values {
		args[i] = v
	}
	return q.Call("priorityIn", args...)
}

// ParentID filters by parent_id dimension
func (q *TypedQuery[T]) ParentID(value string) *TypedQuery[T] {
	return q.Call("parent_id", value)
}

// ParentIDExists filters for documents with a parent
func (q *TypedQuery[T]) ParentIDExists() *TypedQuery[T] {
	return q.Call("parent_idExists")
}

// ParentIDNotExists filters for documents without a parent
func (q *TypedQuery[T]) ParentIDNotExists() *TypedQuery[T] {
	return q.Call("parent_idNotExists")
}

// WithFilter adds a dimension filter to the query
func (q *TypedQuery[T]) WithFilter(dimension string, value interface{}) *TypedQuery[T] {
	q.filters[dimension] = value
	return q
}

// OrderBy sets the order for results
func (q *TypedQuery[T]) OrderBy(fields ...string) *TypedQuery[T] {
	q.orderBy = fields
	return q
}

// Limit sets the maximum number of results
func (q *TypedQuery[T]) Limit(n int) *TypedQuery[T] {
	q.limit = n
	return q
}

// Offset sets the number of results to skip
func (q *TypedQuery[T]) Offset(n int) *TypedQuery[T] {
	q.offset = n
	return q
}

// Find executes the query and returns all matching documents
func (q *TypedQuery[T]) Find() ([]T, error) {
	// Pass all filters to the database - no more client-side filtering needed
	processedFilters := make(map[string]interface{})
	for key, value := range q.filters {
		processedFilters[key] = value
	}

	// Build OrderBy clauses
	orderClauses := make([]OrderClause, len(q.orderBy))
	for i, field := range q.orderBy {
		descending := false
		if strings.HasPrefix(field, "-") {
			descending = true
			field = field[1:]
		}
		orderClauses[i] = OrderClause{
			Column:     field,
			Descending: descending,
		}
	}

	// Build ListOptions
	opts := ListOptions{
		Filters: processedFilters,
		OrderBy: orderClauses,
	}
	if q.limit > 0 {
		opts.Limit = &q.limit
	}
	if q.offset > 0 {
		opts.Offset = &q.offset
	}

	// Execute the query
	docs, err := q.store.store.List(opts)
	if err != nil {
		return nil, err
	}

	// Convert documents to typed results (no client-side filtering needed)
	results := make([]T, 0, len(docs))
	for _, doc := range docs {
		item := new(T)

		// Set Document embedded field
		itemValue := reflect.ValueOf(item).Elem()
		for i := 0; i < itemValue.NumField(); i++ {
			field := itemValue.Field(i)
			if field.Type() == reflect.TypeOf(Document{}) {
				field.Set(reflect.ValueOf(doc))
				break
			}
		}

		// Populate dimension fields
		if err := q.store.populateDimensions(item, doc.Dimensions); err != nil {
			return nil, fmt.Errorf("failed to populate dimensions: %w", err)
		}

		results = append(results, *item)
	}

	return results, nil
}

// First returns the first matching document or an error if none found
func (q *TypedQuery[T]) First() (*T, error) {
	results, err := q.Limit(1).Find()
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no documents found")
	}

	return &results[0], nil
}

// Get returns exactly one document or an error
func (q *TypedQuery[T]) Get() (*T, error) {
	results, err := q.Find()
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no documents found")
	}

	if len(results) > 1 {
		return nil, fmt.Errorf("expected exactly one document, found %d", len(results))
	}

	return &results[0], nil
}

// Count returns the number of matching documents
func (q *TypedQuery[T]) Count() (int, error) {
	// Use Find to get filtered results
	results, err := q.Find()
	if err != nil {
		return 0, err
	}

	return len(results), nil
}

// Exists returns true if at least one matching document exists
func (q *TypedQuery[T]) Exists() (bool, error) {
	count, err := q.Count()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
