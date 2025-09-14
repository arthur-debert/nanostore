# Nanostore

A document and ID store library that uses SQLite to manage document storage with dynamically generated user-facing, contiguous IDs.

## Features

- Dynamic ID generation using SQL window functions
- Hierarchical document structure with parent-child relationships
- Status-based ID namespacing (e.g., pending: 1, 2, 3; completed: c1, c2, c3)
- Configurable dimensions for custom taxonomies
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

// Create a new store with default configuration
store, err := nanostore.New("mydata.db", nanostore.DefaultTestConfig())
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// Add a document
uuid, err := store.Add("My Document", nil, nil)

// List documents
docs, err := store.List(nanostore.ListOptions{})

// Complete a document (note: IDs may shift after this!)
err = store.SetStatus(uuid, nanostore.StatusCompleted)
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

## Examples

### Todo Application

A full-featured hierarchical todo list application demonstrating nanostore's capabilities:

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

See `examples/apps/todo/README.md` for more details.

## License

MIT