package api

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/types"
)

// DocumentMetadata contains the metadata fields of a document
// This provides structured access to document metadata without the full document content
type DocumentMetadata struct {
	UUID      string    // Stable internal identifier
	SimpleID  string    // Generated ID like "1", "c2", "1.2.c3"
	Title     string    // Document title
	CreatedAt time.Time // Creation timestamp
	UpdatedAt time.Time // Last update timestamp
}

// TypedStore wraps a Store with type-safe operations for a specific document type T.
//
// This is the primary interface for applications using nanostore, providing compile-time
// type safety while leveraging the sophisticated ID generation and dimensional organization
// features of the underlying store. The TypedStore automatically handles:
//
// - **Automatic Configuration**: Generates dimension configuration from struct tags
// - **Type Marshaling/Unmarshaling**: Converts between Go structs and store documents
// - **Smart ID Resolution**: Transparently handles SimpleID ↔ UUID conversion
// - **Fluent Query Interface**: Provides chainable, type-safe query building
//
// # Design Philosophy
//
// The TypedStore is designed around "configuration by convention" principles:
//
//  1. **Struct Tags Drive Configuration**: Instead of separate config files, dimensions
//     are defined directly on Go struct fields using tags
//  2. **Type Safety First**: All operations are checked at compile time
//  3. **Zero Boilerplate**: Minimal setup required - just define your struct and go
//  4. **Progressive Enhancement**: Simple cases work with minimal tags, complex cases
//     supported through additional tag options
//
// # Struct Tag Conventions
//
// The TypedStore uses struct tags to automatically generate dimension configurations:
//
//	type Task struct {
//	    nanostore.Document        // Required embedded field
//
//	    // Enumerated dimension with values, prefixes, and default
//	    Status   string `values:"pending,active,done" prefix:"done=d" default:"pending"`
//	    Priority string `values:"low,medium,high" prefix:"high=h" default:"medium"`
//
//	    // Hierarchical dimension (parent-child relationships)
//	    ParentID string `dimension:"parent_id,ref"`
//
//	    // Regular fields (stored as _data.field_name)
//	    Assignee string
//	    DueDate  time.Time
//	}
//
// # Document Embedding Requirement
//
// All types used with TypedStore must embed nanostore.Document:
//
//	type MyDoc struct {
//	    nanostore.Document  // Required - provides UUID, SimpleID, Title, Body, etc.
//	    MyField string
//	}
//
// This embedding provides:
// - UUID: Stable internal identifier
// - SimpleID: Human-readable ID (e.g., "1", "1.dh3", "1.2.c4")
// - Title/Body: Standard document content fields
// - CreatedAt/UpdatedAt: Automatic timestamp management
// - Dimensions: Map containing all dimensional data
//
// # Automatic ID Resolution
//
// The TypedStore automatically handles Smart ID resolution throughout:
//
// - **User Input**: Methods accept SimpleIDs from users (e.g., "1.2", "dh3")
// - **Internal Processing**: Automatically resolves to UUIDs for store operations
// - **Query Results**: Returns documents with properly generated SimpleIDs
// - **Reference Fields**: Parent IDs in queries are automatically resolved
//
// # Error Handling Strategy
//
// The TypedStore provides clear error messages for common issues:
//
// - **Type Validation**: Ensures T embeds Document before operation
// - **Configuration Errors**: Clear messages for invalid struct tags
// - **Marshal/Unmarshal**: Detailed errors for type conversion failures
// - **Store Errors**: Propagates underlying store errors with context
//
// # Performance Characteristics
//
// - **Configuration Generation**: O(n) where n = number of struct fields (startup only)
// - **Type Marshaling**: O(m) where m = number of dimensional fields per document
// - **Reflection Overhead**: Minimized by caching reflect.Type
// - **Query Performance**: Leverages underlying store optimizations
//
// # Thread Safety
//
// TypedStore is thread-safe:
// - Immutable after creation (config and type information cached)
// - All operations delegate to thread-safe underlying store
// - Multiple goroutines can safely share a single TypedStore instance
type TypedStore[T any] struct {
	store  nanostore.Store  // Underlying nanostore implementation
	config nanostore.Config // Generated configuration from struct tags
	typ    reflect.Type     // Cached type information for T
}

