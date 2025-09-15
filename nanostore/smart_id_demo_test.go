package nanostore_test

import (
	"fmt"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

// TestSmartIDDetectionDemo demonstrates the smart ID detection feature
func TestSmartIDDetectionDemo(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create some test documents
	docUUID, err := store.Add("My Task", nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// List to get user-facing ID
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	userFacingID := docs[0].UserFacingID

	fmt.Printf("Created document:\n")
	fmt.Printf("  UUID: %s\n", docUUID)
	fmt.Printf("  User-facing ID: %s\n", userFacingID)
	fmt.Printf("\n")

	// Demonstrate that both ID types work seamlessly
	fmt.Printf("Smart ID Detection in action:\n")

	// Update using UUID (works as before)
	fmt.Printf("1. Updating with UUID (%s)...\n", docUUID)
	err = store.Update(docUUID, nanostore.UpdateRequest{
		Title: stringPtr("Updated via UUID"),
	})
	if err != nil {
		t.Errorf("failed to update with UUID: %v", err)
	} else {
		fmt.Printf("   ✓ Success!\n")
	}

	// Update using user-facing ID (new capability!)
	fmt.Printf("2. Updating with user-facing ID (%s)...\n", userFacingID)
	err = store.Update(userFacingID, nanostore.UpdateRequest{
		Title: stringPtr("Updated via user-facing ID"),
	})
	if err != nil {
		t.Errorf("failed to update with user-facing ID: %v", err)
	} else {
		fmt.Printf("   ✓ Success!\n")
	}

	// Set status using user-facing ID
	fmt.Printf("3. Setting status with user-facing ID (%s)...\n", userFacingID)
	err = nanostore.TestSetStatusUpdate(store, userFacingID, "completed")
	if err != nil {
		t.Errorf("failed to set status with user-facing ID: %v", err)
	} else {
		fmt.Printf("   ✓ Success!\n")
	}

	// Get fresh list to see the completed document
	docs, err = store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	completedUserID := docs[0].UserFacingID

	// Delete using the new completed user-facing ID
	fmt.Printf("4. Deleting with completed user-facing ID (%s)...\n", completedUserID)
	err = store.Delete(completedUserID, false)
	if err != nil {
		t.Errorf("failed to delete with user-facing ID: %v", err)
	} else {
		fmt.Printf("   ✓ Success!\n")
	}

	fmt.Printf("\nDemo complete! Smart ID detection allows seamless use of both UUID and user-facing IDs.\n")
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}
