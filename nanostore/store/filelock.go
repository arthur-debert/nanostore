package store

import (
	"context"
	"time"

	"github.com/gofrs/flock"
)

// FileLock defines the interface for file locking operations
type FileLock interface {
	// TryLockContext attempts to acquire an exclusive lock with retries
	TryLockContext(ctx context.Context, retryInterval time.Duration) (bool, error)

	// Unlock releases the lock
	Unlock() error
}

// FileLockFactory creates FileLock instances
type FileLockFactory interface {
	// New creates a new FileLock for the given path
	New(path string) FileLock
}

// FlockWrapper wraps github.com/gofrs/flock for our interface
type FlockWrapper struct {
	flock *flock.Flock
}

// TryLockContext implements FileLock.TryLockContext
func (f *FlockWrapper) TryLockContext(ctx context.Context, retryInterval time.Duration) (bool, error) {
	return f.flock.TryLockContext(ctx, retryInterval)
}

// Unlock implements FileLock.Unlock
func (f *FlockWrapper) Unlock() error {
	return f.flock.Unlock()
}

// FlockFactory is the default factory implementation using flock
type FlockFactory struct{}

// New implements FileLockFactory.New
func (f *FlockFactory) New(path string) FileLock {
	return &FlockWrapper{
		flock: flock.New(path),
	}
}
