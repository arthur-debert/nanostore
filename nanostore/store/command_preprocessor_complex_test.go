package store

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// EXCEPTION: This is an internal package test that needs access to unexported types.
// It cannot use the standard fixture approach but should still follow other best practices where possible.

import (
	"os"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/types"
)

// Complex test structures to test nested resolution

type ComplexCommand struct {
	ID            string `id:"true"`
	NestedCommand *NestedCommand
	Commands      []SubCommand
	Metadata      map[string]interface{}
}

type NestedCommand struct {
	ParentID    string `id:"true"`
	RefID       string `id:"true"`
	DeepNested  *DeeplyNestedCommand
	StringField string
	IntField    int
	TimeField   time.Time
}

type DeeplyNestedCommand struct {
	AncestorID   string `id:"true"`
	DataMap      map[string]interface{}
	PointerField *string `id:"true"`
}

type SubCommand struct {
	SubID     string `id:"true"`
	SubData   string
	SubNested *NestedCommand
}

func TestCommandPreprocessorComplexStructs(t *testing.T) {
	// Setup test store
	store := createTestStoreWithDocuments(t)
	preprocessor := newCommandPreprocessor(store)

	// Get test document IDs
	docs := getTestDocuments(t, store)
	doc1UUID := docs[0].UUID
	doc1SimpleID := docs[0].SimpleID
	doc2UUID := docs[1].UUID
	doc2SimpleID := docs[1].SimpleID
	doc3UUID := docs[2].UUID
	doc3SimpleID := docs[2].SimpleID

	t.Run("DeeplyNestedStructResolution", func(t *testing.T) {
		ptrID := doc3SimpleID
		cmd := &ComplexCommand{
			ID: doc1SimpleID,
			NestedCommand: &NestedCommand{
				ParentID:    doc2SimpleID,
				RefID:       doc3SimpleID,
				StringField: "test",
				IntField:    42,
				TimeField:   time.Now(),
				DeepNested: &DeeplyNestedCommand{
					AncestorID:   doc1SimpleID,
					PointerField: &ptrID,
					DataMap: map[string]interface{}{
						"key1": "value1",
						"key2": 123,
					},
				},
			},
			Commands: []SubCommand{
				{
					SubID:   doc2SimpleID,
					SubData: "data1",
					SubNested: &NestedCommand{
						ParentID: doc3SimpleID,
						RefID:    doc1SimpleID,
					},
				},
				{
					SubID:   doc3SimpleID,
					SubData: "data2",
				},
			},
			Metadata: map[string]interface{}{
				"meta_key": "meta_value",
				"count":    10,
			},
		}

		err := preprocessor.preprocessCommand(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify all IDs were resolved
		if cmd.ID != doc1UUID {
			t.Errorf("expected ID %s, got %s", doc1UUID, cmd.ID)
		}
		if cmd.NestedCommand.ParentID != doc2UUID {
			t.Errorf("expected NestedCommand.ParentID %s, got %s", doc2UUID, cmd.NestedCommand.ParentID)
		}
		if cmd.NestedCommand.RefID != doc3UUID {
			t.Errorf("expected NestedCommand.RefID %s, got %s", doc3UUID, cmd.NestedCommand.RefID)
		}
		if cmd.NestedCommand.DeepNested.AncestorID != doc1UUID {
			t.Errorf("expected DeepNested.AncestorID %s, got %s", doc1UUID, cmd.NestedCommand.DeepNested.AncestorID)
		}
		if *cmd.NestedCommand.DeepNested.PointerField != doc3UUID {
			t.Errorf("expected DeepNested.PointerField %s, got %s", doc3UUID, *cmd.NestedCommand.DeepNested.PointerField)
		}

		// Verify slice elements were processed
		if len(cmd.Commands) != 2 {
			t.Fatalf("expected 2 commands, got %d", len(cmd.Commands))
		}
		if cmd.Commands[0].SubID != doc2UUID {
			t.Errorf("expected Commands[0].SubID %s, got %s", doc2UUID, cmd.Commands[0].SubID)
		}
		if cmd.Commands[0].SubNested.ParentID != doc3UUID {
			t.Errorf("expected Commands[0].SubNested.ParentID %s, got %s", doc3UUID, cmd.Commands[0].SubNested.ParentID)
		}
		if cmd.Commands[1].SubID != doc3UUID {
			t.Errorf("expected Commands[1].SubID %s, got %s", doc3UUID, cmd.Commands[1].SubID)
		}

		// Verify non-ID fields remain unchanged
		if cmd.NestedCommand.StringField != "test" {
			t.Errorf("StringField changed: %s", cmd.NestedCommand.StringField)
		}
		if cmd.NestedCommand.IntField != 42 {
			t.Errorf("IntField changed: %d", cmd.NestedCommand.IntField)
		}
		if cmd.Metadata["meta_key"] != "meta_value" {
			t.Errorf("Metadata changed")
		}
	})

	t.Run("NilFieldHandling", func(t *testing.T) {
		cmd := &ComplexCommand{
			ID:            doc1SimpleID,
			NestedCommand: nil, // nil pointer
			Commands:      nil, // nil slice
			Metadata:      nil, // nil map
		}

		err := preprocessor.preprocessCommand(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should handle nils gracefully
		if cmd.ID != doc1UUID {
			t.Errorf("expected ID %s, got %s", doc1UUID, cmd.ID)
		}
		if cmd.NestedCommand != nil {
			t.Error("expected NestedCommand to remain nil")
		}
		if cmd.Commands != nil {
			t.Error("expected Commands to remain nil")
		}
		if cmd.Metadata != nil {
			t.Error("expected Metadata to remain nil")
		}
	})

	t.Run("MixedValidAndInvalidIDs", func(t *testing.T) {
		invalidID := "invalid-simple-id"
		cmd := &ComplexCommand{
			ID: doc1SimpleID, // Valid
			NestedCommand: &NestedCommand{
				ParentID: invalidID,    // Invalid
				RefID:    doc2SimpleID, // Valid
				DeepNested: &DeeplyNestedCommand{
					AncestorID: "another-invalid", // Invalid
				},
			},
		}

		err := preprocessor.preprocessCommand(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Valid IDs should be resolved
		if cmd.ID != doc1UUID {
			t.Errorf("expected ID %s, got %s", doc1UUID, cmd.ID)
		}
		if cmd.NestedCommand.RefID != doc2UUID {
			t.Errorf("expected RefID %s, got %s", doc2UUID, cmd.NestedCommand.RefID)
		}

		// Invalid IDs should remain unchanged
		if cmd.NestedCommand.ParentID != invalidID {
			t.Errorf("expected ParentID to remain %s, got %s", invalidID, cmd.NestedCommand.ParentID)
		}
		if cmd.NestedCommand.DeepNested.AncestorID != "another-invalid" {
			t.Errorf("expected AncestorID to remain 'another-invalid', got %s", cmd.NestedCommand.DeepNested.AncestorID)
		}
	})

	t.Run("EmptyStructHandling", func(t *testing.T) {
		cmd := &ComplexCommand{}

		err := preprocessor.preprocessCommand(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Empty strings should remain empty
		if cmd.ID != "" {
			t.Errorf("expected empty ID, got %s", cmd.ID)
		}
	})

	t.Run("AlreadyResolvedUUIDs", func(t *testing.T) {
		cmd := &ComplexCommand{
			ID: doc1UUID, // Already a UUID
			NestedCommand: &NestedCommand{
				ParentID: doc2UUID, // Already a UUID
				RefID:    doc3UUID, // Already a UUID
			},
		}

		err := preprocessor.preprocessCommand(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// UUIDs should remain unchanged
		if cmd.ID != doc1UUID {
			t.Errorf("UUID changed: expected %s, got %s", doc1UUID, cmd.ID)
		}
		if cmd.NestedCommand.ParentID != doc2UUID {
			t.Errorf("UUID changed: expected %s, got %s", doc2UUID, cmd.NestedCommand.ParentID)
		}
		if cmd.NestedCommand.RefID != doc3UUID {
			t.Errorf("UUID changed: expected %s, got %s", doc3UUID, cmd.NestedCommand.RefID)
		}
	})
}

// Helper to create a test store with some documents
func createTestStoreWithDocuments(t *testing.T) *jsonFileStore {
	t.Helper()

	config := &testConfig{
		dimensions: []types.DimensionConfig{
			{
				Name:         "status",
				Type:         types.Enumerated,
				Values:       []string{"active", "inactive"},
				DefaultValue: "active",
			},
			{
				Name:     "location",
				Type:     types.Hierarchical,
				RefField: "parent_id",
			},
		},
	}

	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(tmpfile.Name()) })
	_ = tmpfile.Close()

	store, err := newJSONFileStore(tmpfile.Name(), config)
	if err != nil {
		t.Fatal(err)
	}

	// Add test documents
	_, err = store.Add("Doc1", map[string]interface{}{"status": "active"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Add("Doc2", map[string]interface{}{"status": "active"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Add("Doc3", map[string]interface{}{"status": "inactive"})
	if err != nil {
		t.Fatal(err)
	}

	return store
}

// Helper to get test documents
func getTestDocuments(t *testing.T, store *jsonFileStore) []types.Document {
	t.Helper()

	docs, err := store.List(types.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) < 3 {
		t.Fatalf("expected at least 3 documents, got %d", len(docs))
	}
	return docs
}
