package store_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/arthur-debert/nanostore/nanostore/storage"
	"github.com/arthur-debert/nanostore/nanostore/store"
	"github.com/arthur-debert/nanostore/types"
)

// ExampleFileSystemMocking demonstrates how to use mock file systems for testing
func Example_fileSystemMocking() {
	// Create a mock file system and lock factory
	mockFS := store.NewMockFileSystem()
	mockLockFactory := store.NewMockFileLockFactory()

	// Create a test configuration
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:         "status",
				Type:         types.Enumerated,
				Values:       []string{"pending", "in_progress", "done"},
				DefaultValue: "pending",
			},
			{
				Name:     "parent",
				Type:     types.Hierarchical,
				RefField: "parent_id",
			},
		},
	}

	// Create a store with the mock file system
	testStore, err := store.NewWithOptions("tasks.json", &config,
		store.WithFileSystem(mockFS),
		store.WithFileLockFactory(mockLockFactory),
		store.WithTimeFunc(func() time.Time {
			return time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		}),
	)
	if err != nil {
		panic(err)
	}
	defer func() { _ = testStore.Close() }()

	// Add some tasks
	_, err = testStore.Add("Design API", map[string]interface{}{"status": "done"})
	if err != nil {
		panic(err)
	}
	_, err = testStore.Add("Write tests", map[string]interface{}{"status": "in_progress"})
	if err != nil {
		panic(err)
	}
	_, err = testStore.Add("Write docs", map[string]interface{}{"status": "pending"})
	if err != nil {
		panic(err)
	}

	// The mock file system allows us to inspect what was written
	content, ok := mockFS.GetFileContent("tasks.json")
	if !ok {
		panic("file not found")
	}
	var data storage.StoreData
	err = json.Unmarshal(content, &data)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Documents saved: %d\n", len(data.Documents))
	if len(data.Documents) > 0 {
		fmt.Printf("First document title: %s\n", data.Documents[0].Title)
	}

	// We can also simulate file system errors for error handling tests
	mockFS.WriteFileError = errors.New("disk full")

	// This operation will fail due to the simulated error
	_, err = testStore.Add("Another task", map[string]interface{}{"status": "pending"})
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}

	// Output:
	// Documents saved: 3
	// First document title: Design API
	// Expected error: failed to save: failed to write temp file: disk full
}

// ExampleConcurrentAccess demonstrates testing concurrent access with mock locks
func Example_concurrentAccess() {
	mockFS := store.NewMockFileSystem()
	mockLockFactory := store.NewMockFileLockFactory()

	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{Name: "status", Type: types.Enumerated, Values: []string{"active", "done"}},
		},
	}

	// Create a store
	testStore, _ := store.NewWithOptions("concurrent.json", &config,
		store.WithFileSystem(mockFS),
		store.WithFileLockFactory(mockLockFactory),
	)
	defer func() { _ = testStore.Close() }()

	// Get the lock used by the store
	lock := mockLockFactory.GetLock("concurrent.json.lock")

	// Add a document - this will acquire and release the lock
	_, _ = testStore.Add("Task 1", map[string]interface{}{"status": "active"})

	// Check lock statistics
	fmt.Printf("Lock attempts: %d\n", lock.LockAttempts)
	fmt.Printf("Unlock attempts: %d\n", lock.UnlockAttempts)
	fmt.Printf("Currently locked: %v\n", lock.IsLocked())

	// Output:
	// Lock attempts: 2
	// Unlock attempts: 2
	// Currently locked: false
}

// This example shows that with the new abstractions, we can:
// 1. Test file operations without touching the real file system (faster, more reliable)
// 2. Simulate error conditions that would be hard to reproduce with real files
// 3. Verify the exact content being written
// 4. Test concurrent access patterns deterministically
// 5. Run tests in parallel without file conflicts
