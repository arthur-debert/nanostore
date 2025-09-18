package nanostore_test

import (
	"os"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestIDGeneration(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"todo", "done"},
				DefaultValue: "todo",
				Prefixes: map[string]string{
					"todo": "t",
					"done": "d",
				},
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Get test store for time control
	testStore := nanostore.AsTestStore(store)
	if testStore == nil {
		t.Fatal("store doesn't support testing features")
	}

	// Set deterministic time
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	currentTime := baseTime
	testStore.SetTimeFunc(func() time.Time {
		t := currentTime
		currentTime = currentTime.Add(1 * time.Hour)
		return t
	})

	t.Run("BasicIDGeneration", func(t *testing.T) {
		// Add todo items - should get t1, t2, t3
		id1, _ := store.Add("Task 1", map[string]interface{}{"status": "todo"})
		id2, _ := store.Add("Task 2", map[string]interface{}{"status": "todo"})
		id3, _ := store.Add("Task 3", map[string]interface{}{"status": "todo"})

		// Add done items - should get d1, d2
		id4, _ := store.Add("Done 1", map[string]interface{}{"status": "done"})
		id5, _ := store.Add("Done 2", map[string]interface{}{"status": "done"})

		// List all and check IDs
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Create a map for easier checking
		idMap := make(map[string]string)
		for _, doc := range docs {
			idMap[doc.UUID] = doc.SimpleID
		}

		// Check the IDs
		expectedIDs := map[string]string{
			id1: "1",  // First in todo partition (canonical)
			id2: "2",  // Second in todo partition
			id3: "3",  // Third in todo partition
			id4: "d1", // First in done partition
			id5: "d2", // Second in done partition
		}

		for uuid, expectedID := range expectedIDs {
			if actualID, exists := idMap[uuid]; !exists {
				t.Errorf("UUID %s not found in results", uuid)
			} else if actualID != expectedID {
				t.Errorf("UUID %s: expected ID %s, got %s", uuid, expectedID, actualID)
			}
		}
	})

	t.Run("IDResolveUUID", func(t *testing.T) {
		// Test resolving simple IDs back to UUIDs
		testCases := []string{"1", "2", "3", "d1", "d2"}

		for _, simpleID := range testCases {
			uuid, err := store.ResolveUUID(simpleID)
			if err != nil {
				t.Errorf("failed to resolve ID %s: %v", simpleID, err)
			} else if uuid == "" {
				t.Errorf("resolved empty UUID for ID %s", simpleID)
			}
		}

		// Test non-existent ID
		_, err := store.ResolveUUID("nonexistent")
		if err == nil {
			t.Error("expected error for non-existent ID")
		}
	})
}

func TestHierarchicalIDGeneration(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("HierarchicalIDs", func(t *testing.T) {
		// Add root items
		root1, _ := store.Add("Root 1", nil)
		root2, _ := store.Add("Root 2", nil)

		// Add children of root1
		child1, _ := store.Add("Child 1.1", map[string]interface{}{"parent_uuid": root1})
		child2, _ := store.Add("Child 1.2", map[string]interface{}{"parent_uuid": root1})

		// Add children of root2
		child3, _ := store.Add("Child 2.1", map[string]interface{}{"parent_uuid": root2})

		// Add grandchild
		grandchild, _ := store.Add("Grandchild 1.1.1", map[string]interface{}{"parent_uuid": child1})

		// List all and check IDs
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Create a map for easier checking
		idMap := make(map[string]string)
		for _, doc := range docs {
			idMap[doc.UUID] = doc.SimpleID
			t.Logf("%s -> %s", doc.Title, doc.SimpleID)
		}

		// Check the hierarchical IDs
		expectedIDs := map[string]string{
			root1:      "1",
			root2:      "2",
			child1:     "1.1",
			child2:     "1.2",
			child3:     "2.1",
			grandchild: "1.1.1",
		}

		for uuid, expectedID := range expectedIDs {
			if actualID, exists := idMap[uuid]; !exists {
				t.Errorf("UUID %s not found in results", uuid)
			} else if actualID != expectedID {
				t.Errorf("UUID %s: expected ID %s, got %s", uuid, expectedID, actualID)
			}
		}
	})
}

func TestIDGenerationWithMultipleDimensions(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "type",
				Type:         nanostore.Enumerated,
				Values:       []string{"bug", "feature"},
				DefaultValue: "bug",
				Prefixes: map[string]string{
					"bug":     "b",
					"feature": "f",
				},
			},
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "high"},
				DefaultValue: "low",
				Prefixes: map[string]string{
					"low":  "l",
					"high": "h",
				},
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("MultiplePrefixes", func(t *testing.T) {
		// Add items with different combinations
		_, _ = store.Add("Bug Low 1", map[string]interface{}{
			"type":     "bug",
			"priority": "low",
		})
		_, _ = store.Add("Bug Low 2", map[string]interface{}{
			"type":     "bug",
			"priority": "low",
		})
		_, _ = store.Add("Bug High 1", map[string]interface{}{
			"type":     "bug",
			"priority": "high",
		})
		_, _ = store.Add("Feature High 1", map[string]interface{}{
			"type":     "feature",
			"priority": "high",
		})

		// List all and check IDs
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Log all IDs to see the pattern
		for _, doc := range docs {
			t.Logf("%s (%s, %s) -> %s", 
				doc.Title, 
				doc.Dimensions["type"], 
				doc.Dimensions["priority"], 
				doc.SimpleID)
		}

		// Count occurrences of each ID pattern
		idPatterns := make(map[string]int)
		for _, doc := range docs {
			// Extract the prefix part (everything before the number)
			id := doc.SimpleID
			var prefix string
			for i, ch := range id {
				if ch >= '0' && ch <= '9' {
					prefix = id[:i]
					break
				}
			}
			idPatterns[prefix]++
		}

		// We should have different prefixes for different combinations
		// With partition-based IDs:
		// - Canonical (bug+low) gets no prefix: "1", "2"
		// - Only priority differs: "h1" (high priority)
		// - Both differ: "fh1" (feature+high)
		expectedPatterns := map[string]int{
			"": 2,   // bug + low (canonical)
			"h": 1,  // bug + high
			"fh": 1, // feature + high
		}

		for pattern, count := range expectedPatterns {
			if actualCount, exists := idPatterns[pattern]; !exists {
				t.Errorf("expected pattern %s not found", pattern)
			} else if actualCount != count {
				t.Errorf("pattern %s: expected count %d, got %d", pattern, count, actualCount)
			}
		}
	})
}