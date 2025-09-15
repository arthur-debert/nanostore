package nanostore

// DefaultTestConfig returns the TodoConfig for backward compatibility in tests
// New tests should define their own domain-specific configurations
func DefaultTestConfig() Config {
	return TodoConfig()
}

// NewTestStore creates a store with the default test configuration
func NewTestStore(dbPath string) (Store, error) {
	return New(dbPath, DefaultTestConfig())
}

// TestAdd is a convenience method for tests that adds a document with default dimensions
func TestAdd(store Store, title string, parentID *string) (string, error) {
	dimensions := make(map[string]interface{})
	if parentID != nil {
		dimensions["parent_uuid"] = *parentID
	}
	return store.Add(title, dimensions)
}

// TestSetStatusUpdate is a helper function for tests to set the status dimension of a document
// This is a convenience wrapper around the generic Update method for backward compatibility
func TestSetStatusUpdate(store Store, id string, status string) error {
	return store.Update(id, UpdateRequest{
		Dimensions: map[string]string{"status": status},
	})
}