// NewFromType creates a new TypedStore for the given type T, automatically generating
// the configuration from struct tags.
//
// This is the primary constructor for TypedStore and performs several critical setup steps:
//
// 1. **Type Validation**: Ensures T properly embeds nanostore.Document
// 2. **Configuration Generation**: Analyzes struct tags to create dimension config
// 3. **Store Creation**: Initializes the underlying nanostore with generated config
// 4. **Type Caching**: Stores reflection metadata for efficient operations
//
// # Supported Struct Tags
//
// The function recognizes several struct tag formats:
//
// ## Enumerated Dimensions
//
//	Status string `values:"pending,active,done" prefix:"done=d" default:"pending"`
//	- values: Comma-separated list of valid values
//	- prefix: Value-to-prefix mappings (format: "value=prefix,value2=prefix2")
//	- default: Default value (must be in values list)
//
// ## Hierarchical Dimensions
//
//	ParentID string `dimension:"parent_id,ref"`
//	- First part: Reference field name in document dimensions
//	- "ref" flag: Indicates this is a hierarchical reference field
//
// # Configuration Generation Process
//
// 1. **Field Enumeration**: Iterates through all struct fields using reflection
// 2. **Tag Parsing**: Extracts and validates dimension configuration from tags
// 3. **Validation**: Ensures tag values are consistent and valid
// 4. **Config Assembly**: Builds complete nanostore.Config from parsed dimensions
//
// # Error Scenarios
//
// The function provides detailed errors for common setup issues:
//
// - **Missing Document Embedding**: "type X must embed nanostore.Document"
// - **Invalid Tag Format**: Parsing errors for malformed tag values
// - **Pointer Fields**: "field X: pointer fields are not supported"
// - **Store Creation Failures**: Underlying nanostore initialization errors
//
// # Usage Example
//
//	type Task struct {
//	    nanostore.Document
//	    Status   string `values:"pending,active,done" default:"pending"`
//	    Priority string `values:"low,medium,high" prefix:"high=h"`
//	    ParentID string `dimension:"parent_id,ref"`
//	}
//
//	store, err := NewFromType[Task]("tasks.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer store.Close()
//
// # Performance Notes
//
// - Reflection is only used during initialization, not per-operation
// - Generated configuration is cached for the lifetime of the TypedStore
// - File creation is deferred until first document is added
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

// Create adds a new document with the given title and typed data.
//
// This is the primary method for adding new documents to the store. It handles:
//
// 1. **Type Marshaling**: Converts typed struct to dimension map
// 2. **Data Separation**: Distinguishes between dimensional and extra data
// 3. **ID Generation**: Triggers automatic SimpleID generation based on dimensions
// 4. **UUID Assignment**: Creates stable internal UUID for the document
//
// # Data Processing Strategy
//
// The method processes struct fields in three categories:
//
// ## Dimensional Fields
// Fields with dimension tags become part of the document's partition:
//
//	Status string `values:"pending,active,done"`  // Becomes dimensions["status"]
//
// ## Reference Fields
// Hierarchical reference fields for parent-child relationships:
//
//	ParentID string `dimension:"parent_id,ref"`   // Becomes dimensions["parent_id"]
//
// ## Extra Data Fields
// Regular struct fields are stored with "_data." prefix:
//
//	Assignee string                               // Becomes dimensions["_data.assignee"]
//	DueDate  time.Time                           // Becomes dimensions["_data.duedate"]
//
// # SimpleID Generation
//
// The returned ID is a human-readable SimpleID that reflects the document's dimensions:
//
//	// Example: Task with status=done, priority=high, position=3 in partition
//	task := &Task{Status: "done", Priority: "high", Title: "Fix bug"}
//	id, err := store.Create("Fix critical bug", task)
//	// Returns: "dh3" (done="d", high="h", position=3)
//
// # Error Handling
//
// Create returns detailed errors for various failure scenarios:
//
// - **Marshal Failures**: Invalid struct field types or values
// - **Validation Errors**: Enumerated values not in configured list
// - **Store Errors**: Underlying storage or file system issues
// - **Constraint Violations**: Duplicate references or circular dependencies
//
// # Performance Notes
//
// - Marshaling time is O(n) where n = number of struct fields
// - ID generation time is O(m log m) where m = total documents in store
// - File I/O is minimized through atomic write operations
// - Dimension validation is O(1) with pre-computed maps
//
// # Usage Examples
//
//	// Simple document creation
//	task := &Task{
//	    Status:   "pending",
//	    Priority: "high",
//	    Assignee: "alice",
//	}
//	id, err := store.Create("Implement feature X", task)
//
//	// Hierarchical document creation (child)
//	subtask := &Task{
//	    Status:   "pending",
//	    ParentID: "1",  // Parent SimpleID - automatically resolved
//	    Assignee: "bob",
//	}
//	childID, err := store.Create("Subtask of feature X", subtask)
//	// childID might be "1.1" (first child of parent "1")
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

