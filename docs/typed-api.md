# Nanostore TypedStore API

The TypedStore API provides compile-time type safety and automatic configuration generation for nanostore. Define your data model with struct tags and get a fully configured, type-safe document store.

## Problem It Solves

Before TypedStore, working with nanostore required manual configuration and type assertions:

```go
// Manual configuration was verbose and error-prone
config := nanostore.Config{
    Dimensions: []nanostore.DimensionConfig{
        {
            Name:         "status",
            Type:         nanostore.Enumerated,
            Values:       []string{"pending", "active", "done"},
            DefaultValue: "pending",
        },
        // ... more configuration
    },
}
store, err := nanostore.New("data.json", config)

// Working with documents required type assertions
func getDocumentStatus(doc nanostore.Document) string {
    if status, ok := doc.Dimensions["status"].(string); ok {
        return status
    }
    return "pending"  // fallback
}
```

## Solution: TypedStore with Struct Tags

Define your data model once with struct tags, get automatic configuration:

```go
type Task struct {
    nanostore.Document  // Required embedding
    
    Status   string `values:"pending,active,done" default:"pending"`
    Priority string `values:"low,medium,high" default:"medium"`
    ParentID string `dimension:"parent_id,ref"`  // ref = hierarchical reference
    
    // Custom fields (no tags needed)
    Description string
    AssignedTo  string
    DueDate     time.Time
}

// One line creates a fully configured store
store, err := nanostore.NewFromType[Task]("tasks.json")
```

## Features

### 1. Type-Safe Document Creation

```go
task := &Task{
    Status:      "active",
    Priority:    "high",
    ParentID:    parentID,  // Supports UUID or user-facing ID
    Description: "Implement new feature",
    AssignedTo:  "alice",
}

id, err := store.Create("Implement feature", task)
// Returns human-friendly ID like "h1" (high priority, position 1)
```

### 2. Fluent Query Interface

```go
// Type-safe, chainable queries
activeTasks, err := store.Query().
    Status("active").
    Priority("high").
    OrderBy("created_at").
    Find()

for _, task := range activeTasks {
    fmt.Printf("%s: %s (%s priority)\n", 
        task.SimpleID,  // Generated ID like "h1", "h2"
        task.Title,     // Direct access, no type assertions
        task.Priority)  // Type safe
}
```

### 3. Simple CRUD Operations

```go
// Get by ID (supports both UUID and SimpleID)
task, err := store.Get("h1")

// Update with type safety
task.Status = "completed"
task.AssignedTo = "bob"
err = store.Update(task.SimpleID, task)

// Delete with cascade support
err = store.Delete("h1", true)  // Delete task and subtasks
```

### 4. Automatic Configuration

- Dimension configuration generated from struct tags
- Default values applied automatically
- Validation built-in (enum values, required fields)
- No manual configuration needed

### 5. Smart ID Resolution

Hierarchical references work with both UUIDs and user-facing IDs:

```go
type Task struct {
    ParentID string `dimension:"parent_id,ref"`
}

// Both work automatically:
task.ParentID = "123e4567-e89b-12d3-a456-426614174000"  // Internal UUID
task.ParentID = "h1"                                    // User-facing SimpleID

// Query children automatically resolves IDs
children, err := store.Query().ParentID("h1").Find()
```

## Tag Syntax

### Enumerated Dimensions

```go
Status string `values:"pending,active,done" default:"pending" prefix:"done=d,active=a"`
```

- `values` - Comma-separated list of valid values (required)
- `default` - Default value when not specified
- `prefix` - Value-to-prefix mappings for ID generation

### Hierarchical Dimensions

```go
ParentID string `dimension:"parent_id,ref"`
```

- First part: field name in dimensions map
- `ref` flag: marks as hierarchical reference

### Regular Fields

Fields without dimension tags are stored as custom data with `_data.` prefix automatically.

## Implementation Details

### Marshaling (struct → dimensions)

1. Extracts fields with `dimension` tags
2. Skips zero values (except false for bools)
3. Returns `map[string]interface{}` for use with store methods

### Unmarshaling (document → struct)

1. Populates embedded `Document` fields
2. Maps dimensions to struct fields based on tags
3. Applies defaults for missing dimensions
4. Handles type conversions (string→bool, string→int, etc.)

## TypedStore API

```go
// Create a typed store from struct definition
func NewFromType[T any](filePath string) (*TypedStore[T], error)

// TypedStore methods
type TypedStore[T any] struct {
    // CRUD operations
    Create(title string, data *T) (string, error)
    Get(id string) (*T, error)
    Update(id string, data *T) error
    Delete(id string, cascade bool) error
    
    // Bulk operations
    DeleteByDimension(filters map[string]interface{}) (int, error)
    UpdateByDimension(filters map[string]interface{}, data *T) (int, error)
    
    // Querying
    Query() *TypedQuery[T]
    
    // Lifecycle
    Close() error
}

// Query builder for type-safe filtering
type TypedQuery[T any] struct {
    // Filtering methods (generated based on your struct)
    Status(value string) *TypedQuery[T]
    Priority(value string) *TypedQuery[T]
    ParentID(id string) *TypedQuery[T]
    Search(text string) *TypedQuery[T]
    
    // Ordering and pagination
    OrderBy(column string) *TypedQuery[T]
    OrderByDesc(column string) *TypedQuery[T]
    Limit(n int) *TypedQuery[T]
    Offset(n int) *TypedQuery[T]
    
    // Terminal operations
    Find() ([]T, error)
    First() (*T, error)
    Count() (int, error)
    Exists() (bool, error)
}
```

## Best Practices

1. **Embed `nanostore.Document`** to get access to UUID, Title, Body, timestamps
2. **Use meaningful field names** that match your domain
3. **Set appropriate defaults** in tags for better ergonomics
4. **Mark parent references with `ref`** for smart ID support
5. **Keep zero values in mind** - they're omitted during marshaling

## Migration Example

Before (manual configuration):
```go
// Lots of manual configuration
config := nanostore.Config{
    Dimensions: []nanostore.DimensionConfig{
        {Name: "status", Type: nanostore.Enumerated, Values: []string{"pending", "done"}},
        {Name: "parent", Type: nanostore.Hierarchical, RefField: "parent_uuid"},
    },
}
store, err := nanostore.New("todos.json", config)

// Manual dimension handling
func GetActiveTodos() ([]Todo, error) {
    docs, _ := store.List(nanostore.ListOptions{
        Filters: map[string]interface{}{"status": "active"},
    })
    var todos []Todo
    for _, doc := range docs {
        status, _ := doc.Dimensions["status"].(string)
        priority, _ := doc.Dimensions["priority"].(string)
        todos = append(todos, Todo{
            Document: doc,
            Status:   status,
            Priority: priority,
        })
    }
    return todos, nil
}
```

After (TypedStore):
```go
type Todo struct {
    nanostore.Document
    Status   string `values:"pending,active,done" default:"pending"`
    Priority string `values:"low,medium,high" default:"medium"`
    ParentID string `dimension:"parent_id,ref"`
}

// One line setup
store, err := nanostore.NewFromType[Todo]("todos.json")

// One line queries
func GetActiveTodos() ([]Todo, error) {
    return store.Query().Status("active").Find()
}
```

**Result**: 95% less code, compile-time safety, and automatic configuration!