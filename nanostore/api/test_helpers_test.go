package api_test

import "github.com/arthur-debert/nanostore/nanostore"

// TodoItem represents a todo item with hierarchical support
// Used across multiple test files
type TodoItem struct {
	nanostore.Document

	Status   string `values:"pending,active,done" prefix:"done=d" default:"pending"`
	Priority string `values:"low,medium,high" prefix:"high=h" default:"medium"`
	Activity string `values:"active,archived,deleted" default:"active"`
	ParentID string `dimension:"parent_id,ref"`
}
