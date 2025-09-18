package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

// UniverseData provides typed access to the test fixture data
type UniverseData struct {
	// Root level documents
	PersonalRoot nanostore.Document // ID: "1" - Personal tasks root
	WorkRoot     nanostore.Document // ID: "2" - Work projects root
	ArchivedRoot nanostore.Document // ID: "3" - Archived root with active children

	// Personal hierarchy
	BuyGroceries    nanostore.Document // ID: "1.1" - Under PersonalRoot
	ExerciseRoutine nanostore.Document // ID: "h1.2" - High priority
	ReadBook        nanostore.Document // ID: "d1.3" - Done status

	// Grocery subtasks
	Milk  nanostore.Document // ID: "h1.1.1" - High priority
	Bread nanostore.Document // ID: "d1.1.2" - Done status

	// Work hierarchy
	TeamMeeting      nanostore.Document // ID: "h2.1" - High priority meeting
	CodeReview       nanostore.Document // ID: "2.2" - Medium priority
	DeployProduction nanostore.Document // ID: "ah2.3" - Archived, high priority

	// Meeting subtasks
	PrepareAgenda nanostore.Document // ID: "dh2.1.1" - Done, high priority
	UpdateSlides  nanostore.Document // ID: "2.1.2" - Active

	// Deep nesting (5 levels)
	Level3Task nanostore.Document // ID: "2.1.1.1"
	Level4Task nanostore.Document // ID: "2.1.1.1.1"
	Level5Task nanostore.Document // ID: "dh2.1.1.1.1.1"

	// Edge cases
	EmptyTitle   nanostore.Document // Empty title document
	SpecialChars nanostore.Document // Special characters in title
	UnicodeEmoji nanostore.Document // Unicode/emoji in title

	// Search test documents
	PackForTrip nanostore.Document // ID: "h4" - For search testing
	PackLunch   nanostore.Document // ID: "d5" - For search testing

	// Mixed state parent
	MixedParent  nanostore.Document // ID: "6" - Has children in different states
	ActiveChild  nanostore.Document // ID: "h6.1"
	PendingChild nanostore.Document // ID: "6.2"
	DoneChild    nanostore.Document // ID: "d6.3"

	// Deleted parent with active children
	DeletedParent nanostore.Document // ID: "7" - Deleted but has active children
	OrphanChild   nanostore.Document // ID: "h7.1" - Active child of deleted parent

	// All documents map for easy access by UUID
	ByUUID map[string]nanostore.Document
}

