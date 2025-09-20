# Storage Layer Fix Plan

## Current Problem
We have duplicate storage implementations:
1. `impl_store_json.go` (812 lines) - still contains all file I/O logic
2. `storage/internal/json_storage.go` (310 lines) - duplicate implementation

This adds 310 unnecessary lines instead of extracting functionality.

## Root Cause
The Storage interface we created is too high-level (document-oriented) and doesn't match how impl_store_json.go actually works:
- impl_store_json.go loads ALL documents into memory and saves them as a batch
- The Storage interface assumes individual document operations (Save, Update, Delete)
- This mismatch led to creating a duplicate implementation instead of extracting

## Correct Approach
The storage layer should be a thin persistence layer that:
1. Handles file I/O and locking
2. Manages the JSON serialization/deserialization
3. Does NOT manage documents, UUIDs, or business logic

## Implementation Plan

### Option 1: Simple File Storage (Recommended)
Create a minimal storage interface that matches current usage:
```go
type Storage interface {
    // Load reads the entire store data from disk
    Load() (*StoreData, error)
    
    // Save writes the entire store data to disk
    Save(data *StoreData) error
    
    // Close releases resources
    Close() error
}

type StoreData struct {
    Documents []types.Document
    Metadata  Metadata
}
```

This matches exactly how impl_store_json.go works today.

### Option 2: Delete Current Storage Implementation
Since the current storage implementation duplicates functionality:
1. Delete `storage/internal/json_storage.go`
2. Delete `storage/json.go` 
3. Keep only `storage/storage.go` with the interface
4. Implement the simpler interface in impl_store_json.go

## Benefits
- Removes 310+ lines of duplicate code
- Keeps the refactoring focused on organization, not rewriting
- Maintains current behavior exactly
- Can still swap storage backends later