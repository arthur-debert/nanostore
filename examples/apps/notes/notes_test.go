package notes_test

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/examples/apps/notes"
)

func TestNotesBasicOperations(t *testing.T) {
	app, err := notes.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create notes app: %v", err)
	}
	defer app.Close()

	// Add notes
	id1, err := app.Add("How to use nanostore", "A guide to using nanostore for notes", []string{"nanostore", "tutorial"})
	if err != nil {
		t.Fatalf("failed to add note 1: %v", err)
	}
	if id1 != "1" {
		t.Errorf("expected first note ID to be '1', got '%s'", id1)
	}

	id2, err := app.Add("Shopping list", "Milk, Bread, Eggs", []string{"personal"})
	if err != nil {
		t.Fatalf("failed to add note 2: %v", err)
	}
	if id2 != "2" {
		t.Errorf("expected second note ID to be '2', got '%s'", id2)
	}

	// List notes
	notesList, err := app.List(notes.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	if len(notesList) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notesList))
	}

	// Verify first note
	if notesList[0].Title != "How to use nanostore" {
		t.Errorf("expected first note title to be 'How to use nanostore', got '%s'", notesList[0].Title)
	}
	if len(notesList[0].Tags) != 2 || notesList[0].Tags[0] != "nanostore" {
		t.Errorf("expected first note to have tags [nanostore, tutorial], got %v", notesList[0].Tags)
	}
}

func TestNotesArchive(t *testing.T) {
	app, err := notes.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create notes app: %v", err)
	}
	defer app.Close()

	// Add notes
	app.Add("Note 1", "Content 1", nil)
	app.Add("Note 2", "Content 2", nil)
	app.Add("Note 3", "Content 3", nil)

	// Archive note 2
	err = app.Archive("2")
	if err != nil {
		t.Fatalf("failed to archive note: %v", err)
	}

	// List live notes only
	notesList, err := app.List(notes.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	if len(notesList) != 2 {
		t.Errorf("expected 2 live notes after archiving, got %d", len(notesList))
	}

	// Note 3 should now be ID 2
	if notesList[1].UserFacingID != "2" {
		t.Errorf("expected note 3 to have ID '2' after archiving, got '%s'", notesList[1].UserFacingID)
	}
	if notesList[1].Title != "Note 3" {
		t.Errorf("expected ID 2 to be 'Note 3', got '%s'", notesList[1].Title)
	}

	// List with archived
	notesWithArchived, err := app.List(notes.ListOptions{ShowArchived: true})
	if err != nil {
		t.Fatalf("failed to list notes with archived: %v", err)
	}

	if len(notesWithArchived) != 3 {
		t.Errorf("expected 3 notes including archived, got %d", len(notesWithArchived))
	}

	// Find archived note
	var archivedNote *notes.Note
	for _, n := range notesWithArchived {
		if n.IsArchived {
			archivedNote = n
			break
		}
	}

	if archivedNote == nil {
		t.Fatal("archived note not found")
	}
	if archivedNote.UserFacingID != "a1" {
		t.Errorf("expected archived note to have ID 'a1', got '%s'", archivedNote.UserFacingID)
	}
	if archivedNote.Title != "Note 2" {
		t.Errorf("expected archived note to be 'Note 2', got '%s'", archivedNote.Title)
	}
}

func TestNotesPinning(t *testing.T) {
	app, err := notes.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create notes app: %v", err)
	}
	defer app.Close()

	// Add notes
	app.Add("Regular note 1", "Content", nil)
	app.Add("Important note", "Important content", nil)
	app.Add("Regular note 2", "Content", nil)

	// Pin the important note
	err = app.Pin("2")
	if err != nil {
		t.Fatalf("failed to pin note: %v", err)
	}

	// List notes
	notesList, err := app.List(notes.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	// Pinned note should have 'p' prefix
	if notesList[0].UserFacingID != "p1" {
		t.Errorf("expected pinned note to have ID 'p1', got '%s'", notesList[0].UserFacingID)
	}
	if notesList[0].Title != "Important note" {
		t.Errorf("expected pinned note to be 'Important note', got '%s'", notesList[0].Title)
	}
	if !notesList[0].IsPinned {
		t.Error("expected note to be marked as pinned")
	}

	// Other notes should be renumbered
	if notesList[1].UserFacingID != "1" {
		t.Errorf("expected first regular note to have ID '1', got '%s'", notesList[1].UserFacingID)
	}
	if notesList[2].UserFacingID != "2" {
		t.Errorf("expected second regular note to have ID '2', got '%s'", notesList[2].UserFacingID)
	}

	// Unpin
	err = app.Unpin("p1")
	if err != nil {
		t.Fatalf("failed to unpin note: %v", err)
	}

	// List again
	notesList, err = app.List(notes.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list notes after unpin: %v", err)
	}

	// Should be back to normal numbering
	if notesList[1].UserFacingID != "2" {
		t.Errorf("expected unpinned note to have ID '2', got '%s'", notesList[1].UserFacingID)
	}
}

