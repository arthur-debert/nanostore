package migration

import (
	"testing"

	"github.com/arthur-debert/nanostore/types"
)

func TestAPIRenameField(t *testing.T) {
	// Setup test documents
	docs := []types.Document{
		{
			UUID:  "1",
			Title: "Doc 1",
			Dimensions: map[string]interface{}{
				"old_status": "active",
				"priority":   "high",
			},
		},
		{
			UUID:  "2",
			Title: "Doc 2",
			Dimensions: map[string]interface{}{
				"old_status": "pending",
				"priority":   "low",
			},
		},
	}

	api := NewAPI()

	t.Run("successful rename", func(t *testing.T) {
		// Make a copy of docs for this test
		testDocs := make([]types.Document, len(docs))
		copy(testDocs, docs)
		for i := range testDocs {
			testDocs[i].Dimensions = make(map[string]interface{})
			for k, v := range docs[i].Dimensions {
				testDocs[i].Dimensions[k] = v
			}
		}

		result := api.RenameField(testDocs, types.Config{}, "old_status", "status", Options{
			DryRun:  false,
			Verbose: false,
		})

		if !result.Success {
			t.Errorf("expected success, got failure")
		}
		if result.Code != CodeSuccess {
			t.Errorf("expected code %d, got %d", CodeSuccess, result.Code)
		}
		if result.Stats.ModifiedDocs != 2 {
			t.Errorf("expected 2 modified docs, got %d", result.Stats.ModifiedDocs)
		}

		// Verify rename in documents
		for _, doc := range testDocs {
			if _, exists := doc.Dimensions["status"]; !exists {
				t.Errorf("status field not found in doc %s", doc.UUID)
			}
			if _, exists := doc.Dimensions["old_status"]; exists {
				t.Errorf("old_status field still exists in doc %s", doc.UUID)
			}
		}
	})

	t.Run("dry run", func(t *testing.T) {
		// Make a copy of docs for this test
		testDocs := make([]types.Document, len(docs))
		copy(testDocs, docs)
		for i := range testDocs {
			testDocs[i].Dimensions = make(map[string]interface{})
			for k, v := range docs[i].Dimensions {
				testDocs[i].Dimensions[k] = v
			}
		}

		result := api.RenameField(testDocs, types.Config{}, "old_status", "status", Options{
			DryRun:  true,
			Verbose: false,
		})

		if !result.Success {
			t.Errorf("expected success, got failure")
		}

		// Verify no changes in dry run
		for i, doc := range testDocs {
			if _, exists := doc.Dimensions["status"]; exists {
				t.Errorf("status field found in doc %s during dry run", doc.UUID)
			}
			if _, exists := doc.Dimensions["old_status"]; !exists {
				t.Errorf("old_status field missing in doc %s during dry run", doc.UUID)
			}

			// Compare with original
			if testDocs[i].Dimensions["old_status"] != docs[i].Dimensions["old_status"] {
				t.Error("document was modified during dry run")
			}
		}
	})
}
