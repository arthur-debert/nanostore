package api

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/arthur-debert/nanostore/nanostore"
)

// TypedStore wraps a Store with type-safe operations for a specific document type T
type TypedStore[T any] struct {
	store  nanostore.Store
	config nanostore.Config
	typ    reflect.Type
}

// NewFromType creates a new TypedStore for the given type T, automatically generating
// the configuration from struct tags
func NewFromType[T any](filePath string) (*TypedStore[T], error) {
	var zero T
	typ := reflect.TypeOf(zero)

	// Ensure T embeds Document
	if !embedsDocument(typ) {
		return nil, fmt.Errorf("type %s must embed nanostore.Document", typ.Name())
	}

	// Generate config from struct tags
	config, err := generateConfigFromType(typ)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config: %w", err)
	}

	// Create underlying store
	store, err := nanostore.New(filePath, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	return &TypedStore[T]{
		store:  store,
		config: config,
		typ:    typ,
	}, nil
}

// Create adds a new document with the given title and typed data
func (ts *TypedStore[T]) Create(title string, data *T) (string, error) {
	dimensions, extraData, err := MarshalDimensions(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal dimensions: %w", err)
	}

	// Store extra data in dimensions with a special prefix
	for key, value := range extraData {
		dimensions["_data."+key] = value
	}

	return ts.store.Add(title, dimensions)
}

// Get retrieves a document by ID and unmarshals it into the typed structure
func (ts *TypedStore[T]) Get(id string) (*T, error) {
	// First try to resolve if it's a simple ID
	uuid := id
	if resolvedUUID, err := ts.store.ResolveUUID(id); err == nil {
		uuid = resolvedUUID
	}

	// List with UUID filter to get the document
	docs, err := ts.store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"uuid": uuid},
	})
	if err != nil {
		return nil, err
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("document not found: %s", id)
	}

	var result T
	if err := UnmarshalDimensions(docs[0], &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	return &result, nil
}

// Update modifies an existing document with typed data
func (ts *TypedStore[T]) Update(id string, data *T) error {
	dimensions, extraData, err := MarshalDimensions(data)
	if err != nil {
		return fmt.Errorf("failed to marshal dimensions: %w", err)
	}

	// Store extra data in dimensions with a special prefix
	for key, value := range extraData {
		dimensions["_data."+key] = value
	}

	// Extract title and body if they're set
	var req nanostore.UpdateRequest
	req.Dimensions = dimensions

	// Use reflection to check if Title or Body fields are set
	val := reflect.ValueOf(data).Elem()
	if docField := val.FieldByName("Document"); docField.IsValid() {
		doc := docField.Interface().(nanostore.Document)
		if doc.Title != "" {
			req.Title = &doc.Title
		}
		if doc.Body != "" {
			req.Body = &doc.Body
		}
	}

	return ts.store.Update(id, req)
}

// Delete removes a document and optionally its children
func (ts *TypedStore[T]) Delete(id string, cascade bool) error {
	return ts.store.Delete(id, cascade)
}

// Query returns a new typed query builder
func (ts *TypedStore[T]) Query() *TypedQuery[T] {
	return &TypedQuery[T]{
		store: ts.store,
		options: nanostore.ListOptions{
			Filters: make(map[string]interface{}),
		},
	}
}

// Close closes the underlying store
func (ts *TypedStore[T]) Close() error {
	return ts.store.Close()
}

// TypedQuery provides a fluent interface for building type-safe queries
type TypedQuery[T any] struct {
	store   nanostore.Store
	options nanostore.ListOptions
}

// Activity filters by activity value
func (tq *TypedQuery[T]) Activity(value string) *TypedQuery[T] {
	tq.options.Filters["activity"] = value
	return tq
}

// Status filters by status value
func (tq *TypedQuery[T]) Status(value string) *TypedQuery[T] {
	tq.options.Filters["status"] = value
	return tq
}

// StatusIn filters by multiple status values
func (tq *TypedQuery[T]) StatusIn(values ...string) *TypedQuery[T] {
	tq.options.Filters["status"] = values
	return tq
}

// StatusNot excludes a specific status
func (tq *TypedQuery[T]) StatusNot(value string) *TypedQuery[T] {
	// Get all possible status values from config
	// For now, we'll implement a simple exclusion by listing other values
	// In a real implementation, you'd want NOT support in the query layer
	allStatuses := []string{"pending", "active", "done"}
	var includeStatuses []string
	for _, s := range allStatuses {
		if s != value {
			includeStatuses = append(includeStatuses, s)
		}
	}
	if len(includeStatuses) > 0 {
		tq.options.Filters["status"] = includeStatuses
	}
	return tq
}

// Priority filters by priority value
func (tq *TypedQuery[T]) Priority(value string) *TypedQuery[T] {
	tq.options.Filters["priority"] = value
	return tq
}

// ParentID filters by parent ID
func (tq *TypedQuery[T]) ParentID(id string) *TypedQuery[T] {
	// Try to resolve user-facing ID first
	if uuid, err := tq.store.ResolveUUID(id); err == nil {
		tq.options.Filters["parent_id"] = uuid
	} else {
		tq.options.Filters["parent_id"] = id
	}
	return tq
}

// ParentIDNotExists filters for documents without a parent
func (tq *TypedQuery[T]) ParentIDNotExists() *TypedQuery[T] {
	// We need to filter in post-processing since the store doesn't
	// support "not exists" queries directly
	// For now, we'll get all and filter
	// In production, you'd add proper NOT EXISTS support to the store
	tq.options.Filters["__parent_not_exists__"] = true
	return tq
}

