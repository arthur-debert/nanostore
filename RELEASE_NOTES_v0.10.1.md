# Nanostore v0.10.1 Release Notes

## Export Functionality

This release introduces comprehensive export functionality for nanostore, enabling applications to create portable archives of their data.

### Key Features

- **Zip Archive Export**: Creates structured archives containing both database metadata and individual document content
- **Flexible Filtering**: Export all documents, specific IDs, or use dimension-based filters
- **CLI Integration**: Easy-to-integrate Cobra commands for adding export to any application
- **Two-Phase Design**: Testable architecture that separates data generation from archive creation

### How It Works

The export feature creates a zip archive with:
- `db.json`: Complete database representation with all documents and metadata
- Individual `.txt` files: One per document, named as `<uuid>-<order>-<title>.txt`

### Usage

#### Programmatic API

```go
import "github.com/arthur-debert/nanostore/nanostore"

// Export all documents
archivePath, err := nanostore.Export(store, nanostore.ExportOptions{})

// Export specific documents
archivePath, err := nanostore.Export(store, nanostore.ExportOptions{
    IDs: []string{"1", "h2", "3.1"},
})

// Export with dimension filters
archivePath, err := nanostore.Export(store, nanostore.ExportOptions{
    DimensionFilters: map[string]interface{}{
        "status": "completed",
        "priority": "high",
    },
})

// Export to specific path
err := nanostore.ExportToPath(store, options, "/path/to/backup.zip")
```

#### CLI Integration

Applications using Cobra can add export functionality with one line:

```go
rootCmd.AddCommand(nanostore.CreateExportCommand("/path/to/store.json", config))
```

This provides users with:
```bash
myapp export                    # Export all documents
myapp export 1 c2              # Export specific documents
myapp export --output backup.zip # Export to specific file
```

### Example Implementation

See the included `samples/todos` application for a complete example of integrating export functionality into a hierarchical todo manager.

### Building and Testing

Use the included build script to compile the sample CLIs:
```bash
./build.sh
./.bin/todos add "My task"
./.bin/todos export --output my-backup.zip
```

### Documentation

Detailed documentation available in `docs/export.txt` covering:
- Export structure and file naming conventions
- Programming interface with examples
- CLI integration patterns
- Testing strategies
- Error handling

This release maintains backward compatibility while adding powerful data portability features to nanostore applications.