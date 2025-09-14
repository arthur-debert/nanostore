// Package notes implements a note-taking application using nanostore.
// It demonstrates status-based ID prefixes, tagging, and pinning.
package notes

import (
	"fmt"
	"strings"

	"github.com/arthur-debert/nanostore/nanostore"
)

// NoteStatus represents the lifecycle state of a note
type NoteStatus string

const (
	StatusLive     NoteStatus = "live"
	StatusArchived NoteStatus = "archived"
	StatusDeleted  NoteStatus = "deleted"
)

// Notes wraps nanostore to provide note-specific functionality
type Notes struct {
	store nanostore.Store
}

// NotesConfig creates a nanostore configuration for the notes app
func NotesConfig() nanostore.Config {
	// Use default config and map our statuses to pending/completed
	return nanostore.DefaultTestConfig()
}

// New creates a new Notes instance
func New(dbPath string) (*Notes, error) {
	store, err := nanostore.New(dbPath, NotesConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	return &Notes{store: store}, nil
}

// Close releases resources
func (n *Notes) Close() error {
	return n.store.Close()
}

// Add creates a new note with optional tags
func (n *Notes) Add(title, content string, tags []string) (string, error) {
	// Map live status to pending
	dimensions := map[string]string{
		"status": "pending",
	}

	// Store body with tags appended
	body := content
	if len(tags) > 0 {
		body = content + "\n\n#tags: " + strings.Join(tags, ", ")
	}

	uuid, err := n.store.Add(title, nil, dimensions)
	if err != nil {
		return "", fmt.Errorf("failed to add note: %w", err)
	}

	// Update with body content
	if body != "" {
		err = n.store.Update(uuid, nanostore.UpdateRequest{
			Body: &body,
		})
		if err != nil {
			return "", fmt.Errorf("failed to update note body: %w", err)
		}
	}

	// Get the user-facing ID by listing
	docs, err := n.store.List(nanostore.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get documents: %w", err)
	}

	for _, doc := range docs {
		if doc.UUID == uuid {
			return doc.UserFacingID, nil
		}
	}

	return "", fmt.Errorf("could not find user-facing ID for new note")
}

// Archive moves a note to archived status
func (n *Notes) Archive(userFacingID string) error {
	uuid, err := n.store.ResolveUUID(userFacingID)
	if err != nil {
		return fmt.Errorf("failed to resolve ID '%s': %w", userFacingID, err)
	}

	// Map archived to completed with 'a' prefix
	updates := nanostore.UpdateRequest{
		Dimensions: map[string]string{
			"status": "completed",
		},
	}

	err = n.store.Update(uuid, updates)
	if err != nil {
		return fmt.Errorf("failed to archive note: %w", err)
	}

	return nil
}

// Unarchive moves a note back to live status
func (n *Notes) Unarchive(userFacingID string) error {
	uuid, err := n.store.ResolveUUID(userFacingID)
	if err != nil {
		return fmt.Errorf("failed to resolve ID '%s': %w", userFacingID, err)
	}

	// Map back to pending
	updates := nanostore.UpdateRequest{
		Dimensions: map[string]string{
			"status": "pending",
		},
	}

	err = n.store.Update(uuid, updates)
	if err != nil {
		return fmt.Errorf("failed to unarchive note: %w", err)
	}

	return nil
}

// Delete soft-deletes a note
func (n *Notes) Delete(userFacingID string) error {
	uuid, err := n.store.ResolveUUID(userFacingID)
	if err != nil {
		return fmt.Errorf("failed to resolve ID '%s': %w", userFacingID, err)
	}

	// For delete, we'll actually delete from the store
	err = n.store.Delete(uuid, false)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	return nil
}

// ListOptions configures how notes are listed
type ListOptions struct {
	ShowArchived bool
	ShowDeleted  bool
	Search       string
	Tags         []string
}

// Note represents a note with display information
type Note struct {
	nanostore.Document
	Tags       []string
	IsPinned   bool
	IsArchived bool
	IsDeleted  bool
}

// List returns notes based on the provided options
func (n *Notes) List(opts ListOptions) ([]*Note, error) {
	// Build status filter
	statuses := []nanostore.Status{nanostore.Status(StatusLive)}
	if opts.ShowArchived {
		statuses = append(statuses, nanostore.Status(StatusArchived))
	}
	if opts.ShowDeleted {
		statuses = append(statuses, nanostore.Status(StatusDeleted))
	}

	listOpts := nanostore.ListOptions{
		FilterByStatus: statuses,
		FilterBySearch: opts.Search,
	}

	docs, err := n.store.List(listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}

	// Convert to Notes and apply tag filter if needed
	var notes []*Note
	for _, doc := range docs {
		note := &Note{
			Document: doc,
			// Infer status from user-facing ID prefix
			IsPinned:   strings.HasPrefix(doc.UserFacingID, "p"),
			IsArchived: strings.HasPrefix(doc.UserFacingID, "a"),
			IsDeleted:  doc.Status == nanostore.Status(StatusDeleted),
		}

		// Parse tags from body if present
		if doc.Body != "" {
			if tagIdx := strings.Index(doc.Body, "\n\n#tags: "); tagIdx != -1 {
				tagLine := doc.Body[tagIdx+9:]
				if endIdx := strings.Index(tagLine, "\n"); endIdx != -1 {
					tagLine = tagLine[:endIdx]
				}
				tags := strings.Split(tagLine, ", ")
				for i := range tags {
					tags[i] = strings.TrimSpace(tags[i])
				}
				note.Tags = tags
			}
		}

		// Apply tag filter
		if len(opts.Tags) > 0 {
			if !hasAnyTag(note.Tags, opts.Tags) {
				continue
			}
		}

		notes = append(notes, note)
	}

	return notes, nil
}

// Search is a convenience method for searching notes
func (n *Notes) Search(query string, showArchived bool) ([]*Note, error) {
	return n.List(ListOptions{
		ShowArchived: showArchived,
		Search:       query,
	})
}

// UpdateTags updates the tags for a note
func (n *Notes) UpdateTags(userFacingID string, tags []string) error {
	uuid, err := n.store.ResolveUUID(userFacingID)
	if err != nil {
		return fmt.Errorf("failed to resolve ID '%s': %w", userFacingID, err)
	}

	// Get current note to preserve content
	docs, err := n.store.List(nanostore.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get documents: %w", err)
	}

	var currentBody string
	for _, doc := range docs {
		if doc.UUID == uuid {
			currentBody = doc.Body
			break
		}
	}

	// Remove old tags if present
	if tagIdx := strings.Index(currentBody, "\n\n#tags: "); tagIdx != -1 {
		currentBody = currentBody[:tagIdx]
	}

	// Add new tags
	newBody := currentBody
	if len(tags) > 0 {
		newBody = currentBody + "\n\n#tags: " + strings.Join(tags, ", ")
	}

	updates := nanostore.UpdateRequest{
		Body: &newBody,
	}

	err = n.store.Update(uuid, updates)
	if err != nil {
		return fmt.Errorf("failed to update tags: %w", err)
	}

	return nil
}

// Helper functions

func hasAnyTag(noteTags, searchTags []string) bool {
	for _, searchTag := range searchTags {
		for _, noteTag := range noteTags {
			if strings.EqualFold(noteTag, searchTag) {
				return true
			}
		}
	}
	return false
}

// FormatList formats notes for display
func FormatList(notes []*Note, showStatus bool) string {
	var sb strings.Builder

	for _, note := range notes {
		// Status indicator
		status := ""
		if showStatus {
			if note.IsDeleted {
				status = " [DELETED]"
			} else if note.IsArchived {
				status = " [ARCHIVED]"
			}
		}

		// Format tags
		tags := ""
		if len(note.Tags) > 0 {
			tags = fmt.Sprintf(" [%s]", strings.Join(note.Tags, ", "))
		}

		sb.WriteString(fmt.Sprintf("%s. %s%s%s\n",
			note.UserFacingID, note.Title, tags, status))

		// Show content preview if available (excluding tags)
		if note.Body != "" {
			preview := note.Body
			// Remove tags from preview
			if tagIdx := strings.Index(preview, "\n\n#tags: "); tagIdx != -1 {
				preview = preview[:tagIdx]
			}
			if len(preview) > 60 {
				preview = preview[:57] + "..."
			}
			if preview != "" {
				sb.WriteString(fmt.Sprintf("   %s\n", preview))
			}
		}
	}

	return sb.String()
}
