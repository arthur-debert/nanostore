package main

import (
	"os"
	"path/filepath"
	"samples-nanonotes/app"
	"testing"
)

func TestNoteOperations(t *testing.T) {
	// Create temporary data file
	tempDir := t.TempDir()
	dataFile := filepath.Join(tempDir, "test-notes.json")

	noteApp, err := app.NewNoteApp(dataFile)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer noteApp.Close()

	t.Run("add note", func(t *testing.T) {
		id, err := noteApp.AddNote("Test Note", "This is a test note body")
		if err != nil {
			t.Fatalf("failed to add note: %v", err)
		}

		// Verify note was created
		note, err := noteApp.GetNote(id)
		if err != nil {
			t.Fatalf("failed to get note: %v", err)
		}

		if note.Title != "Test Note" {
			t.Errorf("note title = %q, want %q", note.Title, "Test Note")
		}
		if note.Body != "This is a test note body" {
			t.Errorf("note body = %q, want %q", note.Body, "This is a test note body")
		}
		if note.Status != "active" {
			t.Errorf("note status = %q, want %q", note.Status, "active")
		}
		if note.Pinned {
			t.Error("new note should not be pinned by default")
		}
	})

	t.Run("delete note", func(t *testing.T) {
		id, err := noteApp.AddNote("Note to Delete", "")
		if err != nil {
			t.Fatalf("failed to add note: %v", err)
		}

		err = noteApp.DeleteNote(id)
		if err != nil {
			t.Fatalf("failed to delete note: %v", err)
		}

		// Verify note is marked as deleted
		note, err := noteApp.GetNote(id)
		if err != nil {
			t.Fatalf("failed to get deleted note: %v", err)
		}

		if note.Status != "deleted" {
			t.Errorf("note status = %q, want %q", note.Status, "deleted")
		}
	})

	t.Run("pin and unpin note", func(t *testing.T) {
		id, err := noteApp.AddNote("Note to Pin", "")
		if err != nil {
			t.Fatalf("failed to add note: %v", err)
		}

		// Pin the note
		err = noteApp.PinNote(id)
		if err != nil {
			t.Fatalf("failed to pin note: %v", err)
		}

		note, err := noteApp.GetNote(id)
		if err != nil {
			t.Fatalf("failed to get pinned note: %v", err)
		}

		if !note.Pinned {
			t.Error("note should be pinned")
		}

		// Unpin the note
		err = noteApp.UnpinNote(id)
		if err != nil {
			t.Fatalf("failed to unpin note: %v", err)
		}

		note, err = noteApp.GetNote(id)
		if err != nil {
			t.Fatalf("failed to get unpinned note: %v", err)
		}

		if note.Pinned {
			t.Error("note should not be pinned")
		}
	})
}

func TestListNotes(t *testing.T) {
	tempDir := t.TempDir()
	dataFile := filepath.Join(tempDir, "test-notes.json")

	noteApp, err := app.NewNoteApp(dataFile)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer noteApp.Close()

	// Add test notes
	_, _ = noteApp.AddNote("Active Note 1", "Body 1")
	_, _ = noteApp.AddNote("Active Note 2", "Body 2")
	deletedNote, _ := noteApp.AddNote("Deleted Note", "Body 3")
	pinnedNote, _ := noteApp.AddNote("Pinned Note", "Body 4")

	noteApp.DeleteNote(deletedNote)
	noteApp.PinNote(pinnedNote)

	t.Run("list active notes only", func(t *testing.T) {
		notes, err := noteApp.ListNotes(false, false)
		if err != nil {
			t.Fatalf("failed to list notes: %v", err)
		}

		if len(notes) != 3 {
			t.Errorf("got %d notes, want 3", len(notes))
		}

		// Verify deleted note is not included
		for _, note := range notes {
			if note.UUID == deletedNote {
				t.Error("deleted note should not be in active list")
			}
		}
	})

	t.Run("list all notes including deleted", func(t *testing.T) {
		notes, err := noteApp.ListNotes(true, false)
		if err != nil {
			t.Fatalf("failed to list all notes: %v", err)
		}

		if len(notes) != 4 {
			t.Errorf("got %d notes, want 4", len(notes))
		}
	})

	t.Run("list pinned notes only", func(t *testing.T) {
		notes, err := noteApp.ListNotes(false, true)
		if err != nil {
			t.Fatalf("failed to list pinned notes: %v", err)
		}

		if len(notes) != 1 {
			t.Errorf("got %d notes, want 1", len(notes))
		}

		if len(notes) > 0 && notes[0].UUID != pinnedNote {
			t.Errorf("got note %s, want %s", notes[0].UUID, pinnedNote)
		}
	})
}