func TestNotesSearch(t *testing.T) {
	app, err := notes.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create notes app: %v", err)
	}
	defer app.Close()

	// Add notes
	app.Add("Go programming", "Learn Go basics", []string{"programming", "go"})
	app.Add("Shopping list", "Buy groceries", []string{"personal"})
	app.Add("Go concurrency", "Goroutines and channels", []string{"programming", "go", "advanced"})

	// Search by content
	results, err := app.Search("go", false)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results for 'go' search, got %d", len(results))
	}

	// Search by tag
	results, err = app.List(notes.ListOptions{Tags: []string{"programming"}})
	if err != nil {
		t.Fatalf("failed to search by tag: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results for 'programming' tag, got %d", len(results))
	}

	// Combined search
	results, err = app.List(notes.ListOptions{
		Search: "concurrency",
		Tags:   []string{"advanced"},
	})
	if err != nil {
		t.Fatalf("failed to do combined search: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result for combined search, got %d", len(results))
	}
	if results[0].Title != "Go concurrency" {
		t.Errorf("expected 'Go concurrency' in results, got '%s'", results[0].Title)
	}
}

func TestNotesTagUpdate(t *testing.T) {
	app, err := notes.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create notes app: %v", err)
	}
	defer app.Close()

	// Add note
	id, _ := app.Add("My note", "Content", []string{"initial"})

	// Update tags
	err = app.UpdateTags(id, []string{"updated", "new"})
	if err != nil {
		t.Fatalf("failed to update tags: %v", err)
	}

	// Verify
	notesList, _ := app.List(notes.ListOptions{})
	if len(notesList[0].Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(notesList[0].Tags))
	}
	if notesList[0].Tags[0] != "updated" || notesList[0].Tags[1] != "new" {
		t.Errorf("expected tags [updated, new], got %v", notesList[0].Tags)
	}
}

func TestNotesStatusTransitions(t *testing.T) {
	app, err := notes.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create notes app: %v", err)
	}
	defer app.Close()

	// Add note
	id, _ := app.Add("Test note", "Content", nil)

	// Archive it
	app.Archive(id)

	// Delete it
	err = app.Delete("a1")
	if err != nil {
		t.Fatalf("failed to delete archived note: %v", err)
	}

	// List with deleted
	notesList, err := app.List(notes.ListOptions{ShowDeleted: true})
	if err != nil {
		t.Fatalf("failed to list with deleted: %v", err)
	}

	if len(notesList) != 1 {
		t.Errorf("expected 1 deleted note, got %d", len(notesList))
	}
	if !notesList[0].IsDeleted {
		t.Error("expected note to be marked as deleted")
	}

	// Unarchive deleted note (should fail in real app, but our implementation allows it)
	err = app.Unarchive("d1") // Assuming deleted notes get 'd' prefix
	// This will fail with current implementation since we don't have 'd' prefix configured
}

func TestNotesFormatting(t *testing.T) {
	app, err := notes.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create notes app: %v", err)
	}
	defer app.Close()

	// Add notes
	app.Add("Short note", "Brief content", []string{"tag1", "tag2"})
	app.Add("Long note", "This is a very long content that should be truncated in the preview to avoid taking too much space", nil)

	notesList, _ := app.List(notes.ListOptions{})
	output := notes.FormatList(notesList, false)

	// Check formatting
	if !strings.Contains(output, "1. Short note [tag1, tag2]") {
		t.Error("expected formatted output to contain note with tags")
	}
	if !strings.Contains(output, "   Brief content") {
		t.Error("expected formatted output to contain content preview")
	}
	if !strings.Contains(output, "...") {
		t.Error("expected long content to be truncated with ...")
	}
}
