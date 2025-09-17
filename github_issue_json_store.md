# Redesign: JSON File Store with Smart IDs

## Problem

The current SQL-based implementation has become too complex:
- ID generation requires complex window functions that don't work well with ORMs
- The Ent migration resulted in fetching all data into memory anyway
- For the target use case (<1000 items), SQL overhead isn't justified

## Solution

Implement nanostore as a simple JSON file store with these core principles:

### 1. Keep Existing Schema Definition
```go
type TodoItem struct {
    Document
    Status   string `values:"pending,active,done" prefix:"done=d" default:"pending"`
    Priority string `values:"low,medium,high" prefix:"high=h" default:"medium"`
    ParentID string `dimension:"parent_id,ref"`
}
```

### 2. JSON File Storage
- Always use file storage (no in-memory option)
- Simple JSON structure: `{"items": [...], "metadata": {...}}`
- File locking for concurrent access (3 second timeout, 3 retry attempts)

### 3. Smart ID Generation via Canonical View
- Define a "canonical view" (e.g., `status=pending, orderBy=created_at`)
- Build ID mapping from this canonical view: `{"1": "uuid-123", "1.1": "uuid-456"}`
- All ID resolution goes through this canonical map
- Different queries show different IDs (expected behavior)

### 4. Simple Operations
```go
// All operations: lock → read → modify → write → unlock
store, err := nanostore.New[TodoItem]("todos.json")
id, err := store.Create("Buy milk", &TodoItem{})
todos, err := store.Query().Status("pending").Find() 
err := store.Update("1.2", &TodoItem{Status: "done"})
```

## Benefits
- **Simple**: ~500 lines of code vs thousands
- **Fast**: 2ms operations for typical usage
- **Correct**: File locking handles concurrency
- **Debuggable**: Just a JSON file you can inspect

## Non-Goals
- Large datasets (>1000 items)
- Complex SQL queries  
- Multiple concurrent writers

This design optimizes for the 99% use case: terminal apps managing 10s-100s of hierarchical items with human-friendly IDs.