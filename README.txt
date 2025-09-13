Nanostore - Document and ID Store Library
==========================================

See docs/doc-id-store.txt for the original design specification.

1. Motivation

The problem: CLI tools need user-friendly, contiguous IDs (1, 2, 3...) but 
traditional CRUD systems generate non-contiguous UUIDs or auto-incrementing 
primary keys that become sparse after deletions. Users expect "todo complete 1"
to work, not "todo complete 7f2e8c9a-1234-...".

Example scenario: A todo list where users can create nested todos, mark them 
complete, and reference them by simple numbers. When a user runs "list", they 
see:

    1. Fix login bug
    2. Write documentation  
        2.1. API docs
        2.2. User guide
    3. Deploy to staging
    c1. Review PR #123 (completed)

The user should be able to type "complete 2.1" to mark "API docs" as done, 
and "complete 2" should handle the parent relationship appropriately.

The traditional approach requires complex orchestration between a document 
store and an ID manager, with imperative filtering/sorting in application 
code. This creates architectural complexity and performance bottlenecks.

2. Design Decisions

Architecture: Single SQLite database handles both document storage and dynamic 
ID generation. IDs are view properties, not data properties - calculated at 
query time using SQL window functions.

Key tradeoffs:
- SQLite over distributed database: Targeting small-scale interactive 
  applications on workstations. No concurrency handling needed - single user,
  single process model is the sweet spot.
- Row number generation in SQL vs. application code: SQL window functions 
  (ROW_NUMBER() OVER PARTITION) handle the heavy lifting. Clean, declarative,
  and leverages the database's optimization.
- Hierarchical ID performance: For 3+ levels deep, we don't optimize UUID 
  resolution. Acceptable because typical use cases (todo lists, note taking)
  rarely exceed 2-3 levels of nesting.
- CGO dependency: mattn/go-sqlite3 introduces CGO, complicating cross-compilation
  slightly. Tradeoff accepted for SQLite's query capabilities.

Internal UUID vs. user-facing ID separation eliminates re-indexing on writes.
Writes are simple INSERT/UPDATE operations. Reads generate fresh IDs each time.

The ResolveUUID method recursively walks hierarchical IDs ("1.2.3") by 
executing multiple queries. Not optimized for deep hierarchies but handles 
common cases efficiently.

3. Features

- Document CRUD with hierarchical parent-child relationships
- Dynamic user-facing ID generation (1, 2, c1, 1.2, etc.)
- Status-based filtering and ID partitioning (pending vs completed items)
- Search across document title and body
- Cascade deletion (optional) for removing parent + children
- Parent relationship updates with circular reference prevention
- Bidirectional ID resolution (user ID -> UUID, UUID -> user ID in context)
- Transactional safety for all operations
- Schema migrations with embedded SQL files

4. API

Schema setup for a todo application (embedded in the library):

    sql/schema/001_initial.sql:
    CREATE TABLE documents (
        uuid TEXT PRIMARY KEY,
        title TEXT NOT NULL,
        body TEXT DEFAULT '',
        status TEXT NOT NULL DEFAULT 'pending' 
            CHECK (status IN ('pending', 'completed')),
        parent_uuid TEXT,
        created_at INTEGER NOT NULL,
        updated_at INTEGER NOT NULL,
        FOREIGN KEY (parent_uuid) REFERENCES documents(uuid) ON DELETE CASCADE
    );
    
    sql/schema/002_indexes.sql:
    CREATE INDEX idx_documents_status ON documents(status, created_at);
    CREATE INDEX idx_documents_parent ON documents(parent_uuid, created_at);
    CREATE INDEX idx_documents_search ON documents(title, body);

High-level usage for a todo application:

    store, err := nanostore.New("todos.db")  // Auto-runs migrations
    
    // Add root items
    parentID, _ := store.Add("Project Alpha", nil)
    store.Add("Write tests", &parentID)
    store.Add("Deploy", &parentID)
    
    // List current view
    docs, _ := store.List(nanostore.ListOptions{
        FilterByStatus: []nanostore.Status{nanostore.StatusPending},
    })
    
    // User sees:
    // 1. Project Alpha
    //   1.1. Write tests  
    //   1.2. Deploy
    
    // Complete a subtask
    uuid, _ := store.ResolveUUID("1.1")  // "Write tests" UUID
    store.SetStatus(uuid, nanostore.StatusCompleted)
    
    // Update parent relationship
    store.Update(uuid, nanostore.UpdateRequest{
        Title:    stringPtr("Write comprehensive tests"),
        ParentID: nil,  // Move to root level
    })
    
    // Delete with cascade
    store.Delete(parentID, true)  // Removes parent + all children

The library handles the complexity of maintaining contiguous IDs across 
different filtered views while providing a clean interface for typical 
document management operations.

Core types: Store (main handle), ListOptions (query parameters), 
Document (returned items with generated IDs), UpdateRequest (modification 
parameters).

Status-based partitioning means completed items get "c" prefixed IDs (c1, c2)
while pending items get regular numbers (1, 2). Hierarchical nesting uses 
dot notation (1.2.3).