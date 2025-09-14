Nanostore - Configurable Document and ID Store Library
======================================================

See docs/doc-id-store.txt for the original design specification.

1. Motivation

The problem: CLI tools need user-friendly, contiguous IDs (1, 2, 3...) but 
traditional CRUD systems generate non-contiguous UUIDs or auto-incrementing 
primary keys that become sparse after deletions. Users expect "todo complete 1"
to work, not "todo complete 7f2e8c9a-1234-...".

Additionally, applications need flexibility in how they partition and prefix
their IDs. A todo app might want 'p' for priority items, a project manager 
might want 'u' for urgent tasks, a note-taking app might want 'd' for drafts.

Example scenario: A project management tool where tasks have multiple 
dimensions - priority, status, team - and users need intuitive IDs:

    1. Setup CI/CD pipeline
    h1. Fix production bug (high priority)
    h2. Security audit (high priority) 
        h2.1. Review auth flow
        h2.2. Pen testing
    u1. Customer complaint (urgent)
    uw1. API breaking (urgent + work team)

The user should be able to type "complete h2.1" to mark the auth review done,
and the system should understand the hierarchical and dimensional context.

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
- **Configurable dimension system** - define your own ID partitioning logic
- Dynamic user-facing ID generation with custom prefixes (h1, uw2, p1.d3, etc.)
- Multiple dimension types:
  - Enumerated: predefined values with optional single-letter prefixes
  - Hierarchical: parent-child relationships with dot notation
- Alphabetical prefix ordering (hc1 and ch1 resolve to same document)
- Search across document title and body
- Cascade deletion (optional) for removing parent + children
- Parent relationship updates with circular reference prevention
- Bidirectional ID resolution (user ID -> UUID, UUID -> user ID in context)
- Transactional safety for all operations
- Dynamic schema generation based on configuration

4. API

Configuration for a project management system:

    config := nanostore.Config{
        Dimensions: []nanostore.DimensionConfig{
            {
                Name:         "priority",
                Type:         nanostore.Enumerated,
                Values:       []string{"low", "normal", "high", "urgent"},
                Prefixes:     map[string]string{"high": "h", "urgent": "u"},
                DefaultValue: "normal",
            },
            {
                Name:         "status",
                Type:         nanostore.Enumerated,
                Values:       []string{"todo", "in_progress", "done", "blocked"},
                Prefixes:     map[string]string{"in_progress": "p", "done": "d", "blocked": "b"},
                DefaultValue: "todo",
            },
            {
                Name:     "parent",
                Type:     nanostore.Hierarchical,
                RefField: "parent_task_id",
            },
        },
    }
    
    store, err := nanostore.New("tasks.db", config)

High-level usage:

    // Add tasks with dimension values
    epic, _ := store.Add("Q1 Product Launch", nil, nil) // Uses defaults
    task1, _ := store.Add("Design mockups", &epic, nil) // Inherits parent, uses defaults
    task2, _ := store.Add("Implement backend", &epic, map[string]string{
        "priority": "high",  // Set high priority
    })
    
    // Update status - changes ID prefix
    store.SetStatus(task1, nanostore.Status("done"))       // Now: 1.d1
    store.SetStatus(task2, nanostore.Status("in_progress")) // Now: 1.hp1 (high priority + in_progress)
    
    // List all documents
    docs, _ := store.List(nanostore.ListOptions{})
    
    // User sees:
    // 1. Q1 Product Launch
    //   1.d1. Design mockups (done)
    //   1.hp1. Implement backend (high priority, in_progress)
    
    // Resolve any ID format (including permutations)
    uuid, _ := store.ResolveUUID("1.hp1")  // Backend task UUID
    uuid, _ := store.ResolveUUID("1.ph1")  // Same document!
    
    // Update with dimension awareness
    store.Update(uuid, nanostore.UpdateRequest{
        Title: stringPtr("Implement REST API"),
        Dimensions: map[string]string{
            "priority": "urgent",  // Escalate to urgent
        },
    })
    
    // Delete with cascade
    store.Delete(epic, true)  // Removes parent + all children

The library dynamically generates the schema based on your configuration,
handles ID generation with proper partitioning, and maintains consistency
across all operations.

Core types:
- Config: Defines dimensions and their properties
- DimensionConfig: Individual dimension specification
- Store: Main handle for operations
- Document: Returned items with generated IDs based on dimensions

ID generation follows alphabetical ordering of dimension names. For example,
with priority and status dimensions, a high-priority done task gets ID "hd1"
regardless of whether you think of it as "high+done" or "done+high".