// fixtureDocument represents the JSON structure in universe.json
type fixtureDocument struct {
	UUID       string                 `json:"uuid"`
	Title      string                 `json:"title"`
	Body       string                 `json:"body"`
	Dimensions map[string]interface{} `json:"dimensions"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

type fixtureData struct {
	Documents []fixtureDocument `json:"documents"`
}

// LoadUniverse loads the test fixture and returns a store populated with the universe data
func LoadUniverse(t *testing.T) (nanostore.Store, *UniverseData) {
	t.Helper()

	// Create temporary store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(tmpfile.Name()) })
	_ = tmpfile.Close()

	// Define the configuration matching our test data
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "active", "done"},
				Prefixes:     map[string]string{"done": "d"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "medium", "high"},
				Prefixes:     map[string]string{"high": "h"},
				DefaultValue: "medium",
			},
			{
				Name:         "category",
				Type:         nanostore.Enumerated,
				Values:       []string{"personal", "work", "other"},
				DefaultValue: "other",
			},
			{
				Name:         "activity",
				Type:         nanostore.Enumerated,
				Values:       []string{"active", "archived", "deleted"},
				DefaultValue: "active",
			},
			{
				Name:     "parent_id",
				Type:     nanostore.Hierarchical,
				RefField: "parent_id",
			},
		},
	}

	// Create store
	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	// Load fixture data
	fixturePath := filepath.Join("..", "testdata", "universe.json")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture file: %v", err)
	}

	var fixture fixtureData
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("failed to parse fixture: %v", err)
	}

	// Import documents into store preserving UUIDs
	universe := &UniverseData{
		ByUUID: make(map[string]nanostore.Document),
	}

	// First pass: Create a mapping from fixture UUIDs to actual UUIDs
	fixtureToActualUUID := make(map[string]string)

	// Process documents in order to ensure parents are created before children
	// First add all root documents (no parent_id)
	for _, doc := range fixture.Documents {
		if _, hasParent := doc.Dimensions["parent_id"]; hasParent {
			continue // Skip documents with parents for now
		}

		// Add root document to store
		actualUUID, err := store.Add(doc.Title, doc.Dimensions)
		if err != nil {
			t.Fatalf("failed to add document %s: %v", doc.UUID, err)
		}

		// Update with body if present
		if doc.Body != "" {
			err = store.Update(actualUUID, nanostore.UpdateRequest{
				Body: &doc.Body,
			})
			if err != nil {
				t.Fatalf("failed to update body for document %s: %v", doc.UUID, err)
			}
		}

		// Map fixture UUID to actual UUID
		fixtureToActualUUID[doc.UUID] = actualUUID
	}

	// Multiple passes for hierarchical documents
	maxPasses := 10
	for pass := 0; pass < maxPasses; pass++ {
		added := false
		for _, doc := range fixture.Documents {
			parentFixtureID, hasParent := doc.Dimensions["parent_id"]
			if !hasParent {
				continue // Already processed roots
			}

			// Check if we already processed this document
			if _, alreadyProcessed := fixtureToActualUUID[doc.UUID]; alreadyProcessed {
				continue
			}

			// Check if parent has been processed
			actualParentUUID, parentProcessed := fixtureToActualUUID[parentFixtureID.(string)]
			if !parentProcessed {
				continue // Wait for parent to be processed
			}

			// Update dimensions to use actual parent UUID
			dimensions := make(map[string]interface{})
			for k, v := range doc.Dimensions {
				dimensions[k] = v
			}
			dimensions["parent_id"] = actualParentUUID

			// Add document to store
			actualUUID, err := store.Add(doc.Title, dimensions)
			if err != nil {
				t.Fatalf("failed to add document %s: %v", doc.UUID, err)
			}

			// Update with body if present
			if doc.Body != "" {
				err = store.Update(actualUUID, nanostore.UpdateRequest{
					Body: &doc.Body,
				})
				if err != nil {
					t.Fatalf("failed to update body for document %s: %v", doc.UUID, err)
				}
			}

			// Map fixture UUID to actual UUID
			fixtureToActualUUID[doc.UUID] = actualUUID
			added = true
		}

		if !added {
			break // No more documents to process
		}
	}

	// Now retrieve all documents and build universe data structure
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Create a map from actual UUID to document
	actualUUIDToDoc := make(map[string]nanostore.Document)
	for _, doc := range docs {
		actualUUIDToDoc[doc.UUID] = doc
	}

	// Build universe data structure using fixture UUIDs as keys
	for fixtureUUID, actualUUID := range fixtureToActualUUID {
		doc := actualUUIDToDoc[actualUUID]
		universe.ByUUID[fixtureUUID] = doc

		// Map specific documents to named fields
		switch fixtureUUID {
		case "root-1":
			universe.PersonalRoot = doc
		case "root-2":
			universe.WorkRoot = doc
		case "root-3":
			universe.ArchivedRoot = doc
		case "personal-1":
			universe.BuyGroceries = doc
		case "personal-2":
			universe.ExerciseRoutine = doc
		case "personal-3":
			universe.ReadBook = doc
		case "work-1":
			universe.TeamMeeting = doc
		case "work-2":
			universe.CodeReview = doc
		case "work-3":
			universe.DeployProduction = doc
		case "grocery-1":
			universe.Milk = doc
		case "grocery-2":
			universe.Bread = doc
		case "meeting-1":
			universe.PrepareAgenda = doc
		case "meeting-2":
			universe.UpdateSlides = doc
		case "edge-empty":
			universe.EmptyTitle = doc
		case "edge-special":
			universe.SpecialChars = doc
		case "edge-unicode":
			universe.UnicodeEmoji = doc
		case "search-1":
			universe.PackForTrip = doc
		case "search-2":
			universe.PackLunch = doc
		case "mixed-parent":
			universe.MixedParent = doc
		case "mixed-child-1":
			universe.ActiveChild = doc
		case "mixed-child-2":
			universe.PendingChild = doc
		case "mixed-child-3":
			universe.DoneChild = doc
		case "deleted-parent":
			universe.DeletedParent = doc
		case "orphan-1":
			universe.OrphanChild = doc
		case "deep-1":
			universe.Level3Task = doc
		case "deep-2":
			universe.Level4Task = doc
		case "deep-3":
			universe.Level5Task = doc
		case "archived-child-1":
			// Not mapped to a specific field but available in ByUUID
		}
	}

	return store, universe
}

// GetActiveDocuments returns all documents with activity="active"
func (u *UniverseData) GetActiveDocuments() []nanostore.Document {
	var active []nanostore.Document
	for _, doc := range u.ByUUID {
		if doc.Dimensions["activity"] == "active" {
			active = append(active, doc)
		}
	}
	return active
}

// GetDocumentsByStatus returns all documents with the given status
func (u *UniverseData) GetDocumentsByStatus(status string) []nanostore.Document {
	var docs []nanostore.Document
	for _, doc := range u.ByUUID {
		if doc.Dimensions["status"] == status {
			docs = append(docs, doc)
		}
	}
	return docs
}

// GetChildrenOf returns all direct children of the given parent UUID
func (u *UniverseData) GetChildrenOf(parentUUID string) []nanostore.Document {
	var children []nanostore.Document
	// Find the parent document first to get its UUID
	parent := u.ByUUID[parentUUID]
	for _, doc := range u.ByUUID {
		// Parent references are stored as UUIDs in dimensions
		if doc.Dimensions["parent_id"] == parent.UUID {
			children = append(children, doc)
		}
	}
	return children
}

// GetRootDocuments returns all documents without a parent
func (u *UniverseData) GetRootDocuments() []nanostore.Document {
	var roots []nanostore.Document
	for _, doc := range u.ByUUID {
		if _, hasParent := doc.Dimensions["parent_id"]; !hasParent {
			roots = append(roots, doc)
		}
	}
	return roots
}
