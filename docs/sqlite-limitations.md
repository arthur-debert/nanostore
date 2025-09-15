# SQLite Limitations in Nanostore

This document describes SQLite limitations that affect nanostore's behavior and the design decisions made to work within these constraints.

## Concurrent Write Limitations

SQLite uses a single-writer, multiple-reader model. This means:

- **Only one write transaction** can be active at a time
- **Multiple read transactions** can run concurrently
- Write attempts while another write is in progress will fail with `SQLITE_BUSY`

### Impact on Nanostore

Since nanostore is designed for single-user applications (CLI tools, desktop apps), this limitation aligns well with the intended use case. The codebase includes skipped tests that document this behavior:

- `TestConcurrentWrites` - Demonstrates that concurrent writes fail
- `TestConcurrentMixedOperations` - Shows read/write concurrency limitations
- `TestConcurrentHierarchicalOperations` - Tests hierarchical operations under concurrency
- `TestConcurrentTransactionIsolation` - Documents transaction isolation behavior

These tests remain in the codebase as documentation but are skipped during normal test runs.

### Working with the Limitation

For applications that need concurrent operations:

1. **Use read operations concurrently** - Multiple goroutines can read simultaneously
2. **Serialize write operations** - Use a single writer goroutine or mutex
3. **Handle SQLITE_BUSY errors** - Implement retry logic with exponential backoff
4. **Consider WAL mode** - Write-Ahead Logging can improve concurrency

Example of handling busy errors:

```go
func retryOnBusy(fn func() error) error {
    maxRetries := 5
    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        
        if strings.Contains(err.Error(), "database is locked") {
            time.Sleep(time.Duration(i*10) * time.Millisecond)
            continue
        }
        
        return err
    }
    return fmt.Errorf("max retries exceeded")
}
```

## Transaction Behavior

SQLite automatically wraps each SQL statement in a transaction if one isn't already active. Nanostore leverages this for consistency:

- Each operation (Add, Update, Delete) is atomic
- Failed operations automatically roll back
- No partial updates are possible

The `TestDatabaseConsistencyAfterPanic` test verifies that SQLite properly handles cleanup even when the application crashes.

## Performance Considerations

1. **Batch Operations** - Currently each operation is a separate transaction. For bulk operations, consider:
   - Implementing batch methods that use a single transaction
   - Using prepared statements for repeated operations

2. **Index Usage** - Nanostore creates appropriate indexes, but be aware:
   - SQLite can only use one index per table per query
   - Composite indexes may be beneficial for complex queries

3. **Database Size** - SQLite performs well up to several GB, but consider:
   - VACUUM operations for databases with frequent deletions
   - Page size tuning for large documents

## Platform-Specific Behavior

Some behaviors vary by operating system:

- **File permissions** - Read-only database handling differs between OS
- **File locking** - Mechanism varies (POSIX locks on Unix, mandatory locks on Windows)
- **Path handling** - Case sensitivity depends on the filesystem

The test suite handles these appropriately by skipping platform-dependent tests where necessary.

## Best Practices

1. **Always close the store** - Ensures proper cleanup of resources
2. **Handle errors appropriately** - Don't assume operations will succeed
3. **Use transactions wisely** - Group related operations when possible
4. **Monitor database size** - Plan for maintenance operations
5. **Test with realistic data** - Performance characteristics change with scale