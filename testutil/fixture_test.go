package testutil

import (
	"testing"
)

func TestLoadUniverse(t *testing.T) {
	// Load the fixture
	store, universe := LoadUniverse(t)

	// Verify store is not nil
	if store == nil {
		t.Fatal("store should not be nil")
	}

	// Verify universe data is populated
	if universe == nil {
		t.Fatal("universe should not be nil")
	}

	// Check that we have documents
	if len(universe.ByUUID) == 0 {
		t.Fatal("universe should contain documents")
	}

	// Verify specific documents exist
	if universe.PersonalRoot.Title != "Personal Tasks" {
		t.Errorf("PersonalRoot title incorrect: got %q", universe.PersonalRoot.Title)
	}

	if universe.WorkRoot.Title != "Work Projects" {
		t.Errorf("WorkRoot title incorrect: got %q", universe.WorkRoot.Title)
	}

	// Check document counts
	activeCount := len(universe.GetActiveDocuments())
	if activeCount != 25 { // Based on our fixture data
		t.Errorf("expected 25 active documents, got %d", activeCount)
	}

	// Check root documents
	roots := universe.GetRootDocuments()
	if len(roots) != 10 { // Based on our fixture data
		t.Errorf("expected 10 root documents, got %d", len(roots))
	}

	// Check status filtering
	pendingDocs := universe.GetDocumentsByStatus("pending")
	if len(pendingDocs) != 8 { // Based on our fixture data
		t.Errorf("expected 8 pending documents, got %d", len(pendingDocs))
	}

	// Verify hierarchical relationships
	personalChildren := universe.GetChildrenOf("root-1")
	if len(personalChildren) != 3 {
		t.Errorf("expected 3 children for PersonalRoot, got %d", len(personalChildren))
	}

	workChildren := universe.GetChildrenOf("root-2")
	if len(workChildren) != 3 {
		t.Errorf("expected 3 children for WorkRoot, got %d", len(workChildren))
	}

	// Verify specific ID patterns
	if universe.ReadBook.SimpleID == "" {
		t.Error("ReadBook should have a SimpleID")
	}

	if universe.Level5Task.SimpleID == "" {
		t.Error("Level5Task should have a SimpleID")
	}

	t.Logf("Loaded %d documents successfully", len(universe.ByUUID))
	t.Logf("Active documents: %d", activeCount)
	t.Logf("Root documents: %d", len(roots))
	t.Logf("Pending documents: %d", len(pendingDocs))
	t.Logf("Sample IDs - Read book: %s, Level 5: %s", universe.ReadBook.SimpleID, universe.Level5Task.SimpleID)
}
