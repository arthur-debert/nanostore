package todo_test

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/examples/apps/todo"
)

func TestTodoBasicOperations(t *testing.T) {
	app, err := todo.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create todo app: %v", err)
	}
	defer app.Close()

	// Add root items
	groceriesID, err := app.Add("Groceries", nil)
	if err != nil {
		t.Fatalf("failed to add groceries: %v", err)
	}
	if groceriesID != "1" {
		t.Errorf("expected groceries ID to be '1', got '%s'", groceriesID)
	}

	tripID, err := app.Add("Pack for Trip", nil)
	if err != nil {
		t.Fatalf("failed to add trip: %v", err)
	}
	if tripID != "2" {
		t.Errorf("expected trip ID to be '2', got '%s'", tripID)
	}

	// Add sub-items to groceries
	milkID, err := app.Add("Milk", &groceriesID)
	if err != nil {
		t.Fatalf("failed to add milk: %v", err)
	}
	if milkID != "1.1" {
		t.Errorf("expected milk ID to be '1.1', got '%s'", milkID)
	}

	breadID, err := app.Add("Bread", &groceriesID)
	if err != nil {
		t.Fatalf("failed to add bread: %v", err)
	}
	if breadID != "1.2" {
		t.Errorf("expected bread ID to be '1.2', got '%s'", breadID)
	}

	eggsID, err := app.Add("Eggs", &groceriesID)
	if err != nil {
		t.Fatalf("failed to add eggs: %v", err)
	}
	if eggsID != "1.3" {
		t.Errorf("expected eggs ID to be '1.3', got '%s'", eggsID)
	}

	// List todos (should show only pending)
	items, err := app.List(todo.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}

	output := todo.FormatTree(items, "", false)
	expected := `○ 1. Groceries
  ○ 1.1. Milk
  ○ 1.2. Bread
  ○ 1.3. Eggs
○ 2. Pack for Trip
`
	if output != expected {
		t.Errorf("unexpected list output:\nGot:\n%s\nExpected:\n%s", output, expected)
	}
}

func TestTodoCompletion(t *testing.T) {
	app, err := todo.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create todo app: %v", err)
	}
	defer app.Close()

	// Setup
	groceriesID, _ := app.Add("Groceries", nil)
	app.Add("Milk", &groceriesID)
	breadID, _ := app.Add("Bread", &groceriesID)
	app.Add("Eggs", &groceriesID)

	// Complete bread (1.2)
	err = app.Complete(breadID)
	if err != nil {
		t.Fatalf("failed to complete bread: %v", err)
	}

	// List pending only - bread should be gone, eggs should be 1.2 now
	items, err := app.List(todo.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}

	output := todo.FormatTree(items, "", false)
	expected := `○ 1. Groceries
  ○ 1.1. Milk
  ○ 1.2. Eggs
`
	if output != expected {
		t.Errorf("unexpected list output after completion:\nGot:\n%s\nExpected:\n%s", output, expected)
	}

	// List all - should show completed bread as 1.c1
	items, err = app.List(todo.ListOptions{ShowAll: true})
	if err != nil {
		t.Fatalf("failed to list all todos: %v", err)
	}

	output = todo.FormatTree(items, "", true)
	if !strings.Contains(output, "◐ 1. Groceries") {
		t.Errorf("expected groceries to show mixed status symbol ◐")
	}
	if !strings.Contains(output, "● 1.c1. Bread") {
		t.Errorf("expected completed bread to show as '● 1.c1. Bread'")
	}
}

func TestTodoReopen(t *testing.T) {
	app, err := todo.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create todo app: %v", err)
	}
	defer app.Close()

	// Setup
	groceriesID, _ := app.Add("Groceries", nil)
	app.Add("Milk", &groceriesID)
	breadID, _ := app.Add("Bread", &groceriesID)
	app.Add("Eggs", &groceriesID)

	// Complete and then reopen bread
	app.Complete(breadID)

	// Get the completed ID
	items, _ := app.List(todo.ListOptions{ShowAll: true})
	var completedBreadID string
	for _, item := range items {
		for _, child := range item.Children {
			if child.Title == "Bread" && child.IsCompleted {
				completedBreadID = child.UserFacingID
				break
			}
		}
	}

	if completedBreadID != "1.c1" {
		t.Errorf("expected completed bread ID to be '1.c1', got '%s'", completedBreadID)
	}

	// Reopen bread
	err = app.Reopen(completedBreadID)
	if err != nil {
		t.Fatalf("failed to reopen bread: %v", err)
	}

	// List pending - bread should be back in its original position
	items, err = app.List(todo.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}

	output := todo.FormatTree(items, "", false)
	// Bread returns to position 1.2 (its original creation order)
	expected := `○ 1. Groceries
  ○ 1.1. Milk
  ○ 1.2. Bread
  ○ 1.3. Eggs
`
	if output != expected {
		t.Errorf("unexpected list output after reopen:\nGot:\n%s\nExpected:\n%s", output, expected)
	}
}

