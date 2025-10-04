# Migration Guide: Nanostore v0.11

Nanostore v0.11 introduces a **major API consolidation** - the Store API is now the single, unified interface. This migration guide helps you upgrade from previous versions.

## Breaking Changes Summary

🚨 **Major API Changes:**
- Direct Store API (`nanostore.New()`) has been **removed**
- Store is renamed to **Store** - it's now the only API
- `api.New[T]()` becomes `api.New[T]()`
- Simplified package structure and imports

## Quick Migration

### Before (v0.10.x)
```go
import (
    "github.com/arthur-debert/nanostore/nanostore"
    "github.com/arthur-debert/nanostore/nanostore/api"
)

// Store API
type Task struct {
    nanostore.Document
    Status string `values:"pending,done" default:"pending"`
}

store, err := api.New[Task]("tasks.json")
```

### After (v0.11+)
```go
import (
    "github.com/arthur-debert/nanostore/nanostore"
    "github.com/arthur-debert/nanostore/nanostore/api"
)

// Single unified API
type Task struct {
    nanostore.Document
    Status string `values:"pending,done" default:"pending"`
}

store, err := api.New[Task]("tasks.json")
```

## Migration Steps

### 1. Update Function Calls

Replace all instances:
- `api.New[T]()` → `api.New[T]()`
- Store references → Store

### 2. Remove Direct Store API Usage

If you were using the Direct Store API (`nanostore.New()`), you must migrate to the typed API:

**Old Direct Store Pattern:**
```go
// ❌ No longer available
config := nanostore.Config{
    Dimensions: []nanostore.DimensionConfig{
        {Name: "status", Values: []string{"pending", "done"}},
    },
}
store, err := nanostore.New("data.json", config)
```

**New Unified Pattern:**
```go
// ✅ Define structure with tags
type Item struct {
    nanostore.Document
    Status string `values:"pending,done" default:"pending"`
}

store, err := api.New[Item]("data.json")
```

### 3. Update Documentation References

- Replace "Store" with "Store" in comments and docs
- Update import examples
- Remove references to "Direct Store API"

## Benefits of v0.11

✅ **Simplified API**: Single way to create and use stores  
✅ **Better Performance**: Optimized single-path implementation  
✅ **Type Safety**: All operations are compile-time checked  
✅ **Cleaner Imports**: Reduced cognitive overhead  
✅ **Future-Proof**: Built on the superior typed foundation  

## Complete Feature Parity

The new unified API provides **100% of previous functionality**:

### All Operations Available
```go
// Document operations
id, err := store.Create("Task", &task)
task, err := store.Get(id)
err = store.Update(id, task)
err = store.Delete(id)

// Querying with fluent interface
tasks, err := store.Query().Status("pending").Find()

// Bulk operations
count, err := store.UpdateByUUIDs(uuids, updateData)
count, err := store.DeleteByUUIDs(uuids)

// Raw access when needed
id, err := store.AddRaw("Title", dimensions)
doc, err := store.GetRaw(id)

// Testing utilities
err = store.SetTimeFunc(fixedTime)

// Advanced queries
tasks, err := store.List(types.ListOptions{...})
```

### Enhanced Type Safety
```go
// Before: Runtime errors possible
status := doc.Dimensions["status"].(string) // Panic if wrong type

// After: Compile-time safety
status := task.Status // Always correct type
```

## Breaking Change Details

### Removed APIs

These functions are **no longer available**:
- `nanostore.New(filePath, config)` 
- All `store.Store` interface methods from direct API
- Manual `nanostore.Config` construction

### Replaced APIs

| Old (v0.10.x) | New (v0.11+) |
|----------------|--------------|
| `api.New[T]()` | `api.New[T]()` |
| Store terminology | Store terminology |

## Migration Checklist

- [ ] Replace `api.New[T]()` with `api.New[T]()`
- [ ] Convert any Direct Store API usage to typed structs with tags
- [ ] Update variable names from `typedStore` to `store`
- [ ] Update documentation and comments
- [ ] Remove manual `nanostore.Config` objects
- [ ] Test all functionality works with new API
- [ ] Update import statements if needed

## Field Naming Consistency

v0.11 also includes field naming standardization:

### Consistent snake_case
```go
// ✅ All methods now use snake_case consistently
store.Query().Data("created_by", "alice")
store.List(types.ListOptions{
    OrderBy: []types.OrderClause{
        {Column: "_data.created_by"},
    },
})
```

### Better Validation
```go
// Invalid field names now return clear errors
tasks, err := store.Query().Data("nonexistent", "value").Find()
// Error: "field 'nonexistent' not found in Task, available: [created_by, status]"
```

## Example: Complete Migration

### Before (v0.10.x with Store)
```go
package main

import (
    "log"
    "github.com/arthur-debert/nanostore/nanostore"
    "github.com/arthur-debert/nanostore/nanostore/api"
)

type Task struct {
    nanostore.Document
    Status   string `values:"pending,active,done" default:"pending"`
    Priority string `values:"low,medium,high" prefix:"high=h"`
}

func main() {
    typedStore, err := api.New[Task]("tasks.json")
    if err != nil {
        log.Fatal(err)
    }
    defer typedStore.Close()
    
    id, err := typedStore.Create("Buy groceries", &Task{
        Status:   "pending",
        Priority: "high",
    })
}
```

### After (v0.11+)
```go
package main

import (
    "log"
    "github.com/arthur-debert/nanostore/nanostore"
    "github.com/arthur-debert/nanostore/nanostore/api"
)

type Task struct {
    nanostore.Document
    Status   string `values:"pending,active,done" default:"pending"`
    Priority string `values:"low,medium,high" prefix:"high=h"`
}

func main() {
    store, err := api.New[Task]("tasks.json")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    id, err := store.Create("Buy groceries", &Task{
        Status:   "pending",
        Priority: "high",
    })
}
```

## Getting Help

- Review the updated [API Reference](./typed-api.md)
- Check [sample applications](../samples/) for complete examples  
- See [GitHub issue #76](https://github.com/arthur-debert/nanostore/issues/76) for technical details
- Ask questions in GitHub Discussions

**The migration to a single, unified API provides a cleaner, more maintainable foundation for Nanostore's future development.** 🚀