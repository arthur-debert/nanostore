package nanostore_test

import (
	"fmt"
	"log"
	"sort"

	"github.com/arthur-debert/nanostore/nanostore"
)

func ExampleNew() {
	// Define custom dimensions for a project management system
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "normal", "high", "urgent"},
				Prefixes:     map[string]string{"high": "h", "urgent": "u"},
				DefaultValue: "normal",
			},
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"backlog", "todo", "in_progress", "done"},
				Prefixes:     map[string]string{"in_progress": "p", "done": "d"},
				DefaultValue: "backlog",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_task_id",
			},
		},
	}

	// Create store with custom configuration
	store, err := nanostore.New(":memory:", config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Add some tasks
	epic, _ := store.Add("Q1 Product Launch", nil)
	task1, _ := store.Add("Design mockups", map[string]interface{}{"parent_uuid": epic})
	task2, _ := store.Add("Implement backend", map[string]interface{}{"parent_uuid": epic})

	// Update statuses
	_ = store.Update(task1, nanostore.UpdateRequest{
		Dimensions: map[string]interface{}{"status": "done"},
	})
	_ = store.Update(task2, nanostore.UpdateRequest{
		Dimensions: map[string]interface{}{"status": "in_progress"},
	})

	// List all documents
	docs, _ := store.List(nanostore.ListOptions{})

	// Sort by ID for consistent output
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].UserFacingID < docs[j].UserFacingID
	})

	for _, doc := range docs {
		fmt.Printf("ID: %s, Title: %s\n", doc.UserFacingID, doc.Title)
	}

	// Output:
	// ID: 1, Title: Q1 Product Launch
	// ID: 1.d1, Title: Design mockups
	// ID: 1.p1, Title: Implement backend
}
