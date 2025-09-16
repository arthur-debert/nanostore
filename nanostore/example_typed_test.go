package nanostore_test

import (
	"fmt"
	"log"

	"github.com/arthur-debert/nanostore/nanostore"
)

// Define your domain model with dimension tags
type Task struct {
	nanostore.Document        // Embed document fields
	Status             string `dimension:"status,default=pending"`
	Priority           string `dimension:"priority,default=medium"`
	Assignee           string `dimension:"assignee"`
	ParentID           string `dimension:"parent_id,ref"` // ref indicates hierarchical reference
}

func Example_typedAPI() {
	// Create store with configuration
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "in_progress", "completed", "archived"},
				Prefixes:     map[string]string{"completed": "c", "archived": "a"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "medium", "high", "urgent"},
				Prefixes:     map[string]string{"high": "h", "urgent": "u"},
				DefaultValue: "medium",
			},
			{
				Name:         "assignee",
				Type:         nanostore.Enumerated,
				Values:       []string{"unassigned", "alice", "bob", "charlie"},
				DefaultValue: "unassigned",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_id",
			},
		},
	}

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// BEFORE: The old boilerplate way
	// id, err := store.Add("Implement feature", map[string]interface{}{
	//     "status": "pending",
	//     "priority": "high",
	//     "assignee": "alice",
	// })
	// ...
	// doc := docs[0]
	// status, _ := doc.Dimensions["status"].(string)
	// priority, _ := doc.Dimensions["priority"].(string)
	// if status == "" {
	//     status = "pending"
	// }

	// AFTER: Clean typed API

	// Create a task using typed struct - no more map[string]interface{}!
	task := &Task{
		Status:   "in_progress",
		Priority: "high",
		Assignee: "alice",
	}

	parentID, err := nanostore.AddTyped(store, "Implement login feature", task)
	if err != nil {
		log.Fatal(err)
	}

	// Add subtasks with parent reference - smart ID resolution works!
	subtask := &Task{
		Priority: "urgent",
		Assignee: "bob",
		ParentID: parentID, // Can use UUID or user-facing ID
	}

	_, err = nanostore.AddTyped(store, "Add OAuth support", subtask)
	if err != nil {
		log.Fatal(err)
	}

	// List typed documents - no more type assertions!
	tasks, err := nanostore.ListTyped[Task](store, nanostore.ListOptions{
		Filters: map[string]interface{}{"status": "in_progress"},
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, t := range tasks {
		// Direct field access - no more doc.Dimensions["status"].(string)!
		fmt.Printf("Task: %s\n", t.Title)
		fmt.Printf("  Status: %s\n", t.Status)     // Clean field access
		fmt.Printf("  Priority: %s\n", t.Priority) // Type safe
		fmt.Printf("  Assignee: %s\n", t.Assignee) // No type assertions

		// Defaults are handled automatically
		// If status wasn't set, it would be "pending"
	}

	// Update using typed struct
	taskUpdate := &Task{
		Status:   "completed",
		Assignee: "charlie",
	}

	err = nanostore.UpdateTyped(store, parentID, taskUpdate)
	if err != nil {
		log.Fatal(err)
	}

	// You can still use the regular API when needed
	newTitle := "Implement authentication"
	err = store.Update(parentID, nanostore.UpdateRequest{
		Title: &newTitle,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Output:
	// Task: Implement login feature
	//   Status: in_progress
	//   Priority: high
	//   Assignee: alice
}

func ExampleMarshalDimensions() {
	// You can also use the marshaling functions directly

	task := &Task{
		Status:   "completed",
		Priority: "low",
		Assignee: "alice",
		ParentID: "parent-123",
	}

	dimensions, err := nanostore.MarshalDimensions(task)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Dimensions: %+v\n", dimensions)

	// Output:
	// Dimensions: map[assignee:alice parent_id:parent-123 priority:low status:completed]
}

func ExampleUnmarshalDimensions() {
	// Convert a document back to your typed struct

	doc := nanostore.Document{
		UUID:         "abc-123",
		UserFacingID: "h1",
		Title:        "Important task",
		Dimensions: map[string]interface{}{
			"status":   "in_progress",
			"priority": "high",
			"assignee": "bob",
		},
	}

	var task Task
	err := nanostore.UnmarshalDimensions(doc, &task)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Task: %s (Status: %s, Priority: %s)\n",
		task.Title, task.Status, task.Priority)

	// Output:
	// Task: Important task (Status: in_progress, Priority: high)
}