func TestTodoSearch(t *testing.T) {
	app, err := todo.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create todo app: %v", err)
	}
	defer app.Close()

	// Setup
	groceriesID, _ := app.Add("Groceries", nil)
	app.Add("Milk", &groceriesID)
	app.Add("Bread", &groceriesID)
	app.Add("Eggs", &groceriesID)

	tripID, _ := app.Add("Pack for Trip", nil)
	app.Add("Clothes", &tripID)
	app.Add("Camera Gear", &tripID)
	app.Add("Passport", &tripID)

	// Search for 'r' - should find Groceries, Bread, Trip, Camera Gear, Passport
	items, err := app.Search("r", false)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	// Count matching items
	count := countItems(items)
	if count != 5 { // Groceries, Bread, Pack for Trip, Camera Gear, Passport
		t.Errorf("expected 5 items matching 'r', got %d", count)
	}

	// Search for items containing both 'r' and 'g' patterns
	// This would need OR logic which nanostore doesn't support directly
	// So let's search for 'Gear' instead
	items, err = app.Search("Gear", false)
	if err != nil {
		t.Fatalf("failed to search for Gear: %v", err)
	}

	count = countItems(items)
	if count != 2 { // Pack for Trip (parent context) and Camera Gear
		t.Errorf("expected 2 items for 'Gear' search, got %d", count)
	}
}

func TestTodoSearchWithCompleted(t *testing.T) {
	app, err := todo.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create todo app: %v", err)
	}
	defer app.Close()

	// Setup
	groceriesID, _ := app.Add("Groceries", nil)
	app.Add("Milk", &groceriesID)
	breadID, _ := app.Add("Bread", &groceriesID)
	app.Add("Eggs", &groceriesID)

	// Complete bread
	app.Complete(breadID)

	// Search for 'r' with ShowAll
	items, err := app.Search("r", true)
	if err != nil {
		t.Fatalf("failed to search with ShowAll: %v", err)
	}

	// Should find Groceries and completed Bread
	output := todo.FormatTree(items, "", true)
	if !strings.Contains(output, "◐ 1. Groceries") {
		t.Errorf("expected to find Groceries with mixed status")
	}
	if !strings.Contains(output, "● 1.c1. Bread") {
		t.Errorf("expected to find completed Bread as '● 1.c1. Bread'")
	}
}

func TestTodoMove(t *testing.T) {
	app, err := todo.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create todo app: %v", err)
	}
	defer app.Close()

	// Setup two root categories
	homeID, _ := app.Add("Home Tasks", nil)
	workID, _ := app.Add("Work Tasks", nil)

	// Add item to home
	laundryID, _ := app.Add("Do Laundry", &homeID)

	// Move laundry to work (just for testing)
	err = app.Move(laundryID, &workID)
	if err != nil {
		t.Fatalf("failed to move task: %v", err)
	}

	// List and verify
	items, _ := app.List(todo.ListOptions{})
	output := todo.FormatTree(items, "", false)

	// Laundry should now be under Work Tasks as 2.1
	if !strings.Contains(output, "○ 2. Work Tasks\n  ○ 2.1. Do Laundry") {
		t.Errorf("expected laundry to be under work tasks as 2.1")
	}
}

func TestTodoDelete(t *testing.T) {
	app, err := todo.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create todo app: %v", err)
	}
	defer app.Close()

	// Setup
	groceriesID, _ := app.Add("Groceries", nil)
	app.Add("Milk", &groceriesID)
	app.Add("Bread", &groceriesID)

	// Try to delete parent without cascade - should fail
	err = app.Delete("1", false)
	if err == nil {
		t.Errorf("expected error when deleting parent without cascade")
	}

	// Delete parent with cascade
	err = app.Delete("1", true)
	if err != nil {
		t.Fatalf("failed to delete with cascade: %v", err)
	}

	// List should be empty
	items, _ := app.List(todo.ListOptions{})
	if len(items) != 0 {
		t.Errorf("expected no items after cascade delete, got %d", len(items))
	}
}

// Helper function to count all items in tree
func countItems(items []*todo.TodoItem) int {
	count := len(items)
	for _, item := range items {
		count += countItems(item.Children)
	}
	return count
}
