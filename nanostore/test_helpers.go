package nanostore

// TestAdd is a convenience method for tests that adds a document with default dimensions
func TestAdd(store Store, title string, parentID *string) (string, error) {
	dimensions := make(map[string]interface{})
	if parentID != nil {
		dimensions["parent_uuid"] = *parentID
	}
	return store.Add(title, dimensions)
}
