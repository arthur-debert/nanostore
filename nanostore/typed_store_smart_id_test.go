package nanostore_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

// TestTask is a test type for typed store tests
type TestTask struct {
	nanostore.Document
	Status   string `values:"todo,done" default:"todo"`
	Priority string `values:"low,high" default:"low"`
}

func TestTypedStoreUpdateWithSmartID(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := nanostore.NewFromType[TestTask](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create a task
	uuid, err := store.Create("Test Task", &TestTask{
		Status:   "todo",
		Priority: "high",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get the task to retrieve its SimpleID
	tasks, err := store.Query().Find()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	simpleID := tasks[0].SimpleID

	// Test 1: Update using UUID should still work
	err = store.Update(uuid, &TestTask{
		Document: nanostore.Document{Title: "Updated via UUID"},
		Status:   "done",
		Priority: "high",
	})
	if err != nil {
		t.Errorf("Update with UUID failed: %v", err)
	}

	// Verify update
	task, err := store.Get(uuid)
	if err != nil {
		t.Fatal(err)
	}
	if task.Title != "Updated via UUID" || task.Status != "done" {
		t.Errorf("expected updated values, got title=%q status=%q", task.Title, task.Status)
	}

	// Test 2: Update using SimpleID should work
	err = store.Update(simpleID, &TestTask{
		Document: nanostore.Document{Title: "Updated via SimpleID"},
		Status:   "todo",
		Priority: "low",
	})
	if err != nil {
		t.Errorf("Update with SimpleID failed: %v", err)
	}

	// Verify update
	task, err = store.Get(simpleID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Title != "Updated via SimpleID" || task.Status != "todo" || task.Priority != "low" {
		t.Errorf("expected updated values, got title=%q status=%q priority=%q", task.Title, task.Status, task.Priority)
	}

	// Test 3: Update with invalid ID should fail
	err = store.Update("invalid-id", &TestTask{})
	if err == nil {
		t.Error("expected error for invalid ID, got nil")
	}
}

func TestTypedStoreDeleteWithSmartID(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := nanostore.NewFromType[TestTask](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create tasks
	uuid1, err := store.Create("Task 1", &TestTask{
		Status:   "todo",
		Priority: "high",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Create("Task 2", &TestTask{
		Status:   "done",
		Priority: "low",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get all tasks to find SimpleIDs
	tasks, err := store.Query().Find()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	// Find the simple ID for task 1
	var simpleID string
	for _, task := range tasks {
		if task.UUID == uuid1 {
			simpleID = task.SimpleID
			break
		}
	}

	// Test 1: Delete using UUID should work
	err = store.Delete(uuid1, false)
	if err != nil {
		t.Errorf("Delete with UUID failed: %v", err)
	}

	// Verify deletion
	tasks, err = store.Query().Find()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task after delete, got %d", len(tasks))
	}

	// Re-create task for next test
	_, err = store.Create("Task 1", &TestTask{
		Status:   "todo",
		Priority: "high",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get the new simple ID
	tasks, err = store.Query().Find()
	if err != nil {
		t.Fatal(err)
	}
	// Find the task we just created (it should have a different simple ID now)
	for _, task := range tasks {
		if task.Title == "Task 1" {
			simpleID = task.SimpleID
			break
		}
	}

	// Test 2: Delete using SimpleID should work
	err = store.Delete(simpleID, false)
	if err != nil {
		t.Errorf("Delete with SimpleID failed: %v", err)
	}

	// Verify deletion
	tasks, err = store.Query().Find()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task after delete, got %d", len(tasks))
	}

	// Test 3: Delete with invalid ID should fail
	err = store.Delete("invalid-id", false)
	if err == nil {
		t.Error("expected error for invalid ID, got nil")
	}
}
