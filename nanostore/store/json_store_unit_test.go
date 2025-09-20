package store

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore/storage"
	"github.com/arthur-debert/nanostore/types"
	"github.com/google/uuid"
)

// TestJSONStoreWithMockFS demonstrates unit testing with mock file system
func TestJSONStoreWithMockFS(t *testing.T) {
	t.Run("creates new store with empty file system", func(t *testing.T) {
		// Setup mock file system
		mockFS := NewMockFileSystem()
		mockLockFactory := NewMockFileLockFactory()

		// Create store with mocks
		config := &mockTestConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"pending", "done"}, DefaultValue: "pending"},
			},
		}

		store, err := NewWithOptions("test.json", config,
			WithFileSystem(mockFS),
			WithFileLockFactory(mockLockFactory),
		)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Verify store is initialized but file doesn't exist yet
		if mockFS.FileExists("test.json") {
			t.Error("expected file not to exist initially")
		}

		// Add a document
		id, err := store.Add("Test Document", map[string]interface{}{"status": "pending"})
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Verify file was created
		if !mockFS.FileExists("test.json") {
			t.Error("expected file to exist after add")
		}

		// Verify content
		content, ok := mockFS.GetFileContent("test.json")
		if !ok {
			t.Fatal("failed to get file content")
		}

		var data storage.StoreData
		if err := json.Unmarshal(content, &data); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if len(data.Documents) != 1 {
			t.Errorf("expected 1 document, got %d", len(data.Documents))
		}

		if data.Documents[0].UUID != id {
			t.Errorf("expected UUID %s, got %s", id, data.Documents[0].UUID)
		}
	})

	t.Run("handles file system errors gracefully", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockLockFactory := NewMockFileLockFactory()

		config := &mockTestConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"pending", "done"}, DefaultValue: "pending"},
			},
		}

		// Test read error
		mockFS.ReadFileError = errors.New("disk read error")

		// Pre-populate file so it exists
		testData := &storage.StoreData{
			Documents: []types.Document{},
			Metadata: storage.Metadata{
				Version:   "1.0",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		data, _ := json.MarshalIndent(testData, "", "  ")
		_ = mockFS.WriteFile("test.json", data, 0644)

		// Try to create store - should fail on load
		_, err := NewWithOptions("test.json", config,
			WithFileSystem(mockFS),
			WithFileLockFactory(mockLockFactory),
		)
		if err == nil {
			t.Error("expected error due to read failure")
		}
		if !errors.Is(err, mockFS.ReadFileError) {
			t.Errorf("expected read error, got: %v", err)
		}
	})

	t.Run("atomic write with mock file system", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockLockFactory := NewMockFileLockFactory()

		config := &mockTestConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"pending", "done"}, DefaultValue: "pending"},
			},
		}

		store, err := NewWithOptions("test.json", config,
			WithFileSystem(mockFS),
			WithFileLockFactory(mockLockFactory),
		)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Add a document
		_, err = store.Add("Test Document", map[string]interface{}{"status": "pending"})
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Verify temp file was created and renamed
		if mockFS.FileExists("test.json.tmp") {
			t.Error("temp file should not exist after successful write")
		}
		if !mockFS.FileExists("test.json") {
			t.Error("main file should exist after write")
		}

		// Simulate rename failure
		mockFS.RenameError = errors.New("rename failed")

		// Try to add another document
		_, err = store.Add("Another Document", map[string]interface{}{"status": "done"})
		if err == nil {
			t.Error("expected error due to rename failure")
		}

		// Verify temp file was cleaned up
		if mockFS.FileExists("test.json.tmp") {
			t.Error("temp file should be cleaned up after rename failure")
		}
	})

	t.Run("concurrent access with mock locks", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockLockFactory := NewMockFileLockFactory()

		config := &mockTestConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"pending", "done"}, DefaultValue: "pending"},
			},
		}

		store, err := NewWithOptions("test.json", config,
			WithFileSystem(mockFS),
			WithFileLockFactory(mockLockFactory),
			WithTimeFunc(func() time.Time { return time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC) }),
		)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Get the lock used by the store
		lock := mockLockFactory.GetLock("test.json.lock")

		// Verify lock is acquired during operations
		if lock.LockAttempts != 1 {
			t.Errorf("expected 1 lock attempt during initialization, got %d", lock.LockAttempts)
		}

		// Add a document - should acquire lock again
		_, err = store.Add("Test Document", map[string]interface{}{"status": "pending"})
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Should have acquired lock for save operation
		if lock.LockAttempts < 2 {
			t.Errorf("expected at least 2 lock attempts after add, got %d", lock.LockAttempts)
		}

		// Verify lock is released
		if lock.IsLocked() {
			t.Error("lock should be released after operation")
		}
	})

	t.Run("loads existing data correctly", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockLockFactory := NewMockFileLockFactory()

		// Pre-populate file system with test data
		existingData := &storage.StoreData{
			Documents: []types.Document{
				{
					UUID:       uuid.New().String(),
					Title:      "Existing Document",
					Dimensions: map[string]interface{}{"status": "done"},
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				},
			},
			Metadata: storage.Metadata{
				Version:   "1.0",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		data, _ := json.MarshalIndent(existingData, "", "  ")
		_ = mockFS.WriteFile("test.json", data, 0644)

		config := &mockTestConfig{
			dimensions: []types.DimensionConfig{
				{Name: "status", Type: types.Enumerated, Values: []string{"pending", "done"}, DefaultValue: "pending"},
			},
		}

		store, err := NewWithOptions("test.json", config,
			WithFileSystem(mockFS),
			WithFileLockFactory(mockLockFactory),
		)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// List documents
		docs, err := store.List(types.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		if len(docs) != 1 {
			t.Fatalf("expected 1 document, got %d", len(docs))
		}

		if docs[0].Title != "Existing Document" {
			t.Errorf("expected title 'Existing Document', got '%s'", docs[0].Title)
		}
	})
}

// mockTestConfig implements the Config interface for testing
type mockTestConfig struct {
	dimensions []types.DimensionConfig
}

func (tc *mockTestConfig) GetDimensionSet() *types.DimensionSet {
	return types.DimensionSetFromConfig(types.Config{Dimensions: tc.dimensions})
}