func TestSearchNotes(t *testing.T) {
	tempDir := t.TempDir()
	dataFile := filepath.Join(tempDir, "test-notes.json")

	noteApp, err := app.NewNoteApp(dataFile)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer noteApp.Close()

	// Add test notes
	noteApp.AddNote("Meeting with John", "Discuss Q1 goals")
	noteApp.AddNote("Shopping list", "Buy groceries")
	deletedID, _ := noteApp.AddNote("Meeting notes", "Old meeting")
	noteApp.DeleteNote(deletedID)

	t.Run("search active notes", func(t *testing.T) {
		notes, err := noteApp.SearchNotes("meeting", false)
		if err != nil {
			t.Fatalf("failed to search notes: %v", err)
		}

		if len(notes) != 1 {
			t.Errorf("got %d notes, want 1", len(notes))
		}

		if len(notes) > 0 && notes[0].Title != "Meeting with John" {
			t.Errorf("got note %q, want %q", notes[0].Title, "Meeting with John")
		}
	})

	t.Run("search all notes including deleted", func(t *testing.T) {
		notes, err := noteApp.SearchNotes("meeting", true)
		if err != nil {
			t.Fatalf("failed to search all notes: %v", err)
		}

		if len(notes) != 2 {
			t.Errorf("got %d notes, want 2", len(notes))
		}
	})
}

func TestClean(t *testing.T) {
	tempDir := t.TempDir()
	dataFile := filepath.Join(tempDir, "test-notes.json")

	noteApp, err := app.NewNoteApp(dataFile)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer noteApp.Close()

	// Add and delete some notes
	id1, _ := noteApp.AddNote("Note 1", "")
	id2, _ := noteApp.AddNote("Note 2", "")
	id3, _ := noteApp.AddNote("Note 3", "")

	noteApp.DeleteNote(id1)
	noteApp.DeleteNote(id2)

	// Clean deleted notes
	count, err := noteApp.Clean()
	if err != nil {
		t.Fatalf("failed to clean notes: %v", err)
	}

	if count != 2 {
		t.Errorf("cleaned %d notes, want 2", count)
	}

	// Verify only active note remains
	notes, err := noteApp.ListNotes(true, false)
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	if len(notes) != 1 {
		t.Errorf("got %d notes, want 1", len(notes))
	}

	if notes[0].UUID != id3 {
		t.Errorf("wrong note remained: got %s, want %s", notes[0].UUID, id3)
	}
}

func TestCountNotes(t *testing.T) {
	tempDir := t.TempDir()
	dataFile := filepath.Join(tempDir, "test-notes.json")

	noteApp, err := app.NewNoteApp(dataFile)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer noteApp.Close()

	// Add various notes
	noteApp.AddNote("Active 1", "")
	noteApp.AddNote("Active 2", "")
	deletedID, _ := noteApp.AddNote("Deleted", "")
	pinnedID, _ := noteApp.AddNote("Pinned", "")

	noteApp.DeleteNote(deletedID)
	noteApp.PinNote(pinnedID)

	total, active, deleted, pinned, err := noteApp.CountNotes()
	if err != nil {
		t.Fatalf("failed to count notes: %v", err)
	}

	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if active != 3 {
		t.Errorf("active = %d, want 3", active)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}
	if pinned != 1 {
		t.Errorf("pinned = %d, want 1", pinned)
	}
}

func TestFileCreation(t *testing.T) {
	tempDir := t.TempDir()
	dataFile := filepath.Join(tempDir, "new-notes.json")

	// Ensure file doesn't exist
	if _, err := os.Stat(dataFile); !os.IsNotExist(err) {
		t.Fatal("data file should not exist before app creation")
	}

	noteApp, err := app.NewNoteApp(dataFile)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer noteApp.Close()

	// Add a note to trigger file creation
	_, err = noteApp.AddNote("First Note", "")
	if err != nil {
		t.Fatalf("failed to add note: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		t.Error("data file should exist after adding note")
	}
}
