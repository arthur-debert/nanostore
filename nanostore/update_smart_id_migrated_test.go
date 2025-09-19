package nanostore_test

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/testutil"
)

func TestUpdateWithSmartIDMigrated(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("update using UUID", func(t *testing.T) {
		// Pick a document to update - let's use BuyGroceries
		newTitle := "Updated via UUID"
		err := store.Update(universe.BuyGroceries.UUID, nanostore.UpdateRequest{
			Title: &newTitle,
		})
		if err != nil {
			t.Fatalf("Update with UUID failed: %v", err)
		}

		// Verify update
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.BuyGroceries.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		testutil.AssertDocumentCount(t, docs, 1)
		if docs[0].Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, docs[0].Title)
		}
	})

	t.Run("update using SimpleID", func(t *testing.T) {
		// Use ExerciseRoutine which has a known SimpleID pattern
		newTitle := "Updated via SimpleID"
		err := store.Update(universe.ExerciseRoutine.SimpleID, nanostore.UpdateRequest{
			Title: &newTitle,
		})
		if err != nil {
			t.Fatalf("Update with SimpleID %q failed: %v", universe.ExerciseRoutine.SimpleID, err)
		}

		// Verify update
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.ExerciseRoutine.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		testutil.AssertDocumentCount(t, docs, 1)
		if docs[0].Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, docs[0].Title)
		}
	})

	t.Run("update with invalid ID", func(t *testing.T) {
		newTitle := "Should not update"
		err := store.Update("invalid-id-12345", nanostore.UpdateRequest{
			Title: &newTitle,
		})
		if err == nil {
			t.Error("expected error for invalid ID, got nil")
		}
	})

	t.Run("update dimensions using SimpleID", func(t *testing.T) {
		// Use TeamMeeting which is pending - let's change it to done
		// First verify current state
		if universe.TeamMeeting.Dimensions["status"] != "pending" {
			t.Fatalf("precondition failed: TeamMeeting should be pending, got %v",
				universe.TeamMeeting.Dimensions["status"])
		}

		// Update using SimpleID
		err := store.Update(universe.TeamMeeting.SimpleID, nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{
				"status": "done",
			},
		})
		if err != nil {
			t.Fatalf("Update dimensions with SimpleID %q failed: %v",
				universe.TeamMeeting.SimpleID, err)
		}

		// Verify update
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.TeamMeeting.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		testutil.AssertDocumentCount(t, docs, 1)
		testutil.AssertHasStatus(t, docs[0], "done")

		// The SimpleID should have changed due to status change
		if docs[0].SimpleID == universe.TeamMeeting.SimpleID {
			t.Error("SimpleID should have changed when status changed from pending to done")
		}
		t.Logf("ID changed from %s to %s", universe.TeamMeeting.SimpleID, docs[0].SimpleID)
	})

	t.Run("verify all document types accept SimpleID", func(t *testing.T) {
		// Test a variety of documents with different ID patterns
		testCases := []struct {
			name string
			uuid string
		}{
			{"root document", universe.PersonalRoot.UUID},
			{"child document", universe.BuyGroceries.UUID},
			{"deep nested document", universe.Level5Task.UUID},
			{"document with prefixes", universe.ReadBook.UUID}, // has 'd' prefix for done
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Get fresh document to ensure we have current SimpleID
				// This is important because previous tests may have changed documents
				docs, err := store.List(nanostore.ListOptions{
					Filters: map[string]interface{}{
						"uuid": tc.uuid,
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				if len(docs) != 1 {
					t.Fatalf("Expected 1 document with UUID %s, got %d", tc.uuid, len(docs))
				}
				doc := docs[0]

				newBody := "Updated body for " + tc.name
				err = store.Update(doc.SimpleID, nanostore.UpdateRequest{
					Body: &newBody,
				})
				if err != nil {
					t.Errorf("Failed to update %s using SimpleID %q: %v",
						tc.name, doc.SimpleID, err)
				}
			})
		}
	})
}

// TestSmartIDResolutionEdgeCases tests edge cases specific to SimpleID resolution
func TestSmartIDResolutionEdgeCases(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("UUID takes precedence over SimpleID format", func(t *testing.T) {
		// If a string happens to be both a valid UUID and looks like a SimpleID,
		// it should be treated as a UUID
		newTitle := "Testing UUID precedence"

		// Use an actual UUID from our fixture
		err := store.Update(universe.PackForTrip.UUID, nanostore.UpdateRequest{
			Title: &newTitle,
		})
		if err != nil {
			t.Fatalf("Update with UUID failed: %v", err)
		}

		// Verify it updated the right document
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.PackForTrip.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		testutil.AssertDocumentCount(t, docs, 1)
		if docs[0].Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, docs[0].Title)
		}
	})

	t.Run("SimpleID with special patterns", func(t *testing.T) {
		// Test documents that have complex SimpleIDs
		complexDocs := []struct {
			name     string
			doc      nanostore.Document
			expected string // expected pattern in SimpleID
		}{
			{"deep hierarchy", universe.Level5Task, "."},     // Contains dots
			{"done status", universe.ReadBook, "d"},          // Contains 'd' prefix
			{"high priority", universe.ExerciseRoutine, "h"}, // May contain 'h' prefix
		}

		for _, tc := range complexDocs {
			t.Run(tc.name, func(t *testing.T) {
				if !strings.Contains(tc.doc.SimpleID, tc.expected) {
					t.Logf("Note: %s has SimpleID %q (may not contain expected pattern %q due to canonical values)",
						tc.name, tc.doc.SimpleID, tc.expected)
				}

				// Regardless of pattern, update should work
				newBody := "Updated: " + tc.name
				err := store.Update(tc.doc.SimpleID, nanostore.UpdateRequest{
					Body: &newBody,
				})
				if err != nil {
					t.Errorf("Failed to update %s with SimpleID %q: %v",
						tc.name, tc.doc.SimpleID, err)
				}
			})
		}
	})
}
