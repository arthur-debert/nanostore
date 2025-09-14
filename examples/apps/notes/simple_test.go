package notes_test

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/examples/apps/notes"
)

func TestSimpleNotesScenario(t *testing.T) {
	app, err := notes.NewSimple(":memory:")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// Add notes as per spec
	app.Add("How to use nanostore for a notes app", "", []string{"nanostore", "shell"})
	app.Add("Why simple IDs matter for shell", "", []string{"shell", "ux"})

	// List initial
	notesList, err := app.List(false)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	output := notes.FormatSimpleList(notesList)
	expected := `1. How to use nanostore for a notes app [nanostore, shell]
2. Why simple IDs matter for shell [shell, ux]
`

	if output != expected {
		t.Errorf("Initial list mismatch:\nGot:\n%s\nExpected:\n%s", output, expected)
	}

	// Archive note 1
	err = app.Archive("1")
	if err != nil {
		t.Fatalf("failed to archive: %v", err)
	}

	// List again - should only show one
	notesList, err = app.List(false)
	if err != nil {
		t.Fatalf("failed to list after archive: %v", err)
	}

	output = notes.FormatSimpleList(notesList)
	if !strings.Contains(output, "1. Why simple IDs matter") {
		t.Errorf("Expected remaining note to be renumbered to 1, got:\n%s", output)
	}

	// List with archived
	notesList, err = app.List(true)
	if err != nil {
		t.Fatalf("failed to list all: %v", err)
	}

	if len(notesList) != 2 {
		t.Errorf("Expected 2 notes total, got %d", len(notesList))
	}

	// Find archived note
	var archived *notes.SimpleNote
	for _, n := range notesList {
		if n.IsArchived {
			archived = n
			break
		}
	}

	if archived == nil {
		t.Fatal("Archived note not found")
	}

	if !strings.HasPrefix(archived.UserFacingID, "c") {
		t.Errorf("Expected archived note to have 'c' prefix, got '%s'", archived.UserFacingID)
	}
}
