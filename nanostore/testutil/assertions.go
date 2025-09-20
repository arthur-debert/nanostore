package testutil

import (
	"fmt"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

// AssertDocumentCount checks that the slice contains the expected number of documents
func AssertDocumentCount(t *testing.T, docs []nanostore.Document, expected int, context ...string) {
	t.Helper()
	if len(docs) != expected {
		ctx := ""
		if len(context) > 0 {
			ctx = " " + context[0]
		}
		t.Errorf("expected %d documents%s, got %d", expected, ctx, len(docs))
	}
}

// AssertDocumentExists verifies that a document with the given UUID exists in the slice
func AssertDocumentExists(t *testing.T, docs []nanostore.Document, uuid string) {
	t.Helper()
	for _, doc := range docs {
		if doc.UUID == uuid {
			return
		}
	}
	t.Errorf("document %s not found in results", uuid)
}

// AssertDocumentNotExists verifies that a document with the given UUID does not exist in the slice
func AssertDocumentNotExists(t *testing.T, docs []nanostore.Document, uuid string) {
	t.Helper()
	for _, doc := range docs {
		if doc.UUID == uuid {
			t.Errorf("document %s should not be in results", uuid)
			return
		}
	}
}

// AssertChildCount verifies that a parent has the expected number of direct children
func AssertChildCount(t *testing.T, store nanostore.Store, parentUUID string, expected int) {
	t.Helper()
	children, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{
			"parent_id": parentUUID,
		},
	})
	if err != nil {
		t.Fatalf("failed to query children: %v", err)
	}

	if len(children) != expected {
		t.Errorf("expected %d children for parent %s, got %d", expected, parentUUID, len(children))
	}
}

// AssertPendingChildCount counts children with status="pending" for a given parent
func AssertPendingChildCount(t *testing.T, store nanostore.Store, parentUUID string, expected int) {
	t.Helper()
	children, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{
			"parent_id": parentUUID,
			"status":    "pending",
		},
	})
	if err != nil {
		t.Fatalf("failed to query pending children: %v", err)
	}

	if len(children) != expected {
		t.Errorf("expected %d pending children for parent %s, got %d", expected, parentUUID, len(children))
	}
}

// AssertActiveChildCount counts children with status="active" for a given parent
func AssertActiveChildCount(t *testing.T, store nanostore.Store, parentUUID string, expected int) {
	t.Helper()
	children, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{
			"parent_id": parentUUID,
			"status":    "active",
		},
	})
	if err != nil {
		t.Fatalf("failed to query active children: %v", err)
	}

	if len(children) != expected {
		t.Errorf("expected %d active children for parent %s, got %d", expected, parentUUID, len(children))
	}
}

// AssertDoneChildCount counts children with status="done" for a given parent
func AssertDoneChildCount(t *testing.T, store nanostore.Store, parentUUID string, expected int) {
	t.Helper()
	children, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{
			"parent_id": parentUUID,
			"status":    "done",
		},
	})
	if err != nil {
		t.Fatalf("failed to query done children: %v", err)
	}

	if len(children) != expected {
		t.Errorf("expected %d done children for parent %s, got %d", expected, parentUUID, len(children))
	}
}

// AssertIsRoot verifies that a document has no parent
func AssertIsRoot(t *testing.T, doc nanostore.Document) {
	t.Helper()
	if _, hasParent := doc.Dimensions["parent_id"]; hasParent {
		t.Errorf("document %s should be a root (no parent_id), but has parent_id=%v",
			doc.UUID, doc.Dimensions["parent_id"])
	}
}

// AssertHasParent verifies that a document has a specific parent
func AssertHasParent(t *testing.T, doc nanostore.Document, expectedParentUUID string) {
	t.Helper()
	parentID, hasParent := doc.Dimensions["parent_id"]
	if !hasParent {
		t.Errorf("document %s should have parent %s, but has no parent_id",
			doc.UUID, expectedParentUUID)
		return
	}

	if parentID != expectedParentUUID {
		t.Errorf("document %s should have parent %s, but has parent %v",
			doc.UUID, expectedParentUUID, parentID)
	}
}

// AssertAllHaveDimension verifies that all documents have a specific dimension value
func AssertAllHaveDimension(t *testing.T, docs []nanostore.Document, dimension, value string) {
	t.Helper()
	for _, doc := range docs {
		actual, exists := doc.Dimensions[dimension]
		if !exists {
			t.Errorf("document %s missing dimension %s", doc.UUID, dimension)
			continue
		}
		if actual != value {
			t.Errorf("document %s: expected %s=%s, got %s=%v",
				doc.UUID, dimension, value, dimension, actual)
		}
	}
}

// AssertDimensionValues verifies that a document has all expected dimension values
func AssertDimensionValues(t *testing.T, doc nanostore.Document, expected map[string]string) {
	t.Helper()
	for dim, expectedVal := range expected {
		actual, exists := doc.Dimensions[dim]
		if !exists {
			t.Errorf("document %s missing dimension %s", doc.UUID, dim)
			continue
		}
		if fmt.Sprint(actual) != expectedVal {
			t.Errorf("document %s: expected %s=%s, got %s=%v",
				doc.UUID, dim, expectedVal, dim, actual)
		}
	}
}

// AssertHasStatus verifies that a document has a specific status
func AssertHasStatus(t *testing.T, doc nanostore.Document, status string) {
	t.Helper()
	actual, exists := doc.Dimensions["status"]
	if !exists {
		t.Errorf("document %s missing status dimension", doc.UUID)
		return
	}
	if actual != status {
		t.Errorf("document %s: expected status=%s, got %v", doc.UUID, status, actual)
	}
}

