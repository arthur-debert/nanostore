package api

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

// Test struct for reproducing the body persistence issue from GitHub issue #90
type TestDocForBodyPersistence struct {
	nanostore.Document
	Status string `values:"active,inactive" default:"active"`
}

func TestBodyPersistenceInCreateMethod(t *testing.T) {
	// Create temporary file
	tmpfile, err := os.CreateTemp("", "test_body_persistence*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create store
	store, err := New[TestDocForBodyPersistence](tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Test case 1: Body content in embedded Document
	t.Run("body_content_in_embedded_document", func(t *testing.T) {
		originalBody := "This is the body content that should be persisted"

		testDoc := &TestDocForBodyPersistence{
			Document: nanostore.Document{
				Title: "Test Document",
				Body:  originalBody,
			},
			Status: "active",
		}

		// Debug: Check if extractDocumentFields works
		title, body, found := extractDocumentFields(testDoc)
		t.Logf("extractDocumentFields result: title=%q, body=%q, found=%v", title, body, found)

		if !found {
			t.Fatal("extractDocumentFields should find embedded Document")
		}
		if body != originalBody {
			t.Errorf("extractDocumentFields body mismatch: got %q, want %q", body, originalBody)
		}

		// Create document
		simpleID, err := store.Create("Test Document", testDoc)
		if err != nil {
			t.Fatalf("Failed to create document: %v", err)
		}

		t.Logf("Created document with ID: %s", simpleID)
		t.Logf("Original body: %q", testDoc.Body)

		// Retrieve the document
		retrieved, err := store.Get(simpleID)
		if err != nil {
			t.Fatalf("Failed to retrieve document: %v", err)
		}

		t.Logf("Retrieved body: %q", retrieved.Body)

		// This should pass but currently fails according to issue #90
		if retrieved.Body != originalBody {
			t.Errorf("Body persistence failed: expected %q, got %q", originalBody, retrieved.Body)
		}

		// Also verify other fields work correctly
		if retrieved.Status != "active" {
			t.Errorf("Status mismatch: expected 'active', got %q", retrieved.Status)
		}
		if retrieved.Title != "Test Document" {
			t.Errorf("Title mismatch: expected 'Test Document', got %q", retrieved.Title)
		}
	})

	// Test case 2: Empty body should also work
	t.Run("empty_body_content", func(t *testing.T) {
		testDoc := &TestDocForBodyPersistence{
			Document: nanostore.Document{
				Title: "Test Document 2",
				Body:  "", // Empty body
			},
			Status: "inactive",
		}

		simpleID, err := store.Create("Test Document 2", testDoc)
		if err != nil {
			t.Fatalf("Failed to create document: %v", err)
		}

		retrieved, err := store.Get(simpleID)
		if err != nil {
			t.Fatalf("Failed to retrieve document: %v", err)
		}

		if retrieved.Body != "" {
			t.Errorf("Empty body should remain empty, got %q", retrieved.Body)
		}
	})

	// Test case 3: No embedded document body (only title parameter)
	t.Run("title_parameter_only", func(t *testing.T) {
		testDoc := &TestDocForBodyPersistence{
			Status: "active",
		}

		simpleID, err := store.Create("Parameter Title", testDoc)
		if err != nil {
			t.Fatalf("Failed to create document: %v", err)
		}

		retrieved, err := store.Get(simpleID)
		if err != nil {
			t.Fatalf("Failed to retrieve document: %v", err)
		}

		if retrieved.Title != "Parameter Title" {
			t.Errorf("Title mismatch: expected 'Parameter Title', got %q", retrieved.Title)
		}

		// Body should be empty when not specified
		if retrieved.Body != "" {
			t.Errorf("Body should be empty when not specified, got %q", retrieved.Body)
		}
	})
}

// Test the extractDocumentFields function directly to isolate the issue
func TestExtractDocumentFieldsFunction(t *testing.T) {
	testCases := []struct {
		name        string
		input       interface{}
		expectTitle string
		expectBody  string
		expectFound bool
	}{
		{
			name: "embedded_document_with_body",
			input: &TestDocForBodyPersistence{
				Document: nanostore.Document{
					Title: "Test Title",
					Body:  "Test Body Content",
				},
				Status: "active",
			},
			expectTitle: "Test Title",
			expectBody:  "Test Body Content",
			expectFound: true,
		},
		{
			name: "embedded_document_empty_body",
			input: &TestDocForBodyPersistence{
				Document: nanostore.Document{
					Title: "Test Title",
					Body:  "",
				},
				Status: "active",
			},
			expectTitle: "Test Title",
			expectBody:  "",
			expectFound: true,
		},
		{
			name: "no_embedded_document",
			input: &struct {
				Status string
			}{
				Status: "active",
			},
			expectTitle: "",
			expectBody:  "",
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			title, body, found := extractDocumentFields(tc.input)

			if title != tc.expectTitle {
				t.Errorf("Title mismatch: got %q, want %q", title, tc.expectTitle)
			}
			if body != tc.expectBody {
				t.Errorf("Body mismatch: got %q, want %q", body, tc.expectBody)
			}
			if found != tc.expectFound {
				t.Errorf("Found mismatch: got %v, want %v", found, tc.expectFound)
			}
		})
	}
}