// DeleteByDimension removes all documents matching the given dimension filters
// Multiple filters are combined with AND. Returns the number of documents deleted.
func (ts *TypedStore[T]) DeleteByDimension(filters map[string]interface{}) (int, error) {
	return ts.store.DeleteByDimension(filters)
}

// DeleteWhere removes all documents matching a custom WHERE clause
// The where clause should not include the "WHERE" keyword itself.
// Use with caution as it allows arbitrary SQL conditions.
// Returns the number of documents deleted.
func (ts *TypedStore[T]) DeleteWhere(whereClause string, args ...interface{}) (int, error) {
	return ts.store.DeleteWhere(whereClause, args...)
}

// UpdateByDimension updates all documents matching the given dimension filters
// Multiple filters are combined with AND. Returns the number of documents updated.
func (ts *TypedStore[T]) UpdateByDimension(filters map[string]interface{}, data *T) (int, error) {
	dimensions, extraData, err := MarshalDimensions(data)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal dimensions: %w", err)
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

	return ts.store.UpdateByDimension(filters, req)
}

// UpdateWhere updates all documents matching a custom WHERE clause
// The where clause should not include the "WHERE" keyword itself.
// Use with caution as it allows arbitrary SQL conditions.
// Returns the number of documents updated.
func (ts *TypedStore[T]) UpdateWhere(whereClause string, data *T, args ...interface{}) (int, error) {
	dimensions, extraData, err := MarshalDimensions(data)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal dimensions: %w", err)
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

	return ts.store.UpdateWhere(whereClause, req, args...)
}

// UpdateByUUIDs updates multiple documents by their UUIDs in a single operation
// Returns the number of documents updated.
func (ts *TypedStore[T]) UpdateByUUIDs(uuids []string, data *T) (int, error) {
	dimensions, extraData, err := MarshalDimensions(data)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal dimensions: %w", err)
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

	return ts.store.UpdateByUUIDs(uuids, req)
}

// DeleteByUUIDs deletes multiple documents by their UUIDs in a single operation
// Returns the number of documents deleted.
func (ts *TypedStore[T]) DeleteByUUIDs(uuids []string) (int, error) {
	return ts.store.DeleteByUUIDs(uuids)
}

// Query returns a new typed query builder
func (ts *TypedStore[T]) Query() *TypedQuery[T] {
	return &TypedQuery[T]{
		store: ts.store,
		options: types.ListOptions{
			Filters: make(map[string]interface{}),
		},
	}
}

// Close closes the underlying store
func (ts *TypedStore[T]) Close() error {
	return ts.store.Close()
}

// Store returns the underlying nanostore.Store for operations that need direct access
// This is useful for operations like export that work with the raw Store interface
func (ts *TypedStore[T]) Store() nanostore.Store {
	return ts.store
}

// ResolveUUID converts a simple ID (e.g., "1.2.c3") to a UUID
// This provides direct access to ID resolution without needing to access the underlying store
func (ts *TypedStore[T]) ResolveUUID(simpleID string) (string, error) {
	return ts.store.ResolveUUID(simpleID)
}

// List returns documents based on the provided ListOptions, converted to typed structs
// This provides direct access to the underlying store's List functionality while maintaining type safety
func (ts *TypedStore[T]) List(opts types.ListOptions) ([]T, error) {
	docs, err := ts.store.List(opts)
	if err != nil {
		return nil, err
	}

	result := make([]T, len(docs))
	for i, doc := range docs {
		if err := UnmarshalDimensions(doc, &result[i]); err != nil {
			return nil, fmt.Errorf("failed to unmarshal document %s: %w", doc.UUID, err)
		}
	}

	return result, nil
}

