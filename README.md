# Nanostore

A document store library that generates human-friendly, hierarchical IDs for interactive applications. Nanostore uses JSON file storage with automatic ID generation, making it ideal for command-line tools, personal productivity applications, and small-scale interactive systems.

## Key Features

- **Human-Friendly IDs**: Generates sequential IDs like `1`, `1.1`, `h2.3` instead of UUIDs
- **Hierarchical Structure**: Built-in parent-child relationships with automatic ID nesting
- **Dynamic Prefixes**: IDs change based on status (e.g., `1` becomes `d1` when completed)
- **Type-Safe API**: Define your data model with struct tags, get compile-time safety and automatic configuration
- **JSON Storage**: Human-readable persistence with file locking for concurrent access
- **Zero Dependencies**: Simple deployment with single JSON file storage

## The Problem Nanostore Solves

Interactive command-line applications need IDs that users can easily type and remember. While UUIDs work great for internal systems, asking users to type `dbf15ed6-bcd4-4528-8831-5bf56039d327` is poor UX.

Nanostore generates stable, sequential IDs that maintain hierarchical relationships:

```bash
$ todo list
  ○ 1. Groceries
    ○ 1.1. Milk
    ○ 1.2. Bread
  ○ 2. Pack for Trip
    ○ 2.1. Clothes
    ○ h2.2. Passport  # h = high priority

$ todo complete 1.1
$ todo list
  ○ 1. Groceries
    ● d1.1. Milk      # d = done, maintains hierarchy
    ○ 1.2. Bread
  ○ 2. Pack for Trip
    ○ 2.1. Clothes
    ○ h2.2. Passport
```

## Quick Start

### Type-Safe API with Struct Tags

```go
import (
    "github.com/arthur-debert/nanostore/nanostore"
    "github.com/arthur-debert/nanostore/nanostore/api"
)

// Define your data model with struct tags
type TodoItem struct {
    nanostore.Document  // Required embedding
    
    Status   string `values:"pending,active,done" prefix:"done=d" default:"pending"`
    Priority string `values:"low,medium,high" prefix:"high=h" default:"medium"`
    ParentID string `dimension:"parent_id,ref"`
    
    // Non-dimension fields stored as custom data
    AssignedTo  string
    DueDate     time.Time
    Description string
}

// Create typed store - configuration generated automatically from struct tags
store, err := api.NewFromType[TodoItem]("todos.json")
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// Type-safe operations
id, err := store.Create("Buy groceries", &TodoItem{
    Priority:    "high",
    AssignedTo:  "alice",
    Description: "Weekly shopping",
})

// Type-safe queries with fluent interface
urgentTasks, err := store.Query().
    Priority("high").
    Status("pending").
    OrderBy("created_at").
    Find()

// Update with type safety  
task, err := store.Get(id)
task.Status = "done"
err = store.Update(id, task)

// Output:
fmt.Printf("%s. %s (assigned to %s)\n", task.SimpleID, task.Title, task.AssignedTo)
// h1. Buy groceries (assigned to alice)
```

## Core Concepts

### Dimensions

Dimensions define how IDs are partitioned and generated:

- **Enumerated**: Predefined values with optional prefixes (`status: pending,done`)
- **Hierarchical**: Parent-child relationships (`parent_id` references)

### ID Generation

IDs are generated dynamically based on:
1. **Canonical View**: Default filters (e.g., show only active items)
2. **Partitioning**: Group by dimension values  
3. **Ordering**: Sequential numbering within each partition
4. **Prefixes**: Applied based on dimension values

### Two-Tier ID System

- **Internal UUIDs**: Stable, never change, used for storage
- **User-Facing IDs**: Generated dynamically, hierarchical, human-friendly

## When to Use Nanostore

### ✅ Great For

- Personal productivity tools (todo lists, note managers)
- Command-line applications requiring user input
- Small team tools with modest data (< 10k items)
- Applications needing hierarchical organization
- Prototypes requiring quick human-readable IDs

### ❌ Not Suitable For

- Web applications with concurrent users  
- High-volume data processing
- Applications requiring complex analytical queries
- Multi-tenant systems with strict isolation
- Systems needing ACID transactions

## Installation

```bash
go get github.com/arthur-debert/nanostore/nanostore
```

## Documentation

- **[Problem & Design](docs/design-and-problem-space.txt)**: Why nanostore exists and how it works
- **[Technical Architecture](docs/technical-architecture.txt)**: Implementation details and performance characteristics  
- **[In-Depth Guide](docs/in-depth-guide.txt)**: Complete tutorial building a hierarchical todo application with the type-safe API

## Architecture

Nanostore is organized as modular packages:

```
github.com/arthur-debert/nanostore/
├── types/       # Core data structures and interfaces
├── search/      # Search and filtering functionality  
├── nanostore/   # Main package with TypedStore API
│   └── api/     # Type-safe declarative API implementation
└── docs/        # Comprehensive documentation and guides
```

The main entry point is the `TypedStore[T]` API in the `nanostore` package, which provides type-safe operations with automatic configuration generation from struct tags.

## Performance & Limitations

- **Scale**: Optimized for hundreds to low thousands of documents
- **Memory**: Entire dataset loaded into memory for operations  
- **Concurrency**: Single-writer with file locking, not designed for concurrent access
- **Storage**: JSON file format, human-readable but not space-efficient
- **Query**: O(n) filtering, O(n log n) ID generation where n = document count

## Examples

See `docs/in-depth-guide.txt` for a complete todo application implementation demonstrating:

- Hierarchical task organization
- Status transitions with ID prefixes  
- Type-safe queries and updates
- Search and filtering
- Bulk operations

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

---

**Nanostore**: Making interactive applications more human-friendly, one ID at a time.