// ParentIDStartsWith filters for documents whose parent ID starts with a prefix
// Useful for finding all descendants of a node
func (tq *TypedQuery[T]) ParentIDStartsWith(prefix string) *TypedQuery[T] {
	// This would need custom support in the store layer
	// For now, we'll skip implementation
	return tq
}

// Search adds text search filtering
func (tq *TypedQuery[T]) Search(text string) *TypedQuery[T] {
	tq.options.FilterBySearch = text
	return tq
}

// OrderBy adds ordering
func (tq *TypedQuery[T]) OrderBy(column string) *TypedQuery[T] {
	tq.options.OrderBy = append(tq.options.OrderBy, nanostore.OrderClause{
		Column:     column,
		Descending: false,
	})
	return tq
}

// OrderByDesc adds descending ordering
func (tq *TypedQuery[T]) OrderByDesc(column string) *TypedQuery[T] {
	tq.options.OrderBy = append(tq.options.OrderBy, nanostore.OrderClause{
		Column:     column,
		Descending: true,
	})
	return tq
}

// Limit sets the maximum number of results
func (tq *TypedQuery[T]) Limit(n int) *TypedQuery[T] {
	tq.options.Limit = &n
	return tq
}

// Offset sets the number of results to skip
func (tq *TypedQuery[T]) Offset(n int) *TypedQuery[T] {
	tq.options.Offset = &n
	return tq
}

// Find executes the query and returns typed results
func (tq *TypedQuery[T]) Find() ([]T, error) {
	// Check for special filters
	parentNotExists := false
	if _, ok := tq.options.Filters["__parent_not_exists__"]; ok {
		parentNotExists = true
		delete(tq.options.Filters, "__parent_not_exists__")
	}

	docs, err := tq.store.List(tq.options)
	if err != nil {
		return nil, err
	}

	results := make([]T, 0, len(docs))
	for _, doc := range docs {
		// Apply post-processing filters
		if parentNotExists {
			// Check if parent_id exists in dimensions
			if _, hasParent := doc.Dimensions["parent_id"]; hasParent {
				continue // Skip documents with parent
			}
		}

		var typed T
		if err := UnmarshalDimensions(doc, &typed); err != nil {
			return nil, fmt.Errorf("failed to unmarshal document: %w", err)
		}
		results = append(results, typed)
	}

	return results, nil
}

// First returns the first matching document
func (tq *TypedQuery[T]) First() (*T, error) {
	limit := 1
	tq.options.Limit = &limit

	results, err := tq.Find()
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no documents found")
	}

	return &results[0], nil
}

// Count returns the number of matching documents
func (tq *TypedQuery[T]) Count() (int, error) {
	// Use Find to get filtered results including post-processing
	results, err := tq.Find()
	if err != nil {
		return 0, err
	}

	return len(results), nil
}

// Exists returns true if any matching documents exist
func (tq *TypedQuery[T]) Exists() (bool, error) {
	// Set limit to 1 for efficiency
	limit := 1
	tq.options.Limit = &limit

	results, err := tq.Find()
	if err != nil {
		return false, err
	}

	return len(results) > 0, nil
}

// Helper functions

// embedsDocument checks if a type embeds nanostore.Document
func embedsDocument(typ reflect.Type) bool {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous && field.Type == reflect.TypeOf(nanostore.Document{}) {
			return true
		}
	}

	return false
}

// generateConfigFromType creates a Config from struct tags
func generateConfigFromType(typ reflect.Type) (nanostore.Config, error) {
	var config nanostore.Config

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip embedded Document field
		if field.Anonymous && field.Type == reflect.TypeOf(nanostore.Document{}) {
			continue
		}

		// Check for pointer fields (not allowed)
		if field.Type.Kind() == reflect.Ptr {
			return config, fmt.Errorf("field %s: pointer fields are not supported", field.Name)
		}

		// Look for field tags in different formats
		if tagValue := field.Tag.Get("values"); tagValue != "" {
			// Parse enumerated dimension from tags like:
			// `values:"pending,active,done" prefix:"done=d" default:"pending"`
			dimConfig := nanostore.DimensionConfig{
				Name: strings.ToLower(field.Name),
				Type: nanostore.Enumerated,
			}

			// Parse values
			dimConfig.Values = strings.Split(tagValue, ",")
			for i := range dimConfig.Values {
				dimConfig.Values[i] = strings.TrimSpace(dimConfig.Values[i])
			}

			// Parse default
			if defaultVal := field.Tag.Get("default"); defaultVal != "" {
				dimConfig.DefaultValue = defaultVal
			}

			// Parse prefixes
			if prefixTag := field.Tag.Get("prefix"); prefixTag != "" {
				dimConfig.Prefixes = make(map[string]string)
				// Parse formats like "done=d" or "done=d,active=a"
				for _, p := range strings.Split(prefixTag, ",") {
					parts := strings.Split(strings.TrimSpace(p), "=")
					if len(parts) == 2 {
						dimConfig.Prefixes[parts[0]] = parts[1]
					}
				}
			}

			config.Dimensions = append(config.Dimensions, dimConfig)
		} else if dimTag := field.Tag.Get("dimension"); dimTag != "" {
			// Parse dimension tag like: `dimension:"parent_id,ref"`
			parts := strings.Split(dimTag, ",")
			dimName := parts[0]

			// Check if it's a reference field
			isRef := false
			for _, part := range parts[1:] {
				if part == "ref" {
					isRef = true
					break
				}
			}

			if isRef {
				// Hierarchical dimension
				config.Dimensions = append(config.Dimensions, nanostore.DimensionConfig{
					Name:     field.Name + "_hierarchy",
					Type:     nanostore.Hierarchical,
					RefField: dimName,
				})
			}
		}
	}

	return config, nil
}