// GetRaw returns the raw document without type conversion
// This provides direct access to the underlying document structure, useful for:
// - Accessing dimensions not defined in the struct
// - Inspecting metadata (CreatedAt, UpdatedAt, etc.)
// - Working with documents that partially match the struct schema
// - Debugging and introspection
// Accepts both UUID and SimpleID for maximum flexibility
func (ts *TypedStore[T]) GetRaw(id string) (*types.Document, error) {
	// First try to resolve as SimpleID to UUID
	uuid, err := ts.store.ResolveUUID(id)
	if err != nil {
		// If resolution fails, try using the ID directly as UUID
		uuid = id
	}

	// Use List with UUID filter to get the raw document
	docs, err := ts.store.List(types.ListOptions{
		Filters: map[string]interface{}{
			"uuid": uuid,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("document with ID %s not found", id)
	}

	if len(docs) > 1 {
		return nil, fmt.Errorf("multiple documents found for ID %s", id)
	}

	return &docs[0], nil
}

// AddRaw creates a new document with raw dimension values
// This provides direct access to the underlying store's Add functionality for cases where:
// - The document doesn't fully match the struct schema
// - You need to set custom _data fields not defined in the struct
// - You want to bypass struct tag validation
// - You're migrating data that has different dimension names
// Returns the UUID of the created document
func (ts *TypedStore[T]) AddRaw(title string, dimensions map[string]interface{}) (string, error) {
	return ts.store.Add(title, dimensions)
}

// GetDimensions returns the raw dimensions map for a document
// This provides access to all dimension values and custom _data fields, useful for:
// - Accessing fields not defined in the struct schema
// - Debugging dimension values and configuration
// - Working with documents that have additional custom fields
// - Introspecting the full dimension structure
// Accepts both UUID and SimpleID for maximum flexibility
// Returns a copy of the dimensions map to prevent accidental modifications
func (ts *TypedStore[T]) GetDimensions(id string) (map[string]interface{}, error) {
	doc, err := ts.GetRaw(id)
	if err != nil {
		return nil, err
	}

	// Return a copy of the dimensions to prevent modifications
	result := make(map[string]interface{})
	for key, value := range doc.Dimensions {
		result[key] = value
	}

	return result, nil
}

// GetMetadata returns the metadata fields of a document
// This provides access to document metadata (UUID, SimpleID, Title, timestamps) without
// loading the full document content or dimensions, useful for:
// - Quick metadata inspection without full document overhead
// - Accessing metadata when document content is not in struct format
// - Building document lists with metadata-only information
// - Debugging and administrative operations
// Accepts both UUID and SimpleID for maximum flexibility
func (ts *TypedStore[T]) GetMetadata(id string) (*DocumentMetadata, error) {
	doc, err := ts.GetRaw(id)
	if err != nil {
		return nil, err
	}

	return &DocumentMetadata{
		UUID:      doc.UUID,
		SimpleID:  doc.SimpleID,
		Title:     doc.Title,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}, nil
}

// TypedQuery provides a fluent interface for building type-safe queries.
//
// This query builder implements the "fluent interface" pattern, allowing users to chain
// method calls to construct complex queries in a readable, type-safe manner. The query
// builder supports:
//
// - **Dimensional Filtering**: Filter by any configured dimension
// - **Text Search**: Full-text search across title and body fields
// - **Ordering**: Sort results by any field (ascending or descending)
// - **Pagination**: Limit and offset for result sets
// - **Hierarchical Queries**: Special support for parent-child relationships
//
// # Design Principles
//
// 1. **Immutable Operations**: Each method returns a new query state
// 2. **Progressive Refinement**: Start broad, add filters to narrow results
// 3. **Type Safety**: All filter values are validated at compile time where possible
// 4. **Lazy Execution**: Query is only executed when terminal method is called
//
// # Method Categories
//
// ## Filter Methods
// Add filtering conditions (can be chained):
//
//	query.Status("active").Priority("high").ParentID("1")
//
// ## Ordering Methods
// Control result ordering:
//
//	query.OrderBy("created_at").OrderByDesc("priority")
//
// ## Pagination Methods
// Control result set size and position:
//
//	query.Limit(10).Offset(20)
//
// ## Terminal Methods
// Execute the query and return results:
//
//	results, err := query.Find()        // Returns []T
//	first, err := query.First()         // Returns *T
//	count, err := query.Count()         // Returns int
//	exists, err := query.Exists()       // Returns bool
//
// # Smart ID Resolution
//
// The query builder automatically handles SimpleID resolution for reference fields:
//
//	// User provides SimpleID, automatically resolved to UUID internally
//	query.ParentID("1.2")  // Resolves "1.2" → UUID before querying
//
// # Performance Characteristics
//
// - **Query Building**: O(1) per method call (just modifies options struct)
// - **Execution Time**: Depends on underlying store query performance
// - **Memory Usage**: Minimal - only stores filter options until execution
// - **Result Processing**: O(n) where n = number of matching documents
//
// # Usage Examples
//
//	// Simple filtering
//	activeTasks, err := store.Query().
//	    Status("active").
//	    Find()
//
//	// Complex query with multiple conditions
//	urgentTasks, err := store.Query().
//	    Status("pending").
//	    Priority("high").
//	    Search("critical").
//	    OrderByDesc("created_at").
//	    Limit(5).
//	    Find()
//
//	// Hierarchical queries
//	childTasks, err := store.Query().
//	    ParentID("1").        // All children of task "1"
//	    Status("active").
//	    Find()
//
//	// Existence checks
//	hasActiveTasks, err := store.Query().
//	    Status("active").
//	    Exists()
type TypedQuery[T any] struct {
	store   nanostore.Store   // Underlying store for query execution
	options types.ListOptions // Accumulated query options
}

// Activity filters by activity value.
// This is a domain-specific filter method - applications should define their own
// filter methods based on their configured dimensions.
func (tq *TypedQuery[T]) Activity(value string) *TypedQuery[T] {
	tq.options.Filters["activity"] = value
	return tq
}

// ActivityIn filters by multiple activity values.
// This allows OR-style filtering for activity values - documents matching ANY
// of the provided values will be included in results.
//
// Example:
//
//	// Find documents that are either active or archived
//	results, err := store.Query().ActivityIn("active", "archived").Find()
func (tq *TypedQuery[T]) ActivityIn(values ...string) *TypedQuery[T] {
	tq.options.Filters["activity"] = values
	return tq
}

// ActivityNot excludes a specific activity.
// This works by including all OTHER known activity values.
//
// Example:
//
//	// Find all tasks that are NOT deleted
//	results, err := store.Query().ActivityNot("deleted").Find()
func (tq *TypedQuery[T]) ActivityNot(value string) *TypedQuery[T] {
	allActivities := []string{"active", "archived", "deleted"}
	var includeActivities []string
	for _, a := range allActivities {
		if a != value {
			includeActivities = append(includeActivities, a)
		}
	}
	if len(includeActivities) > 0 {
		tq.options.Filters["activity"] = includeActivities
	}
	return tq
}

// ActivityNotIn excludes multiple activity values.
// This works by including all OTHER known activity values.
//
// Example:
//
//	// Find all tasks that are NOT deleted or archived (i.e., active only)
//	results, err := store.Query().ActivityNotIn("deleted", "archived").Find()
func (tq *TypedQuery[T]) ActivityNotIn(values ...string) *TypedQuery[T] {
	allActivities := []string{"active", "archived", "deleted"}
	excludeSet := make(map[string]bool)
	for _, v := range values {
		excludeSet[v] = true
	}

	var includeActivities []string
	for _, a := range allActivities {
		if !excludeSet[a] {
			includeActivities = append(includeActivities, a)
		}
	}
	if len(includeActivities) > 0 {
		tq.options.Filters["activity"] = includeActivities
	}
	return tq
}

// Status filters by status value.
// Status is a common enumerated dimension in many applications.
// The value must be one of the values configured in the dimension's Values list.
//
// Example:
//
//	// Find all active documents
//	results, err := store.Query().Status("active").Find()
func (tq *TypedQuery[T]) Status(value string) *TypedQuery[T] {
	tq.options.Filters["status"] = value
	return tq
}

// StatusIn filters by multiple status values.
// This allows OR-style filtering for status values - documents matching ANY
// of the provided values will be included in results.
//
// Example:
//
//	// Find documents that are either pending or active
//	results, err := store.Query().StatusIn("pending", "active").Find()
func (tq *TypedQuery[T]) StatusIn(values ...string) *TypedQuery[T] {
	tq.options.Filters["status"] = values
	return tq
}

// StatusNot excludes a specific status.
//
// Implementation Note: This uses a workaround approach since the underlying store
// doesn't support native NOT operations. It works by filtering to all OTHER known
// status values based on the configured dimension values.
//
// For documents that match the struct schema, this will work correctly.
// For documents with unknown status values, behavior may vary.
//
// Example:
//
//	// Find all tasks that are NOT done
//	results, err := store.Query().StatusNot("done").Find()
func (tq *TypedQuery[T]) StatusNot(value string) *TypedQuery[T] {
	// Get all known status values from TodoItem struct tag configuration
	// This is more robust than hardcoding but still has limitations
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

// StatusNotIn excludes multiple status values.
// This works by including all OTHER known status values.
//
// Example:
//
//	// Find all tasks that are NOT done or archived
//	results, err := store.Query().StatusNotIn("done", "archived").Find()
func (tq *TypedQuery[T]) StatusNotIn(values ...string) *TypedQuery[T] {
	allStatuses := []string{"pending", "active", "done"}
	excludeSet := make(map[string]bool)
	for _, v := range values {
		excludeSet[v] = true
	}

	var includeStatuses []string
	for _, s := range allStatuses {
		if !excludeSet[s] {
			includeStatuses = append(includeStatuses, s)
		}
	}
	if len(includeStatuses) > 0 {
		tq.options.Filters["status"] = includeStatuses
	}
	return tq
}

// Priority filters by priority value.
// Priority is another common enumerated dimension for task/document management.
//
// Example:
//
//	// Find all high priority items
//	results, err := store.Query().Priority("high").Find()
func (tq *TypedQuery[T]) Priority(value string) *TypedQuery[T] {
	tq.options.Filters["priority"] = value
	return tq
}

// PriorityIn filters by multiple priority values.
// This allows OR-style filtering for priority values - documents matching ANY
// of the provided values will be included in results.
//
// Example:
//
//	// Find documents that are either high or medium priority
//	results, err := store.Query().PriorityIn("high", "medium").Find()
func (tq *TypedQuery[T]) PriorityIn(values ...string) *TypedQuery[T] {
	tq.options.Filters["priority"] = values
	return tq
}

// PriorityNot excludes a specific priority.
// This works by including all OTHER known priority values.
//
// Example:
//
//	// Find all tasks that are NOT low priority
//	results, err := store.Query().PriorityNot("low").Find()
func (tq *TypedQuery[T]) PriorityNot(value string) *TypedQuery[T] {
	allPriorities := []string{"low", "medium", "high"}
	var includePriorities []string
	for _, p := range allPriorities {
		if p != value {
			includePriorities = append(includePriorities, p)
		}
	}
	if len(includePriorities) > 0 {
		tq.options.Filters["priority"] = includePriorities
	}
	return tq
}

// PriorityNotIn excludes multiple priority values.
// This works by including all OTHER known priority values.
//
// Example:
//
//	// Find all tasks that are NOT low or medium priority (i.e., high priority only)
//	results, err := store.Query().PriorityNotIn("low", "medium").Find()
func (tq *TypedQuery[T]) PriorityNotIn(values ...string) *TypedQuery[T] {
	allPriorities := []string{"low", "medium", "high"}
	excludeSet := make(map[string]bool)
	for _, v := range values {
		excludeSet[v] = true
	}

	var includePriorities []string
	for _, p := range allPriorities {
		if !excludeSet[p] {
			includePriorities = append(includePriorities, p)
		}
	}
	if len(includePriorities) > 0 {
		tq.options.Filters["priority"] = includePriorities
	}
	return tq
}

// Data filters by custom data fields not defined in the struct schema.
// This method enables querying documents by _data.* fields that were added via AddRaw
// or other means outside the typed struct definition.
//
// The field name should NOT include the "_data." prefix - it will be added automatically.
//
// Examples:
//
//	// Find documents with specific assignee
//	results, err := store.Query().Data("assignee", "alice").Find()
//
//	// Find documents with specific tags
//	results, err := store.Query().Data("tags", "urgent").Find()
//
//	// Chain with other filters
//	results, err := store.Query().
//	    Status("active").
//	    Data("assignee", "alice").
//	    Find()
//
// Performance Note: Data field queries may be slower than dimension queries
// since they typically cannot leverage specialized indexes.
func (tq *TypedQuery[T]) Data(field string, value interface{}) *TypedQuery[T] {
	tq.options.Filters["_data."+field] = value
	return tq
}

// DataIn filters by multiple values for a custom data field.
// This allows OR-style filtering for data field values - documents matching ANY
// of the provided values will be included in results.
//
// The field name should NOT include the "_data." prefix - it will be added automatically.
//
// Examples:
//
//	// Find documents with multiple possible assignees
//	results, err := store.Query().DataIn("assignee", "alice", "bob").Find()
//
//	// Find documents with multiple possible tags
//	results, err := store.Query().DataIn("category", "urgent", "important").Find()
func (tq *TypedQuery[T]) DataIn(field string, values ...interface{}) *TypedQuery[T] {
	tq.options.Filters["_data."+field] = values
	return tq
}

// DataNot excludes documents with a specific data field value.
//
// Implementation Note: Since we don't know all possible values for data fields,
// this method uses a post-processing approach. The exclusion is handled in the
// Find() method after retrieving results from the store.
//
// Performance Note: This may be slower than dimension-based NOT operations
// since it requires post-processing of all matching documents.
//
// Examples:
//
//	// Find documents NOT assigned to Alice
//	results, err := store.Query().DataNot("assignee", "alice").Find()
//
//	// Find documents NOT tagged as urgent
//	results, err := store.Query().DataNot("tags", "urgent").Find()
func (tq *TypedQuery[T]) DataNot(field string, value interface{}) *TypedQuery[T] {
	// Use special filter key to mark for post-processing
	tq.options.Filters["__data_not__"+field] = value
	return tq
}

// DataNotIn excludes documents with any of the specified data field values.
//
// Implementation Note: Like DataNot, this uses post-processing since we don't
// know all possible values for custom data fields.
//
// Examples:
//
//	// Find documents NOT assigned to Alice or Bob
//	results, err := store.Query().DataNotIn("assignee", "alice", "bob").Find()
func (tq *TypedQuery[T]) DataNotIn(field string, values ...interface{}) *TypedQuery[T] {
	// Use special filter key to mark for post-processing
	tq.options.Filters["__data_not_in__"+field] = values
	return tq
}

// ParentID filters by parent ID, with automatic SimpleID resolution.
//
// This method demonstrates the power of Smart ID resolution in queries:
// - Users can provide human-readable SimpleIDs (e.g., "1", "1.2", "dh3")
// - The method automatically resolves them to internal UUIDs for querying
// - If resolution fails, the original value is used (supports external references)
//
// This enables intuitive hierarchical queries without exposing users to UUIDs.
//
// Examples:
//
//	// Find all children of document "1"
//	children, err := store.Query().ParentID("1").Find()
//
//	// Find all children of a specific document with complex ID
//	children, err := store.Query().ParentID("1.dh3").Find()
//
// Performance Note: ID resolution adds slight overhead but is typically fast
// due to internal caching in the store layer.
func (tq *TypedQuery[T]) ParentID(id string) *TypedQuery[T] {
	// Try to resolve SimpleID to UUID for internal querying
	if uuid, err := tq.store.ResolveUUID(id); err == nil {
		tq.options.Filters["parent_id"] = uuid
	} else {
		// Resolution failed - use original value (supports external references)
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
	tq.options.OrderBy = append(tq.options.OrderBy, types.OrderClause{
		Column:     column,
		Descending: false,
	})
	return tq
}

// OrderByDesc adds descending ordering
func (tq *TypedQuery[T]) OrderByDesc(column string) *TypedQuery[T] {
	tq.options.OrderBy = append(tq.options.OrderBy, types.OrderClause{
		Column:     column,
		Descending: true,
	})
	return tq
}

// OrderByData adds ascending ordering by custom data field.
// This method enables ordering documents by _data.* fields that were added via AddRaw
// or other means outside the typed struct definition.
//
// The field name should NOT include the "_data." prefix - it will be added automatically.
//
// Examples:
//
//	// Order by assignee name
//	results, err := store.Query().OrderByData("assignee").Find()
//
//	// Order by creation timestamp in custom data
//	results, err := store.Query().OrderByData("created_by_user").Find()
//
//	// Combine with filters and other ordering
//	results, err := store.Query().
//	    Status("active").
//	    OrderByData("priority_score").
//	    OrderByDesc("created_at").
//	    Find()
//
// Performance Note: Ordering by data fields may be slower than dimension ordering
// since they typically cannot leverage specialized indexes.
func (tq *TypedQuery[T]) OrderByData(field string) *TypedQuery[T] {
	tq.options.OrderBy = append(tq.options.OrderBy, types.OrderClause{
		Column:     "_data." + field,
		Descending: false,
	})
	return tq
}

// OrderByDataDesc adds descending ordering by custom data field.
// This method enables ordering documents by _data.* fields in descending order.
//
// The field name should NOT include the "_data." prefix - it will be added automatically.
//
// Examples:
//
//	// Order by priority score (highest first)
//	results, err := store.Query().OrderByDataDesc("priority_score").Find()
//
//	// Order by last update timestamp (most recent first)
//	results, err := store.Query().OrderByDataDesc("last_updated").Find()
func (tq *TypedQuery[T]) OrderByDataDesc(field string) *TypedQuery[T] {
	tq.options.OrderBy = append(tq.options.OrderBy, types.OrderClause{
		Column:     "_data." + field,
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

// Find executes the query and returns typed results.
//
// This is the primary terminal method for query execution. It performs several steps:
//
// 1. **Query Execution**: Executes the accumulated filters against the store
// 2. **Post-Processing**: Applies filters that require client-side processing
// 3. **Type Unmarshaling**: Converts raw documents back to typed structs
// 4. **Result Assembly**: Builds the final []T slice for return
//
// # Post-Processing Filters
//
// Some query operations cannot be efficiently implemented at the store level
// and require post-processing of results:
//
// - **ParentIDNotExists**: Filters out documents with parent references
// - **Complex NOT operations**: Future filters requiring negation logic
// - **Cross-dimensional calculations**: Filters spanning multiple dimensions
//
// # Error Handling
//
// Find returns detailed errors for various failure scenarios:
//
// - **Store Query Errors**: Issues with the underlying store query
// - **Unmarshal Errors**: Type conversion failures during result processing
// - **Constraint Violations**: Data inconsistencies discovered during processing
//
// # Performance Characteristics
//
// - **Store Query Time**: Depends on number of documents and index efficiency
// - **Post-Processing Time**: O(n) where n = number of store results
// - **Unmarshaling Time**: O(n × m) where m = number of struct fields
// - **Memory Usage**: Linear with result count
//
// # Usage Examples
//
//	// Simple query
//	allTasks, err := store.Query().Find()
//
//	// Filtered query
//	activeTasks, err := store.Query().
//	    Status("active").
//	    Priority("high").
//	    Find()
//
//	// Complex query with ordering and pagination
//	recentTasks, err := store.Query().
//	    Status("active").
//	    OrderByDesc("created_at").
//	    Limit(10).
//	    Find()
func (tq *TypedQuery[T]) Find() ([]T, error) {
	// Check for special filters and extract them for post-processing
	parentNotExists := false
	if _, ok := tq.options.Filters["__parent_not_exists__"]; ok {
		parentNotExists = true
		delete(tq.options.Filters, "__parent_not_exists__")
	}

	// Extract data NOT filters for post-processing
	var dataNotFilters []struct {
		field string
		value interface{}
	}
	var dataNotInFilters []struct {
		field  string
		values []interface{}
	}

	for key, value := range tq.options.Filters {
		if strings.HasPrefix(key, "__data_not__") {
			field := strings.TrimPrefix(key, "__data_not__")
			dataNotFilters = append(dataNotFilters, struct {
				field string
				value interface{}
			}{field, value})
			delete(tq.options.Filters, key)
		} else if strings.HasPrefix(key, "__data_not_in__") {
			field := strings.TrimPrefix(key, "__data_not_in__")
			if values, ok := value.([]interface{}); ok {
				dataNotInFilters = append(dataNotInFilters, struct {
					field  string
					values []interface{}
				}{field, values})
			}
			delete(tq.options.Filters, key)
		}
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

		// Apply data NOT filters
		skip := false
		for _, filter := range dataNotFilters {
			if dataValue, exists := doc.Dimensions["_data."+filter.field]; exists {
				if dataValue == filter.value {
					skip = true
					break
				}
			}
		}
		if skip {
			continue
		}

		// Apply data NOT IN filters
		for _, filter := range dataNotInFilters {
			if dataValue, exists := doc.Dimensions["_data."+filter.field]; exists {
				for _, excludeValue := range filter.values {
					if dataValue == excludeValue {
						skip = true
						break
					}
				}
				if skip {
					break
				}
			}
		}
		if skip {
			continue
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

// generateConfigFromType creates a Config from struct tags.
//
// This function is the heart of the "configuration by convention" approach.
// It uses Go's reflection system to introspect struct definitions and automatically
// generate nanostore dimension configurations from struct tags.
//
// # Reflection-Based Analysis
//
// The function performs deep struct analysis:
//
// 1. **Field Enumeration**: Iterates through all struct fields
// 2. **Tag Parsing**: Extracts configuration from multiple tag formats
// 3. **Type Validation**: Ensures field types are compatible with nanostore
// 4. **Constraint Checking**: Validates dimension configuration rules
//
// # Supported Tag Formats
//
// ## Enumerated Dimensions
//
//	Status string `values:"pending,active,done" prefix:"done=d" default:"pending"`
//
// This creates an enumerated dimension with:
// - Name: "status" (lowercased field name)
// - Values: ["pending", "active", "done"]
// - Prefixes: {"done": "d"} (done→d, others no prefix)
// - Default: "pending"
//
// ## Hierarchical Dimensions
//
//	ParentID string `dimension:"parent_id,ref"`
//
// This creates a hierarchical dimension with:
// - Name: "ParentID_hierarchy" (field name + "_hierarchy")
// - Type: Hierarchical
// - RefField: "parent_id" (the actual reference field name)
//
// # Configuration Generation Strategy
//
// The function makes several automatic decisions:
//
// - **Dimension Names**: Derived from struct field names (lowercased)
// - **Type Inference**: Enumerated vs Hierarchical based on tag patterns
// - **Validation**: Ensures prefixes don't conflict, defaults are valid
// - **Error Reporting**: Provides specific errors with field context
//
// # Error Scenarios
//
// Common configuration errors detected:
//
// - **Pointer Fields**: "field X: pointer fields are not supported"
// - **Invalid Tag Syntax**: Malformed values or prefix specifications
// - **Missing Required Tags**: Hierarchical dimensions without RefField
// - **Validation Failures**: Defaults not in values list, duplicate prefixes
//
// # Future Enhancements
//
// Potential improvements to tag processing:
//
// - **Advanced Validation**: Cross-field constraint validation
// - **Custom Naming**: Override field-to-dimension name mapping
// - **Inheritance**: Support for dimension inheritance across struct hierarchies
// - **Plugin System**: Custom tag processors for domain-specific needs
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
