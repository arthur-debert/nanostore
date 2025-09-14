package nanostore

// DefaultTestConfig returns a configuration that mimics the old hardcoded behavior
// for tests that rely on status (pending/completed) and parent dimensions.
func DefaultTestConfig() Config {
	return Config{
		Dimensions: []DimensionConfig{
			{
				Name:         "status",
				Type:         Enumerated,
				Values:       []string{"pending", "completed"},
				Prefixes:     map[string]string{"completed": "c"},
				DefaultValue: "pending",
			},
			{
				Name:     "parent",
				Type:     Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}
}

// NewTestStore creates a store with the default test configuration
func NewTestStore(dbPath string) (Store, error) {
	return New(dbPath, DefaultTestConfig())
}

// TestAdd is a convenience method for tests that adds a document with default dimensions
func TestAdd(store Store, title string, parentID *string) (string, error) {
	return store.Add(title, parentID, nil)
}
