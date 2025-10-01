# Migration Guide: Direct Store API → TypedStore API

This guide helps you migrate from the Direct Store API (`nanostore.New()`) to the TypedStore API (`api.NewFromType[T]()`), which provides type safety, automatic configuration, and a superior developer experience.

## Quick Start: Basic Migration

### Before (Direct Store API)

```go
import "github.com/arthur-debert/nanostore/nanostore"

// Manual configuration
config := nanostore.Config{
    Dimensions: []nanostore.DimensionConfig{
        {
            Name:         "status",
            Type:         nanostore.Enumerated,
            Values:       []string{"pending", "active", "done"},
            DefaultValue: "pending",
        },
        {
            Name:         "priority", 
            Type:         nanostore.Enumerated,
            Values:       []string{"low", "medium", "high"},
            DefaultValue: "medium",
        },
    },
}

store, err := nanostore.New("/path/to/store.json", config)
if err != nil {
    log.Fatal(err)
}
```

### After (TypedStore API)

```go
import "github.com/arthur-debert/nanostore/nanostore/api"
import "github.com/arthur-debert/nanostore/nanostore"

// Automatic configuration from struct tags
type Task struct {
    nanostore.Document                                    // Required embedded field
    Status   string `values:"pending,active,done" default:"pending"`
    Priority string `values:"low,medium,high" default:"medium"`
}

store, err := api.NewFromType[Task]("/path/to/store.json")
if err != nil {
    log.Fatal(err)
}
```

## Key Migration Patterns

### 1. Document Operations

#### Creating Documents

**Before:**
```go
uuid, err := store.Add("My Task", map[string]interface{}{
    "status":   "pending",
    "priority": "high",
    "_data.assignee": "alice",
})
```

**After:**
```go
task := &Task{
    Document: nanostore.Document{Title: "My Task"},
    Status:   "pending", 
    Priority: "high",
    Assignee: "alice", // Custom fields stored automatically
}
uuid, err := store.Create(task)
```

#### Retrieving Documents

**Before:**
```go
doc, err := store.GetByID("1")
status := doc.Dimensions["status"].(string)
assignee := doc.Dimensions["_data.assignee"].(string)
```

**After:**
```go
task, err := store.Get("1")
status := task.Status     // Type-safe access
assignee := task.Assignee // Type-safe access
```

#### Updating Documents

**Before:**
```go
err := store.UpdateByID("1", nanostore.UpdateRequest{
    Dimensions: map[string]interface{}{
        "status": "done",
    },
})
```

**After:**
```go
task, err := store.Get("1")
task.Status = "done"
err = store.Update("1", task)
```

### 2. Querying and Filtering

#### Basic Filtering

**Before:**
```go
docs, err := store.List(types.ListOptions{
    Filters: map[string]interface{}{
        "status": "active",
        "priority": "high",
    },
})
```

**After:**
```go
tasks, err := store.Query().
    Status("active").
    Priority("high").
    Find()
```

#### Advanced Filtering

**Before:**
```go
docs, err := store.List(types.ListOptions{
    Filters: map[string]interface{}{
        "status": []string{"pending", "active"}, // OR operation
        "_data.assignee": "alice",
    },
    OrderBy: []types.OrderClause{
        {Column: "created_at", Descending: true},
    },
    Limit: 10,
})
```

**After:**
```go
tasks, err := store.Query().
    StatusIn("pending", "active").        // OR operation
    Data("assignee", "alice").            // Custom data fields
    OrderByDesc("created_at").
    Limit(10).
    Find()
```

### 3. Complex Queries

#### WHERE Clauses

**Before:**
```go
count, err := store.DeleteWhere(
    "status = ? AND created_at < ? AND (_data.assignee IS NULL)",
    "archived", 
    time.Now().AddDate(0, -6, 0),
)
```

**After:**
```go
count, err := store.Query().
    Where("status = ? AND created_at < ? AND (_data.assignee IS NULL)",
          "archived", time.Now().AddDate(0, -6, 0)).
    Delete()
```

#### NOT Operations

**Before:**
```go
docs, err := store.List(types.ListOptions{
    Filters: map[string]interface{}{
        "status": []string{"pending", "active"}, // Everything except "done"
    },
})
```

**After:**
```go
tasks, err := store.Query().
    StatusNot("done").    // Automatically includes all other known values
    Find()
```

## Advanced Migration Scenarios

### 1. Hierarchical Dimensions

**Before:**
```go
config := nanostore.Config{
    Dimensions: []nanostore.DimensionConfig{
        {
            Name:     "parent_id",
            Type:     nanostore.Hierarchical,
            RefField: "parent_id",
        },
    },
}
```

**After:**
```go
type Task struct {
    nanostore.Document
    ParentID string `dimension:"parent_id,ref"`  // Hierarchical dimension
}
```

