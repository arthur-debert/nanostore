# Notes App

A note-taking application that demonstrates how nanostore's generic document store can be configured for a different domain than task management. This example shows how the same underlying system can power a notes app with archiving functionality.

## Features

- **Two-state lifecycle**: live → archived
- **Dynamic ID prefixes**:
  - Live notes: `1`, `2`, `3`...
  - Archived notes: `c1`, `c2`, `c3`...
- **Tagging system**: Organize notes with multiple tags
- **Search**: Search by content
- **Delete**: Hard delete removes notes completely

## Key Differences from Todo App

1. **Flat structure**: No hierarchical relationships (todo app has parent-child)
2. **Tag-based organization**: Uses tags stored in body instead of parent-child relationships
3. **Archive vs Complete**: Uses different terminology but same underlying status system
4. **Hard delete**: Delete removes items completely rather than soft delete

## Usage

### Building

```bash
cd cmd
go build -o notes
```

### Commands

```bash
# Add a note with tags
./notes add "Meeting notes" -c "Discussed project timeline" -t "work,meetings"

# List notes
./notes list
./notes list --archived  # Include archived notes

# Archive and unarchive
./notes archive 1
./notes unarchive c1

# Delete (permanent)
./notes delete 2

# Update tags
./notes tag 1 "work,important,followup"

# Search
./notes search "project"
./notes search "meeting" --archived
```

## ID Generation Examples

Starting with these notes:
```
1. Project ideas
2. Meeting notes
3. Shopping list
```

After archiving note 2:
```
1. Project ideas
2. Shopping list

Archived:
c1. Meeting notes
```

After deleting note 1:
```
1. Shopping list

Archived:
c1. Meeting notes
```

## Implementation Notes

### Configuration

The notes app uses nanostore's TodoConfig() for simplicity, demonstrating how the same configuration can be repurposed:
- "pending" status is used for active notes (no prefix)
- "completed" status is used for archived notes ("c" prefix)
- No hierarchical dimension is used (flat structure)

A custom configuration could define domain-specific dimensions:
```go
config := nanostore.Config{
    Dimensions: []nanostore.DimensionConfig{
        {
            Name:         "status",
            Type:         nanostore.Enumerated,
            Values:       []string{"active", "archived", "pinned"},
            Prefixes:     map[string]string{"archived": "a", "pinned": "p"},
            DefaultValue: "active",
        },
    },
}
```

### Tag Storage

Tags are stored in the `data` field as JSON, allowing flexible tag management without requiring additional database schema.

## Testing

Run the comprehensive test suite:

```bash
go test -v
```

The tests cover:
- Basic CRUD operations
- Status transitions
- Pinning behavior
- Tag filtering
- Search functionality
- ID renumbering scenarios