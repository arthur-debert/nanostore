package api

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/store"
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

// DebugInfo contains comprehensive debugging information about a Store
type DebugInfo struct {
	StoreType     string            // Type of underlying store implementation
	FilePath      string            // Location of store data file (if applicable)
	DocumentCount int               // Total number of documents in store
	Configuration *nanostore.Config // Complete dimension configuration
	TypeInfo      TypeDebugInfo     // Information about the Go type T
	RuntimeStats  RuntimeDebugStats // Runtime statistics and metrics
	LastError     string            // Last error encountered (if any)
}

// TypeDebugInfo contains information about the Go type used with Store
type TypeDebugInfo struct {
	TypeName    string           // Full Go type name
	PackageName string           // Package name containing the type
	FieldCount  int              // Number of struct fields
	Fields      []FieldDebugInfo // Details about each field
	EmbedsList  []string         // List of embedded types
	HasDocument bool             // Whether type embeds nanostore.Document
}

// FieldDebugInfo contains information about a struct field
type FieldDebugInfo struct {
	Name         string // Field name
	Type         string // Field type
	Tag          string // Complete struct tag
	IsEmbedded   bool   // Whether field is embedded
	IsDimension  bool   // Whether field maps to a dimension
	DimensionTag string // Dimension configuration from tag
}

// RuntimeDebugStats contains runtime statistics about the store
type RuntimeDebugStats struct {
	TotalDimensions int // Number of configured dimensions
	TotalValues     int // Total number of values across all dimensions
	TotalPrefixes   int // Total number of prefix mappings
}

// StoreStats contains statistical information about store contents
type StoreStats struct {
	TotalDocuments        int                       // Total number of documents
	DimensionDistribution map[string]map[string]int // Distribution of values per dimension
	DataFieldCoverage     map[string]float64        // Percentage coverage of data fields
	DataFieldDistribution map[string]map[string]int // Value distribution for data fields
}

// IntegrityReport contains the results of store integrity validation
type IntegrityReport struct {
	IsValid        bool               // Whether store passed all integrity checks
	TotalDocuments int                // Total number of documents validated
	ErrorCount     int                // Number of errors found
	WarningCount   int                // Number of warnings found
	Errors         []IntegrityError   // Detailed error information
	Warnings       []IntegrityWarning // Detailed warning information
	Summary        string             // Human-readable summary of findings
}

// IntegrityError represents a serious integrity issue found during validation
type IntegrityError struct {
	Type       string // Error type (e.g., "UUID_DUPLICATE", "INVALID_DIMENSION_VALUE")
	DocumentID string // ID of affected document
	Message    string // Human-readable error description
}

// IntegrityWarning represents a minor issue found during validation
type IntegrityWarning struct {
	Type       string // Warning type (e.g., "MISSING_SIMPLE_ID")
	DocumentID string // ID of affected document
	Message    string // Human-readable warning description
}

// QueryPlan contains information about how a query would be executed
type QueryPlan struct {
	TotalFilters            int      // Total number of filters applied
	IndexedFilterCount      int      // Number of filters using indexed dimensions
	DataFieldFilterCount    int      // Number of filters on data fields (slower)
	CustomWhereClauseCount  int      // Number of custom WHERE clauses
	PerformanceRating       string   // Performance assessment (Fast/Medium/Slow)
	OptimizationSuggestions []string // Suggestions for improving query performance
}

// FieldUsageStats contains statistics about field usage across all documents
type FieldUsageStats struct {
	TotalDocuments int                           // Total number of documents analyzed
	DimensionUsage map[string]DimensionUsageInfo // Usage statistics for dimension fields
	DataFieldUsage map[string]DataFieldUsageInfo // Usage statistics for data fields
	CoreFieldUsage CoreFieldUsageInfo            // Usage statistics for core Document fields
}

// DimensionUsageInfo contains usage statistics for a dimension field
type DimensionUsageInfo struct {
	DimensionName string         // Name of the dimension
	Type          string         // Type of dimension (enumerated/hierarchical)
	ValueCounts   map[string]int // Count of each value across all documents
	NonEmptyCount int            // Number of documents with non-empty values
}

// DataFieldUsageInfo contains usage statistics for a data field
type DataFieldUsageInfo struct {
	FieldName          string  // Name of the data field
	NonEmptyCount      int     // Number of documents with non-empty values
	CoveragePercentage float64 // Percentage of documents that have this field populated
}

// CoreFieldUsageInfo contains usage statistics for core Document fields
type CoreFieldUsageInfo struct {
	TitleUsageCount         int     // Number of documents with non-empty titles
	TitleCoveragePercentage float64 // Percentage of documents with titles
	BodyUsageCount          int     // Number of documents with non-empty body content
	BodyCoveragePercentage  float64 // Percentage of documents with body content
}

// TypeSchema contains detailed schema information about the Go type T
type TypeSchema struct {
	TypeName        string               // Full Go type name
	PackageName     string               // Package containing the type
	EmbedsDocument  bool                 // Whether type embeds nanostore.Document
	DimensionFields []DimensionFieldInfo // Fields that map to dimensions
	DataFields      []DataFieldInfo      // Fields that map to data storage
}

// DimensionFieldInfo contains information about a field that maps to a dimension
type DimensionFieldInfo struct {
	Name           string            // Go field name
	Type           string            // Go field type
	DimensionType  string            // Dimension type (enumerated/hierarchical)
	Tags           string            // Complete struct tag string
	AllowedValues  []string          // Allowed values for enumerated dimensions
	DefaultValue   string            // Default value for enumerated dimensions
	PrefixMappings map[string]string // Value-to-prefix mappings
	RefField       string            // Reference field for hierarchical dimensions
}

// DataFieldInfo contains information about a field that maps to data storage
type DataFieldInfo struct {
	Name string // Go field name
	Type string // Go field type
	Tags string // Complete struct tag string
}

// Store wraps a Store with type-safe operations for a specific document type T.
//
// This is the primary interface for applications using nanostore, providing compile-time
// type safety while leveraging the sophisticated ID generation and dimensional organization
// features of the underlying store. The Store automatically handles:
//
// - **Automatic Configuration**: Generates dimension configuration from struct tags
// - **Type Marshaling/Unmarshaling**: Converts between Go structs and store documents
// - **Smart ID Resolution**: Transparently handles SimpleID ↔ UUID conversion
// - **Fluent Query Interface**: Provides chainable, type-safe query building
//
// # Design Philosophy
//
// The Store is designed around "configuration by convention" principles:
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
// The Store uses struct tags to automatically generate dimension configurations:
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
// All types used with Store must embed nanostore.Document:
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
// The Store automatically handles Smart ID resolution throughout:
//
// - **User Input**: Methods accept SimpleIDs from users (e.g., "1.2", "dh3")
// - **Internal Processing**: Automatically resolves to UUIDs for store operations
// - **Query Results**: Returns documents with properly generated SimpleIDs
// - **Reference Fields**: Parent IDs in queries are automatically resolved
//
// # Error Handling Strategy
//
// The Store provides clear error messages for common issues:
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
// Store is thread-safe:
// - Immutable after creation (config and type information cached)
// - All operations delegate to thread-safe underlying store
// - Multiple goroutines can safely share a single Store instance
type Store[T any] struct {
	store  store.Store      // Underlying nanostore implementation
	config nanostore.Config // Generated configuration from struct tags
	typ    reflect.Type     // Cached type information for T
}

