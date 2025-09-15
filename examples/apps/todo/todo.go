// Package todo implements a hierarchical todo list application using nanostore.
// It demonstrates how nanostore's dynamic ID generation works in practice.
package todo

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arthur-debert/nanostore/nanostore"
)

// Todo wraps nanostore to provide todo-specific functionality
type Todo struct {
	store nanostore.Store
}

// todoConfig returns a configuration suitable for todo applications
func todoConfig() nanostore.Config {
	return nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "completed"},
				Prefixes:     map[string]string{"completed": "c"},
				DefaultValue: "pending",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}
}

// New creates a new Todo instance
func New(dbPath string) (*Todo, error) {
	// Use the todo-specific configuration
	store, err := nanostore.New(dbPath, todoConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	return &Todo{store: store}, nil
}

// SetStatus is a todo-specific helper function to set the status dimension of a document
// This is equivalent to: store.Update(id, UpdateRequest{Dimensions: {"status": status}})
func SetStatus(store nanostore.Store, id string, status string) error {
	return store.Update(id, nanostore.UpdateRequest{
		Dimensions: map[string]string{"status": status},
	})
}

// Close releases resources
func (t *Todo) Close() error {
	return t.store.Close()
}

// Add creates a new todo item
func (t *Todo) Add(title string, parentID *string) (string, error) {
	// If parentID is a user-facing ID, resolve it first
	if parentID != nil && *parentID != "" {
		uuid, err := t.store.ResolveUUID(*parentID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve parent ID '%s': %w", *parentID, err)
		}
		parentID = &uuid
	}

	dimensions := make(map[string]interface{})
	if parentID != nil {
		dimensions["parent_uuid"] = *parentID
	}
	uuid, err := t.store.Add(title, dimensions)
	if err != nil {
		return "", fmt.Errorf("failed to add todo: %w", err)
	}

	// Get all documents to find the user-facing ID
	docs, err := t.store.List(nanostore.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get user-facing ID: %w", err)
	}

	// Find the document we just added and return its user-facing ID
	for _, doc := range docs {
		if doc.UUID == uuid {
			return doc.UserFacingID, nil
		}
	}

	return "", fmt.Errorf("could not find user-facing ID for new todo")
}

// Complete marks a todo item as completed
func (t *Todo) Complete(userFacingID string) error {
	uuid, err := t.store.ResolveUUID(userFacingID)
	if err != nil {
		return fmt.Errorf("failed to resolve ID '%s': %w", userFacingID, err)
	}

	err = SetStatus(t.store, uuid, "completed")
	if err != nil {
		return fmt.Errorf("failed to complete todo: %w", err)
	}

	return nil
}

// CompleteMultiple marks multiple todo items as completed
// This demonstrates the correct pattern for batch operations with dynamic IDs
func (t *Todo) CompleteMultiple(userFacingIDs []string) error {
	// IMPORTANT: Pre-resolve all IDs to UUIDs before any mutations
	// This prevents ID shifting from affecting subsequent resolutions
	type resolvedItem struct {
		userID string
		uuid   string
	}

	var items []resolvedItem

	// Step 1: Resolve all IDs first
	for _, id := range userFacingIDs {
		uuid, err := t.store.ResolveUUID(id)
		if err != nil {
			// Return error with context about which ID failed
			return fmt.Errorf("failed to resolve ID '%s': %w", id, err)
		}
		items = append(items, resolvedItem{userID: id, uuid: uuid})
	}

	// Step 2: Complete all items using their UUIDs
	var completed []string
	for _, item := range items {
		err := SetStatus(t.store, item.uuid, "completed")
		if err != nil {
			// If we fail partway through, report what was completed
			if len(completed) > 0 {
				return fmt.Errorf("failed to complete ID '%s' (already completed: %v): %w",
					item.userID, completed, err)
			}
			return fmt.Errorf("failed to complete ID '%s': %w", item.userID, err)
		}
		completed = append(completed, item.userID)
	}

	return nil
}

// Reopen marks a completed todo item as pending
func (t *Todo) Reopen(userFacingID string) error {
	uuid, err := t.store.ResolveUUID(userFacingID)
	if err != nil {
		return fmt.Errorf("failed to resolve ID '%s': %w", userFacingID, err)
	}

	err = SetStatus(t.store, uuid, "pending")
	if err != nil {
		return fmt.Errorf("failed to reopen todo: %w", err)
	}

	return nil
}

// Move changes the parent of a todo item
func (t *Todo) Move(userFacingID string, newParentID *string) error {
	uuid, err := t.store.ResolveUUID(userFacingID)
	if err != nil {
		return fmt.Errorf("failed to resolve ID '%s': %w", userFacingID, err)
	}

	// Resolve new parent ID if provided
	var newParentUUID *string
	if newParentID != nil && *newParentID != "" {
		parentUUID, err := t.store.ResolveUUID(*newParentID)
		if err != nil {
			return fmt.Errorf("failed to resolve new parent ID '%s': %w", *newParentID, err)
		}
		newParentUUID = &parentUUID
	}

	updates := nanostore.UpdateRequest{
		Dimensions: map[string]string{"parent_uuid": ""},
	}
	if newParentUUID != nil {
		updates.Dimensions["parent_uuid"] = *newParentUUID
	}

	err = t.store.Update(uuid, updates)
	if err != nil {
		return fmt.Errorf("failed to move todo: %w", err)
	}

	return nil
}

// Delete removes a todo item and optionally its children
func (t *Todo) Delete(userFacingID string, cascade bool) error {
	uuid, err := t.store.ResolveUUID(userFacingID)
	if err != nil {
		return fmt.Errorf("failed to resolve ID '%s': %w", userFacingID, err)
	}

	err = t.store.Delete(uuid, cascade)
	if err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}

	return nil
}

