# Nanostore CLI Implementation Plan v0.14

## Updated for Unified API (v0.14.0+)

This plan updates issue #79 to reflect the unified API architecture where there is only ONE API: `api.Store[T]` created via `api.New[T]()`.

## Core Philosophy

### 1. Direct API Mapping
- CLI commands should be **shell translations** of the Go API methods
- Parameters and returns should **match exactly** with Go API signatures  
- No "form that doesn't match" - maintain 1:1 correspondence

### 2. Generic Command Translation
Instead of implementing commands one-by-one, create a **uniform translation system** using Cobra that:
- Maps CLI verbs to Go methods systematically
- Translates CLI flags to Go parameters automatically  
- Handles type conversions and JSON marshaling uniformly
- Provides consistent error handling and output formatting

### 3. Full API Coverage
Implement **the complete `api.Store[T]` interface** as CLI commands, organized in logical phases.

## Architecture: Generic Command System

### Unified Command Pattern
```bash
nanostore <verb> [arguments] [flags] --type <TypeDef> --db <path>
```

### Key Innovation: Type-Generic CLI
```bash
# Instead of hardcoded commands, use generic type system:
nanostore --type Task create "New Task" --status pending --priority high
nanostore --type Task get 1
nanostore --type Task list --status active
nanostore --type Task update 1 --status done
nanostore --type Note create "Meeting Notes" --category work --tags "meeting,q4"
```

### Universal Flags (All Commands)
```bash
--type, -t <type>      # Type definition (JSON schema or struct name)
--db, -d <path>        # Database file path (env: NANOSTORE_DB)
--format, -f <format>  # Output: table|json|yaml|csv (env: NANOSTORE_FORMAT)
--no-color             # Disable colors (env: NANOSTORE_NOCOLOR)
--quiet, -q            # Suppress headers
--dry-run              # Show what would happen without executing
```

## Complete API Mapping

### Core CRUD Operations
| CLI Command | Go API Method | Parameters |
|-------------|---------------|------------|
| `create <title> [--field=value...]` | `Store.Create(title, data)` | title + struct fields |
| `get <id>` | `Store.Get(id)` | id (SimpleID or UUID) |
| `update <id> [--field=value...]` | `Store.Update(id, data)` | id + partial struct |
| `delete <id> [--cascade]` | `Store.Delete(id, cascade)` | id + cascade flag |

### Bulk Operations  
| CLI Command | Go API Method | Parameters |
|-------------|---------------|------------|
| `update-by-dimension --filter key=value --set field=value` | `Store.UpdateByDimension(filters, data)` | filters + updates |
| `update-where --where "clause" --set field=value [args...]` | `Store.UpdateWhere(clause, data, args)` | SQL clause + updates |
| `update-by-uuids --uuids id1,id2 --set field=value` | `Store.UpdateByUUIDs(uuids, data)` | UUID list + updates |
| `delete-by-dimension --filter key=value` | `Store.DeleteByDimension(filters)` | dimension filters |
| `delete-where --where "clause" [args...]` | `Store.DeleteWhere(clause, args)` | SQL clause + args |
| `delete-by-uuids --uuids id1,id2` | `Store.DeleteByUUIDs(uuids)` | UUID list |

### Query Operations
| CLI Command | Go API Method | Parameters |
|-------------|---------------|------------|
| `list [--filter key=value...] [--sort field] [--limit N]` | `Store.List(opts)` | ListOptions struct |
| `query --where "clause" [args...]` | `Store.Query().Where().Find()` | Query builder pattern |

### Metadata & Introspection
| CLI Command | Go API Method | Parameters |
|-------------|---------------|------------|
| `get-raw <id>` | `Store.GetRaw(id)` | id → raw Document |
| `get-dimensions <id>` | `Store.GetDimensions(id)` | id → dimensions map |
| `get-metadata <id>` | `Store.GetMetadata(id)` | id → DocumentMetadata |
| `resolve-uuid <simpleID>` | `Store.ResolveUUID(simpleID)` | SimpleID → UUID |

### Configuration & Debug
| CLI Command | Go API Method | Parameters |
|-------------|---------------|------------|
| `config` | `Store.GetDimensionConfig()` | → Config struct |
| `debug` | `Store.GetDebugInfo()` | → DebugInfo struct |
| `stats` | `Store.GetStoreStats()` | → StoreStats struct |
| `validate` | `Store.ValidateConfiguration()` | → validation errors |
| `integrity` | `Store.ValidateStoreIntegrity()` | → IntegrityReport |
| `field-stats` | `Store.GetFieldUsageStats()` | → FieldUsageStats |
| `schema` | `Store.GetTypeSchema()` | → TypeSchema |

### Administrative Operations
| CLI Command | Go API Method | Parameters |
|-------------|---------------|------------|
| `add-raw <title> --dimensions '{...}'` | `Store.AddRaw(title, dimensions)` | title + dimensions JSON |
| `add-dimension-value --dim name --value val --prefix p` | `Store.AddDimensionValue(dim, val, prefix)` | dimension config |
| `modify-dimension-default --dim name --default val` | `Store.ModifyDimensionDefault(dim, default)` | dimension config |

## Implementation Strategy

### Phase 1: Generic Command Framework
**Goal**: Create the uniform translation system

```go
// Generic command structure
type Command struct {
    Name     string
    Method   string              // Go method name  
    Args     []ArgSpec          // Required arguments
    Flags    []FlagSpec         // Optional flags  
    Returns  ReturnSpec         // Return type/format
}

// Auto-generate commands from API reflection
func GenerateCommands() []Command {
    // Use reflection on api.Store[T] to extract methods
    // Generate Cobra commands automatically
    // Handle type conversion and JSON marshaling uniformly
}
```

