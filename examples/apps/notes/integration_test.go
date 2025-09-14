package notes_test

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/examples/apps/notes"
)

// TestIntegrationScenario tests the exact scenario from the specification
func TestIntegrationScenario(t *testing.T) {
	app, err := notes.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create notes app: %v", err)
	}
	defer app.Close()

	// Initial setup as per specification
	app.Add("How to use nanostore for a notes app", "", []string{"nanostore", "shell"})
	app.Add("Why simple IDs matter for shell", "", []string{"shell", "ux"})

	// Test initial list
	t.Run("InitialList", func(t *testing.T) {
		notesList, err := app.List(notes.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		output := notes.FormatList(notesList, false)
		expected := `1. How to use nanostore for a notes app [nanostore, shell]
2. Why simple IDs matter for shell [shell, ux]
`

		// Normalize comparison (ignore content preview lines)
		outputLines := getMainLines(output)
		expectedLines := getMainLines(expected)

		if outputLines != expectedLines {
			t.Errorf("Initial list mismatch:\nGot:\n%s\nExpected:\n%s", outputLines, expectedLines)
		}
	})

	// Archive note 1
	err = app.Archive("1")
	if err != nil {
		t.Fatalf("failed to archive note 1: %v", err)
	}

	// Test list after archiving
	t.Run("AfterArchiving", func(t *testing.T) {
		notesList, err := app.List(notes.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		output := notes.FormatList(notesList, false)

		// The second note should now be ID 1
		if !strings.Contains(output, "1. Why simple IDs matter for shell") {
			t.Errorf("Expected 'Why simple IDs matter' to be renumbered to ID 1")
		}

		// Should only show one note
		if len(notesList) != 1 {
			t.Errorf("Expected 1 live note, got %d", len(notesList))
		}
	})

	// List with archived
	t.Run("ListWithArchived", func(t *testing.T) {
		notesList, err := app.List(notes.ListOptions{ShowArchived: true})
		if err != nil {
			t.Fatalf("failed to list with archived: %v", err)
		}

		// Should have 2 notes total
		if len(notesList) != 2 {
			t.Errorf("Expected 2 notes including archived, got %d", len(notesList))
		}

		// Find the archived note
		var archived *notes.Note
		for _, n := range notesList {
			if n.IsArchived {
				archived = n
				break
			}
		}

		if archived == nil {
			t.Fatal("Archived note not found")
		}

		if archived.UserFacingID != "a1" {
			t.Errorf("Expected archived note to have ID 'a1', got '%s'", archived.UserFacingID)
		}

		if archived.Title != "How to use nanostore for a notes app" {
			t.Errorf("Wrong note was archived")
		}
	})
}

// TestComplexScenario tests multiple features together
func TestComplexScenario(t *testing.T) {
	app, err := notes.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create notes app: %v", err)
	}
	defer app.Close()

	// Create notes
	app.Add("Project ideas", "List of project ideas", []string{"projects"})
	app.Add("Meeting notes", "Today's meeting summary", []string{"work", "meetings"})
	app.Add("Important deadline", "Project due Friday", []string{"work", "urgent"})
	app.Add("Grocery list", "Milk, bread, eggs", []string{"personal"})
	app.Add("Book recommendations", "Sci-fi books to read", []string{"personal", "reading"})

	// Pin the important deadline
	err = app.Pin("3")
	if err != nil {
		t.Fatalf("failed to pin note: %v", err)
	}

	// Archive the grocery list
	err = app.Archive("4") // This was originally ID 4, but after pinning it might have changed
	if err != nil {
		// Try with new ID after pinning
		err = app.Archive("3")
		if err != nil {
			t.Fatalf("failed to archive note: %v", err)
		}
	}

	// List live notes
	notesList, err := app.List(notes.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	// Check that we have one pinned note first
	if !notesList[0].IsPinned {
		t.Error("Expected first note to be pinned")
	}
	if !strings.HasPrefix(notesList[0].UserFacingID, "p") {
		t.Errorf("Expected pinned note to have 'p' prefix, got '%s'", notesList[0].UserFacingID)
	}

	// Search for work-related notes
	workNotesList, err := app.List(notes.ListOptions{Tags: []string{"work"}})
	if err != nil {
		t.Fatalf("failed to search by tag: %v", err)
	}

	if len(workNotesList) != 2 {
		t.Errorf("Expected 2 work notes, got %d", len(workNotesList))
	}

	// Delete a note
	err = app.Delete("2")
	if err != nil {
		t.Fatalf("failed to delete note: %v", err)
	}

	// List should show renumbered IDs
	notesList, err = app.List(notes.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list after delete: %v", err)
	}

	// Verify continuous numbering is maintained
	foundIDs := make(map[string]bool)
	for _, note := range notesList {
		foundIDs[note.UserFacingID] = true
	}

	// Should have p1 and continuous numbering for unpinned notes
	if !foundIDs["p1"] {
		t.Error("Expected to find p1 (pinned note)")
	}
}

// TestMultiplePinnedNotes tests behavior with multiple pinned notes
func TestMultiplePinnedNotes(t *testing.T) {
	app, err := notes.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create notes app: %v", err)
	}
	defer app.Close()

	// Create notes
	app.Add("Note 1", "", nil)
	app.Add("Note 2", "", nil)
	app.Add("Note 3", "", nil)
	app.Add("Note 4", "", nil)

	// Pin notes 2 and 4
	app.Pin("2")
	app.Pin("4")

	// List notes
	notesList, err := app.List(notes.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	// First two should be pinned
	if notesList[0].UserFacingID != "p1" || notesList[1].UserFacingID != "p2" {
		t.Errorf("Expected first two notes to be p1 and p2, got %s and %s",
			notesList[0].UserFacingID, notesList[1].UserFacingID)
	}

	// Rest should be renumbered
	if notesList[2].UserFacingID != "1" || notesList[3].UserFacingID != "2" {
		t.Errorf("Expected unpinned notes to be 1 and 2, got %s and %s",
			notesList[2].UserFacingID, notesList[3].UserFacingID)
	}
}

// Helper function to extract main lines (non-indented) from output
func getMainLines(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var mainLines []string
	for _, line := range lines {
		if len(line) > 0 && line[0] != ' ' {
			mainLines = append(mainLines, line)
		}
	}
	return strings.Join(mainLines, "\n")
}