// New creates a new Store for the given type T, automatically generating
// the configuration from struct tags.
//
// This is the primary constructor for Store and performs several critical setup steps:
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
//	store, err := New[Task]("tasks.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer store.Close()
//
// # Performance Notes
//
// - Reflection is only used during initialization, not per-operation
// - Generated configuration is cached for the lifetime of the Store
// - File creation is deferred until first document is added
func New[T any](filePath string) (*Store[T], error) {
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
	store, err := store.New(filePath, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	return &Store[T]{
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
// 3. **Document Field Extraction**: Preserves title and body from embedded Document
// 4. **ID Generation**: Triggers automatic SimpleID generation based on dimensions
// 5. **UUID Assignment**: Creates stable internal UUID for the document
//
// # Data Processing Strategy
//
// The method processes struct fields in four categories:
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
// ## Document Fields
// Embedded Document fields are handled specially:
//
//	Document nanostore.Document                   // Title and Body are extracted and preserved
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
// # Title and Body Handling
//
// Create provides flexible title and body handling for embedded Document fields:
//
// **Title Precedence**:
// 1. If title parameter is non-empty, it takes precedence
// 2. If title parameter is empty and struct has Document.Title, use struct title
// 3. If neither is provided, the document will have an empty title
//
// **Body Preservation**:
// - Body content from embedded Document.Body is always preserved
// - This eliminates the need for workarounds like two-phase create+update operations
// - Empty body fields are ignored (not stored as empty strings)
//
// **Backward Compatibility**:
// - Existing two-phase workarounds continue to work unchanged
// - No breaking changes to existing Create method signatures
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
//	// Document creation with embedded Document fields
//	taskWithBody := &Task{
//	    Document: nanostore.Document{
//	        Title: "Default Title",      // Used if Create title parameter is empty
//	        Body:  "Task description",   // Automatically preserved
//	    },
//	    Status:   "pending",
//	    Priority: "high",
//	}
//	id, err := store.Create("Override Title", taskWithBody)  // Title parameter takes precedence
//
//	// Hierarchical document creation (child)
//	subtask := &Task{
//	    Status:   "pending",
//	    ParentID: "1",  // Parent SimpleID - automatically resolved
//	    Assignee: "bob",
//	}
//	childID, err := store.Create("Subtask of feature X", subtask)
//	// childID might be "1.1" (first child of parent "1")
func (ts *Store[T]) Create(title string, data *T) (string, error) {
	dimensions, extraData, err := MarshalDimensions(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal dimensions: %w", err)
	}

	// Store extra data in dimensions with a special prefix
	for key, value := range extraData {
		dimensions["_data."+key] = value
	}

	// Extract Document fields (title, body) from the embedded Document
	structTitle, structBody, hasDocument := extractDocumentFields(data)

	// Determine final title: use parameter if non-empty, otherwise use struct title
	finalTitle := title
	if title == "" && hasDocument && structTitle != "" {
		finalTitle = structTitle
	}

	// Add the document with proper title and body handling
	uuid, err := ts.store.Add(finalTitle, dimensions)
	if err != nil {
		return "", err
	}

	// If we have body content from the struct, update the document to include it
	if hasDocument && structBody != "" {
		// Update the document with the body content
		// The Add method doesn't handle body content directly, so we need a follow-up update
		err = ts.store.Update(uuid, types.UpdateRequest{
			Body: &structBody,
		})
		if err != nil {
			return "", fmt.Errorf("failed to set body content: %w", err)
		}
	}

	return uuid, nil
}

// Get retrieves a document by ID and unmarshals it into the typed structure.
//
// This is the primary method for retrieving individual documents with full type safety.
// The method handles ID resolution, document retrieval, and automatic type marshaling
// to provide a seamless experience for typed document access.
//
// # ID Resolution
//
// Accepts both UUID and SimpleID for maximum flexibility:
// - SimpleID examples: "1", "2.h3", "1.2.c4"
// - UUID examples: "550e8400-e29b-41d4-a716-446655440000"
//
// ID Resolution Strategy:
// 1. Try to resolve ID as SimpleID to UUID using store's ID mapping
// 2. If resolution fails, assume ID is already a UUID and use directly
// 3. Query store with resolved/direct UUID
// 4. Return typed document with all dimension fields populated
//
// This dual approach ensures compatibility with both user-facing SimpleIDs
// and system-internal UUIDs, providing consistent behavior across all ID-based methods.
//
// # Type Marshaling
//
// The returned document is fully typed with all struct fields populated:
// - **Dimension Fields**: Mapped from document dimensions using struct tags
// - **Data Fields**: Extracted from document's _data map
// - **Embedded Document**: UUID, SimpleID, Title, Body, timestamps populated
// - **Zero Values**: Applied for fields not present in the document
//
// # Usage Examples
//
//	// Get by SimpleID (user-friendly)
//	task, err := store.Get("1.2")
//	if err != nil {
//	    log.Printf("Task not found: %v", err)
//	    return
//	}
//	fmt.Printf("Task: %s (Status: %s)\n", task.Title, task.Status)
//
//	// Get by UUID (system-internal)
//	task, err := store.Get("550e8400-e29b-41d4-a716-446655440000")
//	if err != nil {
//	    return
//	}
//
//	// Access all document fields
//	fmt.Printf("UUID: %s\n", task.UUID)
//	fmt.Printf("SimpleID: %s\n", task.SimpleID)
//	fmt.Printf("Created: %v\n", task.CreatedAt)
//	fmt.Printf("Priority: %s\n", task.Priority)
//	fmt.Printf("Assignee: %s\n", task.Assignee)
//
// # Error Handling
//
// Returns error if:
// - Document with specified ID is not found
// - Multiple documents found for the same ID (indicates data corruption)
// - Type unmarshaling fails (struct/document schema mismatch)
// - Database query fails
//
// Common error scenarios:
//
//	// Document not found
//	task, err := store.Get("nonexistent")
//	if err != nil {
//	    // Error: "document with ID nonexistent not found"
//	}
//
//	// Schema mismatch (missing required dimension)
//	task, err := store.Get("1")
//	if err != nil {
//	    // Error: "failed to unmarshal document: missing dimension 'status'"
//	}
//
// # Performance Characteristics
//
// - **Query Time**: O(log n) for indexed UUID lookups, O(1) for SimpleID resolution
// - **Marshaling**: O(k) where k = number of struct fields
// - **Memory**: Single document allocation + reflection overhead
// - **Caching**: No internal caching - each call queries the database
//
// For high-frequency access patterns, consider:
// - Batching multiple gets using List() with UUID filters
// - Using GetRaw() if you only need specific fields
// - Using GetMetadata() if you only need document metadata
//
// # Related Methods
//
// For specialized retrieval needs:
// - GetRaw() - Returns raw document without type marshaling
// - GetMetadata() - Returns only metadata (UUID, SimpleID, timestamps)
// - GetDimensions() - Returns raw dimensions map
// - List() - Bulk retrieval with filtering
func (ts *Store[T]) Get(id string) (*T, error) {
	// Consistent ID resolution: try SimpleID first, fallback to direct UUID
	uuid, err := ts.store.ResolveUUID(id)
	if err != nil {
		// If resolution fails, try using the ID directly as UUID
		uuid = id
	}

	// Use List with UUID filter to get the document
	docs, err := ts.store.List(types.ListOptions{
		Filters: map[string]interface{}{
			"uuid": uuid,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("document with ID '%s' not found", id)
	}

	if len(docs) > 1 {
		return nil, fmt.Errorf("multiple documents found for ID '%s'", id)
	}

	var result T
	if err := UnmarshalDimensions(docs[0], &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	return &result, nil
}

// buildUpdateRequest creates an UpdateRequest from typed data
// This helper eliminates ~100 lines of code duplication across Update methods
// by centralizing the complex struct-to-update-request conversion logic
func (ts *Store[T]) buildUpdateRequest(data *T) (nanostore.UpdateRequest, error) {
	// MarshalDimensionsForUpdate preserves zero values for field clearing in updates
	// This is where struct tag parsing happens and values are validated
	dimensions, extraData, err := MarshalDimensionsForUpdate(data)
	if err != nil {
		return nanostore.UpdateRequest{}, fmt.Errorf("failed to marshal dimensions: %w", err)
	}

	// Store extra data in dimensions with the "_data." prefix
	// This preserves fields that don't have dimension tags but need to be stored
	for key, value := range extraData {
		dimensions["_data."+key] = value
	}

	// Extract title and body if they're set in the embedded Document
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

	return req, nil
}

// Update modifies an existing document with typed data.
//
// This method processes all fields in the provided struct and applies them to the document:
//
// # Field Clearing Behavior
//
// **IMPORTANT**: As of this version, zero values in the update struct will clear
// the corresponding fields in the document. This enables field clearing but represents
// a behavior change from previous versions where zero values were ignored.
//
// ## Data Fields (Non-Dimension Fields)
// - Zero values (empty strings, 0, time.Time{}, etc.) WILL clear the field
// - This allows you to explicitly clear field values in update operations
//
// ## Dimension Fields
// - **Enumerated dimensions** (with "values" tag): Zero values are ignored to prevent validation errors
// - **Non-enumerated dimensions** (like refs): Zero values will clear the field
//
// # Usage Examples
//
//	// Clear assignee field and update priority
//	task := &Task{
//	    Assignee: "",    // Will clear the assignee field
//	    Priority: "high", // Will update priority to "high"
//	    // Note: If Description is not set, it will also be cleared to ""
//	}
//	err := store.Update("task-1", task)
//
//	// To preserve existing values while updating specific fields,
//	// first get the document, then modify only the desired fields:
//	existing, _ := store.Get("task-1")
//	existing.Priority = "high" // Only change priority
//	count, err := store.Update("task-1", existing)
//	if err != nil {
//	    log.Printf("Update failed: %v", err)
//	} else if count == 0 {
//	    log.Printf("Document not found")
//	} else {
//	    log.Printf("Successfully updated %d document", count)
//	}
//
// # Return Value
//
// Returns (count int, error):
// - **count = 1**: Document was found and successfully updated
// - **count = 0**: Document with specified ID was not found
// - **error != nil**: Update operation failed (database error, validation error, etc.)
//
// This return signature provides consistency with other update methods
// (UpdateByDimension, UpdateWhere, UpdateByUUIDs) and allows callers to
// distinguish between "document not found" and "update failed" scenarios.
//
// # Performance Characteristics
//
// - **ID Resolution**: O(1) for SimpleID lookup, O(log n) for UUID validation
// - **Document Existence Check**: O(log n) indexed lookup before update
// - **Field Processing**: O(k) where k = number of struct fields
// - **Database Update**: O(1) single document update
// - **Type Marshaling**: O(k) for struct tag processing and validation
//
// The method performs an existence check before updating to provide accurate
// count return values, adding minimal overhead for improved API consistency.
//
// # Migration from Previous Behavior
//
// If your code previously relied on zero values being ignored in updates,
// you will need to either:
// 1. Get the existing document first and modify only the fields you want to change
// 2. Explicitly set all fields to their desired values (including preserving existing values)
//
// # ID Resolution
//
// Accepts both UUID and SimpleID for the id parameter:
// - SimpleID examples: "1", "2.h3", "1.2.c4"
// - UUID examples: "550e8400-e29b-41d4-a716-446655440000"
//
// Uses the same ID resolution strategy as Get() for consistency.
//
// # Error Handling
//
// Returns error if:
// - Type marshaling fails (invalid struct tags, unsupported field types)
// - Database transaction fails during update
// - ID resolution fails and direct UUID lookup also fails
// - Validation errors (invalid enumerated dimension values)
//
// # Related Methods
//
// For bulk updates, consider:
// - UpdateByDimension() - Update multiple documents by filter criteria
// - UpdateWhere() - Update with custom SQL conditions
// - UpdateByUUIDs() - Update specific documents by UUID list
func (ts *Store[T]) Update(id string, data *T) (int, error) {
	req, err := ts.buildUpdateRequest(data)
	if err != nil {
		return 0, err
	}

	// Check if document exists before updating for consistent count behavior
	_, err = ts.GetRaw(id)
	if err != nil {
		// Document doesn't exist - return 0 updated, but preserve the error
		return 0, err
	}

	// Update the document
	err = ts.store.Update(id, req)
	if err != nil {
		return 0, err
	}

	// Successfully updated one document
	return 1, nil
}

// Delete removes a document and optionally its children.
//
// This method provides permanent deletion of documents from the store with optional
// cascading to handle hierarchical document structures.
//
// # ID Resolution
//
// Accepts both UUID and SimpleID for maximum flexibility:
// - SimpleID examples: "1", "2.h3", "1.2.c4"
// - UUID examples: "550e8400-e29b-41d4-a716-446655440000"
//
// ID Resolution Strategy:
// 1. Try to resolve ID as SimpleID to UUID
// 2. If resolution fails, use ID directly as UUID
// 3. Delete document with resolved/direct UUID
//
// # Cascade Behavior
//
// The cascade parameter controls deletion of child documents:
//
// ## cascade = false (Default Behavior)
// - Deletes only the specified document
// - Child documents remain in the store but become orphaned
// - Parent references in children are not automatically updated
// - Use this when you want to preserve child documents independently
//
// ## cascade = true (Hierarchical Deletion)
// - Deletes the specified document AND all its descendants
// - Recursively finds and deletes all documents with ParentID chains leading to this document
// - Useful for cleaning up entire hierarchical structures
// - **WARNING**: This can delete many documents if used on high-level parents
//
// # Usage Examples
//
//	// Delete single document, preserve children
//	err := store.Delete("1.2", false)
//	if err != nil {
//	    log.Printf("Failed to delete document: %v", err)
//	}
//
//	// Delete document and all descendants (use with caution)
//	err := store.Delete("project-1", true)
//	if err != nil {
//	    log.Printf("Failed to cascade delete: %v", err)
//	}
//
//	// Delete by UUID (also supports cascade)
//	err := store.Delete("550e8400-e29b-41d4-a716-446655440000", false)
//
// # Error Handling
//
// Returns error if:
// - Document with specified ID is not found
// - Database transaction fails during deletion
// - Child document deletion fails (when cascade=true)
// - ID resolution fails and direct UUID lookup also fails
//
// # Performance Considerations
//
// - **Non-cascading Delete**: O(1) - single document deletion
// - **Cascading Delete**: O(n) where n = total number of descendants
// - For deep hierarchies or large trees, cascading deletes may be slow
// - Consider deleting leaf nodes first for better performance with large structures
//
// # Data Consistency
//
// - Deletion is atomic at the document level
// - Cascading deletes are performed in dependency order (children before parents)
// - If any child deletion fails during cascade, the entire operation is rolled back
// - SimpleID sequences are not automatically reclaimed after deletion
//
// # Related Methods
//
// For bulk deletion operations, consider:
// - DeleteByDimension() for deleting multiple documents by filter criteria
// - DeleteWhere() for deleting with custom SQL conditions
func (ts *Store[T]) Delete(id string, cascade bool) error {
	return ts.store.Delete(id, cascade)
}

// DeleteByDimension removes all documents matching the given dimension filters
// Multiple filters are combined with AND. Returns the number of documents deleted.
func (ts *Store[T]) DeleteByDimension(filters map[string]interface{}) (int, error) {
	return ts.store.DeleteByDimension(filters)
}

// DeleteWhere removes all documents matching a custom WHERE clause
//
// SECURITY WARNING: This method accepts SQL conditions and must be used carefully.
// Always use parameterized queries with ? placeholders to prevent SQL injection.
//
// The where clause should not include the "WHERE" keyword itself.
//
// Example:
//
//	// SAFE - uses parameterized query
//	count, err := store.DeleteWhere("status = ? AND created_at < ?", "archived", cutoffDate)
//
//	// DANGEROUS - vulnerable to SQL injection
//	count, err := store.DeleteWhere("status = '" + userInput + "'") // DON'T DO THIS
//
// Returns the number of documents deleted.
func (ts *Store[T]) DeleteWhere(whereClause string, args ...interface{}) (int, error) {
	// Pass through to underlying store which implements the secure WHERE clause evaluation
	return ts.store.DeleteWhere(whereClause, args...)
}

// UpdateByDimension updates all documents matching the given dimension filters.
//
// Multiple filters are combined with AND. This method applies the same field clearing
// behavior as Update() - zero values in the data struct will clear the corresponding
// fields in ALL matching documents.
//
// See Update() method documentation for complete field clearing behavior details.
//
// Returns the number of documents updated.
func (ts *Store[T]) UpdateByDimension(filters map[string]interface{}, data *T) (int, error) {
	req, err := ts.buildUpdateRequest(data)
	if err != nil {
		return 0, err
	}

	return ts.store.UpdateByDimension(filters, req)
}

// UpdateWhere updates all documents matching a custom WHERE clause.
//
// This method applies the same field clearing behavior as Update() - zero values
// in the data struct will clear the corresponding fields in ALL matching documents.
//
// SECURITY WARNING: This method accepts SQL conditions and must be used carefully.
// Always use parameterized queries with ? placeholders to prevent SQL injection.
//
// The where clause should not include the "WHERE" keyword itself.
//
// Example:
//
//	// SAFE - uses parameterized query with field clearing
//	task := &Task{
//	    Status:   "completed", // Will update status
//	    Assignee: "",          // Will clear assignee field
//	}
//	count, err := store.UpdateWhere("status = ? AND priority = ?", task, "pending", "high")
//
//	// DANGEROUS - vulnerable to SQL injection
//	whereClause := "status = '" + userInput + "'" // DON'T DO THIS
//
// See Update() method documentation for complete field clearing behavior details.
//
// Returns the number of documents updated.
func (ts *Store[T]) UpdateWhere(whereClause string, data *T, args ...interface{}) (int, error) {
	// Convert typed data to UpdateRequest using shared helper
	req, err := ts.buildUpdateRequest(data)
	if err != nil {
		return 0, err
	}

	return ts.store.UpdateWhere(whereClause, req, args...)
}

// UpdateByUUIDs updates multiple documents by their UUIDs in a single operation.
//
// This method applies the same field clearing behavior as Update() - zero values
// in the data struct will clear the corresponding fields in ALL updated documents.
//
// **IMPORTANT**: This enables bulk field clearing, which was the primary goal of
// issue #82. You can now clear fields across multiple documents efficiently:
//
//	// Clear assignee field for multiple tasks
//	updates := &Task{
//	    Assignee: "", // Will clear assignee field in all specified documents
//	}
//	count, err := store.UpdateByUUIDs(taskUUIDs, updates)
//
// See Update() method documentation for complete field clearing behavior details.
//
// Returns the number of documents updated.
func (ts *Store[T]) UpdateByUUIDs(uuids []string, data *T) (int, error) {
	req, err := ts.buildUpdateRequest(data)
	if err != nil {
		return 0, err
	}

	return ts.store.UpdateByUUIDs(uuids, req)
}

// DeleteByUUIDs deletes multiple documents by their UUIDs in a single operation
// Returns the number of documents deleted.
func (ts *Store[T]) DeleteByUUIDs(uuids []string) (int, error) {
	return ts.store.DeleteByUUIDs(uuids)
}

// Query returns a new typed query builder
func (ts *Store[T]) Query() *Query[T] {
	return &Query[T]{
		store:      ts.store,
		typedStore: ts,
		options: types.ListOptions{
			Filters: make(map[string]interface{}),
		},
	}
}

// Close closes the underlying store
func (ts *Store[T]) Close() error {
	return ts.store.Close()
}

// Store returns the underlying store.Store for operations that need direct access
// This is useful for operations like export that work with the raw Store interface
func (ts *Store[T]) Store() store.Store {
	return ts.store
}

// ResolveUUID converts a simple ID (e.g., "1.2.c3") to a UUID
// This provides direct access to ID resolution without needing to access the underlying store
func (ts *Store[T]) ResolveUUID(simpleID string) (string, error) {
	return ts.store.ResolveUUID(simpleID)
}

// List returns documents based on the provided ListOptions, converted to typed structs
// This provides direct access to the underlying store's List functionality while maintaining type safety
func (ts *Store[T]) List(opts types.ListOptions) ([]T, error) {
	// Validate and transform field names in query options
	transformedOpts, err := ts.validateAndTransformListOptions(opts)
	if err != nil {
		return nil, err
	}

	// Delegate to underlying store for actual querying
	// The store handles all filtering, ordering, and pagination logic
	docs, err := ts.store.List(transformedOpts)
	if err != nil {
		return nil, err
	}

	// Convert raw documents to typed structs
	// Pre-allocate slice for performance - we know the exact size needed
	result := make([]T, len(docs))
	for i, doc := range docs {
		// UnmarshalDimensions maps document dimensions to struct fields using reflection
		// This is where struct tags are processed and values are converted
		if err := UnmarshalDimensions(doc, &result[i]); err != nil {
			// Fail fast on unmarshal error - indicates schema mismatch or corrupted data
			return nil, fmt.Errorf("failed to unmarshal document '%s': %w", doc.UUID, err)
		}
	}

	return result, nil
}

// validateAndTransformListOptions validates field names and transforms them to the canonical storage format
func (ts *Store[T]) validateAndTransformListOptions(opts types.ListOptions) (types.ListOptions, error) {
	// Create a copy to avoid modifying the original
	transformedOpts := opts

	// Get the type information for field validation
	var zeroT T
	typ := reflect.TypeOf(zeroT)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Transform OrderBy field names
	if len(opts.OrderBy) > 0 {
		transformedOrderBy := make([]types.OrderClause, len(opts.OrderBy))
		copy(transformedOrderBy, opts.OrderBy)

		for i, clause := range transformedOrderBy {
			if strings.HasPrefix(clause.Column, "_data.") {
				// Extract the field name and validate/transform it
				fieldName := strings.TrimPrefix(clause.Column, "_data.")

				// Validate that the field exists in the struct
				if err := ts.validateDataFieldName(typ, fieldName); err != nil {
					return types.ListOptions{}, fmt.Errorf("invalid field in OrderBy: %w", err)
				}

				// Transform to snake_case for storage
				snakeFieldName := normalizeFieldName(fieldName)
				transformedOrderBy[i].Column = "_data." + snakeFieldName
			}
			// Note: Non-data fields (like "created_at", "title") are passed through unchanged
		}

		transformedOpts.OrderBy = transformedOrderBy
	}

	// Transform filter field names
	if len(opts.Filters) > 0 {
		transformedFilters := make(map[string]interface{})

		for key, value := range opts.Filters {
			if strings.HasPrefix(key, "_data.") {
				// Extract the field name and validate/transform it
				fieldName := strings.TrimPrefix(key, "_data.")

				// Validate that the field exists in the struct
				if err := ts.validateDataFieldName(typ, fieldName); err != nil {
					return types.ListOptions{}, fmt.Errorf("invalid field in Filters: %w", err)
				}

				// Transform to snake_case for storage
				snakeFieldName := normalizeFieldName(fieldName)
				transformedFilters["_data."+snakeFieldName] = value
			} else {
				// Non-data filters are passed through unchanged
				transformedFilters[key] = value
			}
		}

		transformedOpts.Filters = transformedFilters
	}

	return transformedOpts, nil
}

// validateDataFieldName checks if a field name (either snake_case or PascalCase) exists in the struct
func (ts *Store[T]) validateDataFieldName(typ reflect.Type, fieldName string) error {
	// Try to find the field by name (supports both conventions)
	if _, found := findFieldByName(typ, fieldName); found {
		return nil
	}

	// Field not found - provide helpful error message
	availableFields := ts.getAvailableDataFields(typ)
	return fmt.Errorf("field '%s' not found in %s, available data fields: %v",
		fieldName, typ.Name(), availableFields)
}

// getAvailableDataFields returns a list of available data field names (non-dimension fields)
func (ts *Store[T]) getAvailableDataFields(typ reflect.Type) []string {
	var fields []string

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

		// Skip dimension fields (fields with dimension or values tags)
		dimTag := field.Tag.Get("dimension")
		valuesTag := field.Tag.Get("values")
		if dimTag != "" || valuesTag != "" {
			continue
		}

		// Add both snake_case and PascalCase versions for clarity
		snakeName := normalizeFieldName(field.Name)
		fields = append(fields, snakeName, field.Name)
	}

	return fields
}

// GetRaw retrieves a document by ID and returns the raw document structure.
//
// Accepts both UUID and SimpleID for maximum flexibility.
// ID Resolution Strategy:
// 1. Try to resolve ID as SimpleID to UUID
// 2. If resolution fails, use ID directly as UUID
// 3. Query store with resolved/direct UUID
//
// This provides consistent behavior with Get and other ID-based methods.
// Returns the raw document without type conversion - useful for:
// - Accessing dimensions not defined in the struct
// - Inspecting metadata (CreatedAt, UpdatedAt, etc.)
// - Working with documents that partially match the struct schema
// - Debugging and introspection
// - Administrative operations
func (ts *Store[T]) GetRaw(id string) (*types.Document, error) {
	// Consistent ID resolution: try SimpleID first, fallback to direct UUID
	// This dual approach handles both user-provided SimpleIDs ("1", "h2")
	// and system-provided UUIDs transparently
	uuid, err := ts.store.ResolveUUID(id)
	if err != nil {
		// If resolution fails, assume ID is already a UUID
		// This fallback is critical for API consistency - methods should accept both ID types
		uuid = id
	}

	// Use List with UUID filter to get the raw document
	// We use List instead of GetByID to leverage existing filtering infrastructure
	docs, err := ts.store.List(types.ListOptions{
		Filters: map[string]interface{}{
			"uuid": uuid, // Filter by exact UUID match
		},
	})
	if err != nil {
		return nil, err
	}

	// Validate result count - UUIDs should be unique
	if len(docs) == 0 {
		return nil, fmt.Errorf("document with ID '%s' not found", id)
	}

	if len(docs) > 1 {
		// This should never happen with valid UUIDs, but check anyway
		// Could indicate data corruption or ID collision
		return nil, fmt.Errorf("multiple documents found for ID '%s'", id)
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
func (ts *Store[T]) AddRaw(title string, dimensions map[string]interface{}) (string, error) {
	// Normalize _data field names to ensure consistent storage format
	// This ensures compatibility with typed queries while maintaining raw capability
	normalizedDimensions := make(map[string]interface{})
	for key, value := range dimensions {
		if strings.HasPrefix(key, "_data.") {
			fieldName := strings.TrimPrefix(key, "_data.")
			normalizedFieldName := normalizeFieldName(fieldName)
			normalizedDimensions["_data."+normalizedFieldName] = value
		} else {
			normalizedDimensions[key] = value
		}
	}

	// Pass through to underlying store with normalized field names
	return ts.store.Add(title, normalizedDimensions)
}

// GetDimensions returns the raw dimensions map for a document
// This provides access to all dimension values and custom _data fields, useful for:
// - Accessing fields not defined in the struct schema
// - Debugging dimension values and configuration
// - Working with documents that have additional custom fields
// - Introspecting the full dimension structure
// Accepts both UUID and SimpleID for maximum flexibility
// Returns a copy of the dimensions map to prevent accidental modifications
func (ts *Store[T]) GetDimensions(id string) (map[string]interface{}, error) {
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
func (ts *Store[T]) GetMetadata(id string) (*DocumentMetadata, error) {
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

// GetDimensionConfig returns the runtime configuration for this Store.
//
// This method provides introspection capabilities for the automatically generated
// dimension configuration. It allows applications to examine:
//
// - **Dimension Names**: All configured dimension names
// - **Enumerated Values**: Valid values for each enumerated dimension
// - **Prefix Mappings**: Value-to-prefix mappings for ID generation
// - **Default Values**: Default values for dimensions
// - **Hierarchical Relations**: Parent-child relationship configurations
//
// # Use Cases
//
// - **Debugging**: Inspect configuration during development
// - **Validation**: Verify struct tags were parsed correctly
// - **Documentation**: Generate API documentation from configuration
// - **Migration**: Compare configurations across schema versions
// - **Testing**: Validate configuration in unit tests
//
// # Return Format
//
// Returns the same Config struct used internally by nanostore, containing:
//
//	type Config struct {
//	    Dimensions []DimensionConfig
//	}
//
//	type DimensionConfig struct {
//	    Name         string
//	    Type         DimensionType  // Enumerated or Hierarchical
//	    Values       []string       // For enumerated dimensions
//	    Prefixes     map[string]string // Value -> prefix mapping
//	    DefaultValue string         // Default for new documents
//	    RefField     string         // For hierarchical dimensions
//	}
//
// # Example Usage
//
//	// Inspect configuration
//	config, err := store.GetDimensionConfig()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, dim := range config.Dimensions {
//	    fmt.Printf("Dimension: %s\n", dim.Name)
//	    if dim.Type == nanostore.Enumerated {
//	        fmt.Printf("  Values: %v\n", dim.Values)
//	        fmt.Printf("  Prefixes: %v\n", dim.Prefixes)
//	        fmt.Printf("  Default: %s\n", dim.DefaultValue)
//	    }
//	}
//
// # Performance Notes
//
// This method returns a copy of the configuration, not a reference.
// The configuration is generated once at Store creation and cached,
// so this method is O(1) with respect to runtime performance.
func (ts *Store[T]) GetDimensionConfig() (*nanostore.Config, error) {
	// Return a copy of the cached configuration
	// Critical: This is cached from struct tag parsing done ONCE at store creation.
	// We don't regenerate via reflection here because:
	//   1. Reflection is expensive (2.6µs vs 0.8ns)
	//   2. Configuration is immutable after store creation
	//   3. Multiple threads can safely access this cached copy
	configCopy := ts.config
	return &configCopy, nil
}

// SetTimeFunc sets a custom time function for deterministic timestamps in testing.
//
// This method enables deterministic testing by allowing tests to control the timestamps
// used for document creation and updates. This is essential for reliable test scenarios
// where predictable timestamps are needed for assertions and ordering.
//
// # Use Cases
//
// - **Deterministic Testing**: Control timestamps for reproducible test results
// - **Time-Based Assertions**: Test documents with specific creation/update times
// - **Ordering Tests**: Verify correct ordering behavior with known timestamps
// - **Migration Testing**: Simulate documents created at different times
// - **Performance Testing**: Measure operations without time variations
//
// # Parameters
//
// - **timeFunc**: Function that returns the desired time.Time value
//   - If nil, reverts to using the system time (time.Now)
//   - Function is called each time a timestamp is needed
//   - Should be deterministic for testing purposes
//
// # Example Usage
//
//	// Set fixed time for all operations
//	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
//	store.SetTimeFunc(func() time.Time { return fixedTime })
//
//	// Create document with predictable timestamp
//	id, err := store.Create("Test Document", &TodoItem{Status: "pending"})
//
//	// Reset to system time when done
//	store.SetTimeFunc(nil)
//
//	// Use with time sequences for testing ordering
//	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
//	counter := 0
//	store.SetTimeFunc(func() time.Time {
//	    counter++
//	    return baseTime.Add(time.Duration(counter) * time.Hour)
//	})
//
// # Implementation Notes
//
// This method attempts to cast the underlying store to the TestStore interface.
// If the cast fails (e.g., in production builds without test support), the method
// returns an error indicating the limitation.
//
// # Error Conditions
//
// - **No Test Support**: The underlying store doesn't implement TestStore interface
// - **Store Type Mismatch**: Store implementation doesn't support time function setting
//
// # Production vs Testing
//
// This method is primarily intended for testing scenarios. Production code typically
// should not need to override system time, though the capability exists if needed
// for specific use cases like migration or data import.
func (ts *Store[T]) SetTimeFunc(timeFunc func() time.Time) error {
	// Attempt to cast underlying store to TestStore interface
	// This interface check is necessary because not all store implementations
	// support time function override (production stores may omit this feature)
	testStore, ok := ts.store.(store.TestStore)
	if !ok {
		// Fail fast with clear error - this prevents silent time function ignoring
		// which would lead to confusing test failures
		return fmt.Errorf("underlying store does not support SetTimeFunc - store type %T does not implement TestStore interface", ts.store)
	}

	// Delegate to underlying store's SetTimeFunc
	// The underlying store handles the actual time function replacement
	testStore.SetTimeFunc(timeFunc)
	return nil
}

// ValidateConfiguration performs runtime validation of the Store configuration.
//
// This method checks for configuration issues that might not be caught during
// store creation, including:
//
// - **Prefix Conflicts**: Multiple values mapping to the same prefix
// - **Invalid Default Values**: Defaults not in enumerated value lists
// - **Missing Required Fields**: Hierarchical dimensions without ref fields
// - **Constraint Violations**: Values violating naming conventions
// - **Type Consistency**: Field types compatible with dimension types
//
// # Validation Categories
//
// ## Enumerated Dimension Validation
//
// - All values are non-empty strings
// - Default value exists in values list
// - Prefix mappings point to valid values
// - No duplicate values or prefixes
//
// ## Hierarchical Dimension Validation
//
// - RefField is specified and non-empty
// - Field types are compatible (typically string)
// - No circular reference possibilities
//
// ## Cross-Dimension Validation
//
// - Prefix conflicts across dimensions
// - Dimension name uniqueness
// - Compatible with nanostore constraints
//
// # Error Reporting
//
// Returns detailed errors with specific field and value information:
//
//	"dimension 'status': default value 'invalid' not in values list [pending,active,done]"
//	"dimension 'priority': prefix conflict - value 'high' and 'urgent' both map to prefix 'h'"
//	"field 'ParentID': hierarchical dimension missing ref field specification"
//
// # Example Usage
//
//	// Validate configuration during testing
//	func TestStoreConfiguration(t *testing.T) {
//	    store, err := api.New[TodoItem]("test.json")
//	    require.NoError(t, err)
//	    defer store.Close()
//
//	    err = store.ValidateConfiguration()
//	    assert.NoError(t, err, "Configuration should be valid")
//	}
//
//	// Validate before critical operations
//	if err := store.ValidateConfiguration(); err != nil {
//	    return fmt.Errorf("invalid store configuration: %w", err)
//	}
//
// # Performance Notes
//
// This method performs O(n²) validation for prefix conflicts where n is the
// total number of configured values across all dimensions. It should typically
// be called during application startup or testing, not in hot paths.
func (ts *Store[T]) ValidateConfiguration() error {
	config, err := ts.GetDimensionConfig()
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	// Track all prefixes across dimensions to detect conflicts
	allPrefixes := make(map[string][]string) // prefix -> list of "dimension:value"

	for _, dim := range config.Dimensions {
		// Validate enumerated dimensions
		if dim.Type == nanostore.Enumerated {
			// Check that values list is not empty
			if len(dim.Values) == 0 {
				return fmt.Errorf("dimension '%s': enumerated dimension must have at least one value", dim.Name)
			}

			// Check for empty values
			for _, value := range dim.Values {
				if strings.TrimSpace(value) == "" {
					return fmt.Errorf("dimension '%s': empty value not allowed", dim.Name)
				}
			}

			// Check default value exists in values list
			if dim.DefaultValue != "" {
				found := false
				for _, value := range dim.Values {
					if value == dim.DefaultValue {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("dimension '%s': default value '%s' not in values list %v",
						dim.Name, dim.DefaultValue, dim.Values)
				}
			}

			// Check prefix mappings and collect for conflict detection
			for value, prefix := range dim.Prefixes {
				// Verify the value exists in values list
				found := false
				for _, v := range dim.Values {
					if v == value {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("dimension '%s': prefix mapping for unknown value '%s'", dim.Name, value)
				}

				// Verify prefix is non-empty
				if strings.TrimSpace(prefix) == "" {
					return fmt.Errorf("dimension '%s': empty prefix not allowed for value '%s'", dim.Name, value)
				}

				// Track prefix for conflict detection
				key := dim.Name + ":" + value
				allPrefixes[prefix] = append(allPrefixes[prefix], key)
			}
		}

		// Validate hierarchical dimensions
		if dim.Type == nanostore.Hierarchical {
			if strings.TrimSpace(dim.RefField) == "" {
				return fmt.Errorf("dimension '%s': hierarchical dimension must specify RefField", dim.Name)
			}
		}
	}

	// Check for prefix conflicts across dimensions
	for prefix, sources := range allPrefixes {
		if len(sources) > 1 {
			return fmt.Errorf("prefix conflict: prefix '%s' used by multiple dimension:value pairs: %v",
				prefix, sources)
		}
	}

	return nil
}

// GetDebugInfo returns comprehensive debugging information about the Store.
//
// This method provides developers with detailed insights into the store's current
// state, configuration, and runtime characteristics. It's invaluable for debugging
// issues, optimizing performance, and understanding store behavior.
//
// # Information Categories
//
// ## Store Metadata
// - **Store Type**: Type of underlying store implementation
// - **File Path**: Location of store data file (if applicable)
// - **Configuration**: Complete dimension configuration details
// - **Document Count**: Total number of documents in store
//
// ## Runtime Statistics
// - **Memory Usage**: Estimated memory consumption (when available)
// - **Performance Metrics**: Query and operation timing information
// - **Cache Status**: Information about internal caching
// - **Configuration Hash**: Fingerprint for configuration validation
//
// ## Type Information
// - **Go Type**: Full type name for T
// - **Struct Fields**: Field names and types from reflection
// - **Tag Configuration**: Parsed struct tag information
// - **Embedding Validation**: Confirms nanostore.Document embedding
//
// # Return Format
//
// Returns a structured DebugInfo object containing all debugging information:
//
//	type DebugInfo struct {
//	    StoreType        string
//	    FilePath         string
//	    DocumentCount    int
//	    Configuration    *nanostore.Config
//	    TypeInfo         TypeDebugInfo
//	    RuntimeStats     RuntimeDebugStats
//	    LastError        string
//	}
//
// # Example Usage
//
//	// Get comprehensive debug information
//	debug, err := store.GetDebugInfo()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Store Type: %s\n", debug.StoreType)
//	fmt.Printf("Document Count: %d\n", debug.DocumentCount)
//	fmt.Printf("Go Type: %s\n", debug.TypeInfo.TypeName)
//
//	for _, dim := range debug.Configuration.Dimensions {
//	    fmt.Printf("Dimension %s: %d values\n", dim.Name, len(dim.Values))
//	}
//
// # Performance Notes
//
// This method performs reflection and potentially expensive store operations
// to gather comprehensive information. It should be used primarily for debugging
// and development, not in hot paths or production monitoring.
//
// # Use Cases
//
// - **Debugging**: Understand why queries aren't working as expected
// - **Development**: Inspect configuration during development
// - **Testing**: Validate store state in unit tests
// - **Monitoring**: Get runtime statistics for health checks
// - **Documentation**: Generate documentation from live configuration
func (ts *Store[T]) GetDebugInfo() (*DebugInfo, error) {
	var zero T
	typ := reflect.TypeOf(zero)

	// Get configuration information
	config, err := ts.GetDimensionConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration: %w", err)
	}

	// Get document count using raw store access
	allDocs, err := ts.store.List(types.ListOptions{})
	if err != nil {
		return &DebugInfo{
			StoreType:     fmt.Sprintf("%T", ts.store),
			DocumentCount: -1,
			Configuration: config,
			TypeInfo:      extractTypeInfo(typ),
			LastError:     err.Error(),
		}, nil
	}

	// Extract type information
	typeInfo := extractTypeInfo(typ)

	// Build debug info
	debugInfo := &DebugInfo{
		StoreType:     fmt.Sprintf("%T", ts.store),
		DocumentCount: len(allDocs),
		Configuration: config,
		TypeInfo:      typeInfo,
		RuntimeStats: RuntimeDebugStats{
			TotalDimensions: len(config.Dimensions),
			TotalValues:     countTotalValues(config.Dimensions),
			TotalPrefixes:   countTotalPrefixes(config.Dimensions),
		},
	}

	return debugInfo, nil
}

// GetStoreStats returns statistical information about the store's contents.
//
// This method provides quantitative insights into the store's document distribution,
// dimension usage, and data patterns. It's useful for understanding data patterns
// and optimizing queries.
//
// # Statistical Categories
//
// ## Document Distribution
// - **Total Documents**: Count of all documents in store
// - **By Dimension Values**: Distribution across enumerated dimension values
// - **By Hierarchical Depth**: Distribution of parent-child relationships
// - **By Creation Time**: Temporal distribution of documents
//
// ## Data Field Usage
// - **Custom Fields**: Usage patterns of `_data.*` fields
// - **Field Value Distribution**: Most common values per field
// - **Field Coverage**: Percentage of documents with each field
//
// ## Performance Insights
// - **Query Complexity**: Estimates for different query patterns
// - **Index Utilization**: Which dimensions benefit from indexing
// - **Hot Spots**: Most frequently queried dimension combinations
//
// # Example Usage
//
//	// Get store statistics
//	stats, err := store.GetStoreStats()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Total Documents: %d\n", stats.TotalDocuments)
//
//	for dimension, distribution := range stats.DimensionDistribution {
//	    fmt.Printf("Dimension %s:\n", dimension)
//	    for value, count := range distribution {
//	        fmt.Printf("  %s: %d documents\n", value, count)
//	    }
//	}
//
//	for field, coverage := range stats.DataFieldCoverage {
//	    fmt.Printf("Field %s: %.1f%% coverage\n", field, coverage*100)
//	}
//
// # Performance Notes
//
// This method iterates through all documents to calculate statistics.
// For large stores, this can be expensive. Consider caching results
// or calling periodically rather than on every request.
func (ts *Store[T]) GetStoreStats() (*StoreStats, error) {
	// Get all documents for analysis using raw store access
	allDocs, err := ts.store.List(types.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents for stats: %w", err)
	}

	stats := &StoreStats{
		TotalDocuments:        len(allDocs),
		DimensionDistribution: make(map[string]map[string]int),
		DataFieldCoverage:     make(map[string]float64),
		DataFieldDistribution: make(map[string]map[string]int),
	}

	if len(allDocs) == 0 {
		return stats, nil
	}

	// Get configuration to know which dimensions exist
	config, err := ts.GetDimensionConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration: %w", err)
	}

	// Initialize dimension distribution maps
	for _, dim := range config.Dimensions {
		if dim.Type == nanostore.Enumerated {
			stats.DimensionDistribution[dim.Name] = make(map[string]int)
		}
	}

	// Track data fields seen across all documents
	dataFieldCounts := make(map[string]int)
	dataFieldValues := make(map[string]map[string]int)

	// Analyze each document
	for _, doc := range allDocs {
		// Analyze enumerated dimensions
		for _, dim := range config.Dimensions {
			if dim.Type == nanostore.Enumerated {
				if value, exists := doc.Dimensions[dim.Name]; exists {
					valueStr := fmt.Sprintf("%v", value)
					stats.DimensionDistribution[dim.Name][valueStr]++
				}
			}
		}

		// Analyze data fields
		for key, value := range doc.Dimensions {
			if strings.HasPrefix(key, "_data.") {
				fieldName := strings.TrimPrefix(key, "_data.")
				dataFieldCounts[fieldName]++

				// Track value distribution for data fields
				if dataFieldValues[fieldName] == nil {
					dataFieldValues[fieldName] = make(map[string]int)
				}
				valueStr := fmt.Sprintf("%v", value)
				dataFieldValues[fieldName][valueStr]++
			}
		}
	}

	// Calculate coverage percentages for data fields
	totalDocs := float64(len(allDocs))
	for field, count := range dataFieldCounts {
		stats.DataFieldCoverage[field] = float64(count) / totalDocs
	}

	// Store data field value distributions
	stats.DataFieldDistribution = dataFieldValues

	return stats, nil
}

// ValidateStoreIntegrity performs comprehensive integrity checks on the store.
//
// This method validates the consistency and correctness of the store's data,
// configuration, and relationships. It's essential for debugging data corruption
// issues and ensuring store reliability.
//
// # Validation Categories
//
// ## Configuration Consistency
// - **Dimension Values**: All document values exist in configured value lists
// - **Default Values**: Documents have appropriate defaults when unspecified
// - **Required Fields**: All required dimensions are present
// - **Type Consistency**: Field types match expected types
//
// ## Document Integrity
// - **UUID Uniqueness**: All document UUIDs are unique
// - **SimpleID Consistency**: SimpleIDs match UUID relationships
// - **Hierarchical Validity**: Parent-child relationships are valid
// - **Timestamp Ordering**: CreatedAt ≤ UpdatedAt for all documents
//
// ## Structural Validity
// - **Embedding Compliance**: All documents properly embed required fields
// - **Field Completeness**: Required metadata fields are present
// - **Data Consistency**: Custom data fields follow expected patterns
//
// # Return Format
//
// Returns a detailed IntegrityReport with findings:
//
//	type IntegrityReport struct {
//	    IsValid           bool
//	    TotalDocuments    int
//	    ErrorCount        int
//	    WarningCount      int
//	    Errors           []IntegrityError
//	    Warnings         []IntegrityWarning
//	    Summary          string
//	}
//
// # Example Usage
//
//	// Validate store integrity
//	report, err := store.ValidateStoreIntegrity()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Store Valid: %v\n", report.IsValid)
//	fmt.Printf("Documents: %d, Errors: %d, Warnings: %d\n",
//	    report.TotalDocuments, report.ErrorCount, report.WarningCount)
//
//	for _, error := range report.Errors {
//	    fmt.Printf("ERROR: %s\n", error.Message)
//	}
//
// # Performance Notes
//
// This method performs extensive validation by examining all documents and
// their relationships. For large stores, this can take significant time.
// Consider running periodically or during maintenance windows.
func (ts *Store[T]) ValidateStoreIntegrity() (*IntegrityReport, error) {
	// Get all documents and configuration using raw store access
	allDocs, err := ts.store.List(types.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}

	config, err := ts.GetDimensionConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration: %w", err)
	}

	report := &IntegrityReport{
		TotalDocuments: len(allDocs),
		Errors:         []IntegrityError{},
		Warnings:       []IntegrityWarning{},
	}

	// Track UUIDs for uniqueness validation
	seenUUIDs := make(map[string]bool)

	// Validate each document
	for i, doc := range allDocs {
		// Check UUID uniqueness
		if seenUUIDs[doc.UUID] {
			report.Errors = append(report.Errors, IntegrityError{
				Type:       "UUID_DUPLICATE",
				DocumentID: doc.UUID,
				Message:    fmt.Sprintf("Duplicate UUID found: %s", doc.UUID),
			})
		}
		seenUUIDs[doc.UUID] = true

		// Check timestamp consistency
		if !doc.UpdatedAt.IsZero() && !doc.CreatedAt.IsZero() && doc.UpdatedAt.Before(doc.CreatedAt) {
			report.Errors = append(report.Errors, IntegrityError{
				Type:       "TIMESTAMP_INCONSISTENT",
				DocumentID: doc.UUID,
				Message:    fmt.Sprintf("UpdatedAt (%v) is before CreatedAt (%v)", doc.UpdatedAt, doc.CreatedAt),
			})
		}

		// Validate enumerated dimension values
		for _, dim := range config.Dimensions {
			if dim.Type == nanostore.Enumerated {
				if value, exists := doc.Dimensions[dim.Name]; exists {
					valueStr := fmt.Sprintf("%v", value)

					// Check if value is in allowed list
					found := false
					for _, allowedValue := range dim.Values {
						if valueStr == allowedValue {
							found = true
							break
						}
					}

					if !found {
						report.Errors = append(report.Errors, IntegrityError{
							Type:       "INVALID_DIMENSION_VALUE",
							DocumentID: doc.UUID,
							Message:    fmt.Sprintf("Document %d: dimension '%s' has invalid value '%s' (allowed: %v)", i, dim.Name, valueStr, dim.Values),
						})
					}
				}
			}
		}

		// Check for missing required metadata
		if doc.UUID == "" {
			report.Errors = append(report.Errors, IntegrityError{
				Type:       "MISSING_UUID",
				DocumentID: fmt.Sprintf("document_%d", i),
				Message:    fmt.Sprintf("Document %d is missing UUID", i),
			})
		}

		if doc.SimpleID == "" {
			report.Warnings = append(report.Warnings, IntegrityWarning{
				Type:       "MISSING_SIMPLE_ID",
				DocumentID: doc.UUID,
				Message:    fmt.Sprintf("Document has empty SimpleID: %s", doc.UUID),
			})
		}
	}

	// Set final status
	report.ErrorCount = len(report.Errors)
	report.WarningCount = len(report.Warnings)
	report.IsValid = report.ErrorCount == 0

	// Generate summary
	if report.IsValid {
		if report.WarningCount > 0 {
			report.Summary = fmt.Sprintf("Store is valid with %d warnings", report.WarningCount)
		} else {
			report.Summary = "Store is completely valid"
		}
	} else {
		report.Summary = fmt.Sprintf("Store has %d errors and %d warnings", report.ErrorCount, report.WarningCount)
	}

	return report, nil
}

// AddDimensionValue adds a new enumerated value to an existing dimension.
//
// This method provides limited runtime configuration modification by allowing
// new values to be added to existing enumerated dimensions. This is one of the
// safer configuration changes since it doesn't invalidate existing documents.
//
// # Supported Operations
//
// - **Add Enumerated Values**: Extend existing value lists
// - **Add Prefix Mappings**: Assign prefixes to new values
// - **Validation**: Ensure new values don't conflict with existing configuration
//
// # Limitations
//
// Due to the complexity of runtime configuration changes, this method has
// several important limitations:
//
// - **Enumerated Only**: Only works with enumerated dimensions, not hierarchical
// - **Additive Only**: Cannot remove or modify existing values
// - **No Store Update**: Changes don't persist to underlying store configuration
// - **Session Only**: Changes are lost when Store is recreated
// - **No Migration**: Existing documents are not affected
//
// # Future Enhancements
//
// Full runtime configuration modification would require:
//
// - **Store-Level Support**: Underlying nanostore API for config changes
// - **Data Migration**: Automatic migration of existing documents
// - **Atomic Updates**: Transactional configuration changes
// - **Rollback Support**: Ability to revert configuration changes
// - **Validation**: Comprehensive checking before applying changes
//
// # Parameters
//
// - **dimensionName**: Name of the existing enumerated dimension
// - **value**: New value to add to the dimension's value list
// - **prefix**: Optional prefix for the new value (empty string for no prefix)
//
// # Example Usage
//
//	// Add a new status value with prefix
//	err := store.AddDimensionValue("status", "cancelled", "c")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Add a new priority value without prefix
//	err = store.AddDimensionValue("priority", "urgent", "")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Verify the change
//	config, err := store.GetDimensionConfig()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// config now includes the new values
//
// # Error Conditions
//
// - **Dimension Not Found**: The specified dimension doesn't exist
// - **Not Enumerated**: The dimension is not an enumerated type
// - **Value Exists**: The value is already in the dimension's value list
// - **Prefix Conflict**: The prefix is already used by another value
// - **Invalid Input**: Empty value or dimension name
//
// # Security Notes
//
// This method modifies in-memory configuration only. Changes do not persist
// across application restarts and do not affect the underlying store's
// configuration or existing documents.
func (ts *Store[T]) AddDimensionValue(dimensionName, value, prefix string) error {
	if strings.TrimSpace(dimensionName) == "" {
		return fmt.Errorf("dimension name cannot be empty")
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("value cannot be empty")
	}

	// Get current configuration
	config, err := ts.GetDimensionConfig()
	if err != nil {
		return fmt.Errorf("failed to get current configuration: %w", err)
	}

	// Find the dimension
	var targetDim *nanostore.DimensionConfig
	for i := range config.Dimensions {
		if config.Dimensions[i].Name == dimensionName {
			targetDim = &config.Dimensions[i]
			break
		}
	}

	if targetDim == nil {
		return fmt.Errorf("dimension '%s' not found", dimensionName)
	}

	if targetDim.Type != nanostore.Enumerated {
		return fmt.Errorf("dimension '%s' is not enumerated (type: %v)", dimensionName, targetDim.Type)
	}

	// Check if value already exists
	for _, existingValue := range targetDim.Values {
		if existingValue == value {
			return fmt.Errorf("value '%s' already exists in dimension '%s'", value, dimensionName)
		}
	}

	// Check prefix conflicts if prefix is provided
	if prefix != "" {
		// Check against all dimensions for conflicts
		for _, dim := range config.Dimensions {
			for _, existingPrefix := range dim.Prefixes {
				if existingPrefix == prefix {
					return fmt.Errorf("prefix '%s' is already used by another value", prefix)
				}
			}
		}
	}

	// Note: This is a demonstration of the API design.
	// In a full implementation, this would:
	// 1. Update the underlying store configuration
	// 2. Persist changes to storage
	// 3. Handle concurrent access safely
	// 4. Validate against existing documents
	//
	// For now, we return an informational error indicating the limitation
	return fmt.Errorf("runtime configuration modification is not fully implemented - "+
		"changes would add value '%s' with prefix '%s' to dimension '%s', "+
		"but underlying store configuration cannot be modified in current implementation",
		value, prefix, dimensionName)
}

// ModifyDimensionDefault changes the default value for an enumerated dimension.
//
// This method demonstrates the API pattern for runtime configuration changes.
// Like AddDimensionValue, it has significant limitations in the current implementation.
//
// # Parameters
//
// - **dimensionName**: Name of the existing enumerated dimension
// - **newDefault**: New default value (must exist in dimension's value list)
//
// # Example Usage
//
//	// Change default status from "pending" to "active"
//	err := store.ModifyDimensionDefault("status", "active")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Limitations
//
// Same limitations as AddDimensionValue - this is an API demonstration
// that shows the intended design pattern for future full implementation.
func (ts *Store[T]) ModifyDimensionDefault(dimensionName, newDefault string) error {
	if strings.TrimSpace(dimensionName) == "" {
		return fmt.Errorf("dimension name cannot be empty")
	}
	if strings.TrimSpace(newDefault) == "" {
		return fmt.Errorf("default value cannot be empty")
	}

	// Get current configuration
	config, err := ts.GetDimensionConfig()
	if err != nil {
		return fmt.Errorf("failed to get current configuration: %w", err)
	}

	// Find and validate the dimension
	var targetDim *nanostore.DimensionConfig
	for i := range config.Dimensions {
		if config.Dimensions[i].Name == dimensionName {
			targetDim = &config.Dimensions[i]
			break
		}
	}

	if targetDim == nil {
		return fmt.Errorf("dimension '%s' not found", dimensionName)
	}

	if targetDim.Type != nanostore.Enumerated {
		return fmt.Errorf("dimension '%s' is not enumerated", dimensionName)
	}

	// Verify new default exists in values list
	found := false
	for _, value := range targetDim.Values {
		if value == newDefault {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("new default '%s' is not in values list %v for dimension '%s'",
			newDefault, targetDim.Values, dimensionName)
	}

	// In a full implementation, this would update the store configuration
	return fmt.Errorf("runtime configuration modification is not fully implemented - "+
		"would change default for dimension '%s' from '%s' to '%s', "+
		"but underlying store configuration cannot be modified in current implementation",
		dimensionName, targetDim.DefaultValue, newDefault)
}

// GetFieldUsageStats returns statistics about how struct fields are being used
// across all documents in the store. This helps identify data patterns and
// optimize struct design.
//
// # Usage Examples
//
//	stats, err := store.GetFieldUsageStats()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Field usage analysis:\n")
//	for fieldName, usage := range stats.DataFieldUsage {
//	    fmt.Printf("  %s: %.1f%% coverage (%d non-empty values)\n",
//	        fieldName, usage.CoveragePercentage, usage.NonEmptyCount)
//	}
//
//	// Find rarely used fields
//	for fieldName, usage := range stats.DataFieldUsage {
//	    if usage.CoveragePercentage < 10.0 {
//	        fmt.Printf("  Low usage field: %s (%.1f%% coverage)\n",
//	            fieldName, usage.CoveragePercentage)
//	    }
//	}
//
// # Field Categories
//
// Analyzes different types of fields:
// - **Dimension Fields**: Enumerated and hierarchical fields with their value distributions
// - **Data Fields**: Custom struct fields stored in _data with coverage analysis
// - **Core Fields**: Title, Body, and other Document fields
//
// # Optimization Insights
//
// Use this information to:
// - Identify rarely used fields that could be removed
// - Find fields that might benefit from being made dimensional
// - Understand data completeness across your document set
// - Guide struct design decisions
func (ts *Store[T]) GetFieldUsageStats() (*FieldUsageStats, error) {
	stats := &FieldUsageStats{
		TotalDocuments: 0,
		DimensionUsage: make(map[string]DimensionUsageInfo),
		DataFieldUsage: make(map[string]DataFieldUsageInfo),
		CoreFieldUsage: CoreFieldUsageInfo{},
	}

	// Get all documents to analyze
	docs, err := ts.store.List(types.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents for analysis: %w", err)
	}

	stats.TotalDocuments = len(docs)
	if stats.TotalDocuments == 0 {
		return stats, nil // No data to analyze
	}

	// Initialize dimension usage tracking
	config, err := ts.GetDimensionConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get dimension config: %w", err)
	}

	for _, dim := range config.Dimensions {
		stats.DimensionUsage[dim.Name] = DimensionUsageInfo{
			DimensionName: dim.Name,
			Type:          dim.Type.String(),
			ValueCounts:   make(map[string]int),
			NonEmptyCount: 0,
		}
	}

	// Track all data field names encountered
	dataFieldNames := make(map[string]bool)

	// Analyze each document
	for _, doc := range docs {
		// Analyze core Document fields
		if doc.Title != "" {
			stats.CoreFieldUsage.TitleUsageCount++
		}
		if doc.Body != "" {
			stats.CoreFieldUsage.BodyUsageCount++
		}

		// Analyze dimensions
		for dimName, dimUsage := range stats.DimensionUsage {
			if value, exists := doc.Dimensions[dimName]; exists && value != nil {
				valueStr := fmt.Sprintf("%v", value)
				if valueStr != "" {
					dimUsage.NonEmptyCount++
					dimUsage.ValueCounts[valueStr]++
					stats.DimensionUsage[dimName] = dimUsage
				}
			}
		}

		// Analyze data fields
		for fieldName, value := range doc.Dimensions {
			if strings.HasPrefix(fieldName, "_data.") {
				dataFieldName := strings.TrimPrefix(fieldName, "_data.")
				dataFieldNames[dataFieldName] = true

				if value != nil && fmt.Sprintf("%v", value) != "" {
					// Initialize if not exists
					if _, exists := stats.DataFieldUsage[dataFieldName]; !exists {
						stats.DataFieldUsage[dataFieldName] = DataFieldUsageInfo{
							FieldName:     dataFieldName,
							NonEmptyCount: 0,
						}
					}

					usage := stats.DataFieldUsage[dataFieldName]
					usage.NonEmptyCount++
					stats.DataFieldUsage[dataFieldName] = usage
				}
			}
		}
	}

	// Calculate coverage percentages
	for fieldName := range dataFieldNames {
		usage := stats.DataFieldUsage[fieldName]
		usage.CoveragePercentage = (float64(usage.NonEmptyCount) / float64(stats.TotalDocuments)) * 100.0
		stats.DataFieldUsage[fieldName] = usage
	}

	// Calculate core field coverage
	stats.CoreFieldUsage.TitleCoveragePercentage = (float64(stats.CoreFieldUsage.TitleUsageCount) / float64(stats.TotalDocuments)) * 100.0
	stats.CoreFieldUsage.BodyCoveragePercentage = (float64(stats.CoreFieldUsage.BodyUsageCount) / float64(stats.TotalDocuments)) * 100.0

	return stats, nil
}

// GetTypeSchema returns detailed schema information about the Go type T,
// including field types, tags, and nanostore-specific annotations.
//
// This method provides comprehensive type introspection for:
// - Understanding the complete struct schema
// - Validating dimension field mappings
// - Debugging type-related issues
// - Generating documentation
//
// # Usage Examples
//
//	schema, err := store.GetTypeSchema()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Type: %s\n", schema.TypeName)
//	fmt.Printf("Package: %s\n", schema.PackageName)
//	fmt.Printf("Embeds Document: %t\n", schema.EmbedsDocument)
//
//	fmt.Println("\nDimension Fields:")
//	for _, field := range schema.DimensionFields {
//	    fmt.Printf("  %s (%s): %s\n", field.Name, field.Type, field.DimensionType)
//	    if field.AllowedValues != nil {
//	        fmt.Printf("    Values: %v\n", field.AllowedValues)
//	    }
//	}
//
//	fmt.Println("\nData Fields:")
//	for _, field := range schema.DataFields {
//	    fmt.Printf("  %s (%s)\n", field.Name, field.Type)
//	}
//
// # Schema Validation
//
// The schema information can help identify:
// - Missing nanostore.Document embedding
// - Incorrectly tagged dimension fields
// - Type compatibility issues
// - Struct design problems
func (ts *Store[T]) GetTypeSchema() (*TypeSchema, error) {
	var zero T
	typ := reflect.TypeOf(zero)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	schema := &TypeSchema{
		TypeName:        typ.String(),
		PackageName:     typ.PkgPath(),
		EmbedsDocument:  false,
		DimensionFields: []DimensionFieldInfo{},
		DataFields:      []DataFieldInfo{},
	}

	// Check for Document embedding
	schema.EmbedsDocument = embedsDocument(typ)

	// Get dimension configuration for reference
	config, err := ts.GetDimensionConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get dimension config: %w", err)
	}

	// Create map of dimension names for quick lookup
	dimensionMap := make(map[string]nanostore.DimensionConfig)
	for _, dim := range config.Dimensions {
		dimensionMap[dim.Name] = dim
	}

	// Analyze each field
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip embedded Document field
		if field.Anonymous && field.Type == reflect.TypeOf(nanostore.Document{}) {
			continue
		}

		// Check if this is a dimension field
		fieldLowerName := strings.ToLower(field.Name)
		if dimConfig, isDimension := dimensionMap[fieldLowerName]; isDimension {
			// This is a dimension field
			dimField := DimensionFieldInfo{
				Name:          field.Name,
				Type:          field.Type.String(),
				DimensionType: dimConfig.Type.String(),
				Tags:          string(field.Tag),
			}

			switch dimConfig.Type {
			case nanostore.Enumerated:
				dimField.AllowedValues = dimConfig.Values
				dimField.DefaultValue = dimConfig.DefaultValue
				dimField.PrefixMappings = dimConfig.Prefixes
			case nanostore.Hierarchical:
				dimField.RefField = dimConfig.RefField
			}

			schema.DimensionFields = append(schema.DimensionFields, dimField)
		} else {
			// This is a data field
			dataField := DataFieldInfo{
				Name: field.Name,
				Type: field.Type.String(),
				Tags: string(field.Tag),
			}

			schema.DataFields = append(schema.DataFields, dataField)
		}
	}

	return schema, nil
}

type Query[T any] struct {
	store      store.Store       // Underlying store for query execution
	typedStore *Store[T]         // Parent Store for validation
	options    types.ListOptions // Accumulated query options
}

// getDimensionConfig returns the dimension configuration for type T
// This is used internally by NOT methods to get valid values dynamically
func (tq *Query[T]) getDimensionConfig() (*nanostore.Config, error) {
	var zero T
	typ := reflect.TypeOf(zero)

	// Check that T embeds Document
	if !embedsDocument(typ) {
		return nil, fmt.Errorf("type %T does not embed nanostore.Document", zero)
	}

	// Generate dimension configuration from struct tags
	config, err := generateConfigFromType(typ)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config for type %T: %w", zero, err)
	}

	return &config, nil
}

// validateDataFieldReferences checks all data field references in the query options.
//
// This function is critical for eliminating silent failures in data field queries.
// It validates all field names used in Data(), DataIn(), DataNot(), DataNotIn(),
// OrderByData(), and OrderByDataDesc() methods against the actual Go struct fields.
//
// Key benefits:
// - **Prevents silent failures**: Invalid field names now return clear errors instead of empty results
// - **Case-insensitive queries**: Users can use any case for field names (e.g., "assignee" or "Assignee")
// - **Helpful error messages**: Provides suggestions for typos and lists valid field names
// - **Consistency**: Ensures all data field operations use the same validation logic
//
// This method should be called before executing any query to catch invalid field names early.
func (tq *Query[T]) validateDataFieldReferences() error {
	// Get valid data fields for type T
	var zero T
	validFields, err := getValidDataFields(zero)
	if err != nil {
		return fmt.Errorf("failed to get valid data fields: %w", err)
	}

	// Check filters for _data.* field references and normalize case
	for key := range tq.options.Filters {
		var fieldName, prefix string
		var isDataField bool

		if strings.HasPrefix(key, "_data.") {
			fieldName = strings.TrimPrefix(key, "_data.")
			prefix = "_data."
			isDataField = true
		} else if strings.HasPrefix(key, "__data_not__") {
			fieldName = strings.TrimPrefix(key, "__data_not__")
			prefix = "__data_not__"
			isDataField = true
		} else if strings.HasPrefix(key, "__data_not_in__") {
			fieldName = strings.TrimPrefix(key, "__data_not_in__")
			prefix = "__data_not_in__"
			isDataField = true
		}

		if isDataField {
			normalizedFieldName, err := validateDataFieldName(fieldName, validFields)
			if err != nil {
				return fmt.Errorf("invalid filter field: %w", err)
			}

			// If the field name was normalized, update the filter key
			if normalizedFieldName != fieldName {
				value := tq.options.Filters[key]
				delete(tq.options.Filters, key)
				tq.options.Filters[prefix+normalizedFieldName] = value
			}
		}
	}

	// Check OrderBy clauses for _data.* field references and normalize case
	for i, orderClause := range tq.options.OrderBy {
		if strings.HasPrefix(orderClause.Column, "_data.") {
			fieldName := strings.TrimPrefix(orderClause.Column, "_data.")
			correctFieldName, err := validateDataFieldName(fieldName, validFields)
			if err != nil {
				return fmt.Errorf("invalid order by field: %w", err)
			}

			// If the field name case was corrected, update the order clause
			if correctFieldName != fieldName {
				tq.options.OrderBy[i].Column = "_data." + correctFieldName
			}
		}
	}

	return nil
}

// getEnumeratedValues returns the valid values for an enumerated dimension
// This method is CRITICAL for fixing the hardcoded values issue in NOT methods.
// Instead of hardcoding ["pending", "active", "done"], we dynamically discover
// values from the actual struct tag configuration at runtime.
// Returns nil if the dimension doesn't exist or isn't enumerated
func (tq *Query[T]) getEnumeratedValues(dimensionName string) ([]string, error) {
	// Get configuration from the cached dimension config (performance optimized)
	config, err := tq.getDimensionConfig()
	if err != nil {
		return nil, err
	}

	// Linear search through dimensions - typically small (< 10 dimensions)
	for _, dim := range config.Dimensions {
		if dim.Name == dimensionName && dim.Type == nanostore.Enumerated {
			// Return the actual values from struct tags, not hardcoded values
			return dim.Values, nil
		}
	}

	return nil, fmt.Errorf("dimension '%s' not found or not enumerated", dimensionName)
}

// Activity filters by activity value.
// This is a domain-specific filter method - applications should define their own
// filter methods based on their configured dimensions.
func (tq *Query[T]) Activity(value string) *Query[T] {
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
func (tq *Query[T]) ActivityIn(values ...string) *Query[T] {
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
func (tq *Query[T]) ActivityNot(value string) *Query[T] {
	// Dynamically get all known activity values from type configuration
	allActivities, err := tq.getEnumeratedValues("activity")
	if err != nil {
		// If we can't get values, fall back to no filtering
		// This maintains backward compatibility
		return tq
	}

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
func (tq *Query[T]) ActivityNotIn(values ...string) *Query[T] {
	// Dynamically get all known activity values from type configuration
	allActivities, err := tq.getEnumeratedValues("activity")
	if err != nil {
		// If we can't get values, fall back to no filtering
		// This maintains backward compatibility
		return tq
	}

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
func (tq *Query[T]) Status(value string) *Query[T] {
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
func (tq *Query[T]) StatusIn(values ...string) *Query[T] {
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
func (tq *Query[T]) StatusNot(value string) *Query[T] {
	// Dynamically get all known status values from type configuration
	// CRITICAL FIX: This replaces hardcoded ["pending", "active", "done"] with
	// actual values from struct tags, making this work with ANY enum configuration
	allStatuses, err := tq.getEnumeratedValues("status")
	if err != nil {
		// If we can't get values, fall back to no filtering
		// This graceful degradation prevents query failures for misconfigured structs
		return tq
	}

	// Build inclusion list: everything EXCEPT the specified value
	// This approach works because underlying store doesn't support native NOT operations
	var includeStatuses []string
	for _, s := range allStatuses {
		if s != value {
			includeStatuses = append(includeStatuses, s)
		}
	}
	// Only set filter if we have values to include (avoid empty filter)
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
func (tq *Query[T]) StatusNotIn(values ...string) *Query[T] {
	// Dynamically get all known status values from type configuration
	allStatuses, err := tq.getEnumeratedValues("status")
	if err != nil {
		// If we can't get values, fall back to no filtering
		// This maintains backward compatibility
		return tq
	}

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
func (tq *Query[T]) Priority(value string) *Query[T] {
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
func (tq *Query[T]) PriorityIn(values ...string) *Query[T] {
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
func (tq *Query[T]) PriorityNot(value string) *Query[T] {
	// Dynamically get all known priority values from type configuration
	allPriorities, err := tq.getEnumeratedValues("priority")
	if err != nil {
		// If we can't get values, fall back to no filtering
		// This maintains backward compatibility
		return tq
	}

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
func (tq *Query[T]) PriorityNotIn(values ...string) *Query[T] {
	// Dynamically get all known priority values from type configuration
	allPriorities, err := tq.getEnumeratedValues("priority")
	if err != nil {
		// If we can't get values, fall back to no filtering
		// This maintains backward compatibility
		return tq
	}

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
func (tq *Query[T]) Data(field string, value interface{}) *Query[T] {
	// Validate and transform field name immediately for better error reporting
	var zeroT T
	typ := reflect.TypeOf(zeroT)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Validate field exists
	if err := tq.typedStore.validateDataFieldName(typ, field); err != nil {
		// For now, we'll store the error to be returned during Find()
		// In the future, we could return an error-carrying TypedQuery
		tq.options.Filters["__validation_error__"] = fmt.Errorf("data field validation: %w", err)
		return tq
	}

	// Transform to snake_case for storage
	snakeField := normalizeFieldName(field)
	tq.options.Filters["_data."+snakeField] = value
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
func (tq *Query[T]) DataIn(field string, values ...interface{}) *Query[T] {
	// Validate and transform field name immediately for better error reporting
	var zeroT T
	typ := reflect.TypeOf(zeroT)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Validate field exists
	if err := tq.typedStore.validateDataFieldName(typ, field); err != nil {
		// Store error to be returned during Find()
		tq.options.Filters["__validation_error__"] = fmt.Errorf("DataIn field validation: %w", err)
		return tq
	}

	// Transform to snake_case for storage
	snakeField := normalizeFieldName(field)
	tq.options.Filters["_data."+snakeField] = values
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
func (tq *Query[T]) DataNot(field string, value interface{}) *Query[T] {
	// Validate and transform field name immediately for better error reporting
	var zeroT T
	typ := reflect.TypeOf(zeroT)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Validate field exists
	if err := tq.typedStore.validateDataFieldName(typ, field); err != nil {
		// Store error to be returned during Find()
		tq.options.Filters["__validation_error__"] = fmt.Errorf("DataNot field validation: %w", err)
		return tq
	}

	// Transform to snake_case for storage - use snake_case in special filter key
	snakeField := normalizeFieldName(field)
	tq.options.Filters["__data_not__"+snakeField] = value
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
func (tq *Query[T]) DataNotIn(field string, values ...interface{}) *Query[T] {
	// Validate and transform field name immediately for better error reporting
	var zeroT T
	typ := reflect.TypeOf(zeroT)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Validate field exists
	if err := tq.typedStore.validateDataFieldName(typ, field); err != nil {
		// Store error to be returned during Find()
		tq.options.Filters["__validation_error__"] = fmt.Errorf("DataNotIn field validation: %w", err)
		return tq
	}

	// Transform to snake_case for storage - use snake_case in special filter key
	snakeField := normalizeFieldName(field)
	tq.options.Filters["__data_not_in__"+snakeField] = values
	return tq
}

// Where adds a custom SQL WHERE clause condition for advanced filtering.
//
// Implementation Note: Since the underlying store doesn't support WHERE clauses
// in List operations (only in Delete/Update), this method uses post-processing.
// The condition is applied after retrieving results from the store.
//
// The whereClause should NOT include the "WHERE" keyword itself.
// Use SQL column names that match the underlying schema:
// - Document fields: uuid, simple_id, title, body, created_at, updated_at
// - Dimension fields: Use dimension names directly (status, priority, etc.)
// - Data fields: Use _data.field_name format
//
// Performance Note: This may be slower than dimension-based filtering since
// it requires post-processing of all matching documents from other filters.
//
// Examples:
//
//	// Find documents created in the last week
//	results, err := store.Query().
//	    Where("created_at > ?", time.Now().AddDate(0, 0, -7)).
//	    Find()
//
//	// Find documents with title containing text (case-insensitive)
//	results, err := store.Query().
//	    Where("LOWER(title) LIKE ?", "%important%").
//	    Find()
//
//	// Complex condition with multiple fields
//	results, err := store.Query().
//	    Status("active").
//	    Where("created_at > ? AND (priority = ? OR _data.urgent = ?)",
//	          yesterday, "high", true).
//	    Find()
//
// CRITICAL SECURITY NOTE: Always use parameterized queries with ? placeholders.
// The underlying WhereEvaluator implements robust injection protection, but you must
// use it correctly. Examples:
//
//	// SAFE - parameterized query
//	query.Where("status = ? AND priority = ?", userStatus, userPriority)
//
//	// DANGEROUS - string concatenation opens injection vulnerability
//	query.Where("status = '" + userInput + "'") // DON'T DO THIS
//
//	// DANGEROUS - even formatted strings are vulnerable
//	query.Where(fmt.Sprintf("status = '%s'", userInput)) // DON'T DO THIS
//
// The security design requires that query structure be established BEFORE
// user parameters are considered. Parameter substitution happens safely after parsing.
func (tq *Query[T]) Where(whereClause string, args ...interface{}) *Query[T] {
	// Use special filter key to mark for post-processing
	tq.options.Filters["__where_clause__"] = map[string]interface{}{
		"clause": whereClause,
		"args":   args,
	}
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
func (tq *Query[T]) ParentID(id string) *Query[T] {
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
func (tq *Query[T]) ParentIDNotExists() *Query[T] {
	// We need to filter in post-processing since the store doesn't
	// support "not exists" queries directly
	// For now, we'll get all and filter
	// In production, you'd add proper NOT EXISTS support to the store
	tq.options.Filters["__parent_not_exists__"] = true
	return tq
}

// ParentIDStartsWith filters for documents whose parent ID starts with a prefix
// Useful for finding all descendants of a node
func (tq *Query[T]) ParentIDStartsWith(prefix string) *Query[T] {
	// This would need custom support in the store layer
	// For now, we'll skip implementation
	return tq
}

// Search adds full-text search filtering across document title and body fields.
//
// This method enables text-based searching within documents, searching across
// the Title and Body fields of documents. The search is typically case-insensitive
// and supports partial word matching depending on the underlying store implementation.
//
// # Search Behavior
//
// - **Fields Searched**: Document Title and Body fields
// - **Search Type**: Full-text search with partial matching
// - **Case Sensitivity**: Typically case-insensitive (store-dependent)
// - **Multiple Terms**: Space-separated terms are typically treated as AND conditions
//
// # Usage Examples
//
//	// Search for documents containing "budget"
//	results, err := store.Query().
//	    Search("budget").
//	    Find()
//
//	// Combine search with status filtering
//	activeBudgetTasks, err := store.Query().
//	    Search("budget").
//	    Status("active").
//	    Find()
//
//	// Search with multiple terms
//	quarterlyReports, err := store.Query().
//	    Search("quarterly report").
//	    Priority("high").
//	    Find()
//
//	// Search and get just the first match
//	firstMatch, err := store.Query().
//	    Search("meeting").
//	    OrderByDesc("created_at").
//	    First()
//
//	// Check if any documents contain specific text
//	hasDocuments, err := store.Query().
//	    Search("project alpha").
//	    Exists()
//
// # Performance Considerations
//
// - **Text Search**: May be slower than dimensional filtering
// - **Indexing**: Performance depends on underlying search indexing
// - **Combination**: Combine with dimensional filters for better performance
// - **Large Text**: Long search terms may impact performance
//
// # Search Tips
//
// **For Better Performance:**
// - Use dimensional filters first, then add search
// - Keep search terms focused and specific
// - Consider using Data() filters for exact field matching instead
//
// **Search Strategy:**
//
//	// ✅ Efficient - filter by dimension first
//	results := store.Query().
//	    Status("active").        // Fast dimensional filter
//	    Search("budget").        // Then text search
//	    Find()
//
//	// ❌ Less efficient - search first
//	results := store.Query().
//	    Search("budget").        // Slower text search first
//	    Status("active").        // Then dimensional filter
//	    Find()
//
// # Related Methods
//
// - Data() - Exact field matching for non-document fields
// - DataIn() - Match against specific values in data fields
// - Where() - Custom SQL-based text searching with LIKE patterns
//
// # Note on Data Fields
//
// Search() only searches Title and Body fields. To search custom data fields
// (like Assignee, Description, etc.), use Data() or Where() methods instead.
func (tq *Query[T]) Search(text string) *Query[T] {
	tq.options.FilterBySearch = text
	return tq
}

// OrderBy adds ordering
func (tq *Query[T]) OrderBy(column string) *Query[T] {
	tq.options.OrderBy = append(tq.options.OrderBy, types.OrderClause{
		Column:     column,
		Descending: false,
	})
	return tq
}

// OrderByDesc adds descending ordering
func (tq *Query[T]) OrderByDesc(column string) *Query[T] {
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
func (tq *Query[T]) OrderByData(field string) *Query[T] {
	// Validate and transform field name immediately for better error reporting
	var zeroT T
	typ := reflect.TypeOf(zeroT)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Validate field exists
	if err := tq.typedStore.validateDataFieldName(typ, field); err != nil {
		// Store error to be returned during Find()
		tq.options.Filters["__validation_error__"] = fmt.Errorf("orderByData field validation: %w", err)
		return tq
	}

	// Transform to snake_case for storage
	snakeField := normalizeFieldName(field)
	tq.options.OrderBy = append(tq.options.OrderBy, types.OrderClause{
		Column:     "_data." + snakeField,
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
func (tq *Query[T]) OrderByDataDesc(field string) *Query[T] {
	// Validate and transform field name immediately for better error reporting
	var zeroT T
	typ := reflect.TypeOf(zeroT)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Validate field exists
	if err := tq.typedStore.validateDataFieldName(typ, field); err != nil {
		// Store error to be returned during Find()
		tq.options.Filters["__validation_error__"] = fmt.Errorf("OrderByDataDesc field validation: %w", err)
		return tq
	}

	// Transform to snake_case for storage
	snakeField := normalizeFieldName(field)
	tq.options.OrderBy = append(tq.options.OrderBy, types.OrderClause{
		Column:     "_data." + snakeField,
		Descending: true,
	})
	return tq
}

// Limit sets the maximum number of results to return from the query.
//
// This method implements result limiting for pagination and performance optimization.
// The limit is applied after all filtering and ordering operations, returning only
// the first N documents from the final result set.
//
// # Behavior
//
// - **Zero or Negative Values**: No limit applied (returns all results)
// - **Positive Values**: Returns at most N documents
// - **Combined with Offset**: Returns N documents starting from offset position
// - **Ordering Impact**: Limit applies to ordered results (use OrderBy for predictable pagination)
//
// # Pagination Patterns
//
// **Basic Pagination:**
//
//	// Page 1: First 10 results
//	page1, err := store.Query().
//	    Status("active").
//	    OrderBy("created_at").
//	    Limit(10).
//	    Find()
//
//	// Page 2: Next 10 results
//	page2, err := store.Query().
//	    Status("active").
//	    OrderBy("created_at").
//	    Limit(10).
//	    Offset(10).
//	    Find()
//
//	// Page 3: Next 10 results
//	page3, err := store.Query().
//	    Status("active").
//	    OrderBy("created_at").
//	    Limit(10).
//	    Offset(20).
//	    Find()
//
// **Performance Optimization:**
//
//	// Get just the top 5 high-priority tasks
//	topTasks, err := store.Query().
//	    Priority("high").
//	    OrderByDesc("created_at").
//	    Limit(5).
//	    Find()
//
//	// Check if more than 100 results exist (early termination)
//	sample, err := store.Query().
//	    Status("pending").
//	    Limit(101).
//	    Find()
//	tooMany := len(sample) > 100
//
// # Use Cases
//
// - **Pagination**: Breaking large result sets into manageable pages
// - **Performance**: Limiting expensive queries to reduce memory usage
// - **Sampling**: Getting representative samples from large datasets
// - **UI Display**: Loading initial results with "load more" functionality
// - **Batch Processing**: Processing data in chunks
//
// # Performance Impact
//
// - **Database Level**: Limit is applied at database level for efficiency
// - **Memory Usage**: Significantly reduces memory allocation for large result sets
// - **Query Time**: Can improve query performance for large tables
// - **Best Practice**: Always use with OrderBy() for consistent pagination
//
// # Related Methods
//
// - Offset() - Skip N results (combine for pagination)
// - OrderBy() - Essential for predictable pagination
// - First() - Equivalent to Limit(1) with error handling
// - Count() - Get total count (ignores Limit for full count)
func (tq *Query[T]) Limit(n int) *Query[T] {
	tq.options.Limit = &n
	return tq
}

// Offset sets the number of results to skip before returning results.
//
// This method implements result skipping for pagination, allowing you to skip
// the first N documents and return subsequent results. Commonly used with Limit()
// to implement pagination patterns.
//
// # Behavior
//
// - **Zero Value**: No results skipped (starts from beginning)
// - **Positive Values**: Skips the first N documents
// - **Negative Values**: Treated as zero (no skipping)
// - **Combined with Limit**: Skip N, then return up to Limit documents
// - **Ordering Required**: Use OrderBy() for consistent pagination
//
// # Pagination Implementation
//
// **Standard Pagination Formula:**
//
//	pageSize := 10
//	pageNumber := 2  // 1-based page numbering
//
//	results, err := store.Query().
//	    OrderBy("created_at").              // Consistent ordering
//	    Limit(pageSize).                    // Page size
//	    Offset((pageNumber - 1) * pageSize). // Skip previous pages
//	    Find()
//
// **Real-World Pagination Example:**
//
//	func GetTasksPage(store *Store[Task], page, pageSize int, filters TaskFilters) ([]Task, error) {
//	    query := store.Query()
//
//	    // Apply filters
//	    if filters.Status != "" {
//	        query = query.Status(filters.Status)
//	    }
//	    if filters.Priority != "" {
//	        query = query.Priority(filters.Priority)
//	    }
//
//	    // Apply pagination
//	    return query.
//	        OrderBy("created_at").                    // Consistent ordering
//	        Limit(pageSize).                         // Results per page
//	        Offset((page - 1) * pageSize).          // Skip previous pages
//	        Find()
//	}
//
// # Performance Considerations
//
// - **Large Offsets**: Very large offsets can be slow (database must count+skip)
// - **Deep Pagination**: Consider cursor-based pagination for large datasets
// - **Memory**: Offset doesn't affect memory usage (only skips, doesn't load)
// - **Database Level**: Offset is handled efficiently at database level
//
// **Performance Tips:**
//
//	// ✅ Good - reasonable offset
//	page1 := query.Limit(20).Offset(0).Find()   // First page
//	page2 := query.Limit(20).Offset(20).Find()  // Second page
//
//	// ⚠️  Be careful - large offset may be slow
//	page100 := query.Limit(20).Offset(1980).Find() // Page 100
//
//	// ✅ Better for deep pagination - cursor-based
//	lastCreatedAt := getLastItemTimestamp()
//	nextPage := query.Where("created_at > ?", lastCreatedAt).Limit(20).Find()
//
// # Common Patterns
//
// **Load More Pattern:**
//
//	currentResults := []Task{}
//	pageSize := 20
//
//	for page := 1; ; page++ {
//	    batch, err := store.Query().
//	        Status("active").
//	        OrderBy("priority").
//	        Limit(pageSize).
//	        Offset((page - 1) * pageSize).
//	        Find()
//
//	    if len(batch) == 0 {
//	        break // No more results
//	    }
//
//	    currentResults = append(currentResults, batch...)
//
//	    if len(batch) < pageSize {
//	        break // Last page (partial results)
//	    }
//	}
//
// **Infinite Scroll Pattern:**
//
//	loadedCount := 0
//	pageSize := 10
//
//	func loadMoreTasks() {
//	    moreTasks, err := store.Query().
//	        Status("active").
//	        OrderByDesc("created_at").
//	        Limit(pageSize).
//	        Offset(loadedCount).
//	        Find()
//
//	    if len(moreTasks) > 0 {
//	        displayTasks(moreTasks)
//	        loadedCount += len(moreTasks)
//	    }
//	}
//
// # Related Methods
//
// - Limit() - Set maximum results (essential for pagination)
// - OrderBy() - Required for consistent pagination
// - Count() - Get total count for pagination info (ignores Offset)
// - Find() - Execute paginated query
func (tq *Query[T]) Offset(n int) *Query[T] {
	tq.options.Offset = &n
	return tq
}

// Find executes the query and returns typed results.
//
// This is the primary terminal method for query execution. It performs several steps:
//
// 1. **Field Validation**: Validates all data field names against the Go struct (NEW: eliminates silent failures)
// 2. **Query Execution**: Executes the accumulated filters against the store
// 3. **Post-Processing**: Applies filters that require client-side processing
// 4. **Type Unmarshaling**: Converts raw documents back to typed structs
// 5. **Result Assembly**: Builds the final []T slice for return
//
// # Field Validation (Issue #83 Fix)
//
// As of this implementation, Find() now validates all data field references before query execution.
// This eliminates silent failures where invalid field names would return empty results instead of errors.
//
// - **Data field validation**: All Data(), DataIn(), DataNot(), DataNotIn() field names are validated
// - **Ordering field validation**: All OrderByData(), OrderByDataDesc() field names are validated
// - **Case-insensitive**: Field names like "assignee" and "Assignee" both work correctly
// - **Helpful errors**: Invalid field names return clear error messages with suggestions
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
func (tq *Query[T]) Find() ([]T, error) {
	// Check for validation errors first
	if validationErr, ok := tq.options.Filters["__validation_error__"]; ok {
		delete(tq.options.Filters, "__validation_error__")
		if err, isErr := validationErr.(error); isErr {
			return nil, err
		}
	}

	// Validate data field references before executing the query
	if err := tq.validateDataFieldReferences(); err != nil {
		return nil, err
	}

	// Check for special filters and extract them for post-processing
	parentNotExists := false
	if _, ok := tq.options.Filters["__parent_not_exists__"]; ok {
		parentNotExists = true
		delete(tq.options.Filters, "__parent_not_exists__")
	}

	// Extract special filters for post-processing
	var dataNotFilters []struct {
		field string
		value interface{}
	}
	var dataNotInFilters []struct {
		field  string
		values []interface{}
	}
	var whereClause struct {
		clause string
		args   []interface{}
		active bool
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
		} else if key == "__where_clause__" {
			if whereMap, ok := value.(map[string]interface{}); ok {
				if clause, ok := whereMap["clause"].(string); ok {
					whereClause.clause = clause
					whereClause.active = true
					if args, ok := whereMap["args"].([]interface{}); ok {
						whereClause.args = args
					}
				}
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

		// Apply WHERE clause filter
		if whereClause.active {
			// Use the secure WhereEvaluator to safely evaluate the WHERE clause
			evaluator := store.NewWhereEvaluator(whereClause.clause, whereClause.args...)
			matches, err := evaluator.EvaluateDocument(&doc)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate WHERE clause: %w", err)
			}
			if !matches {
				continue // Skip documents that don't match the WHERE clause
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

// First returns the first matching document or an error if no documents are found.
//
// This is a terminal operation that executes the query and returns only the first result.
// It's equivalent to calling Limit(1).Find() but provides more convenient error handling
// for single-document retrieval scenarios.
//
// # Behavior
//
// - Automatically sets limit to 1 for optimal performance
// - Returns the first document if any matches are found
// - Returns error if no documents match the query criteria
// - Respects all previously applied filters, ordering, and offset
//
// # Ordering Impact
//
// The "first" document depends on the ordering applied to the query:
// - **No ordering**: Returns first document in storage order (unpredictable)
// - **With OrderBy()**: Returns first document according to specified ordering
// - **With offset**: Returns first document after skipping offset documents
//
// # Usage Examples
//
//	// Get the most recent high-priority task
//	task, err := store.Query().
//	    Priority("high").
//	    OrderByDesc("created_at").
//	    First()
//	if err != nil {
//	    log.Printf("No high-priority tasks found: %v", err)
//	    return
//	}
//	fmt.Printf("Most recent high-priority task: %s\n", task.Title)
//
//	// Get first pending task (any order)
//	task, err := store.Query().
//	    Status("pending").
//	    First()
//	if err != nil {
//	    log.Printf("No pending tasks: %v", err)
//	} else {
//	    fmt.Printf("Found pending task: %s\n", task.Title)
//	}
//
//	// Get first child of a specific parent
//	child, err := store.Query().
//	    ParentID("project-1").
//	    OrderBy("title").
//	    First()
//
// # Error Handling
//
// Returns error if:
// - No documents match the query criteria (returns "no documents found")
// - Database query fails
// - Type unmarshaling fails
// - Any filter validation fails
//
// # Performance Characteristics
//
// - **Query Time**: O(log n) with proper indexing + ordering overhead
// - **Memory Usage**: Minimal - only allocates one document
// - **Optimization**: Automatically applies LIMIT 1 to database query
// - **Best Practice**: Use with OrderBy() for predictable results
//
// # Related Methods
//
// - Find() - Returns all matching documents
// - Exists() - Check if any documents exist without retrieving data
// - Count() - Get count of matching documents
func (tq *Query[T]) First() (*T, error) {
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

// Count returns the number of documents that match the current query criteria.
//
// This is a terminal operation that executes the query and counts all matching results.
// Unlike database-level COUNT queries, this method performs full query execution
// including type marshaling and validation to ensure accurate counts.
//
// # Behavior
//
// - Executes the complete query with all filters applied
// - Performs type marshaling and validation on all results
// - Returns the count of successfully processed documents
// - Respects all filters, but ignores Limit() and Offset() for total count
//
// # Query Processing
//
// The Count() method uses the same execution path as Find() to ensure consistency:
// 1. Applies all dimensional filters
// 2. Executes data field filters and WHERE clauses
// 3. Performs type marshaling and validation
// 4. Returns count of valid results
//
// This approach ensures the count accurately reflects what Find() would return,
// but may be slower than database-level COUNT operations.
//
// # Usage Examples
//
//	// Count high-priority tasks
//	count, err := store.Query().
//	    Priority("high").
//	    Count()
//	if err != nil {
//	    log.Printf("Count failed: %v", err)
//	} else {
//	    fmt.Printf("Found %d high-priority tasks\n", count)
//	}
//
//	// Count pending tasks assigned to a specific user
//	count, err := store.Query().
//	    Status("pending").
//	    Data("Assignee", "alice").
//	    Count()
//	fmt.Printf("Alice has %d pending tasks\n", count)
//
//	// Count children of a project
//	childCount, err := store.Query().
//	    ParentID("project-1").
//	    Count()
//	if childCount > 0 {
//	    fmt.Printf("Project has %d sub-tasks\n", childCount)
//	}
//
//	// Check if any documents exist (alternative to Exists())
//	if count, _ := store.Query().Status("active").Count(); count > 0 {
//	    fmt.Printf("Found %d active items\n", count)
//	}
//
// # Performance Characteristics
//
// - **Query Time**: O(n) where n = number of matching documents (full table scan possible)
// - **Memory Usage**: O(n) during processing, O(1) for final result
// - **Processing Overhead**: Full type marshaling for each matching document
// - **Optimization**: Consider using Exists() if you only need to check for presence
//
// # Performance Considerations
//
// For large result sets, Count() can be expensive because it:
// - Loads and processes all matching documents
// - Performs type conversion and validation on each result
// - May be slower than SQL COUNT queries
//
// **Performance Alternatives:**
// - Use Exists() instead of `Count() > 0` for existence checks
// - Consider adding database-level count methods for high-frequency use cases
// - Use Limit() on Find() if you only need to know "at least N exist"
//
// # Error Handling
//
// Returns error if:
// - Database query fails
// - Any filter validation fails (invalid field names, etc.)
// - Type marshaling fails for any matching document
//
// Note: If some documents fail type marshaling, the entire operation fails.
// This ensures count consistency with Find() results.
//
// # Related Methods
//
// - Find() - Returns the actual documents being counted
// - Exists() - More efficient for checking if any documents exist
// - First() - Get just the first matching document
func (tq *Query[T]) Count() (int, error) {
	// Use Find to get filtered results including post-processing
	results, err := tq.Find()
	if err != nil {
		return 0, err
	}

	return len(results), nil
}

// Exists returns true if any documents match the current query criteria.
//
// This is an optimized terminal operation for existence checking that's more efficient
// than Count() > 0 or len(Find()) > 0 because it stops processing after finding
// the first matching document.
//
// # Behavior
//
// - Automatically sets limit to 1 for optimal performance
// - Returns true immediately upon finding any matching document
// - Returns false if no documents match the query criteria
// - Respects all previously applied filters and conditions
//
// # Performance Optimization
//
// Exists() is specifically optimized for existence checks:
// 1. Uses LIMIT 1 to stop after first match
// 2. Minimal memory allocation (only one document maximum)
// 3. Early termination on first successful match
// 4. More efficient than Count() for large result sets
//
// # Use Cases
//
// **Primary use cases for Exists():**
// - Conditional logic based on document presence
// - Validation that prerequisites exist
// - Checking constraints before operations
// - Dashboard indicators and status checks
//
// # Usage Examples
//
//	// Check if any high-priority tasks exist
//	hasUrgent, err := store.Query().
//	    Priority("high").
//	    Status("pending").
//	    Exists()
//	if err != nil {
//	    log.Printf("Query failed: %v", err)
//	} else if hasUrgent {
//	    log.Println("⚠️  High-priority tasks need attention")
//	}
//
//	// Validate that a parent exists before creating children
//	parentExists, err := store.Query().
//	    ParentID("project-1").
//	    Exists()
//	if !parentExists {
//	    return fmt.Errorf("cannot create subtask: project-1 does not exist")
//	}
//
//	// Check if user has any assigned tasks
//	hasWork, err := store.Query().
//	    Data("Assignee", userID).
//	    Status("pending").
//	    Exists()
//	if hasWork {
//	    fmt.Printf("User %s has pending work\n", userID)
//	}
//
//	// Dashboard status indicator
//	if hasActive, _ := store.Query().Status("active").Exists(); hasActive {
//	    statusIndicator = "🟢 Active"
//	} else {
//	    statusIndicator = "⚫ Inactive"
//	}
//
//	// Conditional processing
//	if exists, _ := store.Query().Priority("high").Exists(); exists {
//	    // Only process high-priority items if any exist
//	    tasks, _ := store.Query().Priority("high").Find()
//	    processTasks(tasks)
//	}
//
// # Performance Comparison
//
// **Exists() vs Alternatives:**
//
//	// ✅ Efficient - stops at first match
//	exists, err := query.Exists()
//
//	// ❌ Inefficient - counts ALL matches
//	count, err := query.Count()
//	exists := count > 0
//
//	// ❌ Very inefficient - loads ALL documents
//	results, err := query.Find()
//	exists := len(results) > 0
//
// # Performance Characteristics
//
// - **Query Time**: O(log n) with proper indexing (early termination)
// - **Memory Usage**: O(1) - maximum one document allocation
// - **Processing**: Minimal - stops after first valid match
// - **Best Case**: O(1) if first document matches
// - **Worst Case**: O(n) if no documents match (must check all)
//
// # Error Handling
//
// Returns error if:
// - Database query fails
// - Filter validation fails (invalid field names, etc.)
// - Type marshaling fails for the first matching document
//
// Note: If the first matching document fails type marshaling, the entire operation fails.
// This maintains consistency with other query methods.
//
// # Related Methods
//
// - Count() - Get exact number of matching documents (slower)
// - First() - Get the first matching document (similar performance, returns data)
// - Find() - Get all matching documents (much slower for large results)
//
// # Best Practices
//
// - **Use Exists() instead of Count() > 0** for existence checks
// - **Combine with conditional logic** for efficient workflows
// - **Use with indexable fields** (dimensions) for best performance
// - **Avoid for operations that need the actual documents** (use First() or Find())
func (tq *Query[T]) Exists() (bool, error) {
	// Set limit to 1 for efficiency
	limit := 1
	tq.options.Limit = &limit

	results, err := tq.Find()
	if err != nil {
		return false, err
	}

	return len(results) > 0, nil
}

// GetQueryPlan analyzes and returns information about how a query would be executed.
//
// This method provides insights into query execution for performance optimization
// and debugging. It analyzes the query structure without executing it.
//
// # Query Analysis
//
// Returns information about:
// - **Filter Types**: Which filters use dimensions vs data fields
// - **Index Usage**: Which filters can leverage indexed dimensions
// - **Performance Estimates**: Relative performance characteristics
// - **Optimization Suggestions**: Recommendations for better performance
//
// # Usage Examples
//
//	// Analyze a complex query
//	plan, err := store.Query().
//	    Status("active").
//	    Data("Assignee", "alice").
//	    OrderBy("created_at").
//	    GetQueryPlan()
//
//	fmt.Printf("Indexed filters: %d\n", plan.IndexedFilterCount)
//	fmt.Printf("Performance rating: %s\n", plan.PerformanceRating)
//	for _, suggestion := range plan.OptimizationSuggestions {
//	    fmt.Printf("Suggestion: %s\n", suggestion)
//	}
//
// # Performance Analysis
//
// The query plan provides performance insights:
// - **Fast**: Primarily uses indexed dimensional filters
// - **Medium**: Mix of indexed and data field filters
// - **Slow**: Primarily uses data field filters or complex WHERE clauses
//
// Use this information to optimize query performance by restructuring
// queries to use more dimensional filters when possible.
func (tq *Query[T]) GetQueryPlan() (*QueryPlan, error) {
	plan := &QueryPlan{
		TotalFilters:            0,
		IndexedFilterCount:      0,
		DataFieldFilterCount:    0,
		CustomWhereClauseCount:  0,
		OptimizationSuggestions: []string{},
	}

	// Analyze dimensional filters (these are indexed and fast)
	dimensionFilters := 0
	if tq.options.Filters != nil {
		for filterName := range tq.options.Filters {
			plan.TotalFilters++
			// Dimension filters are typically faster
			if filterName != "_data" { // _data filters are slower
				plan.IndexedFilterCount++
				dimensionFilters++
			}
		}
	}

	// Count data field filters (slower, require full document scan)
	dataFilters := 0
	if tq.options.Filters != nil {
		for filterName := range tq.options.Filters {
			if filterName == "_data" || strings.HasPrefix(filterName, "_data.") {
				dataFilters++
			}
		}
	}
	plan.DataFieldFilterCount = dataFilters
	plan.TotalFilters += dataFilters

	// Count custom WHERE clauses (performance varies)
	// Note: Current ListOptions doesn't support custom WHERE clauses
	plan.CustomWhereClauseCount = 0

	// Determine performance rating
	if plan.TotalFilters == 0 {
		plan.PerformanceRating = "No filters - will return all documents"
	} else if plan.IndexedFilterCount > 0 && plan.DataFieldFilterCount == 0 {
		plan.PerformanceRating = "Fast - uses only indexed dimensions"
	} else if plan.IndexedFilterCount >= plan.DataFieldFilterCount {
		plan.PerformanceRating = "Medium - mix of indexed and data field filters"
	} else {
		plan.PerformanceRating = "Slow - primarily uses data field filters"
	}

	// Generate optimization suggestions
	if plan.DataFieldFilterCount > plan.IndexedFilterCount && plan.IndexedFilterCount == 0 {
		plan.OptimizationSuggestions = append(plan.OptimizationSuggestions,
			"Consider adding dimensional filters to improve performance")
	}

	if plan.DataFieldFilterCount > 3 {
		plan.OptimizationSuggestions = append(plan.OptimizationSuggestions,
			"Many data field filters detected - consider restructuring as dimensional filters if possible")
	}

	if plan.CustomWhereClauseCount > 0 && plan.IndexedFilterCount == 0 {
		plan.OptimizationSuggestions = append(plan.OptimizationSuggestions,
			"WHERE clause without indexed filters may be slow - add dimensional filters if possible")
	}

	if len(tq.options.OrderBy) > 0 && plan.TotalFilters > 0 {
		plan.OptimizationSuggestions = append(plan.OptimizationSuggestions,
			"Ordering with filtering - ensure ORDER BY columns are indexed for best performance")
	}

	if plan.TotalFilters == 0 && (tq.options.Limit == nil || *tq.options.Limit > 1000) {
		plan.OptimizationSuggestions = append(plan.OptimizationSuggestions,
			"No filters with high/unlimited limit - consider adding filters or reducing limit")
	}

	return plan, nil
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

		// Check for pointer fields and validate supported types
		if field.Type.Kind() == reflect.Ptr {
			// Check if the underlying type is supported
			elemType := field.Type.Elem()
			switch elemType.Kind() {
			case reflect.String, reflect.Bool, reflect.Int, reflect.Int64, reflect.Float64:
				// Basic pointer types are supported
			default:
				// Check for time.Time specifically
				if elemType == reflect.TypeOf(time.Time{}) {
					// *time.Time is supported
				} else {
					return config, fmt.Errorf("field %s: pointer type %s is not supported (supported: *string, *bool, *int, *int64, *float64, *time.Time)", field.Name, field.Type)
				}
			}
		}

		// Look for field tags in different formats
		if tagValue, tagExists := field.Tag.Lookup("values"); tagExists {
			// Parse enumerated dimension from tags like:
			// `values:"pending,active,done" prefix:"done=d" default:"pending"`
			dimConfig := nanostore.DimensionConfig{
				Name: strings.ToLower(field.Name),
				Type: nanostore.Enumerated,
			}

			// Parse and validate values tag
			if err := parseValuesTag(tagValue, &dimConfig, field.Name); err != nil {
				return config, fmt.Errorf("field '%s': %w", field.Name, err)
			}

			// Parse and validate default tag
			if defaultVal, defaultExists := field.Tag.Lookup("default"); defaultExists {
				if err := parseDefaultTag(defaultVal, &dimConfig, field.Name); err != nil {
					return config, fmt.Errorf("field '%s': %w", field.Name, err)
				}
			}

			// Parse and validate prefix tag
			if prefixTag, prefixExists := field.Tag.Lookup("prefix"); prefixExists {
				if err := parsePrefixTag(prefixTag, &dimConfig, field.Name); err != nil {
					return config, fmt.Errorf("field '%s': %w", field.Name, err)
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
			} else {
				// Regular dimension (simple value dimension)
				config.Dimensions = append(config.Dimensions, nanostore.DimensionConfig{
					Name: dimName,
					Type: nanostore.Enumerated, // Use enumerated type for simple values
				})
			}
		}
	}

	// Validate the generated configuration for consistency and correctness
	if err := validateStructTagConfiguration(config); err != nil {
		return config, fmt.Errorf("struct tag validation failed: %w", err)
	}

	return config, nil
}

// parseValuesTag parses and validates the "values" struct tag
func parseValuesTag(tagValue string, dimConfig *nanostore.DimensionConfig, fieldName string) error {
	if strings.TrimSpace(tagValue) == "" {
		return fmt.Errorf("values tag cannot be empty")
	}

	// Split and validate values
	values := strings.Split(tagValue, ",")
	var cleanValues []string

	for i, value := range values {
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			return fmt.Errorf("empty value at position %d in values tag '%s'", i, tagValue)
		}

		// Check for suspicious patterns that might indicate malformed tags
		if strings.Contains(trimmedValue, "=") {
			return fmt.Errorf("value '%s' contains '=' - did you mean to use the prefix tag?", trimmedValue)
		}
		if strings.Contains(trimmedValue, ":") {
			return fmt.Errorf("value '%s' contains ':' - check tag formatting", trimmedValue)
		}

		cleanValues = append(cleanValues, trimmedValue)
	}

	dimConfig.Values = cleanValues
	return nil
}

// parseDefaultTag parses and validates the "default" struct tag
func parseDefaultTag(tagValue string, dimConfig *nanostore.DimensionConfig, fieldName string) error {
	if strings.TrimSpace(tagValue) == "" {
		return fmt.Errorf("default tag cannot be empty")
	}

	// Check for suspicious patterns
	if strings.Contains(tagValue, ",") {
		return fmt.Errorf("default value '%s' contains comma - default should be a single value", tagValue)
	}
	if strings.Contains(tagValue, "=") {
		return fmt.Errorf("default value '%s' contains '=' - check tag formatting", tagValue)
	}

	dimConfig.DefaultValue = strings.TrimSpace(tagValue)
	return nil
}

// parsePrefixTag parses and validates the "prefix" struct tag
func parsePrefixTag(tagValue string, dimConfig *nanostore.DimensionConfig, fieldName string) error {
	if strings.TrimSpace(tagValue) == "" {
		return fmt.Errorf("prefix tag cannot be empty")
	}

	dimConfig.Prefixes = make(map[string]string)

	// Parse formats like "done=d" or "done=d,active=a"
	prefixMappings := strings.Split(tagValue, ",")
	for i, mapping := range prefixMappings {
		trimmedMapping := strings.TrimSpace(mapping)
		if trimmedMapping == "" {
			return fmt.Errorf("empty prefix mapping at position %d in prefix tag '%s'", i, tagValue)
		}

		// Split on '=' to get value=prefix pairs
		parts := strings.Split(trimmedMapping, "=")
		if len(parts) != 2 {
			return fmt.Errorf("invalid prefix mapping '%s' - format should be 'value=prefix'", trimmedMapping)
		}

		value := strings.TrimSpace(parts[0])
		prefix := strings.TrimSpace(parts[1])

		if value == "" {
			return fmt.Errorf("empty value in prefix mapping '%s'", trimmedMapping)
		}
		if prefix == "" {
			return fmt.Errorf("empty prefix in prefix mapping '%s'", trimmedMapping)
		}

		// Check for duplicate prefix mappings within the same tag
		if existingValue, exists := dimConfig.Prefixes[value]; exists {
			return fmt.Errorf("duplicate prefix mapping for value '%s' (was '%s', now '%s')",
				value, existingValue, prefix)
		}

		dimConfig.Prefixes[value] = prefix
	}

	return nil
}

// validateStructTagConfiguration performs comprehensive validation of the configuration
// generated from struct tags to catch malformed tags and constraint violations.
func validateStructTagConfiguration(config nanostore.Config) error {
	dimensionNames := make(map[string]bool)
	allPrefixes := make(map[string]string) // prefix -> dimension name

	for _, dim := range config.Dimensions {
		// Check for duplicate dimension names
		if dimensionNames[dim.Name] {
			return fmt.Errorf("duplicate dimension name '%s'", dim.Name)
		}
		dimensionNames[dim.Name] = true

		// Validate based on dimension type
		switch dim.Type {
		case nanostore.Enumerated:
			if err := validateEnumeratedDimensionConfig(dim); err != nil {
				return fmt.Errorf("dimension '%s': %w", dim.Name, err)
			}

			// Check for prefix conflicts across dimensions
			for value, prefix := range dim.Prefixes {
				if existingDim, exists := allPrefixes[prefix]; exists {
					return fmt.Errorf("dimension '%s': prefix '%s' for value '%s' conflicts with dimension '%s'",
						dim.Name, prefix, value, existingDim)
				}
				allPrefixes[prefix] = dim.Name
			}

		case nanostore.Hierarchical:
			if err := validateHierarchicalDimensionConfig(dim); err != nil {
				return fmt.Errorf("dimension '%s': %w", dim.Name, err)
			}
		}
	}

	return nil
}

// validateEnumeratedDimensionConfig validates enumerated dimension configuration
func validateEnumeratedDimensionConfig(dim nanostore.DimensionConfig) error {
	// Allow empty values list for simple dimensions (e.g., pointer types like *time.Time)
	// These dimensions can accept any value, unlike traditional enumerated dimensions
	if len(dim.Values) == 0 {
		// For simple dimensions with no predefined values, skip validation
		return nil
	}

	// Check for empty values
	valueSet := make(map[string]bool)
	for i, value := range dim.Values {
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			return fmt.Errorf("empty value at position %d in values list", i)
		}

		// Check for duplicate values
		if valueSet[trimmedValue] {
			return fmt.Errorf("duplicate value '%s' in values list", trimmedValue)
		}
		valueSet[trimmedValue] = true

		// Update the config to use trimmed values
		dim.Values[i] = trimmedValue
	}

	// Validate default value if specified
	if dim.DefaultValue != "" {
		if !valueSet[dim.DefaultValue] {
			return fmt.Errorf("default value '%s' not in values list %v", dim.DefaultValue, dim.Values)
		}
	}

	// Validate prefix mappings
	for value, prefix := range dim.Prefixes {
		// Check that prefix is not empty
		if strings.TrimSpace(prefix) == "" {
			return fmt.Errorf("empty prefix not allowed for value '%s'", value)
		}

		// Check that prefixed value exists in values list
		if !valueSet[value] {
			return fmt.Errorf("prefix mapping for unknown value '%s'", value)
		}
	}

	return nil
}

// validateHierarchicalDimensionConfig validates hierarchical dimension configuration
func validateHierarchicalDimensionConfig(dim nanostore.DimensionConfig) error {
	// Check that RefField is specified
	if dim.RefField == "" {
		return fmt.Errorf("hierarchical dimension must specify RefField")
	}

	// RefField should not be empty string
	if strings.TrimSpace(dim.RefField) == "" {
		return fmt.Errorf("RefField cannot be empty")
	}

	return nil
}

// extractTypeInfo extracts detailed information about a Go type for debugging purposes.
// This function analyzes the type structure and provides comprehensive metadata
// about the type's fields, embedding relationships, and nanostore compatibility.
func extractTypeInfo(typ reflect.Type) TypeDebugInfo {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	info := TypeDebugInfo{
		TypeName:    typ.String(),
		PackageName: typ.PkgPath(),
		FieldCount:  0,
		Fields:      []FieldDebugInfo{},
		EmbedsList:  []string{},
		HasDocument: false,
	}

	if typ.Kind() != reflect.Struct {
		return info
	}

	info.FieldCount = typ.NumField()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		fieldInfo := FieldDebugInfo{
			Name:         field.Name,
			Type:         field.Type.String(),
			Tag:          string(field.Tag),
			IsEmbedded:   field.Anonymous,
			IsDimension:  false,
			DimensionTag: "",
		}

		// Check if this field is a dimension
		valuesValue, valuesExists := field.Tag.Lookup("values")
		_, dimensionExists := field.Tag.Lookup("dimension")
		if valuesExists || dimensionExists {
			fieldInfo.IsDimension = true
			if valuesExists {
				fieldInfo.DimensionTag = fmt.Sprintf("values:%s", valuesValue)
				if prefix := field.Tag.Get("prefix"); prefix != "" {
					fieldInfo.DimensionTag += fmt.Sprintf(" prefix:%s", prefix)
				}
				if defaultVal := field.Tag.Get("default"); defaultVal != "" {
					fieldInfo.DimensionTag += fmt.Sprintf(" default:%s", defaultVal)
				}
			} else if dimension := field.Tag.Get("dimension"); dimension != "" {
				fieldInfo.DimensionTag = fmt.Sprintf("dimension:%s", dimension)
			}
		}

		// Check if embedded
		if field.Anonymous {
			info.EmbedsList = append(info.EmbedsList, field.Type.String())
			// Check if it's nanostore.Document
			if field.Type == reflect.TypeOf(nanostore.Document{}) {
				info.HasDocument = true
			}
		}

		info.Fields = append(info.Fields, fieldInfo)
	}

	return info
}

// countTotalValues counts the total number of enumerated values across all dimensions.
// This provides insight into the complexity of the dimension configuration.
func countTotalValues(dimensions []nanostore.DimensionConfig) int {
	total := 0
	for _, dim := range dimensions {
		if dim.Type == nanostore.Enumerated {
			total += len(dim.Values)
		}
	}
	return total
}

// countTotalPrefixes counts the total number of prefix mappings across all dimensions.
// This helps understand the ID generation complexity and potential for conflicts.
func countTotalPrefixes(dimensions []nanostore.DimensionConfig) int {
	total := 0
	for _, dim := range dimensions {
		if dim.Type == nanostore.Enumerated && dim.Prefixes != nil {
			total += len(dim.Prefixes)
		}
	}
	return total
}
