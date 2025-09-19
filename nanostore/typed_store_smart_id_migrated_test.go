package nanostore_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/testutil"
)

// TestTaskMigrated is a test type for typed store tests
type TestTaskMigrated struct {
	nanostore.Document
	Status   string `values:"todo,done" default:"todo"`
	Priority string `values:"low,high" default:"low"`
}

func TestTypedStoreUpdateWithSmartIDMigrated(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := nanostore.NewFromType[TestTaskMigrated](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create a task
	uuid, err := store.Create("Test Task", &TestTaskMigrated{
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

	t.Run("UpdateUsingUUID", func(t *testing.T) {
		err = store.Update(uuid, &TestTaskMigrated{
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
	})

	t.Run("UpdateUsingSimpleID", func(t *testing.T) {
		err = store.Update(simpleID, &TestTaskMigrated{
			Document: nanostore.Document{Title: "Updated via SimpleID"},
			Status:   "todo",
			Priority: "low",
		})
		if err != nil {
			t.Errorf("Update with SimpleID failed: %v", err)
		}

		// Verify update
		task, err := store.Get(simpleID)
		if err != nil {
			t.Fatal(err)
		}
		if task.Title != "Updated via SimpleID" || task.Status != "todo" || task.Priority != "low" {
			t.Errorf("expected updated values, got title=%q status=%q priority=%q", task.Title, task.Status, task.Priority)
		}
	})

	t.Run("UpdateWithInvalidID", func(t *testing.T) {
		err = store.Update("invalid-id", &TestTaskMigrated{})
		if err == nil {
			t.Error("expected error for invalid ID, got nil")
		}
	})
}

func TestTypedStoreDeleteWithSmartIDMigrated(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := nanostore.NewFromType[TestTaskMigrated](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("DeleteUsingUUID", func(t *testing.T) {
		// Create a task
		uuid, err := store.Create("Task to delete by UUID", &TestTaskMigrated{
			Status:   "todo",
			Priority: "high",
		})
		if err != nil {
			t.Fatal(err)
		}

		// Verify task exists
		tasks, err := store.Query().Find()
		if err != nil {
			t.Fatal(err)
		}
		initialCount := len(tasks)

		// Delete using UUID
		err = store.Delete(uuid, false)
		if err != nil {
			t.Errorf("Delete with UUID failed: %v", err)
		}

		// Verify deletion
		tasks, err = store.Query().Find()
		if err != nil {
			t.Fatal(err)
		}
		if len(tasks) != initialCount-1 {
			t.Errorf("expected %d tasks after delete, got %d", initialCount-1, len(tasks))
		}
	})

	t.Run("DeleteUsingSimpleID", func(t *testing.T) {
		// Create a task
		uuid, err := store.Create("Task to delete by SimpleID", &TestTaskMigrated{
			Status:   "done",
			Priority: "low",
		})
		if err != nil {
			t.Fatal(err)
		}

		// Get the simple ID
		tasks, err := store.Query().Find()
		if err != nil {
			t.Fatal(err)
		}
		var simpleID string
		for _, task := range tasks {
			if task.UUID == uuid {
				simpleID = task.SimpleID
				break
			}
		}
		if simpleID == "" {
			t.Fatal("could not find SimpleID for created task")
		}
		initialCount := len(tasks)

		// Delete using SimpleID
		err = store.Delete(simpleID, false)
		if err != nil {
			t.Errorf("Delete with SimpleID failed: %v", err)
		}

		// Verify deletion
		tasks, err = store.Query().Find()
		if err != nil {
			t.Fatal(err)
		}
		if len(tasks) != initialCount-1 {
			t.Errorf("expected %d tasks after delete, got %d", initialCount-1, len(tasks))
		}
	})

	t.Run("DeleteWithInvalidID", func(t *testing.T) {
		err := store.Delete("invalid-id", false)
		if err == nil {
			t.Error("expected error for invalid ID, got nil")
		}
	})
}

// TestTypedStoreSmartIDWithFixtureMigrated tests typed store behavior with fixture data
func TestTypedStoreSmartIDWithFixtureMigrated(t *testing.T) {
	// Load fixture to verify smart IDs work consistently
	store, universe := testutil.LoadUniverse(t)

	t.Run("VerifySimpleIDsExist", func(t *testing.T) {
		// Get document by UUID first
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.TeamMeeting.UUID,
			},
		})
		if err != nil || len(docs) != 1 {
			t.Fatal("failed to get TeamMeeting")
		}

		simpleID := docs[0].SimpleID
		if simpleID == "" {
			t.Error("expected SimpleID to be populated")
		}

		// Verify SimpleID is populated (format may vary based on implementation)
		t.Logf("TeamMeeting SimpleID: %q", simpleID)
	})

	t.Run("UpdateBySimpleID", func(t *testing.T) {
		// Get a document with known parent to test hierarchical SimpleID
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.CodeReview.UUID,
			},
		})
		if err != nil || len(docs) != 1 {
			t.Fatal("failed to get CodeReview")
		}

		simpleID := docs[0].SimpleID
		newTitle := "Code Review Updated"

		// Update using SimpleID
		err = store.Update(simpleID, nanostore.UpdateRequest{
			Title: &newTitle,
		})
		if err != nil {
			t.Fatalf("failed to update by SimpleID: %v", err)
		}

		// Verify update by getting with UUID
		updatedDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.CodeReview.UUID,
			},
		})
		if err != nil || len(updatedDocs) != 1 {
			t.Fatal("failed to get updated doc")
		}

		if updatedDocs[0].Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, updatedDocs[0].Title)
		}
	})
}
