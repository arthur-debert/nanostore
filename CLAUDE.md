# CLAUDE.md - Development Guidelines for Nanostore

## Project Overview

Nanostore is a document and ID store library that replaces pkg/idm and parts of pkg/too/store. It uses SQLite to manage document storage and dynamically generate user-facing, contiguous IDs.

## Development Guidelines

### Branching Strategy - CRITICAL

1. **NEVER work directly on main** for non-trivial changes
2. **ALWAYS create a feature branch** for:
   - Architectural changes (type system, interfaces)
   - New features or significant refactoring
   - Any work requiring multiple commits
3. **Sequential PR workflow**:
   - Complete and merge PR before starting dependent work
   - If new work depends on unmerged PR, WAIT or work on something else
   - DO NOT create multiple overlapping PRs
4. **Branch naming**: `<type>/<description>` (e.g., `fix/sql-injection`, `refactor/type-safety`)

### Git Commit Practices

1. **Granular Commits**: Make small, focused commits for each logical change
2. **Issue References**: Always reference the GitHub issue in commit messages using `#<issue-number>`
3. **Commit Message Format**:
   ```
   <type>: <subject> (#<issue>)
   
   <optional body>
   ```
   Types: feat, fix, docs, style, refactor, test, chore

### Code Organization

1. **SQL Files**: 
   - Keep SQL in separate `.sql` files under `sql/` directory
   - Use go:embed to include SQL files
   - Never mix multiline SQL strings in Go code
   
2. **Package Structure**:
   - Public API in `api.go` and `types.go` only
   - Implementation details in `internal/engine/`
   - Clear separation between public and internal

3. **Testing**:
   - Test driven development
   - Often the same go file can be split in several test files
   - In tests the path encodes information - ensure significant mapping between name and content
   - Be sharp on the test file theme
   - Avoid brittle tests, don't test for things like messages or several pieces of data if one will do
   - Use in-memory SQLite for unit tests
   - Include fixtures for complex scenarios

### SQL Development

1. **Query Organization**:
   - One query per file
   - Descriptive filenames (e.g., `list_by_status.sql`)
   - Use SQL comments to document complex logic

2. **Schema Changes**:
   - Sequential migration files (001_initial.sql, 002_indexes.sql)
   - Forward-only migrations
   - Test migrations with real data

### Implementation Order

Follow the phases defined in the GitHub issue:
1. Foundation (structure, embedding, schema)
2. Core Operations (CRUD)
3. ID Generation (window functions)
4. ID Resolution (hierarchical parsing)
5. Testing (comprehensive coverage)

### Quality Checks

Before committing:
1. Run tests: `go test ./...`
2. Check formatting: `go fmt ./...`
3. Verify SQL syntax in .sql files
4. Ensure no SQL strings in Go code

### Key Design Principles

1. **IDs are transient**: Generated at query time, not stored
2. **SQL does the heavy lifting**: Use window functions for ID generation
3. **Clean API surface**: Minimal public interface
4. **Testability**: Every SQL query should be testable in isolation

## Current Task Tracking

Use GitHub issue #1 for tracking implementation progress. Update issue checkboxes as phases are completed.
