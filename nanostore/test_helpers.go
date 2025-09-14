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
