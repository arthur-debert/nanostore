package nanostore

// Command types for preprocessing

// UpdateCommand represents an update operation
type UpdateCommand struct {
	ID      string `id:"true"`
	Request UpdateRequest
}

// DeleteCommand represents a delete operation
type DeleteCommand struct {
	ID      string `id:"true"`
	Cascade bool
}

// AddCommand represents an add operation
type AddCommand struct {
	Title      string
	Dimensions map[string]interface{}
}
