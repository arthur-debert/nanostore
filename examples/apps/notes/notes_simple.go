// Package notes implements a simplified note-taking application using nanostore.
// This demonstrates how the generic nanostore library can be easily leveraged
// for specific applications. While nanostore is domain-agnostic, it provides
// all the building blocks needed for a note-taking app with archiving features.
package notes

import (
	"fmt"
	"strings"

	"github.com/arthur-debert/nanostore/nanostore"
)

// SimpleNotes wraps nanostore to provide note-specific functionality
type SimpleNotes struct {
	store nanostore.Store
}

// NewSimple creates a new SimpleNotes instance using default config
// We use the TodoConfig here which provides "pending"/"completed" status values
// that map well to active/archived notes
func NewSimple(dbPath string) (*SimpleNotes, error) {
	store, err := nanostore.New(dbPath, nanostore.TodoConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	return &SimpleNotes{store: store}, nil
}

// Close releases resources
func (n *SimpleNotes) Close() error {
	return n.store.Close()
}

// Add creates a new note (pending status)
func (n *SimpleNotes) Add(title, content string, tags []string) (string, error) {
	// Store tags in body
	body := content
	if len(tags) > 0 {
		body = content + "\n\n#tags: " + strings.Join(tags, ", ")
	}

	uuid, err := n.store.Add(title, nil)
	if err != nil {
		return "", fmt.Errorf("failed to add note: %w", err)
	}

	// Update with body
	if body != "" {
		err = n.store.Update(uuid, nanostore.UpdateRequest{
			Body: &body,
		})
		if err != nil {
			return "", fmt.Errorf("failed to update body: %w", err)
		}
	}

	// Get user-facing ID
	docs, _ := n.store.List(nanostore.ListOptions{})
	for _, doc := range docs {
		if doc.UUID == uuid {
			return doc.UserFacingID, nil
		}
	}

	return "", fmt.Errorf("could not find user-facing ID")
}

// Archive moves a note to completed status (gets 'c' prefix)
// This leverages nanostore's dimension system - archived notes get a different
// ID prefix making them visually distinct
func (n *SimpleNotes) Archive(userFacingID string) error {
	uuid, err := n.store.ResolveUUID(userFacingID)
	if err != nil {
		return fmt.Errorf("failed to resolve ID: %w", err)
	}

	return nanostore.SetStatus(n.store, uuid, "completed")
}

// Unarchive moves a note back to pending status
func (n *SimpleNotes) Unarchive(userFacingID string) error {
	uuid, err := n.store.ResolveUUID(userFacingID)
	if err != nil {
		return fmt.Errorf("failed to resolve ID: %w", err)
	}

	return nanostore.SetStatus(n.store, uuid, "pending")
}

// Delete removes a note
func (n *SimpleNotes) Delete(userFacingID string) error {
	uuid, err := n.store.ResolveUUID(userFacingID)
	if err != nil {
		return fmt.Errorf("failed to resolve ID: %w", err)
	}

	return n.store.Delete(uuid, false)
}

// SimpleNote represents a note
type SimpleNote struct {
	nanostore.Document
	Tags       []string
	IsArchived bool
}

// List returns notes
// This demonstrates filtering using nanostore's generic Filters map
func (n *SimpleNotes) List(showArchived bool) ([]*SimpleNote, error) {
	opts := nanostore.ListOptions{}
	if !showArchived {
		// Filter to show only active (pending) notes
		opts.Filters = map[string]interface{}{"status": "pending"}
	}

	docs, err := n.store.List(opts)
	if err != nil {
		return nil, err
	}

	var notes []*SimpleNote
	for _, doc := range docs {
		note := &SimpleNote{
			Document:   doc,
			IsArchived: doc.GetStatus() == "completed",
		}

		// Parse tags
		if doc.Body != "" {
			if tagIdx := strings.Index(doc.Body, "\n\n#tags: "); tagIdx != -1 {
				tagLine := doc.Body[tagIdx+9:]
				if endIdx := strings.Index(tagLine, "\n"); endIdx != -1 {
					tagLine = tagLine[:endIdx]
				}
				note.Tags = strings.Split(tagLine, ", ")
			}
		}

		notes = append(notes, note)
	}

	return notes, nil
}

// FormatSimpleList formats notes for display
func FormatSimpleList(notes []*SimpleNote) string {
	var sb strings.Builder

	for _, note := range notes {
		tags := ""
		if len(note.Tags) > 0 {
			tags = fmt.Sprintf(" [%s]", strings.Join(note.Tags, ", "))
		}

		sb.WriteString(fmt.Sprintf("%s. %s%s\n",
			note.UserFacingID, note.Title, tags))
	}

	return sb.String()
}
