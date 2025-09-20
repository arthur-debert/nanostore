package store

import (
	"context"
	"sync"
	"time"
)

// MockFileLock provides a mock implementation of FileLock for testing
type MockFileLock struct {
	mu          sync.Mutex
	isLocked    bool
	lockError   error
	unlockError error

	// For tracking lock attempts
	LockAttempts   int
	UnlockAttempts int
}

// TryLockContext implements FileLock.TryLockContext
func (m *MockFileLock) TryLockContext(ctx context.Context, retryInterval time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.LockAttempts++

	if m.lockError != nil {
		return false, m.lockError
	}

	// Simulate already locked
	if m.isLocked {
		return false, nil
	}

	m.isLocked = true
	return true, nil
}

// Unlock implements FileLock.Unlock
func (m *MockFileLock) Unlock() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.UnlockAttempts++

	if m.unlockError != nil {
		return m.unlockError
	}

	m.isLocked = false
	return nil
}

// IsLocked returns whether the lock is currently held (for testing)
func (m *MockFileLock) IsLocked() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isLocked
}

// SetLockError sets an error to be returned on lock attempts (for testing)
func (m *MockFileLock) SetLockError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lockError = err
}

// SetUnlockError sets an error to be returned on unlock attempts (for testing)
func (m *MockFileLock) SetUnlockError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unlockError = err
}

// MockFileLockFactory creates MockFileLock instances
type MockFileLockFactory struct {
	mu    sync.Mutex
	locks map[string]*MockFileLock

	// Default errors to inject
	DefaultLockError   error
	DefaultUnlockError error
}

// NewMockFileLockFactory creates a new mock factory
func NewMockFileLockFactory() *MockFileLockFactory {
	return &MockFileLockFactory{
		locks: make(map[string]*MockFileLock),
	}
}

// New implements FileLockFactory.New
func (f *MockFileLockFactory) New(path string) FileLock {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Return existing lock for the same path
	if lock, exists := f.locks[path]; exists {
		return lock
	}

	// Create new mock lock
	lock := &MockFileLock{
		lockError:   f.DefaultLockError,
		unlockError: f.DefaultUnlockError,
	}
	f.locks[path] = lock

	return lock
}

// GetLock returns the mock lock for a path (for testing)
func (f *MockFileLockFactory) GetLock(path string) *MockFileLock {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.locks[path]
}

// Reset clears all locks (for testing)
func (f *MockFileLockFactory) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.locks = make(map[string]*MockFileLock)
}
