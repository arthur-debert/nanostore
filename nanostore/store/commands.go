package store

import "github.com/arthur-debert/nanostore/types"

// Command types for preprocessing

// UpdateCommand represents an update operation
type UpdateCommand struct {
	ID      string `id:"true"`
	Request types.UpdateRequest
}

// DeleteCommand represents a delete operation
type DeleteCommand struct {
	ID      string `id:"true"`
	Cascade bool
}

// AddCommand represents an add operation
type AddCommand struct {
	Title      string
	Body       string
	Dimensions map[string]interface{}
}
