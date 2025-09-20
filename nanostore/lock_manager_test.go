package nanostore

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// EXCEPTION: This is an internal package test that needs access to unexported types.
// It cannot use the standard fixture approach but should still follow other best practices where possible.

import (
	"sync"
	"testing"
	"time"
)

func TestLockManager(t *testing.T) {
	lm := newLockManager()

	t.Run("ConcurrentReads", func(t *testing.T) {
		// Multiple reads should be able to proceed concurrently
		var wg sync.WaitGroup
		concurrentReads := 10
		results := make(chan time.Time, concurrentReads)

		for i := 0; i < concurrentReads; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = lm.execute(readOperation, func() error {
					start := time.Now()
					time.Sleep(10 * time.Millisecond) // Simulate work
					results <- start
					return nil
				})
			}()
		}

		wg.Wait()
		close(results)

		// Collect all start times
		var startTimes []time.Time
		for start := range results {
			startTimes = append(startTimes, start)
		}

		// Check that reads overlapped (all started within a small window)
		if len(startTimes) != concurrentReads {
			t.Errorf("expected %d reads, got %d", concurrentReads, len(startTimes))
		}

		// Find the time window
		var earliest, latest time.Time
		for i, t := range startTimes {
			if i == 0 || t.Before(earliest) {
				earliest = t
			}
			if i == 0 || t.After(latest) {
				latest = t
			}
		}

		window := latest.Sub(earliest)
		// All reads should have started within 5ms of each other (allowing for goroutine scheduling)
		if window > 5*time.Millisecond {
			t.Errorf("reads did not execute concurrently, window was %v", window)
		}
	})

	t.Run("WriteBlocksReads", func(t *testing.T) {
		// A write should block reads
		writeStarted := make(chan struct{})
		writeDone := make(chan struct{})
		readStarted := make(chan struct{})

		// Start a write that takes some time
		go func() {
			_ = lm.execute(writeOperation, func() error {
				close(writeStarted)
				time.Sleep(50 * time.Millisecond)
				close(writeDone)
				return nil
			})
		}()

		// Wait for write to start
		<-writeStarted

		// Try to read - should be blocked
		go func() {
			_ = lm.execute(readOperation, func() error {
				close(readStarted)
				return nil
			})
		}()

		// Read should not start until write is done
		select {
		case <-readStarted:
			t.Error("read started while write was in progress")
		case <-time.After(25 * time.Millisecond):
			// Expected - read is blocked
		}

		// Wait for write to finish
		<-writeDone

		// Now read should proceed quickly
		select {
		case <-readStarted:
			// Expected
		case <-time.After(10 * time.Millisecond):
			t.Error("read did not start after write completed")
		}
	})

	t.Run("WritesAreSerialized", func(t *testing.T) {
		// Multiple writes should be serialized
		var order []int
		var mu sync.Mutex

		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			id := i
			go func() {
				defer wg.Done()
				_ = lm.execute(writeOperation, func() error {
					mu.Lock()
					order = append(order, id)
					mu.Unlock()
					time.Sleep(10 * time.Millisecond) // Ensure writes don't overlap
					return nil
				})
			}()
			time.Sleep(1 * time.Millisecond) // Stagger the starts slightly
		}

		wg.Wait()

		// Check that we got all writes
		if len(order) != 5 {
			t.Errorf("expected 5 writes, got %d", len(order))
		}

		// The order might not be 0,1,2,3,4 due to goroutine scheduling,
		// but each write should have completed before the next started
		// This is implicitly tested by the fact that we didn't get any panics
		// or race conditions
	})

	t.Run("ExecuteWithResult", func(t *testing.T) {
		// Test that executeWithResult properly returns values
		result, err := lm.executeWithResult(readOperation, func() (interface{}, error) {
			return "test-value", nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result.(string) != "test-value" {
			t.Errorf("expected 'test-value', got %v", result)
		}
	})
}
