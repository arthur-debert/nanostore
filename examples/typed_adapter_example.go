// Package examples shows how to use the typed API to eliminate boilerplate
package examples

import (
	"github.com/arthur-debert/nanostore/nanostore"
)

// TodoItem represents a todo task with typed dimension access
type TodoItem struct {
	nanostore.Document        // Embeds UUID, Title, Body, etc.
	Status             string `dimension:"status,default=pending"`
	Priority           string `dimension:"priority,default=medium"`
	ParentID           string `dimension:"parent_id,ref"` // ref indicates this is a hierarchical reference
}

// TodoAdapter wraps nanostore with typed operations
type TodoAdapter struct {
	store nanostore.Store
}

// NewTodoAdapter creates a new typed adapter
func NewTodoAdapter(dbPath string) (*TodoAdapter, error) {
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "completed"},
				Prefixes:     map[string]string{"completed": "c"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "medium", "high"},
				Prefixes:     map[string]string{"high": "h"},
				DefaultValue: "medium",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_id",
			},
		},
	}

	store, err := nanostore.New(dbPath, config)
	if err != nil {
		return nil, err
	}

	return &TodoAdapter{store: store}, nil
}

// BEFORE: The old boilerplate way
// func (t *TodoAdapter) GetPendingTasks() ([]OldTodoItem, error) {
//     docs, err := t.store.List(nanostore.ListOptions{
//         Filters: map[string]interface{}{"status": "pending"},
//     })
//     if err != nil {
//         return nil, err
//     }
//
//     var todos []OldTodoItem
//     for _, doc := range docs {
//         // Lots of boilerplate type assertions
//         status, _ := doc.Dimensions["status"].(string)
//         if status == "" {
//             status = "pending"
//         }
//         priority, _ := doc.Dimensions["priority"].(string)
//         if priority == "" {
//             priority = "medium"
//         }
//         parentID, _ := doc.Dimensions["parent_id"].(string)
//
//         todos = append(todos, OldTodoItem{
//             ID:       doc.UUID,
//             Title:    doc.Title,
//             Status:   status,
//             Priority: priority,
//             ParentID: parentID,
//         })
//     }
//     return todos, nil
// }

// AFTER: Clean typed API

// GetPendingTasks returns all pending tasks with direct field access
func (t *TodoAdapter) GetPendingTasks() ([]TodoItem, error) {
	return nanostore.ListTyped[TodoItem](t.store, nanostore.ListOptions{
		Filters: map[string]interface{}{"status": "pending"},
	})
}

// AddTask creates a new task - no more manual dimension mapping
func (t *TodoAdapter) AddTask(title string, priority string, parentID string) (*TodoItem, error) {
	task := &TodoItem{
		Status:   "pending", // Will use default if not specified
		Priority: priority,
		ParentID: parentID, // Can be empty string, UUID, or user-facing ID
	}

	id, err := nanostore.AddTyped(t.store, title, task)
	if err != nil {
		return nil, err
	}

	// Fetch the created task to get the full document
	docs, err := t.store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"uuid": id},
	})
	if err != nil || len(docs) == 0 {
		return nil, err
	}

	var created TodoItem
	err = nanostore.UnmarshalDimensions(docs[0], &created)
	return &created, err
}

// CompleteTask marks a task as completed
func (t *TodoAdapter) CompleteTask(id string) error {
	update := &TodoItem{
		Status: "completed",
	}
	return nanostore.UpdateTyped(t.store, id, update)
}

// GetTasksByPriority returns tasks filtered by priority
func (t *TodoAdapter) GetTasksByPriority(priority string) ([]TodoItem, error) {
	return nanostore.ListTyped[TodoItem](t.store, nanostore.ListOptions{
		Filters: map[string]interface{}{"priority": priority},
	})
}

// GetSubtasks returns all subtasks of a parent
func (t *TodoAdapter) GetSubtasks(parentID string) ([]TodoItem, error) {
	return nanostore.ListTyped[TodoItem](t.store, nanostore.ListOptions{
		Filters: map[string]interface{}{"parent_id": parentID},
	})
}

// Example of how clean the code becomes
func ExampleUsage() {
	adapter, _ := NewTodoAdapter(":memory:")

	// Add a parent task
	parent, _ := adapter.AddTask("Build feature", "high", "")

	// Add subtasks - parent.UUID works, or you could use parent.UserFacingID
	_, _ = adapter.AddTask("Write tests", "high", parent.UUID)
	_, _ = adapter.AddTask("Update docs", "medium", parent.UUID)

	// Get all high priority tasks
	highPriorityTasks, _ := adapter.GetTasksByPriority("high")
	for _, task := range highPriorityTasks {
		// Direct field access - no type assertions!
		println(task.Title, task.Status, task.Priority)
	}

	// Complete the parent task
	_ = adapter.CompleteTask(parent.UUID)

	// Get subtasks - clean and type-safe
	subtasks, _ := adapter.GetSubtasks(parent.UUID)
	for _, subtask := range subtasks {
		if subtask.Status == "pending" { // Direct comparison!
			println("Still need to:", subtask.Title)
		}
	}
}