### 2. Complex Prefix Mappings

**Before:**
```go
config := nanostore.Config{
    Dimensions: []nanostore.DimensionConfig{
        {
            Name:     "status",
            Values:   []string{"pending", "active", "done", "archived"},
            Prefixes: map[string]string{
                "done":     "d",
                "archived": "a", 
                "active":   "act",
            },
        },
    },
}
```

**After:**
```go
type Task struct {
    nanostore.Document
    Status string `values:"pending,active,done,archived" prefix:"done=d,archived=a,active=act"`
}
```

### 3. Custom Data Fields

**Before:**
```go
uuid, err := store.Add("Task", map[string]interface{}{
    "status": "pending",
    "_data.assignee": "alice",
    "_data.estimate": 5,
    "_data.tags": []string{"urgent", "backend"},
})
```

**After:**
```go
type Task struct {
    nanostore.Document
    Status   string   `values:"pending,active,done"`
    Assignee string   // Automatically stored as _data.assignee
    Estimate int      // Automatically stored as _data.estimate  
    Tags     []string // Automatically stored as _data.tags
}

task := &Task{
    Document: nanostore.Document{Title: "Task"},
    Status:   "pending",
    Assignee: "alice", 
    Estimate: 5,
    Tags:     []string{"urgent", "backend"},
}
uuid, err := store.Create(task)
```

## Raw Operations (When You Need Them)

For cases where you need direct access to the underlying document structure:

### Adding Documents Without Type Constraints

**Before:**
```go
uuid, err := store.Add("Legacy Doc", map[string]interface{}{
    "old_field": "value",
    "status": "unknown_status", // Not in enum
})
```

**After:**
```go
uuid, err := store.AddRaw("Legacy Doc", map[string]interface{}{
    "old_field": "value",
    "status": "unknown_status", // Bypasses validation
})
```

### Accessing Raw Document Data

**Before:**
```go
doc, err := store.GetByID("1")
allDimensions := doc.Dimensions
```

**After:**
```go
doc, err := store.GetRaw("1")           // Raw document access
dimensions, err := store.GetDimensions("1") // Just dimensions
metadata, err := store.GetMetadata("1")     // Just metadata
```

## Testing Migration

### Time Function Override

**Before:**
```go
testStore := store.(nanostore.TestStore)
testStore.SetTimeFunc(func() time.Time {
    return fixedTime
})
```

**After:**
```go
err := store.SetTimeFunc(func() time.Time {
    return fixedTime
})
```

## Advanced Operations Migration

### ID Resolution

**Before:**
```go
uuid, err := store.ResolveUUID("1.2.c3")
```

**After:**
```go
uuid, err := store.ResolveUUID("1.2.c3")  // Same interface!
```

### Bulk UUID Operations

**Before:**
```go
// Update multiple documents by UUIDs
count, err := store.UpdateByUUIDs([]string{uuid1, uuid2}, nanostore.UpdateRequest{
    Dimensions: map[string]interface{}{
        "status": "completed",
    },
})

// Delete multiple documents by UUIDs  
count, err := store.DeleteByUUIDs([]string{uuid1, uuid2})
```

**After:**
```go
// Update multiple documents by UUIDs - Type-safe!
updateData := &Task{Status: "completed"}
count, err := store.UpdateByUUIDs([]string{uuid1, uuid2}, updateData)

// Delete multiple documents by UUIDs - Same interface!
count, err := store.DeleteByUUIDs([]string{uuid1, uuid2})
```

### Custom ListOptions (Full Flexibility)

**Before:**
```go
docs, err := store.List(types.ListOptions{
    Filters: map[string]interface{}{
        "status": []string{"pending", "active"},
        "_data.assignee": "alice",
    },
    FilterBySearch: "important",
    OrderBy: []types.OrderClause{
        {Column: "created_at", Descending: true},
        {Column: "priority", Descending: false},
    },
    Limit:  &[]int{10}[0],
    Offset: &[]int{20}[0],
})
```

**After:**
```go
// Option 1: Direct ListOptions (same flexibility)
tasks, err := store.List(types.ListOptions{
    Filters: map[string]interface{}{
        "status": []string{"pending", "active"},
        "_data.assignee": "alice",
    },
    FilterBySearch: "important",
    OrderBy: []types.OrderClause{
        {Column: "created_at", Descending: true},
        {Column: "priority", Descending: false},
    },
    Limit:  &[]int{10}[0],
    Offset: &[]int{20}[0],
})

// Option 2: Fluent Query Builder (enhanced experience)
tasks, err := store.Query().
    StatusIn("pending", "active").
    Data("assignee", "alice").
    Search("important").
    OrderByDesc("created_at").
    OrderBy("priority").
    Limit(10).
    Offset(20).
    Find()
```

