# CLI Bulk Operations Implementation Plan

## Overview

Implement bulk operations for the nanostore CLI by leveraging the existing query infrastructure. Bulk operations should work exactly like existing commands - they're just different API methods that consume the same parsed query system.

## Implementation Strategy

### Core Insight

Bulk operations use the **exact same query syntax** as existing commands (`list`, `create`, `update`). No special parsing needed - just different API method calls.

### Key Components

#### 1. Extend MethodExecutor.ExecuteCommand()

Add bulk operation cases that use existing query system:

- `update-by-dimension`
- `update-where`
- `delete-by-dimension`
- `delete-where`
- `update-by-uuids`
- `delete-by-uuids`

#### 2. Add One Helper Function

```go
// queryToDimensionFilters converts query conditions to dimension filters map
func (me *MethodExecutor) queryToDimensionFilters(query *Query) map[string]interface{}
```

#### 3. Extend ReflectionExecutor

Add bulk operation methods that use existing reflection infrastructure.

### CLI Usage Examples

```bash
# Update by dimension - use --sql/--data for clearer separation
nano-db update-by-dimension --sql --status=pending --priority=high --data --status=completed --assignee=john

# Update by WHERE clause - same operators as existing queries  
nano-db update-where --sql --status=pending --and --priority__gte=3 --data --assignee=john --status=completed

# Delete by dimension - same filter syntax (no --data needed)
nano-db delete-by-dimension --sql --status=archived --priority=low

# Delete by WHERE clause - same operators (no --data needed)
nano-db delete-where --sql --created_at__lt=2023-01-01 --or --status=archived

# UUID operations - use --data to separate UUIDs from update data
nano-db update-by-uuids "uuid1,uuid2,uuid3" --data --status=completed --assignee=alice
nano-db delete-by-uuids "uuid1,uuid2,uuid3"
```

### New --sql/--data Operator Design

**BREAKING CHANGE**: All bulk update operations now use `--sql` and `--data` operators for clearer separation.

**Why this change:**

- Eliminates confusion with `--update` operator
- Makes intent crystal clear: `--sql` = WHERE criteria, `--data` = update data
- Maintains consistency with existing query operators
- More intuitive for API users

**How it works:**

- Everything between `--sql` and `--data`: WHERE criteria (what to find)
- Everything after `--data`: update data (what to change)
- Uses existing query operators (`__gte`, `__lt`, `__contains`, etc.)
- Leverages existing `BuildWhereFromQuery()` infrastructure

**Examples:**

```bash
# Simple case
nano-db update-where --sql --status=pending --data --status=completed

# Complex filtering with multiple updates
nano-db update-where \
  --sql \
  --priority__gte=3 \
  --and \
  --created_at__lt=2023-01-01 \
  --data \
  --status=archived \
  --assignee=alice \
  --tags=urgent

# Works with all operators
nano-db update-where --sql --priority__gte=3 --or --category=bug --data --status=in-progress

# Multiple conditions
nano-db update-where --sql --status=pending --and --priority=high --data --assignee=john --tags=assigned
```

## Implementation Plan

### Phase 1: Single Method Implementation

1. Start with `update-by-dimension` as the simplest case
2. Implement end-to-end: code + tests + verification
3. Commit and push with key learnings
4. Document patterns for other methods

### Phase 2: Iterative Implementation

5. Implement each remaining method individually
6. Code + tests + verification for each
7. Commit and push after each method
8. Use established patterns from Phase 1

### Phase 3: Final Integration

9. Write comprehensive PR
10. Final testing and documentation

## Testing Strategy

**NO integration tests or shelling out**

**Unit testing approach:**

- Craft full CLI command strings with options
- Feed to Cobra parser
- Mock the point where query would run
- Assert correct query parameters were generated
- Test CLI string → query conversion, NOT query results

**Test what we control:**

- Query parsing accuracy
- Parameter conversion
- Method invocation
- Error handling

**Don't test:**

- Actual database operations
- Query execution results
- Integration with external systems

## Key Advantages

1. **Consistency**: Uses exact same query syntax as existing commands
2. **No Duplication**: Leverages existing infrastructure
3. **Power**: Supports all existing operators and logic
4. **Security**: Uses existing parameterized query system
5. **Minimal Code**: ~50 lines vs hundreds in naive approach

## Reused Components

- ✅ Query parsing (`parseFilters()`)
- ✅ Query structure (`Query`, `FilterGroup`, `FilterCondition`)
- ✅ Data conversion (`queryToDataMap()`)
- ✅ SQL generation (`BuildWhereFromQuery()`)
- ✅ Type reflection (`ExecuteMethod()`)
- ✅ Output formatting (`OutputFormatter`)
- ✅ Error handling patterns
- ✅ Dry run support

## New Components

- ✅ One helper function (`queryToDimensionFilters()`)
- ✅ Six executor cases (in `ExecuteCommand()`)
- ✅ Six reflection methods (in `ReflectionExecutor`)

## Success Criteria

1. All bulk operations work with existing query syntax
2. Comprehensive unit tests for CLI → query conversion
3. No integration tests or shell execution
4. Each method implemented and tested individually
5. Clear patterns established for future extensions