**Deliverables**:
- [ ] Generic command generator using Go reflection
- [ ] Uniform type system (JSON schema or struct definitions)
- [ ] Consistent flag parsing and validation
- [ ] Universal output formatting (table/json/yaml/csv)
- [ ] Environment variable integration

### Phase 2: Core Operations  
**Goal**: CRUD + basic query operations

**Deliverables**:
- [ ] `create`, `get`, `update`, `delete` commands
- [ ] `list` with filtering and sorting
- [ ] Basic `query` operations
- [ ] Type-safe field validation
- [ ] Error handling with helpful messages

### Phase 3: Advanced Operations
**Goal**: Bulk operations and metadata access

**Deliverables**:
- [ ] All bulk update/delete operations
- [ ] Raw data access commands
- [ ] Metadata and introspection commands
- [ ] UUID resolution utilities

### Phase 4: Administrative Features
**Goal**: Configuration and maintenance operations

**Deliverables**:
- [ ] Configuration introspection
- [ ] Debug and statistics commands  
- [ ] Integrity validation
- [ ] Dynamic dimension management

## Type System Design

### Option 1: JSON Schema Files
```bash
# Define types in JSON files
cat > task.json <<EOF
{
  "type": "Task",
  "dimensions": {
    "status": {"values": ["pending","active","done"], "default": "pending"},
    "priority": {"values": ["low","medium","high"], "default": "medium"}
  },
  "fields": {
    "description": "string",
    "assignee": "string",
    "due_date": "*time.Time"
  }
}
EOF

nanostore --type task.json create "New Task" --status pending
```

### Option 2: Built-in Type Registry  
```bash
# Register types in CLI
nanostore register-type Task task.json
nanostore --type Task create "New Task" --status pending
```

### Option 3: Go Struct Embedding
```bash
# Reference Go structs directly (advanced)
nanostore --type "myapp.Task" create "New Task" --status pending
```

## Generic Query Builder

Instead of hardcoded query commands, create a **universal query interface**:

```bash
# Generic query syntax matching Go Query API exactly
nanostore query \
  --where "status = ? AND priority = ?" "active" "high" \
  --or-where "assignee = ?" "john" \
  --not "description LIKE ?" "%test%" \
  --order-by "created_at DESC" \
  --limit 10

# Maps directly to:
store.Query().
  Where("status = ? AND priority = ?", "active", "high").
  OrWhere("assignee = ?", "john").
  Not("description LIKE ?", "%test%").
  OrderBy("created_at DESC").
  Limit(10).
  Find()
```

## Benefits of Generic Approach

### 1. **Complete API Coverage**
- Every Go method gets a CLI command automatically
- No "missing functionality" gaps
- Future API additions get CLI support automatically

### 2. **Consistency**
- All commands follow same patterns
- Uniform error handling and output formatting  
- Predictable parameter mapping

### 3. **Type Safety**
- Compile-time verification of field names and types
- Validation before database operations
- Clear error messages for type mismatches

### 4. **Maintainability**  
- One implementation handles all commands
- Changes to core API automatically reflected in CLI
- Reduced code duplication

### 5. **Extensibility**
- Easy to add new output formats
- Plugin system for custom types
- Integration with external tools

## Implementation Notes

### Cobra Integration
```go
// Auto-generate Cobra commands
func (c *Command) ToCobraCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   c.GenerateUsage(),
        Short: c.GenerateShortDesc(), 
        RunE:  c.GenerateRunFunc(),
    }
    
    // Add flags automatically based on API signature
    c.AddFlags(cmd)
    
    return cmd
}
```

### Error Handling
- Map Go errors to appropriate exit codes
- Provide context-specific help messages
- Suggest corrections for common mistakes

### Output Formatting
- Table format for human readability
- JSON for machine processing  
- YAML for configuration files
- CSV for spreadsheet import

## Success Criteria

**Phase 1 Complete:**
- [ ] Generic command framework operational
- [ ] Type system working with at least one type
- [ ] Basic CRUD operations functional
- [ ] Output formatting implemented

**Phase 2 Complete:**
- [ ] All core Store methods accessible via CLI
- [ ] Query operations fully functional
- [ ] Type validation working
- [ ] Documentation and help system complete

**Phase 3 Complete:**
- [ ] Complete feature parity with Go API
- [ ] Advanced operations and bulk updates
- [ ] Administrative commands operational
- [ ] Integration testing complete

## Questions Answered

> **Can't we do a solution that (using cobra) translates command & params uniformly to every command?**

**YES!** This plan specifically addresses that with the Generic Command Framework in Phase 1. Instead of implementing commands one-by-one, we create a reflection-based system that:

1. **Analyzes the `api.Store[T]` interface** using Go reflection
2. **Generates Cobra commands automatically** for each public method
3. **Maps CLI flags to Go parameters** systematically  
4. **Handles type conversion and validation** uniformly
5. **Provides consistent output formatting** across all commands

This eliminates the need to hand-code each command and ensures perfect API alignment.

## Migration from Issue #79

This plan **completely replaces** the approach in issue #79 by:

- ✅ **Removing TypedStore references** (now just `api.Store[T]`)
- ✅ **Creating generic command system** instead of one-by-one implementation
- ✅ **Mapping CLI directly to Go API** with exact parameter matching
- ✅ **Supporting full API coverage** from day one
- ✅ **Enabling type-generic operations** for any struct type

The result is a more powerful, maintainable, and complete CLI that grows automatically with the API.