### Testing Utilities

**Before:**
```go
testStore := store.(nanostore.TestStore)
testStore.SetTimeFunc(func() time.Time {
    return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
})
```

**After:**
```go
err := store.SetTimeFunc(func() time.Time {
    return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
})
if err != nil {
    t.Fatalf("SetTimeFunc failed: %v", err)  // Better error handling
}
```

## Migration Checklist

### Step 1: Define Your Types
- [ ] Create struct types that embed `nanostore.Document`
- [ ] Add struct tags for dimensions: `values:"..."`, `default:"..."`, `prefix:"..."`
- [ ] Include custom fields as regular struct fields

### Step 2: Update Store Creation
- [ ] Replace `nanostore.New(filePath, config)` with `api.NewFromType[T](filePath)`
- [ ] Remove manual configuration objects
- [ ] Update import statements

### Step 3: Migrate Operations
- [ ] Replace `Add()` with `Create()`
- [ ] Replace `GetByID()` with `Get()`
- [ ] Replace `UpdateByID()` with `Update()`
- [ ] Replace `List()` with `Query().Find()`

### Step 4: Update Query Patterns
- [ ] Replace filter maps with method chains
- [ ] Use typed methods like `Status()`, `Priority()` instead of generic filters
- [ ] Migrate complex filters to `Where()` clauses
- [ ] Update NOT operations to use `StatusNot()`, etc.
- [ ] Verify `ResolveUUID()` calls (same interface)
- [ ] Update bulk operations: `UpdateByUUIDs()`, `DeleteByUUIDs()` (now type-safe)
- [ ] Migrate `List(ListOptions)` calls (unchanged interface)
- [ ] Update testing utilities: `SetTimeFunc()` (better error handling)

### Step 5: Test and Validate
- [ ] Run existing tests with new TypedStore code
- [ ] Verify all functionality works as expected
- [ ] Check performance characteristics
- [ ] Validate type safety catches errors at compile time

## Common Gotchas

### 1. Document Embedding is Required
```go
// ❌ Wrong - missing embedded Document
type Task struct {
    Status string `values:"pending,done"`
}

// ✅ Correct - includes embedded Document
type Task struct {
    nanostore.Document
    Status string `values:"pending,done"`
}
```

### 2. Struct Tags vs Manual Configuration
```go
// ❌ Manual config no longer needed
config := nanostore.Config{...}
store, err := nanostore.New(filePath, config)

// ✅ Configuration from struct tags
type Task struct {
    nanostore.Document
    Status string `values:"pending,done" default:"pending"`
}
store, err := api.NewFromType[Task](filePath)
```

### 3. Type Assertions No Longer Needed
```go
// ❌ Old way requires type assertions
status := doc.Dimensions["status"].(string)

// ✅ New way is type-safe
status := task.Status
```

## Benefits After Migration

✅ **Type Safety**: Compile-time checking prevents runtime errors  
✅ **Automatic Configuration**: No manual dimension configuration needed  
✅ **IntelliSense Support**: IDE autocomplete for all fields and methods  
✅ **Reduced Boilerplate**: Less code to write and maintain  
✅ **Better Performance**: Optimized operations and caching  
✅ **Consistent API**: Single way to do things reduces confusion  

## Frequently Asked Questions

### Q: Does TypedStore support all Direct Store operations?
**A: Yes!** TypedStore provides 100% feature parity with the Direct Store API. Every operation has an equivalent, often with enhanced type safety.

### Q: Can I still use complex ListOptions?
**A: Absolutely!** TypedStore supports `List(types.ListOptions)` with full flexibility. You also get the enhanced fluent query builder as a bonus.

### Q: Are bulk UUID operations available?
**A: Yes!** `UpdateByUUIDs()` and `DeleteByUUIDs()` are available with improved type safety - you pass typed structs instead of raw maps.

### Q: Can I resolve SimpleIDs to UUIDs?
**A: Yes!** `ResolveUUID()` has the exact same interface as the Direct Store API.

### Q: Do testing utilities work with TypedStore?
**A: Yes!** `SetTimeFunc()` is available with better error handling. No more type assertions needed.

### Q: What if I need raw document access?
**A: Use raw methods!** `AddRaw()`, `GetRaw()`, `GetDimensions()`, `GetMetadata()` provide direct access when needed.

### Q: Is performance the same or better?
**A: Better!** TypedStore includes performance optimizations like intelligent caching and reduced reflection overhead.

## Getting Help

- Check the [TypedStore API Reference](./typed-api.md)
- Review [sample applications](../samples/) for complete examples
- See the [GitHub issue #76](https://github.com/arthur-debert/nanostore/issues/76) for implementation details
- Ask questions in GitHub Discussions

Happy migrating! 🚀