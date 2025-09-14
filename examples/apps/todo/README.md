# Todo App Example

This is a hierarchical todo list application that demonstrates how nanostore's dynamic ID generation works in practice.

## Features

- **Hierarchical Structure**: Todo items can have sub-items
- **Status Management**: Items can be marked as pending or completed
- **Reversible Status**: Completed items can be reopened
- **Dynamic IDs**: IDs automatically adjust based on item status
- **Search**: Find items by title or body content
- **Move Operations**: Items can be moved between parents

## ID Generation Examples

### Default View (Pending Only)

```
○ 1. Groceries
  ○ 1.1. Milk
  ○ 1.2. Bread
  ○ 1.3. Eggs
○ 2. Pack for Trip
  ○ 2.1. Clothes
  ○ 2.2. Camera Gear
  ○ 2.3. Passport
```

### After Completing Bread

When viewing pending only:
```
○ 1. Groceries
  ○ 1.1. Milk
  ○ 1.2. Eggs      # Eggs moved up from 1.3 to 1.2
```

When viewing all items:
```
◐ 1. Groceries       # Mixed status symbol
  ○ 1.1. Milk
  ○ 1.2. Eggs
  ● 1.c1. Bread      # Completed items get 'c' prefix
```

## Command Line Usage

### Build the CLI

```bash
cd examples/apps/todo/cmd
go build -o todo
```

### Commands

```bash
# List todos (pending only by default)
./todo list

# List all todos including completed
./todo list --all

# Add a root todo
./todo add "Groceries"

# Add a sub-item
./todo add "Milk" -p 1

# Complete an item
./todo complete 1.2

# Reopen a completed item
./todo reopen 1.c1

# Search for items
./todo search "Gear"
./todo search "r" --all

# Move an item to a new parent
./todo move 1.2 2

# Delete an item (and optionally its children)
./todo delete 1.2
./todo delete 1 --cascade
```

## Running Tests

```bash
cd examples/apps/todo
go test -v
```

## How It Demonstrates Nanostore

1. **Dynamic ID Generation**: IDs like "1.2" are generated at query time based on the current state
2. **Status Namespacing**: Completed items get a "c" prefix (e.g., "1.c1")
3. **Hierarchical Support**: Multi-level nesting with dot notation (e.g., "1.2.3")
4. **Filtering**: The canonical view shows only pending items by default
5. **Search Integration**: Search respects the current filter and ID generation rules

## Key Insights

- IDs are not stored but generated dynamically based on query context
- Position within a status group is based on creation order
- Reopening an item loses its original position (by design)
- Parent items show mixed status when they have both pending and completed children
- The system handles ID resolution transparently when referencing items