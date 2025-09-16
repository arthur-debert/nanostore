# Nanostore Typed API

The typed API eliminates boilerplate code when working with nanostore documents by using Go struct tags to map between struct fields and dimension values.

## Problem It Solves

Before the typed API, client code was full of boilerplate:

```go
// Extracting dimensions requires type assertions and default handling
func getDocumentStatus(doc nanostore.Document) string {
    if status, ok := doc.Dimensions["status"].(string); ok {
        return status
    }
    return "pending"  // default value
}

// Creating documents requires building maps
dimensions := map[string]interface{}{
    "status": "completed",
    "priority": "high",
    "parent_id": parentID,
}
id, err := store.Add("Task", dimensions)
```

## Solution: Struct Tags

Define your domain types with `dimension` tags:

```go
type Task struct {
    nanostore.Document                          // Embed for base fields
    Status   string `dimension:"status,default=pending"`
    Priority string `dimension:"priority,default=medium"`
    Assignee string `dimension:"assignee"`
    ParentID string `dimension:"parent_id,ref"` // ref = hierarchical reference
}
```

## Features

### 1. Type-Safe Document Creation

```go
task := &Task{
    Status:   "in_progress",
    Priority: "high",
    Assignee: "alice",
    ParentID: parentID,  // Supports UUID or user-facing ID
}

id, err := nanostore.AddTyped(store, "Implement feature", task)
```

### 2. Clean Document Retrieval

```go
// No more type assertions!
tasks, err := nanostore.ListTyped[Task](store, nanostore.ListOptions{
    Filters: map[string]interface{}{"status": "in_progress"},
})

for _, task := range tasks {
    fmt.Printf("%s: %s (%s priority)\n", 
        task.Title,     // Direct access
        task.Status,    // No type assertion
        task.Priority)  // Type safe
}
```

### 3. Simplified Updates

```go
update := &Task{
    Status:   "completed",
    Assignee: "bob",
}

err := nanostore.UpdateTyped(store, taskID, update)
```

### 4. Smart Defaults

- Zero values are omitted (empty strings, nil, 0)
- Exception: `false` for bool fields is preserved
- Default values from tags are applied during unmarshaling

### 5. Reference ID Support

Fields tagged with `ref` work with nanostore's smart ID resolution:

```go
type Task struct {
    ParentID string `dimension:"parent_id,ref"`
}

// Both work:
task.ParentID = "123e4567-e89b-12d3-a456-426614174000"  // UUID
task.ParentID = "h1"                                    // User-facing ID
```

## Tag Syntax

```
dimension:"name[,option1][,option2]"
```

Options:
- `default=value` - Default value when unmarshaling if dimension is missing
- `ref` - Marks field as a hierarchical reference (for parent IDs)
- `-` - Skip this field (won't be included in dimensions)

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

## API Functions

```go
// Marshal struct to dimensions map
func MarshalDimensions(v interface{}) (map[string]interface{}, error)

// Unmarshal document to struct
func UnmarshalDimensions(doc Document, v interface{}) error

// Typed store operations
func AddTyped(s Store, title string, v interface{}) (string, error)
func UpdateTyped(s Store, id string, v interface{}) error
func ListTyped[T any](s Store, opts ListOptions) ([]T, error)
```

## Best Practices

1. **Embed `nanostore.Document`** to get access to UUID, Title, Body, timestamps
2. **Use meaningful field names** that match your domain
3. **Set appropriate defaults** in tags for better ergonomics
4. **Mark parent references with `ref`** for smart ID support
5. **Keep zero values in mind** - they're omitted during marshaling

## Migration Example

Before:
```go
func (a *Adapter) GetTodos() ([]Todo, error) {
    docs, _ := store.List(opts)
    var todos []Todo
    for _, doc := range docs {
        status, _ := doc.Dimensions["status"].(string)
        if status == "" {
            status = "pending"
        }
        // ... more boilerplate ...
    }
    return todos, nil
}
```

After:
```go
func (a *Adapter) GetTodos() ([]Todo, error) {
    return nanostore.ListTyped[Todo](store, opts)
}
```

That's it! The typed API dramatically reduces boilerplate while maintaining full compatibility with the existing nanostore API.