// ListOptions configures how todos are listed
type ListOptions struct {
	ShowAll bool   // Show both pending and completed items
	Search  string // Search filter
}

// TodoItem represents a todo with display information
type TodoItem struct {
	nanostore.Document
	Children    []*TodoItem
	IsCompleted bool
	// For display purposes
	Symbol string // ○ for pending, ● for completed, ◐ for mixed
}

// List returns todos in a hierarchical structure
func (t *Todo) List(opts ListOptions) ([]*TodoItem, error) {
	// IMPORTANT: For hierarchical IDs to work correctly, we need to get ALL documents
	// and filter them ourselves. This is because nanostore regenerates IDs based on
	// the query context - filtering breaks the hierarchical numbering.

	// Always get all documents to preserve hierarchical IDs
	docs, err := t.store.List(nanostore.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list todos: %w", err)
	}

	// Convert to TodoItems and build hierarchy
	items := make([]*TodoItem, len(docs))
	itemMap := make(map[string]*TodoItem)

	for i, doc := range docs {
		item := &TodoItem{
			Document: doc,
			IsCompleted: func() bool {
				status, _ := doc.Dimensions["status"].(string)
				return status == "completed"
			}(),
			Children: []*TodoItem{},
		}

		if item.IsCompleted {
			item.Symbol = "●"
		} else {
			item.Symbol = "○"
		}

		items[i] = item
		itemMap[doc.UUID] = item
	}

	// Build hierarchy
	var roots []*TodoItem
	for _, item := range items {
		parentUUID, hasParent := item.Document.Dimensions["parent_uuid"].(string)
		if !hasParent || parentUUID == "" {
			roots = append(roots, item)
		} else {
			if parent, ok := itemMap[parentUUID]; ok {
				parent.Children = append(parent.Children, item)
			}
		}
	}

	// Sort roots and children by their user-facing IDs
	sortItems(roots)

	// Update symbols for mixed status parents
	updateParentSymbols(roots)

	// Apply search filter if provided
	if opts.Search != "" {
		roots = filterBySearch(roots, opts.Search)
	}

	// Filter out completed items if not showing all
	if !opts.ShowAll {
		roots = filterPending(roots)
	}

	return roots, nil
}

// filterBySearch filters items by search query, keeping parents for context
func filterBySearch(items []*TodoItem, query string) []*TodoItem {
	query = strings.ToLower(query)
	var filtered []*TodoItem

	for _, item := range items {
		// Check if item matches
		matches := strings.Contains(strings.ToLower(item.Title), query) ||
			strings.Contains(strings.ToLower(item.Body), query)

		// Check if any children match
		childMatches := hasMatchingChildren(item.Children, query)

		if matches || childMatches {
			// Clone the item
			filteredItem := &TodoItem{
				Document:    item.Document,
				IsCompleted: item.IsCompleted,
				Symbol:      item.Symbol,
				Children:    nil,
			}

			// Recursively filter children
			if len(item.Children) > 0 {
				filteredItem.Children = filterBySearch(item.Children, query)
			}

			filtered = append(filtered, filteredItem)
		}
	}

	return filtered
}

