package nanostore

import (
	"sync"
)

// operationType defines whether an operation is read or write
type operationType int

const (
	readOperation operationType = iota
	writeOperation
)

// lockManager handles centralized lock management for the store
type lockManager struct {
	mu *sync.RWMutex
}

// newLockManager creates a new lock manager
func newLockManager() *lockManager {
	return &lockManager{
		mu: &sync.RWMutex{},
	}
}

// execute runs a function with appropriate locking based on operation type
func (lm *lockManager) execute(opType operationType, fn func() error) error {
	switch opType {
	case readOperation:
		lm.mu.RLock()
		defer lm.mu.RUnlock()
	case writeOperation:
		lm.mu.Lock()
		defer lm.mu.Unlock()
	}
	return fn()
}

// executeWithResult runs a function with appropriate locking and returns a result
func (lm *lockManager) executeWithResult(opType operationType, fn func() (interface{}, error)) (interface{}, error) {
	switch opType {
	case readOperation:
		lm.mu.RLock()
		defer lm.mu.RUnlock()
	case writeOperation:
		lm.mu.Lock()
		defer lm.mu.Unlock()
	}
	return fn()
}
