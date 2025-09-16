package nanostore_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
)

// Define test document types
type TodoItem struct {
	nanostore.Document
	Status   string `dimension:"status,default=pending"`
	Priority string `dimension:"priority,default=medium"`
	ParentID string `dimension:"parent_id,ref"`
}

type Project struct {
	nanostore.Document
	Title    string // Not a dimension
	Status   string `dimension:"status"`
	Category string `dimension:"category,default=general"`
	Owner    string `dimension:"owner"`
	Active   bool   `dimension:"active,default=true"`
}

func TestMarshalDimensions(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name: "basic todo item",
			input: &TodoItem{
				Status:   "completed",
				Priority: "high",
			},
			expected: map[string]interface{}{
				"status":   "completed",
				"priority": "high",
			},
		},
		{
			name: "todo item with parent",
			input: &TodoItem{
				Status:   "in_progress",
				Priority: "low",
				ParentID: "parent-uuid-123",
			},
			expected: map[string]interface{}{
				"status":    "in_progress",
				"priority":  "low",
				"parent_id": "parent-uuid-123",
			},
		},
		{
			name: "project with bool",
			input: &Project{
				Status:   "active",
				Category: "internal",
				Active:   true,
			},
			expected: map[string]interface{}{
				"status":   "active",
				"category": "internal",
				"active":   true,
			},
		},
		{
			name: "skip zero values without defaults",
			input: &Project{
				Status: "draft",
				Owner:  "", // Zero value, no default
			},
			expected: map[string]interface{}{
				"status": "draft",
				"active": false, // Has explicit false value
			},
		},
		{
			name:    "non-struct input",
			input:   "not a struct",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nanostore.MarshalDimensions(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalDimensions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if len(got) != len(tt.expected) {
				t.Errorf("MarshalDimensions() got %d dimensions, want %d", len(got), len(tt.expected))
				t.Logf("got: %+v", got)
				t.Logf("want: %+v", tt.expected)
			}

			for k, v := range tt.expected {
				if got[k] != v {
					t.Errorf("MarshalDimensions() dimension %s = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestUnmarshalDimensions(t *testing.T) {
	tests := []struct {
		name     string
		doc      nanostore.Document
		target   interface{}
		validate func(t *testing.T, v interface{})
		wantErr  bool
	}{
		{
			name: "basic unmarshal",
			doc: nanostore.Document{
				UUID:         "test-uuid",
				UserFacingID: "1",
				Title:        "Test Todo",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				Dimensions: map[string]interface{}{
					"status":   "completed",
					"priority": "high",
				},
			},
			target: &TodoItem{},
			validate: func(t *testing.T, v interface{}) {
				todo := v.(*TodoItem)
				if todo.Status != "completed" {
					t.Errorf("expected status 'completed', got %s", todo.Status)
				}
				if todo.Priority != "high" {
					t.Errorf("expected priority 'high', got %s", todo.Priority)
				}
				if todo.UUID != "test-uuid" {
					t.Errorf("expected UUID 'test-uuid', got %s", todo.UUID)
				}
			},
		},
		{
			name: "unmarshal with defaults",
			doc: nanostore.Document{
				UUID:      "test-uuid-2",
				Title:     "Test Todo 2",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Dimensions: map[string]interface{}{
					"status": "in_progress",
					// priority not set, should use default
				},
			},
			target: &TodoItem{},
			validate: func(t *testing.T, v interface{}) {
				todo := v.(*TodoItem)
				if todo.Status != "in_progress" {
					t.Errorf("expected status 'in_progress', got %s", todo.Status)
				}
				if todo.Priority != "medium" {
					t.Errorf("expected priority 'medium' (default), got %s", todo.Priority)
				}
			},
		},
		{
			name: "unmarshal with type conversion",
			doc: nanostore.Document{
				UUID:      "test-uuid-3",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Dimensions: map[string]interface{}{
					"status":   "active",
					"category": "work",
					"active":   "true", // String that should convert to bool
				},
			},
			target: &Project{},
			validate: func(t *testing.T, v interface{}) {
				proj := v.(*Project)
				if proj.Status != "active" {
					t.Errorf("expected status 'active', got %s", proj.Status)
				}
				if !proj.Active {
					t.Errorf("expected active to be true")
				}
			},
		},
		{
			name: "unmarshal with parent ref",
			doc: nanostore.Document{
				UUID:      "test-uuid-4",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Dimensions: map[string]interface{}{
					"status":    "pending",
					"priority":  "low",
					"parent_id": "parent-123",
				},
			},
			target: &TodoItem{},
			validate: func(t *testing.T, v interface{}) {
				todo := v.(*TodoItem)
				if todo.ParentID != "parent-123" {
					t.Errorf("expected parent_id 'parent-123', got %s", todo.ParentID)
				}
			},
		},
		{
			name: "unmarshal to non-pointer",
			doc: nanostore.Document{
				Dimensions: map[string]interface{}{},
			},
			target:  TodoItem{}, // Not a pointer
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := nanostore.UnmarshalDimensions(tt.doc, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalDimensions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if tt.validate != nil {
				tt.validate(t, tt.target)
			}
		})
	}
}

func TestTypedStoreOperations(t *testing.T) {
	// Create test configuration
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "in_progress", "completed"},
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

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("AddTyped", func(t *testing.T) {
		todo := &TodoItem{
			Status:   "in_progress",
			Priority: "high",
		}

		id, err := nanostore.AddTyped(store, "Test Task", todo)
		if err != nil {
			t.Fatalf("failed to add typed document: %v", err)
		}

		// Verify by reading back
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		if len(docs) != 1 {
			t.Fatalf("expected 1 document, got %d", len(docs))
		}

		var retrieved TodoItem
		if err := nanostore.UnmarshalDimensions(docs[0], &retrieved); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if retrieved.Status != "in_progress" {
			t.Errorf("expected status 'in_progress', got %s", retrieved.Status)
		}
		if retrieved.Priority != "high" {
			t.Errorf("expected priority 'high', got %s", retrieved.Priority)
		}
		if retrieved.UUID != id {
			t.Errorf("expected UUID %s, got %s", id, retrieved.UUID)
		}
	})

	t.Run("AddTyped with parent reference", func(t *testing.T) {
		// First create a parent
		parent := &TodoItem{
			Status: "pending",
		}
		parentID, err := nanostore.AddTyped(store, "Parent Task", parent)
		if err != nil {
			t.Fatalf("failed to add parent: %v", err)
		}

		// Create child with parent reference
		child := &TodoItem{
			Status:   "pending",
			Priority: "medium",
			ParentID: parentID,
		}
		childID, err := nanostore.AddTyped(store, "Child Task", child)
		if err != nil {
			t.Fatalf("failed to add child: %v", err)
		}

		// Verify the child
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"parent_id": parentID},
		})
		if err != nil {
			t.Fatalf("failed to list children: %v", err)
		}

		if len(docs) != 1 {
			t.Fatalf("expected 1 child, got %d", len(docs))
		}

		var retrieved TodoItem
		if err := nanostore.UnmarshalDimensions(docs[0], &retrieved); err != nil {
			t.Fatalf("failed to unmarshal child: %v", err)
		}

		if retrieved.ParentID != parentID {
			t.Errorf("expected parent_id %s, got %s", parentID, retrieved.ParentID)
		}
		if retrieved.UUID != childID {
			t.Errorf("expected UUID %s, got %s", childID, retrieved.UUID)
		}
	})

	t.Run("UpdateTyped", func(t *testing.T) {
		todo := &TodoItem{
			Status:   "pending",
			Priority: "low",
		}

		id, err := nanostore.AddTyped(store, "Update Test", todo)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Update using typed struct
		updated := &TodoItem{
			Status:   "completed",
			Priority: "high",
		}

		err = nanostore.UpdateTyped(store, id, updated)
		if err != nil {
			t.Fatalf("failed to update: %v", err)
		}

		// Verify update
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		found := false
		for _, doc := range docs {
			if doc.UUID == id {
				found = true
				var retrieved TodoItem
				if err := nanostore.UnmarshalDimensions(doc, &retrieved); err != nil {
					t.Fatalf("failed to unmarshal: %v", err)
				}
				if retrieved.Status != "completed" {
					t.Errorf("expected status 'completed', got %s", retrieved.Status)
				}
				if retrieved.Priority != "high" {
					t.Errorf("expected priority 'high', got %s", retrieved.Priority)
				}
				break
			}
		}

		if !found {
			t.Errorf("updated document not found")
		}
	})

	t.Run("ListTyped", func(t *testing.T) {
		// Clear existing data
		_ = store.Close()
		store, err = nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Add some documents
		todos := []TodoItem{
			{Status: "pending", Priority: "high"},
			{Status: "completed", Priority: "low"},
			{Status: "in_progress", Priority: "medium"},
		}

		for i, todo := range todos {
			_, err := store.Add(fmt.Sprintf("Task %d", i+1), map[string]interface{}{
				"status":   todo.Status,
				"priority": todo.Priority,
			})
			if err != nil {
				t.Fatalf("failed to add document %d: %v", i, err)
			}
		}

		// List typed documents
		retrieved, err := nanostore.ListTyped[TodoItem](store, nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list typed: %v", err)
		}

		if len(retrieved) != 3 {
			t.Fatalf("expected 3 documents, got %d", len(retrieved))
		}

		// Verify all documents have proper typing
		for i, item := range retrieved {
			if item.Title == "" {
				t.Errorf("document %d missing title", i)
			}
			if item.UUID == "" {
				t.Errorf("document %d missing UUID", i)
			}
			if item.Status == "" {
				t.Errorf("document %d missing status", i)
			}
		}

		// Test filtered listing
		completed, err := nanostore.ListTyped[TodoItem](store, nanostore.ListOptions{
			Filters: map[string]interface{}{"status": "completed"},
		})
		if err != nil {
			t.Fatalf("failed to list completed: %v", err)
		}

		if len(completed) != 1 {
			t.Fatalf("expected 1 completed item, got %d", len(completed))
		}

		if completed[0].Status != "completed" {
			t.Errorf("expected status 'completed', got %s", completed[0].Status)
		}
	})
}
