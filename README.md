# Nanostore

A generic document store library that uses SQLite to manage document storage with dynamically generated, human-friendly IDs. Nanostore is domain-agnostic and can be configured for any document-based application.

## Features

- Dynamic ID generation using SQL window functions
- Configurable dimensions for custom taxonomies (enumerated and hierarchical)
- Hierarchical document structure with parent-child relationships
- Flexible ID prefixing based on dimension values (e.g., 1, 2, 3; c1, c2, c3)
- Generic document storage - not tied to any specific domain
- Thread-safe operations

## Important Considerations

- **Dynamic IDs**: User-facing IDs are generated at query time and can shift when document status changes. See [Batch Operations Guide](docs/batch-operations.md) for handling this correctly.

## Installation

```bash
go get github.com/arthur-debert/nanostore/nanostore
```

## Quick Start

```go
import "github.com/arthur-debert/nanostore/nanostore"

// Define your domain-specific configuration
config := nanostore.Config{
    Dimensions: []nanostore.DimensionConfig{
        {
            Name:         "status",
            Type:         nanostore.Enumerated,
            Values:       []string{"active", "archived"},
            Prefixes:     map[string]string{"archived": "a"},
            DefaultValue: "active",
        },
        {
            Name:     "category",
            Type:     nanostore.Hierarchical,
            RefField: "parent_id",
        },
    },
}

// Create a new store
store, err := nanostore.New("mydata.db", config)
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// Add a document
uuid, err := store.Add("My Document", map[string]interface{}{
    "status": "active",
})

// Add a child document
childID, err := store.Add("Child Document", map[string]interface{}{
    "parent_id": uuid,
})

// List documents with filters
docs, err := store.List(nanostore.ListOptions{
    Filters: map[string]interface{}{"status": "active"},
})

// Update document dimensions (note: IDs may shift after this!)
err = store.Update(uuid, nanostore.UpdateRequest{
    Dimensions: map[string]string{"status": "archived"},
})
```

**Note**: When working with multiple documents, always read the [Batch Operations Guide](docs/batch-operations.md) to understand how ID shifting affects batch operations.

## Testing

The project includes comprehensive tests for both the core library and example applications.

### Running Tests

Using the test script (recommended):
```bash
./scripts/test
```

Using go test directly:
```bash
# Run core library tests
go test ./... -v

# Run todo app example tests
cd examples/apps/todo && go test ./... -v
```

Using gotestsum:
```bash
# Install gotestsum if you haven't already
go install gotest.tools/gotestsum@latest

# Run all tests with gotestsum
gotestsum --raw-command -- ./.gotestsum.sh
```

### CI/CD

The GitHub Actions workflow automatically runs all tests on push and pull requests. See `.github/workflows/go.yml` for the configuration.

## Example Applications

Nanostore comes with two example applications that demonstrate how to build domain-specific applications on top of the generic document store:

### Todo Application

A hierarchical todo list with completion status tracking:

```bash
cd examples/apps/todo/cmd
go build -o todo

# Add todos
./todo add "Groceries"
./todo add -p 1 "Milk"
./todo add -p 1 "Bread"

# List todos
./todo list
./todo list --all  # Include completed items

# Complete a todo
./todo complete 1.2

# Search
./todo search "milk"
```

### Notes Application

A note-taking app with archiving and tagging:

```bash
cd examples/apps/notes/cmd
go build -o notes

# Add notes
./notes add "Meeting Notes" --body "Discuss project timeline"
./notes add "Ideas" --tags "project,brainstorm"

# List and search
./notes list
./notes search "project"

# Archive notes
./notes archive 1
./notes list --archived
```

Both examples show how nanostore's generic API can be adapted for specific use cases. See `examples/apps/todo/README.md` and `examples/apps/notes/README.md` for more details.

## Migration from v0.2.0

If you're upgrading from the todo-specific v0.2.0, see [docs/migration-to-nanostore-v0.3.0.txt](docs/migration-to-nanostore-v0.3.0.txt) for a complete migration guide.

## License

MIT