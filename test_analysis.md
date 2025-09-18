# Nanostore Test Organization

## Final Test Organization

Tests have been reorganized to be co-located with the code they test:

### types/ Package
- `canonical_test.go` - Tests CanonicalView type methods
- `partition_test.go` - Tests Partition and DimensionValue types

### nanostore/ids/ Package  
- `id_generator_test.go` - Tests IDGenerator functionality
- `id_transform_test.go` - Tests IDTransformer functionality

### nanostore/stores/ Package
- `file_lock_test.go` - Tests file locking mechanism
- `persistence_test.go` - Tests JSON persistence

### nanostore/api/ Package
- `typed_utility_test.go` - Tests utility functions (isZeroValue, etc.)
- `declarative_test.go` - Tests declarative/typed API
- `declarative_delete_test.go` - Tests delete operations
- `declarative_parent_test.go` - Tests parent/child filtering
- `declarative_query_test.go` - Tests query builder
- `declarative_robustness_test.go` - Tests edge cases
- `query_robustness_test.go` - Tests query robustness
- `test_helpers_test.go` - Shared test types (TodoItem)

### nanostore/ Package (Main Integration Tests)
- `basic_operations_test.go` - Tests core CRUD operations
- `complex_type_validation_test.go` - Tests type validation
- `config_test.go` - Tests configuration
- `config_validation_robustness_test.go` - Tests config validation edge cases
- `datetime_test.go` - Tests datetime field handling
- `dimension_test.go` - Tests dimension helpers
- `filtering_test.go` - Tests filtering functionality
- `non_dimension_fields_test.go` - Tests non-dimension field handling
- `non_dimension_filtering_test.go` - Tests filtering by non-dimension fields
- `ordering_test.go` - Tests ordering functionality
- `pointer_validation_test.go` - Tests pointer field validation
- `transparent_filtering_ordering_test.go` - Tests transparent ID filtering/ordering
- `transparent_verification_test.go` - Tests transparent filtering verification

## Test Organization Principles

1. **Unit tests** are co-located with the code they test
2. **Integration tests** that test the full system remain in the main nanostore package
3. Tests use the `_test` package suffix for external testing
4. Shared test helpers are in `test_helpers_test.go` files

## Benefits

- Tests are easier to find and maintain
- Package responsibilities are clearer
- Related code and tests evolve together
- Better code organization overall