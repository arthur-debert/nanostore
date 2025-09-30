package main

import (
	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// TodoItem represents a todo item with hierarchical support
type TodoItem struct {
	nanostore.Document

	Status   string `values:"pending,active,done" prefix:"done=d" default:"pending"`
	Priority string `values:"low,medium,high" prefix:"high=h" default:"medium"`
	Activity string `values:"active,archived,deleted" default:"active"`
	ParentID string `dimension:"parent_id,ref"`

	// Non-dimension fields
	Description string
	AssignedTo  string
	Tags        string
	DueDate     string // ISO date string (YYYY-MM-DD)
}

// TodoApp manages the todo store and operations
type TodoApp struct {
	store *api.TypedStore[TodoItem]
}

// NewTodoApp creates a new todo application
func NewTodoApp(filePath string) (*TodoApp, error) {
	store, err := api.NewFromType[TodoItem](filePath)
	if err != nil {
		return nil, err
	}

	return &TodoApp{
		store: store,
	}, nil
}

// Close closes the todo store
func (app *TodoApp) Close() error {
	return app.store.Close()
}

// CreateTodo creates a new todo item
func (app *TodoApp) CreateTodo(title string, item *TodoItem) (string, error) {
	return app.store.Create(title, item)
}

// UpdateTodo updates an existing todo item
func (app *TodoApp) UpdateTodo(id string, item *TodoItem) error {
	return app.store.Update(id, item)
}

// GetTodo retrieves a todo by ID
func (app *TodoApp) GetTodo(id string) (*TodoItem, error) {
	return app.store.Get(id)
}

// DeleteTodo deletes a todo
func (app *TodoApp) DeleteTodo(id string, cascade bool) error {
	return app.store.Delete(id, cascade)
}

// GetAllActiveTodos returns all active todos (canonical view)
func (app *TodoApp) GetAllActiveTodos() ([]TodoItem, error) {
	return app.store.Query().
		Activity("active").
		StatusIn("pending", "active").
		Find()
}

// GetAllTodos returns all active todos including completed ones (full view)
func (app *TodoApp) GetAllTodos() ([]TodoItem, error) {
	return app.store.Query().
		Activity("active").
		Find()
}

// GetHighPriorityTodos returns high priority todos
func (app *TodoApp) GetHighPriorityTodos() ([]TodoItem, error) {
	return app.store.Query().
		Priority("high").
		Activity("active").
		Find()
}

// SearchTodos searches for todos containing the given text
func (app *TodoApp) SearchTodos(searchText string) ([]TodoItem, error) {
	return app.store.Query().
		Search(searchText).
		Activity("active").
		Find()
}

// GetRootTodos returns only root-level todos
func (app *TodoApp) GetRootTodos() ([]TodoItem, error) {
	return app.store.Query().
		ParentIDNotExists().
		Activity("active").
		Find()
}

// GetSubtasks returns subtasks of a specific todo
func (app *TodoApp) GetSubtasks(parentID string) ([]TodoItem, error) {
	return app.store.Query().
		ParentID(parentID).
		Activity("active").
		Find()
}

// GetStatistics returns various counts
func (app *TodoApp) GetStatistics() (map[string]int, error) {
	stats := make(map[string]int)

	totalCount, err := app.store.Query().Activity("active").Count()
	if err != nil {
		return nil, err
	}
	stats["total"] = totalCount

	pendingCount, err := app.store.Query().Status("pending").Activity("active").Count()
	if err != nil {
		return nil, err
	}
	stats["pending"] = pendingCount

	activeCount, err := app.store.Query().Status("active").Activity("active").Count()
	if err != nil {
		return nil, err
	}
	stats["active"] = activeCount

	doneCount, err := app.store.Query().Status("done").Count()
	if err != nil {
		return nil, err
	}
	stats["done"] = doneCount

	highPriorityCount, err := app.store.Query().Priority("high").Activity("active").Count()
	if err != nil {
		return nil, err
	}
	stats["high_priority"] = highPriorityCount

	return stats, nil
}
