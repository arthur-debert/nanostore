# Package Organization Analysis for Nanostore

## Overview

This document analyzes the current package structure of nanostore and proposes improvements to eliminate circular dependencies and improve code organization.

## Current Package Structure

```
github.com/arthur-debert/nanostore/
│
├── / (root package)
│   ├── logging.go
│   │   └── Exports: (logging utilities, likely internal)
│   │
│   └── internal/version/
│       └── version.go
│           └── Exports: (version information)
│
├── types/ (core type definitions)
│   ├── Exports:
│   │   ├── Document (struct)
│   │   ├── ListOptions (struct)
│   │   ├── OrderClause (struct)
│   │   ├── UpdateRequest (struct)
│   │   ├── Store (interface)
│   │   ├── DimensionType (type alias)
│   │   ├── Enumerated, Hierarchical (constants)
│   │   ├── DimensionMetadata (struct)
│   │   ├── Dimension (struct)
│   │   ├── DimensionSet (struct with methods)
│   │   ├── DimensionValue (struct)
│   │   ├── Partition (struct)
│   │   ├── PartitionMap (type)
│   │   ├── CanonicalView (struct)
│   │   └── Config (struct)
│   │
│   ├── Files:
│   │   ├── core.go - Document, Store interface, ListOptions
│   │   ├── dimension.go - Dimension types and DimensionSet
│   │   ├── partition.go - Partition types and operations
│   │   ├── canonical.go - CanonicalView for filtering
│   │   ├── config.go - Configuration structures
│   │   └── validation.go - Validation utilities
│   │
│   └── Dependencies: None (base types package)
│
├── nanostore/ (main implementation)
│   ├── Exports:
│   │   ├── New() - Store constructor
│   │   ├── Store (interface, re-exported from types)
│   │   ├── Document (struct, local version)
│   │   ├── Config (struct)
│   │   ├── DimensionConfig (struct)
│   │   ├── ListOptions (struct, local version)
│   │   ├── UpdateRequest (struct, local version)
│   │   └── DimensionType constants (re-exported)
│   │
│   ├── Internal Components:
│   │   ├── impl_store_json.go - JSON file-based Store implementation
│   │   ├── lock_manager.go - File locking for concurrency
│   │   ├── command_preprocessor.go - Smart ID resolution
│   │   └── test_helpers.go - Testing utilities
│   │
│   └── Dependencies:
│       └── types.* (uses types package for core structures)
│
├── search/ (search functionality)
│   ├── Exports:
│   │   ├── Engine (struct implementing Searcher)
│   │   ├── Searcher (interface)
│   │   └── Related types
│   │
│   └── Dependencies:
│       └── types.* (Document, Store)
│
└── samples/todos/ (example application)
```

## Key Issues Identified

### 1. Type Duplication
- `Document` exists in both `types` and `nanostore` packages
- `ListOptions`, `UpdateRequest`, `OrderClause` are duplicated
- Conversion functions like `ToTypesDocument` indicate the duplication problem

### 2. Mixed Responsibilities
The main `nanostore` package contains:
- Store interface definition
- JSON implementation details
- Command preprocessing
- Lock management
- Type definitions

This violates single responsibility principle.

### 3. Circular Dependency Risk
- `nanostore` defines its own types while also importing from `types`
- `command_preprocessor` has complex dependencies that could create cycles
- API packages depend on nanostore types instead of the base types

### 4. Poor Separation of Concerns
- Storage implementation mixed with interface definition
- ID management partially in main package, partially in `ids/`
- Test helpers scattered across packages

## Proposed Package Structure

```
nanostore/
│
├── types/ (shared domain types - no dependencies)
│   ├── Exports:
│   │   ├── Document, SimpleID, UUID types
│   │   ├── Dimension, DimensionSet, DimensionConfig
│   │   ├── ListOptions, UpdateRequest, OrderClause
│   │   ├── Partition, CanonicalView
│   │   └── Config
│   └── Dependencies: None
│
├── storage/ (persistence layer)
│   ├── Exports:
│   │   ├── Storage interface
│   │   ├── NewJSONStorage() → Storage
│   │   └── Transaction interface
│   ├── Dependencies:
│   │   └── types.Document
│   └── Internal: lock_manager.go, file operations
│
├── ids/ (ID management)
│   ├── Exports:
│   │   ├── Generator interface
│   │   ├── Transformer interface
│   │   ├── NewGenerator() → Generator
│   │   └── NewTransformer() → Transformer
│   └── Dependencies:
│       └── types.{Document, Partition, CanonicalView}
│
├── query/ (query processing)
│   ├── Exports:
│   │   ├── Processor interface
│   │   ├── NewProcessor() → Processor
│   │   └── Command types
│   └── Dependencies:
│       ├── types.{Document, ListOptions}
│       └── ids.Transformer
│
├── search/ (already well-separated)
│   ├── Exports: Engine, Searcher interface
│   └── Dependencies: types.Document
│
├── store/ (main orchestrator)
│   ├── Exports:
│   │   ├── Store interface
│   │   └── New() → Store
│   └── Dependencies:
│       ├── All other packages
│       └── Orchestrates everything
│
└── api/ (user-facing APIs)
    ├── Exports: TypedStore[T], declarative APIs
    └── Dependencies: store.Store, types.*
```

