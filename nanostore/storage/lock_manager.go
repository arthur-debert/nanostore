package storage

import (
	"sync"
)

// OperationType defines whether an operation is read or write.
// This distinction allows the lockManager to use appropriate locking
// strategies - read locks (RLock) for concurrent reads, and write locks
// (Lock) for exclusive writes.
type OperationType int

const (
	// ReadOperation indicates an operation that only reads data.
	// Multiple read operations can proceed concurrently.
	ReadOperation OperationType = iota

	// WriteOperation indicates an operation that modifies data.
	// Write operations are exclusive - no other reads or writes
	// can proceed while a write lock is held.
	WriteOperation
)

// LockManager provides centralized lock management for thread-safe store operations.
// It encapsulates the locking strategy, ensuring consistent use of read/write locks
// throughout the store implementation. This centralization prevents common concurrency
// bugs like deadlocks from lock/unlock/relock patterns and ensures all operations
// use the appropriate lock type.
//
// The lockManager uses Go's sync.RWMutex, which allows multiple concurrent readers
// but exclusive writers. This maximizes read throughput while maintaining data
// consistency during writes.
type LockManager struct {
	mu *sync.RWMutex
}

// NewLockManager creates a new lock manager instance.
// The returned LockManager is ready to use and can handle concurrent
// operations immediately.
func NewLockManager() *LockManager {
	return &LockManager{
		mu: &sync.RWMutex{},
	}
}

// execute runs a function with appropriate locking based on operation type.
// It automatically acquires the correct lock (read or write) before executing
// the function and releases it when done.
//
// For read operations:
//   - Acquires a read lock (RLock) allowing concurrent reads
//   - Multiple goroutines can hold read locks simultaneously
//   - Blocks if a write lock is held
//
// For write operations:
//   - Acquires an exclusive write lock
//   - Blocks all other read and write operations
//   - Ensures exclusive access to the protected resources
//
// The lock is automatically released via defer when the function returns,
// ensuring proper cleanup even if the function panics.
//
// Example:
//
//	err := lockManager.Execute(ReadOperation, func() error {
//	    // Safe to read data here
//	    return nil
//	})
func (lm *LockManager) Execute(opType OperationType, fn func() error) error {
	switch opType {
	case ReadOperation:
		lm.mu.RLock()
		defer lm.mu.RUnlock()
	case WriteOperation:
		lm.mu.Lock()
		defer lm.mu.Unlock()
	}
	return fn()
}

// executeWithResult runs a function with appropriate locking and returns a result.
// This is identical to execute() but for functions that return both a result
// and an error. The locking behavior is the same as execute().
//
// The function parameter must return (interface{}, error). The caller is
// responsible for type asserting the returned interface{} to the expected type.
//
// For read operations:
//   - Acquires a read lock (RLock) allowing concurrent reads
//   - Multiple goroutines can hold read locks simultaneously
//   - Blocks if a write lock is held
//
// For write operations:
//   - Acquires an exclusive write lock
//   - Blocks all other read and write operations
//   - Ensures exclusive access to the protected resources
//
// Example:
//
//	result, err := lockManager.ExecuteWithResult(ReadOperation, func() (interface{}, error) {
//	    // Safe to read data here
//	    return someData, nil
//	})
//	if err != nil {
//	    return nil, err
//	}
//	data := result.(ExpectedType)
func (lm *LockManager) ExecuteWithResult(opType OperationType, fn func() (interface{}, error)) (interface{}, error) {
	switch opType {
	case ReadOperation:
		lm.mu.RLock()
		defer lm.mu.RUnlock()
	case WriteOperation:
		lm.mu.Lock()
		defer lm.mu.Unlock()
	}
	return fn()
}
