# Batch Operations in Nanostore

## Overview

When performing batch operations on multiple documents using their user-facing IDs, it's crucial to understand how nanostore's dynamic ID generation affects ID resolution. This document explains the behavior and provides best practices for handling batch operations correctly.

## The ID Shifting Problem

Nanostore generates user-facing IDs dynamically using SQL window functions. These IDs are **not stored** in the database but are calculated at query time based on the current state of documents. This means that when you change a document's status (e.g., from pending to completed), the IDs of remaining documents can shift.

### Example Scenario

Consider three pending todos:
```
1. First todo
2. Second todo  
3. Third todo
```

When you complete "First todo", the IDs immediately shift:
```
1. Second todo (was ID 2)
2. Third todo (was ID 3)
c1. First todo (completed)
```

## The Batch Resolution Problem

This ID shifting behavior can cause issues when performing batch operations. Consider this problematic pattern:

```go
// INCORRECT: Resolving IDs one at a time
ids := []string{"1", "3"}
for _, id := range ids {
    uuid, _ := store.ResolveUUID(id)
    store.SetStatus(uuid, nanostore.StatusCompleted)
}
```

What happens:
1. ID "1" is resolved to "First todo" and completed
2. IDs shift: "Second todo" becomes ID 1, "Third todo" becomes ID 2
3. ID "3" no longer exists - resolution fails!

## Best Practices for Batch Operations

### 1. Pre-resolve All IDs (Recommended)

Always resolve all user-facing IDs to UUIDs before performing any mutations:

```go
// CORRECT: Resolve all IDs first
ids := []string{"1", "3"}
var uuids []string

// Step 1: Resolve all IDs
for _, id := range ids {
    uuid, err := store.ResolveUUID(id)
    if err != nil {
        return fmt.Errorf("failed to resolve ID %s: %w", id, err)
    }
    uuids = append(uuids, uuid)
}

// Step 2: Perform all operations using UUIDs
for _, uuid := range uuids {
    err := store.SetStatus(uuid, nanostore.StatusCompleted)
    if err != nil {
        return fmt.Errorf("failed to complete item: %w", err)
    }
}
```

### 2. Use UUIDs Directly

When building applications on top of nanostore, consider storing and using UUIDs for operations that might be batched:

```go
// Store both user-facing ID and UUID
type TodoItem struct {
    UUID         string
    UserFacingID string
    Title        string
}

// Use UUID for operations
func (app *TodoApp) CompleteMultiple(uuids []string) error {
    for _, uuid := range uuids {
        if err := store.SetStatus(uuid, nanostore.StatusCompleted); err != nil {
            return err
        }
    }
    return nil
}
```

### 3. Reverse Order Processing

If you must resolve IDs individually, process them in reverse order to minimize shifting effects:

```go
// Process IDs in reverse order
ids := []string{"1", "2", "4"}
// Sort in descending order
sort.Slice(ids, func(i, j int) bool {
    return ids[i] > ids[j]
})

// Now process: "4", "2", "1"
for _, id := range ids {
    uuid, _ := store.ResolveUUID(id)
    store.SetStatus(uuid, nanostore.StatusCompleted)
}
```

### 4. Canonical Namespace Pattern

For operations that always work within a specific status namespace, consider using a canonical namespace approach:

```go
// Always work in the pending namespace
func CompleteByPendingIDs(store nanostore.Store, ids []string) error {
    // Get all pending items first
    pending, _ := store.List(nanostore.ListOptions{
        FilterByStatus: []nanostore.Status{nanostore.StatusPending},
    })
    
    // Build ID to UUID map
    idMap := make(map[string]string)
    for _, doc := range pending {
        idMap[doc.UserFacingID] = doc.UUID
    }
    
    // Resolve and complete
    for _, id := range ids {
        if uuid, ok := idMap[id]; ok {
            store.SetStatus(uuid, nanostore.StatusCompleted)
        }
    }
    return nil
}
```

## Testing Batch Operations

When testing batch operations, always verify that the correct items were affected:

```go
func TestBatchComplete(t *testing.T) {
    store, _ := nanostore.New(":memory:", config)
    
    // Create test data
    uuid1, _ := store.Add("Task A", nil, nil)
    uuid2, _ := store.Add("Task B", nil, nil)
    uuid3, _ := store.Add("Task C", nil, nil)
    
    // Complete tasks 1 and 3
    CompleteMultiple(store, []string{"1", "3"})
    
    // Verify by UUID, not by ID!
    doc1, _ := store.Get(uuid1)
    doc3, _ := store.Get(uuid3)
    
    if doc1.Status != nanostore.StatusCompleted {
        t.Error("Task A should be completed")
    }
    if doc3.Status != nanostore.StatusCompleted {
        t.Error("Task C should be completed")
    }
}
```

## Batch Delete Operations

Nanostore provides several methods for efficient bulk deletion of documents:

### 1. DeleteCompleted()

Removes all documents with completed status:

```go
deleted, err := store.DeleteCompleted()
fmt.Printf("Deleted %d completed items\n", deleted)
```

### 2. DeleteByDimension()

Removes all documents matching a specific dimension value:

```go
// Delete by status
deleted, err := store.DeleteByDimension("status", "archived")

// Delete by priority
deleted, err := store.DeleteByDimension("priority", "low")

// Delete by custom dimension
deleted, err := store.DeleteByDimension("category", "spam")
```

### 3. DeleteWhere()

Provides flexible deletion with custom SQL WHERE clauses:

```go
// Simple condition
deleted, err := store.DeleteWhere("priority = ?", "low")

// Multiple conditions
deleted, err := store.DeleteWhere("status = ? AND priority = ?", "draft", "low")

// OR conditions
deleted, err := store.DeleteWhere("status = ? OR category = ?", "archived", "trash")

// Pattern matching
deleted, err := store.DeleteWhere("title LIKE ?", "%DRAFT%")

// IN clause
deleted, err := store.DeleteWhere("category IN (?, ?, ?)", "spam", "trash", "archive")

// Date-based deletion (if you track dates in a dimension)
deleted, err := store.DeleteWhere("created_date < ?", "2024-01-01")
```

### Safety Considerations

1. **Validation**: `DeleteByDimension` validates that the dimension exists and the value is valid for enumerated dimensions
2. **Transactions**: All delete operations use transactions for atomicity
3. **Cascading**: Hierarchical relationships respect `ON DELETE CASCADE` constraints
4. **No Undo**: Delete operations are permanent - consider implementing soft deletes using status dimensions instead

## Summary

- **IDs are dynamic**: They change when document status changes
- **Always pre-resolve**: Resolve all IDs to UUIDs before any mutations
- **Use UUIDs when possible**: They are stable and don't shift
- **Test with UUIDs**: Verify operations using UUIDs, not user-facing IDs
- **Bulk deletes**: Use `DeleteByDimension` or `DeleteWhere` for efficient batch deletions
- **Document the behavior**: Make sure your API users understand ID shifting

By following these practices, you can safely perform batch operations while working with nanostore's dynamic ID system.