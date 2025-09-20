package testutil

import (
	"fmt"
	"github.com/arthur-debert/nanostore/nanostore"
	"testing"
)

func TestFixtureVerification(t *testing.T) {
	store, universe := LoadUniverse(t)

	t.Log("=== FIXTURE VERIFICATION ===")
	t.Logf("Total documents loaded: %d\n", len(universe.ByUUID))

	// List all documents with their IDs
	docs, err := store.List(nanostore.ListOptions{
		OrderBy: []nanostore.OrderClause{
			{Column: "simple_id", Descending: false},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Document listing:")
	t.Log("ID | Title | Status | Priority | Parent")
	t.Log("---|-------|---------|----------|-------")

	for _, doc := range docs {
		parentID := ""
		if pid, ok := doc.Dimensions["parent_id"]; ok {
			parentID = fmt.Sprint(pid)
		}
		t.Logf("%-10s | %-30s | %-8s | %-8s | %s",
			doc.SimpleID,
			doc.Title,
			doc.Dimensions["status"],
			doc.Dimensions["priority"],
			parentID,
		)
	}

	// Verify hierarchical structure
	t.Log("\n=== HIERARCHICAL STRUCTURE ===")
	t.Logf("Personal Root (%s) has %d children",
		universe.PersonalRoot.SimpleID,
		len(universe.GetChildrenOf("root-1")))
	t.Logf("Work Root (%s) has %d children",
		universe.WorkRoot.SimpleID,
		len(universe.GetChildrenOf("root-2")))

	// Check some specific IDs
	t.Log("\n=== ID GENERATION CHECK ===")
	t.Logf("Personal root: %s (expect: 1)", universe.PersonalRoot.SimpleID)
	t.Logf("Work root: %s (expect: 2)", universe.WorkRoot.SimpleID)
	t.Logf("High priority exercise: %s (expect: h1.2)", universe.ExerciseRoutine.SimpleID)
	t.Logf("Done read book: %s (expect: d1.3)", universe.ReadBook.SimpleID)
	t.Logf("Buy groceries: %s (expect: 1.1)", universe.BuyGroceries.SimpleID)
	t.Logf("High priority team meeting: %s (expect: h2.1)", universe.TeamMeeting.SimpleID)

	// Verify the deep nesting
	t.Log("\n=== DEEP NESTING CHECK ===")
	t.Logf("Level 3: %s", universe.Level3Task.SimpleID)
	t.Logf("Level 4: %s", universe.Level4Task.SimpleID)
	t.Logf("Level 5: %s", universe.Level5Task.SimpleID)

	// Verify mixed parent
	t.Log("\n=== MIXED PARENT CHECK ===")
	mixedChildren := universe.GetChildrenOf("mixed-parent")
	t.Logf("Mixed parent has %d children", len(mixedChildren))
	for _, child := range mixedChildren {
		t.Logf("  Child: %s - Status: %s", child.SimpleID, child.Dimensions["status"])
	}
}
