# Changelog

## Test Applications (2024-01-14)

### Added

#### Todo App (`examples/apps/todo/`)
- Hierarchical todo list application demonstrating nanostore's capabilities
- Dynamic ID generation with parent-child relationships (1, 1.1, 1.2, etc.)
- Complete/reopen functionality with automatic ID renumbering
- Comprehensive test suite including unit and integration tests
- CLI interface for interactive usage
- Full documentation in README.md

#### Notes App (`examples/apps/notes/`)
- Note-taking application demonstrating flat document structure
- Archive/unarchive functionality using status-based prefixes
- Tag-based organization with tags stored in note body
- Simple test suite validating core functionality
- CLI interface for interactive usage
- Full documentation in README.md

### Infrastructure Updates
- Updated `scripts/test` to include both example apps
- Added `.gotestsum.sh` for gotestsum support
- Updated GitHub Actions workflow to test example apps in CI/CD
- Created main project README.md with testing documentation

### Key Discoveries
1. Nanostore's ID generation is context-sensitive - filtering breaks hierarchical numbering
2. Solution: Retrieve all documents and filter client-side for hierarchical IDs
3. Both apps validate that nanostore solves the dynamic ID problem it was designed for

### Commits
- `290048f` feat: add todo app example demonstrating nanostore capabilities (#13)
- `15c10ad` chore: update test scripts to include todo app tests
- `f8ac16e` feat: add notes app example demonstrating flat structure with tags
- `c404f5c` fix: remove unsupported features from notes app