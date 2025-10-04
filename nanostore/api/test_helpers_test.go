package api_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)

import "github.com/arthur-debert/nanostore/nanostore"

// TodoItem represents a todo item with hierarchical support
// Used across multiple test files
type TodoItem struct {
	nanostore.Document

	Status   string `values:"pending,active,done" prefix:"done=d" default:"pending"`
	Priority string `values:"low,medium,high" prefix:"high=h" default:"medium"`
	Activity string `values:"active,archived,deleted" default:"active"`
	ParentID string `dimension:"parent_id,ref"`

	// Data fields (stored as _data.* with snake_case names)
	Assignee   string  // Custom field for assignee
	Estimate   int     // Custom field for effort estimate
	Tags       string  // Custom field for tags
	Team       string  // Custom field for team
	Score      float64 // Custom field for score
	Department string  // Custom field for department (used in stress tests)
	Complexity string  // Custom field for complexity (used in stress tests)
}
