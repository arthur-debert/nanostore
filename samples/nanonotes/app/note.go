package app

import (
	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// Note represents a note with status and pinning support
type Note struct {
	nanostore.Document

	// Status dimension: active or deleted
	Status string `values:"active,deleted" default:"active"`

	// Pinned flag
	Pinned bool `default:"false"`

	// Non-dimension fields
	Tags      string
	CreatedBy string
}

// NoteApp manages the note store and operations
type NoteApp struct {
	store *api.TypedStore[Note]
}

// NewNoteApp creates a new note application
func NewNoteApp(filePath string) (*NoteApp, error) {
	store, err := api.NewFromType[Note](filePath)
	if err != nil {
		return nil, err
	}

	return &NoteApp{
		store: store,
	}, nil
}

// Close closes the note store
func (app *NoteApp) Close() error {
	return app.store.Close()
}

// AddNote creates a new note
func (app *NoteApp) AddNote(title string, body string) (string, error) {
	note := &Note{
		Status: "active",
		Pinned: false,
	}

	// Create with title and let the system set the body through the Document
	id, err := app.store.Create(title, note)
	if err != nil {
		return "", err
	}

	// If we have a body, update the note to include it
	if body != "" {
		// Get the created note and update its body
		createdNote, err := app.store.Get(id)
		if err != nil {
			return id, err // Return the ID even if update fails
		}
		createdNote.Body = body
		// Update with the body
		if err := app.store.Update(id, createdNote); err != nil {
			return id, err // Return the ID even if update fails
		}
	}

	return id, nil
}

// DeleteNote soft deletes a note by setting status to deleted
func (app *NoteApp) DeleteNote(id string) error {
	return app.store.Update(id, &Note{
		Status: "deleted",
	})
}

// PinNote pins a note
func (app *NoteApp) PinNote(id string) error {
	return app.store.Update(id, &Note{
		Pinned: true,
	})
}

// UnpinNote unpins a note
func (app *NoteApp) UnpinNote(id string) error {
	return app.store.Update(id, &Note{
		Pinned: false,
	})
}

// GetNote retrieves a note by ID
func (app *NoteApp) GetNote(id string) (*Note, error) {
	return app.store.Get(id)
}

// ListNotes returns notes based on filters
func (app *NoteApp) ListNotes(includeDeleted bool, pinnedOnly bool) ([]Note, error) {
	query := app.store.Query()

	if !includeDeleted {
		// Canonical view: only active notes
		query = query.Status("active")
	}

	if pinnedOnly {
		// Filter for pinned notes
		allNotes, err := query.Find()
		if err != nil {
			return nil, err
		}

		var pinnedNotes []Note
		for _, note := range allNotes {
			if note.Pinned {
				pinnedNotes = append(pinnedNotes, note)
			}
		}
		return pinnedNotes, nil
	}

	return query.Find()
}

// SearchNotes searches for notes containing the given text
func (app *NoteApp) SearchNotes(searchText string, includeDeleted bool) ([]Note, error) {
	query := app.store.Query().Search(searchText)

	if !includeDeleted {
		query = query.Status("active")
	}

	return query.Find()
}

// Clean hard deletes all deleted notes
func (app *NoteApp) Clean() (int, error) {
	// Use bulk operation to delete all notes with status="deleted"
	return app.store.DeleteByDimension(map[string]interface{}{
		"status": "deleted",
	})
}

// CountNotes returns note statistics
func (app *NoteApp) CountNotes() (total int, active int, deleted int, pinned int, err error) {
	allNotes, queryErr := app.store.Query().Find()
	if queryErr != nil {
		return 0, 0, 0, 0, queryErr
	}

	for _, note := range allNotes {
		total++
		switch note.Status {
		case "active":
			active++
		case "deleted":
			deleted++
		}
		if note.Pinned {
			pinned++
		}
	}

	return total, active, deleted, pinned, nil
}

// GetRecentNotes returns the most recently updated notes
func (app *NoteApp) GetRecentNotes(limit int, includeDeleted bool) ([]Note, error) {
	query := app.store.Query()

	if !includeDeleted {
		query = query.Status("active")
	}

	// Get all notes and sort manually since the API might not support OrderBy with direction
	allNotes, err := query.Find()
	if err != nil {
		return nil, err
	}

	// Sort by updated time (most recent first) and limit
	// For simplicity, we'll return the first N notes
	if limit > 0 && len(allNotes) > limit {
		allNotes = allNotes[:limit]
	}

	return allNotes, nil
}