// hasMatchingChildren checks if any children match the search query
func hasMatchingChildren(children []*TodoItem, query string) bool {
	for _, child := range children {
		if strings.Contains(strings.ToLower(child.Title), query) ||
			strings.Contains(strings.ToLower(child.Body), query) {
			return true
		}
		if hasMatchingChildren(child.Children, query) {
			return true
		}
	}
	return false
}

// filterPending recursively filters out completed items
func filterPending(items []*TodoItem) []*TodoItem {
	var filtered []*TodoItem

	for _, item := range items {
		// Skip completed items
		if item.IsCompleted {
			continue
		}

		// Clone the item to avoid modifying the original
		filteredItem := &TodoItem{
			Document:    item.Document,
			IsCompleted: item.IsCompleted,
			Symbol:      "○", // Reset to pending symbol since we're filtering
			Children:    nil,
		}

		// Recursively filter children
		if len(item.Children) > 0 {
			filteredItem.Children = filterPending(item.Children)
		}

		filtered = append(filtered, filteredItem)
	}

	return filtered
}

// sortItems recursively sorts items by their user-facing IDs
func sortItems(items []*TodoItem) {
	// Sort current level
	sort.Slice(items, func(i, j int) bool {
		return compareIDs(items[i].UserFacingID, items[j].UserFacingID)
	})

	// Recursively sort children
	for _, item := range items {
		if len(item.Children) > 0 {
			sortItems(item.Children)
		}
	}
}

// compareIDs compares two user-facing IDs for sorting
func compareIDs(a, b string) bool {
	// Parse IDs to handle numeric and hierarchical comparison
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	for i := 0; i < len(partsA) && i < len(partsB); i++ {
		// Check if one has 'c' prefix (completed) and the other doesn't
		hasCA := strings.HasPrefix(partsA[i], "c")
		hasCB := strings.HasPrefix(partsB[i], "c")

		// If one is completed and the other isn't, pending comes first
		if hasCA != hasCB {
			return !hasCA // pending (no 'c') comes before completed (has 'c')
		}

		// Extract numeric part after any prefix
		numA := extractNumber(partsA[i])
		numB := extractNumber(partsB[i])

		if numA != numB {
			return numA < numB
		}
	}

	return len(partsA) < len(partsB)
}

// extractNumber extracts the numeric part from an ID segment (e.g., "c1" -> 1)
func extractNumber(s string) int {
	// Skip prefix characters
	i := 0
	for i < len(s) && (s[i] < '0' || s[i] > '9') {
		i++
	}

	// Parse number
	num := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		num = num*10 + int(s[i]-'0')
		i++
	}

	return num
}

// updateParentSymbols updates parent symbols to show mixed status
func updateParentSymbols(items []*TodoItem) {
	for _, item := range items {
		if len(item.Children) > 0 {
			hasCompleted := false
			hasPending := false

			checkChildren(item.Children, &hasCompleted, &hasPending)

			if hasCompleted && hasPending {
				item.Symbol = "◐"
			}

			updateParentSymbols(item.Children)
		}
	}
}

func checkChildren(children []*TodoItem, hasCompleted, hasPending *bool) {
	for _, child := range children {
		if child.IsCompleted {
			*hasCompleted = true
		} else {
			*hasPending = true
		}

		if len(child.Children) > 0 {
			checkChildren(child.Children, hasCompleted, hasPending)
		}
	}
}

// Search is a convenience method for searching todos
func (t *Todo) Search(query string, showAll bool) ([]*TodoItem, error) {
	return t.List(ListOptions{
		ShowAll: showAll,
		Search:  query,
	})
}

// FormatTree formats todos in a tree structure for display
func FormatTree(items []*TodoItem, indent string, showCompleted bool) string {
	var sb strings.Builder

	for _, item := range items {
		// Skip completed items if not showing all
		if !showCompleted && item.IsCompleted {
			continue
		}

		sb.WriteString(fmt.Sprintf("%s%s %s. %s\n",
			indent, item.Symbol, item.UserFacingID, item.Title))

		if len(item.Children) > 0 {
			childrenStr := FormatTree(item.Children, indent+"  ", showCompleted)
			sb.WriteString(childrenStr)
		}
	}

	return sb.String()
}
