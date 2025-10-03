package api_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// TestBodyContentHandling represents a test item for body content testing
type TestBodyContentHandling struct {
	nanostore.Document
	Status   string `values:"pending,active,done" default:"pending"`
	Priority string `values:"low,medium,high" default:"medium"`

	// Data fields
	Assignee string
	Tags     string
}

func TestBodyContentHandlingInCreate(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test_body_content*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TestBodyContentHandling](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("Create should handle body content from embedded Document", func(t *testing.T) {
		// This test should FAIL initially - demonstrates the bug
		// Body content provided in the embedded Document should be stored

		item := &TestBodyContentHandling{
			Document: nanostore.Document{
				Title: "Task with Body",
				Body:  "This is important task details that should be stored",
			},
			Status:   "pending",
			Priority: "high",
			Assignee: "alice",
		}

		id, err := store.Create("Override Title", item)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Retrieve the created item
		retrieved, err := store.Get(id)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		// Body content should be preserved (main issue from user feedback)
		if retrieved.Body != "This is important task details that should be stored" {
			t.Errorf("Expected body content to be preserved, got: %q", retrieved.Body)
		}

		// Title parameter should take precedence (existing API contract)
		if retrieved.Title != "Override Title" {
			t.Errorf("Expected title parameter to take precedence, got: %q", retrieved.Title)
		}
	})

	t.Run("Create with title parameter should override struct title", func(t *testing.T) {
		// When both title parameter and struct Title are provided,
		// the title parameter should take precedence

		item := &TestBodyContentHandling{
			Document: nanostore.Document{
				Title: "Original Title",
				Body:  "Body content",
			},
			Status: "active",
		}

		id, err := store.Create("Override Title", item)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		retrieved, err := store.Get(id)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		// Title parameter should win
		if retrieved.Title != "Override Title" {
			t.Errorf("Expected title parameter to override struct title, got: %q", retrieved.Title)
		}

		// Body should still be preserved
		if retrieved.Body != "Body content" {
			t.Errorf("Expected body content to be preserved, got: %q", retrieved.Body)
		}
	})

	t.Run("Create with empty title parameter should use struct title", func(t *testing.T) {
		// When title parameter is empty, struct Title should be used

		item := &TestBodyContentHandling{
			Document: nanostore.Document{
				Title: "Struct Title",
				Body:  "Body from struct",
			},
			Status: "done",
		}

		id, err := store.Create("", item)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		retrieved, err := store.Get(id)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		// Struct title should be used when parameter is empty
		if retrieved.Title != "Struct Title" {
			t.Errorf("Expected struct title to be used when parameter is empty, got: %q", retrieved.Title)
		}

		// Body should be preserved
		if retrieved.Body != "Body from struct" {
			t.Errorf("Expected body content to be preserved, got: %q", retrieved.Body)
		}
	})

	t.Run("Create should not ignore other Document fields", func(t *testing.T) {
		// UUID should be ignored (always generated), but other fields should be handled appropriately

		item := &TestBodyContentHandling{
			Document: nanostore.Document{
				UUID:  "should-be-ignored", // This should be ignored
				Title: "Test Title",
				Body:  "Test Body",
			},
			Status: "active",
		}

		id, err := store.Create("", item)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		retrieved, err := store.Get(id)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		// UUID should be newly generated, not the provided one
		if retrieved.UUID == "should-be-ignored" {
			t.Error("UUID from struct should be ignored and new one generated")
		}

		// Title and Body should be preserved
		if retrieved.Title != "Test Title" {
			t.Errorf("Expected title to be preserved, got: %q", retrieved.Title)
		}

		if retrieved.Body != "Test Body" {
			t.Errorf("Expected body to be preserved, got: %q", retrieved.Body)
		}
	})

	t.Run("Current workaround should still work", func(t *testing.T) {
		// The current two-phase workaround should continue to work
		// to maintain backward compatibility

		// Phase 1: Create without body
		item := &TestBodyContentHandling{
			Status:   "pending",
			Assignee: "bob",
		}

		id, err := store.Create("Task without body", item)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Phase 2: Update with body content
		retrieved, err := store.Get(id)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		retrieved.Body = "Body added after creation"
		_, err = store.Update(id, retrieved)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		// Verify the workaround result
		final, err := store.Get(id)
		if err != nil {
			t.Fatalf("Final get failed: %v", err)
		}

		if final.Body != "Body added after creation" {
			t.Errorf("Expected body from update to be preserved, got: %q", final.Body)
		}
	})
}

func TestCreateMethodSignatureOptions(t *testing.T) {
	// Test different approaches to handle body content in Create

	tmpfile, err := os.CreateTemp("", "test_create_signatures*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TestBodyContentHandling](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("CreateWithBody method should handle body explicitly", func(t *testing.T) {
		// Future enhancement: dedicated method for creating with body
		// This test documents the desired API

		t.Skip("CreateWithBody method not yet implemented - future enhancement")

		// Desired API:
		// id, err := store.CreateWithBody("Title", "Body content", &TestBodyContentHandling{
		//     Status: "active",
		// })
	})

	t.Run("CreateFromStruct method should handle full struct", func(t *testing.T) {
		// Future enhancement: method that processes the entire struct including Document fields
		// This test documents the desired API

		t.Skip("CreateFromStruct method not yet implemented - future enhancement")

		// Desired API:
		// id, err := store.CreateFromStruct(&TestBodyContentHandling{
		//     Document: nanostore.Document{Title: "Title", Body: "Body"},
		//     Status: "active",
		// })
	})
}