// AssertHasPriority verifies that a document has a specific priority
func AssertHasPriority(t *testing.T, doc nanostore.Document, priority string) {
	t.Helper()
	actual, exists := doc.Dimensions["priority"]
	if !exists {
		t.Errorf("document %s missing priority dimension", doc.UUID)
		return
	}
	if actual != priority {
		t.Errorf("document %s: expected priority=%s, got %v", doc.UUID, priority, actual)
	}
}

// AssertOrderedBy verifies that documents are ordered by a specific field
func AssertOrderedBy(t *testing.T, docs []nanostore.Document, field string, ascending bool) {
	t.Helper()
	if len(docs) < 2 {
		return // Nothing to check
	}

	getFieldValue := func(doc nanostore.Document) string {
		switch field {
		case "title":
			return doc.Title
		case "simple_id", "id":
			return doc.SimpleID
		case "uuid":
			return doc.UUID
		default:
			// Check dimensions
			if val, ok := doc.Dimensions[field]; ok {
				return fmt.Sprint(val)
			}
			return ""
		}
	}

	for i := 1; i < len(docs); i++ {
		prev := getFieldValue(docs[i-1])
		curr := getFieldValue(docs[i])

		if ascending && prev > curr {
			t.Errorf("documents not in ascending order by %s: %s > %s at positions %d,%d",
				field, prev, curr, i-1, i)
		} else if !ascending && prev < curr {
			t.Errorf("documents not in descending order by %s: %s < %s at positions %d,%d",
				field, prev, curr, i-1, i)
		}
	}
}

// AssertIDsInOrder verifies that documents appear in a specific ID order
func AssertIDsInOrder(t *testing.T, docs []nanostore.Document, expectedUUIDs []string) {
	t.Helper()
	if len(docs) != len(expectedUUIDs) {
		t.Errorf("expected %d documents, got %d", len(expectedUUIDs), len(docs))
		return
	}

	for i, expectedUUID := range expectedUUIDs {
		if i >= len(docs) {
			break
		}
		if docs[i].UUID != expectedUUID {
			t.Errorf("position %d: expected document %s, got %s",
				i, expectedUUID, docs[i].UUID)
		}
	}
}

// AssertQueryReturns verifies that a query returns exactly the expected documents
func AssertQueryReturns(t *testing.T, store nanostore.Store, opts nanostore.ListOptions, expectedUUIDs ...string) {
	t.Helper()
	results, err := store.List(opts)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != len(expectedUUIDs) {
		t.Errorf("expected %d results, got %d", len(expectedUUIDs), len(results))
	}

	// Create a map for easier lookup
	expectedMap := make(map[string]bool)
	for _, uuid := range expectedUUIDs {
		expectedMap[uuid] = true
	}

	// Check all results are expected
	for _, doc := range results {
		if !expectedMap[doc.UUID] {
			t.Errorf("unexpected document in results: %s", doc.UUID)
		}
		delete(expectedMap, doc.UUID)
	}

	// Check all expected were found
	for uuid := range expectedMap {
		t.Errorf("expected document not found: %s", uuid)
	}
}

// AssertQueryEmpty verifies that a query returns no results
func AssertQueryEmpty(t *testing.T, store nanostore.Store, opts nanostore.ListOptions) {
	t.Helper()
	results, err := store.List(opts)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected empty results, got %d documents", len(results))
		for i, doc := range results {
			t.Logf("  [%d] %s: %s", i, doc.UUID, doc.Title)
		}
	}
}

// AssertSearchFinds verifies that a search query returns the expected number of results
func AssertSearchFinds(t *testing.T, store nanostore.Store, query string, expectedCount int) {
	t.Helper()
	results, err := store.List(nanostore.ListOptions{
		FilterBySearch: query,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != expectedCount {
		t.Errorf("search for %q: expected %d results, got %d",
			query, expectedCount, len(results))
		for i, doc := range results {
			t.Logf("  [%d] %s: %s", i, doc.UUID, doc.Title)
		}
	}
}

// AssertContainsDocument verifies that at least one document in the slice matches a condition
func AssertContainsDocument(t *testing.T, docs []nanostore.Document, predicate func(nanostore.Document) bool, description string) {
	t.Helper()
	for _, doc := range docs {
		if predicate(doc) {
			return // Found it
		}
	}
	t.Errorf("no document matching %s found in %d results", description, len(docs))
}

// AssertAllDocuments verifies that all documents in the slice match a condition
func AssertAllDocuments(t *testing.T, docs []nanostore.Document, predicate func(nanostore.Document) bool, description string) {
	t.Helper()
	for _, doc := range docs {
		if !predicate(doc) {
			t.Errorf("document %s does not match %s", doc.UUID, description)
		}
	}
}

// AssertHierarchyDepth verifies the depth of a document in the hierarchy
// Depth is calculated by counting parent relationships
func AssertHierarchyDepth(t *testing.T, store nanostore.Store, doc nanostore.Document, expectedDepth int) {
	t.Helper()

	depth := 0
	current := doc

	for {
		parentID, hasParent := current.Dimensions["parent_id"]
		if !hasParent {
			break
		}

		depth++
		if depth > 10 { // Safety check for circular references
			t.Fatalf("circular reference detected or depth > 10 for document %s", doc.UUID)
		}

		// Get parent document
		parents, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": parentID,
			},
		})
		if err != nil || len(parents) != 1 {
			t.Fatalf("failed to find parent %v for document %s", parentID, current.UUID)
		}

		current = parents[0]
	}

	if depth != expectedDepth {
		t.Errorf("document %s: expected depth %d, got %d", doc.UUID, expectedDepth, depth)
	}
}
