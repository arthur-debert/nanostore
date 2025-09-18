package nanostore

import (
	"os"
	"testing"
)

func TestCommandPreprocessor(t *testing.T) {
	// Setup test store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	config := Config{
		Dimensions: []DimensionConfig{
			{
				Name:         "status",
				Type:         Enumerated,
				Values:       []string{"active", "done"},
				DefaultValue: "active",
			},
			{
				Name:     "location",
				Type:     Hierarchical,
				RefField: "parent_id",
			},
		},
	}

	store, err := newJSONFileStore(tmpfile.Name(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create test documents
	parentUUID, err := store.Add("Parent", map[string]interface{}{
		"status": "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	childUUID, err := store.Add("Child", map[string]interface{}{
		"status":    "active",
		"parent_id": parentUUID,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get SimpleIDs
	docs, err := store.List(ListOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var parentSimpleID, childSimpleID string
	for _, doc := range docs {
		switch doc.UUID {
		case parentUUID:
			parentSimpleID = doc.SimpleID
		case childUUID:
			childSimpleID = doc.SimpleID
		}
	}

	preprocessor := newCommandPreprocessor(store)

	t.Run("ResolveUpdateCommand", func(t *testing.T) {
		cmd := &UpdateCommand{
			ID: childSimpleID, // Use SimpleID
			Request: UpdateRequest{
				Dimensions: map[string]interface{}{
					"status": "done",
				},
			},
		}

		err := preprocessor.preprocessCommand(cmd)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify ID was resolved to UUID
		if cmd.ID != childUUID {
			t.Errorf("expected ID to be resolved to %s, got %s", childUUID, cmd.ID)
		}
	})

	t.Run("ResolveDeleteCommand", func(t *testing.T) {
		cmd := &DeleteCommand{
			ID:      parentSimpleID, // Use SimpleID
			Cascade: true,
		}

		err := preprocessor.preprocessCommand(cmd)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify ID was resolved to UUID
		if cmd.ID != parentUUID {
			t.Errorf("expected ID to be resolved to %s, got %s", parentUUID, cmd.ID)
		}
	})

	t.Run("ResolveParentIDInDimensions", func(t *testing.T) {
		cmd := &AddCommand{
			Title: "New Child",
			Dimensions: map[string]interface{}{
				"status":    "active",
				"parent_id": parentSimpleID, // Use SimpleID for parent
			},
		}

		err := preprocessor.preprocessCommand(cmd)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify parent_id was resolved to UUID
		if cmd.Dimensions["parent_id"] != parentUUID {
			t.Errorf("expected parent_id to be resolved to %s, got %v", parentUUID, cmd.Dimensions["parent_id"])
		}
	})

	t.Run("PreserveValidUUID", func(t *testing.T) {
		cmd := &UpdateCommand{
			ID: parentUUID, // Already a UUID
			Request: UpdateRequest{
				Dimensions: map[string]interface{}{
					"status": "done",
				},
			},
		}

		err := preprocessor.preprocessCommand(cmd)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify UUID was not changed
		if cmd.ID != parentUUID {
			t.Errorf("expected ID to remain %s, got %s", parentUUID, cmd.ID)
		}
	})

	t.Run("HandleInvalidSimpleID", func(t *testing.T) {
		cmd := &UpdateCommand{
			ID: "invalid-id", // Invalid SimpleID
			Request: UpdateRequest{
				Dimensions: map[string]interface{}{
					"status": "done",
				},
			},
		}

		err := preprocessor.preprocessCommand(cmd)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify invalid ID was kept as-is (allows for external references)
		if cmd.ID != "invalid-id" {
			t.Errorf("expected ID to remain 'invalid-id', got %s", cmd.ID)
		}
	})

	t.Run("ResolveNestedIDs", func(t *testing.T) {
		cmd := &UpdateCommand{
			ID: childSimpleID,
			Request: UpdateRequest{
				Dimensions: map[string]interface{}{
					"status":    "done",
					"parent_id": parentSimpleID, // Nested ID in dimensions
				},
			},
		}

		err := preprocessor.preprocessCommand(cmd)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify both IDs were resolved
		if cmd.ID != childUUID {
			t.Errorf("expected ID to be resolved to %s, got %s", childUUID, cmd.ID)
		}
		if cmd.Request.Dimensions["parent_id"] != parentUUID {
			t.Errorf("expected parent_id to be resolved to %s, got %v", parentUUID, cmd.Request.Dimensions["parent_id"])
		}
	})
}