## Natural Package Themes

Based on functionality analysis, nanostore has these distinct responsibilities:

1. **Storage Layer** (`storage/`)
   - File-based persistence
   - Lock management
   - Transaction handling
   - Should be swappable (JSON today, SQLite tomorrow)

2. **ID Management** (`ids/`)
   - SimpleID generation algorithms
   - ID transformation and resolution
   - Partition-based numbering
   - Should be independent of storage

3. **Document Model** (`document/`)
   - Document type and operations
   - Dimension management
   - Validation rules
   - Core business logic

4. **Query Engine** (`query/`)
   - Command processing
   - SimpleID resolution
   - Filter application
   - Sort and pagination

5. **Search** (`search/`)
   - Full-text search
   - Indexing
   - Result ranking

6. **API Facades** (`api/`)
   - Public interfaces
   - Type-safe wrappers
   - Declarative configuration

7. **Configuration** (`config/`)
   - Store configuration
   - Dimension definitions
   - Validation rules

## Benefits of Proposed Structure

1. **Clear Dependencies**: Unidirectional flow from types → implementations → orchestrator → APIs
2. **No Circular Dependencies**: Each package has a clear role and minimal dependencies
3. **Better Testing**: Each package can be unit tested in isolation
4. **Easier Refactoring**: Clear boundaries make it easier to modify implementations
5. **Swappable Components**: Storage, ID generation, and search can be replaced independently

## Refactoring Approach

### Working Principles

1. **Small, Focused Commits**: Every single change gets its own commit
   - Dehydrate one function → test → commit
   - Move one type → update imports → test → commit
   - Never batch multiple changes

2. **Maintain Passing Tests**: 
   - NEVER use `--no-verify` to skip pre-commit hooks
   - A failing test suite means the refactor is broken
   - Tests are the litmus test for correct refactoring

3. **Move Tests with Code**:
   - When moving code, move its tests in the same commit
   - Split test files if they now test multiple packages
   - Test organization mirrors code organization

4. **Accept Repetitive Work**:
   - Yes, we'll update imports many times
   - Yes, we'll move some things multiple times
   - This is better than one big broken change

### Refactoring Sequence

## Phase 1: Dehydrate Types Package (~450 lines to move out)

Move logic OUT of types package to where it belongs:

1. **Validation Logic** → `internal/validation/`
   - `Validate()`, `validateEnumeratedDim()`, `validateHierarchicalDim()`
   - `IsReservedColumnName()`, `IsValidPrefix()`
   - Update all callers → test → commit

2. **Partition Building** → `nanostore/ids/`
   - `BuildPartitionForDocument()`
   - `ParsePartition()`, `ParseDimensionValue()`
   - Update all callers → test → commit

3. **Matching Logic** → `internal/matching/`
   - `(CanonicalView) Matches()`
   - `(CanonicalView) ExtractFromPartition()`
   - `(CanonicalView) IsCanonicalValue()`
   - Update all callers → test → commit

4. **Factory Functions** → main `nanostore/`
   - `DimensionSetFromConfig()`
   - Other complex builders
   - Update all callers → test → commit

## Phase 2: Redistribute Command Preprocessor

1. **Extract ID Resolution** → `ids/resolver.go`
   - Move reflection-based field discovery
   - Create `IDResolver` interface
   - Test independently → commit

2. **Extract Field Resolution** → `types/field_resolver.go`
   - Create `FieldResolver` for dimension lookups
   - Implement `IsReferenceField()`
   - Test independently → commit

3. **Update Preprocessor**
   - Inject dependencies via interfaces
   - Remove direct store/dimension coupling
   - Test → commit

## Phase 3: Eliminate Type Duplication

1. **Move Core Types** (one at a time):
   - Document → update imports → test → commit
   - ListOptions → update imports → test → commit
   - UpdateRequest → update imports → test → commit
   - Continue for each type...

2. **Remove Conversion Functions**:
   - Remove `ToTypesDocument()`
   - Remove other converters
   - Test → commit

## Phase 4: Extract  Layer

1. **Create `storage/` package**
   - Define Storage interface → commit

2. **Move Implementation**:
   - Move `impl_store_json.go` → test → commit
   - Move `lock_manager.go` → test → commit
   - Update imports → test → commit

## Phase 5: Extract each package

    For each package (ids, query, search):
1. **Create package**
   - Define interfaces and exports
   - Move relevant files (tests included)
   - Update imports → test → commit

## Phase 6: Final Organization

1. **Create orchestrator in `store/`**
   - Implement Store by composing other packages
   - Wire dependencies → test → commit

2. **Update APIs**
   - Update to use new structure
   - Maintain backward compatibility
   - Test → commit

### Expected Outcome

After refactoring:
- `types/`: ~500 lines (pure data structures)
- Clear package boundaries with no circular dependencies
- Each package has a single, clear responsibility
- Full test coverage maintained throughout
