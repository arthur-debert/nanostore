package api_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)
//
// This comprehensive test suite focuses on testing core functionality with
// many input combinations and edge cases as requested for Phase 4.

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore/api"
)

// createFreshStore creates a new store for testing with proper cleanup
func createFreshStore(t *testing.T) (*api.TypedStore[TodoItem], func()) {
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	_ = tmpfile.Close()

	store, err := api.NewFromType[TodoItem](tmpfile.Name())
	if err != nil {
		_ = os.Remove(tmpfile.Name())
		t.Fatal(err)
	}

	cleanup := func() {
		_ = store.Close()
		_ = os.Remove(tmpfile.Name())
	}

	return store, cleanup
}

func TestCoreOperationsWithManyInputs(t *testing.T) {
	// Define comprehensive test data combinations
	statusValues := []string{"pending", "active", "done"}
	priorityValues := []string{"low", "medium", "high"}
	activityValues := []string{"active", "archived", "deleted"}

	// Test titles with various characteristics
	testTitles := []string{
		"Simple Task",
		"Task with Special Characters: !@#$%^&*()",
		"Very Long Task Title That Tests Boundary Conditions And Edge Cases In The System",
		"Unicode Task: 日本語 中文 العربية Русский",
		"Task\nwith\nnewlines",
		"Task\twith\ttabs",
		"",                        // Empty title
		strings.Repeat("x", 1000), // Very long title
	}

	store, cleanup := createFreshStore(t)
	defer cleanup()

	t.Run("CreateWithAllDimensionCombinations", func(t *testing.T) {
		documentCount := 0

		// Test all combinations of valid dimension values
		for _, status := range statusValues {
			for _, priority := range priorityValues {
				for _, activity := range activityValues {
					for i, title := range testTitles {
						taskTitle := fmt.Sprintf("%s [%d]", title, documentCount)

						uuid, err := store.Create(taskTitle, &TodoItem{
							Status:   status,
							Priority: priority,
							Activity: activity,
						})

						if err != nil {
							t.Errorf("failed to create document with combination %s/%s/%s: %v",
								status, priority, activity, err)
							continue
						}

						documentCount++

						// Verify the document was created correctly
						doc, err := store.Get(uuid)
						if err != nil {
							t.Errorf("failed to retrieve created document %s: %v", uuid, err)
							continue
						}

						if doc.Status != status || doc.Priority != priority || doc.Activity != activity {
							t.Errorf("document dimensions mismatch: expected %s/%s/%s, got %s/%s/%s",
								status, priority, activity, doc.Status, doc.Priority, doc.Activity)
						}

						if i < len(testTitles)-2 { // Skip validation for empty and very long titles
							if !strings.Contains(doc.Title, title) {
								t.Errorf("document title mismatch: expected to contain %q, got %q",
									title, doc.Title)
							}
						}
					}
				}
			}
		}

		t.Logf("Successfully created and verified %d documents with all dimension combinations", documentCount)

		// Verify total count
		allDocs, err := store.Query().Find()
		if err != nil {
			t.Fatalf("failed to query all documents: %v", err)
		}

		if len(allDocs) != documentCount {
			t.Errorf("expected %d total documents, found %d", documentCount, len(allDocs))
		}
	})
}

func TestQueryOperationsWithComplexFiltering(t *testing.T) {
	store, cleanup := createFreshStore(t)
	defer cleanup()

	// Create test data with known patterns
	testData := []struct {
		title    string
		status   string
		priority string
		activity string
		data     map[string]interface{}
	}{
		{"High Priority Active", "active", "high", "active", map[string]interface{}{"assignee": "alice", "estimate": 8}},
		{"High Priority Pending", "pending", "high", "active", map[string]interface{}{"assignee": "bob", "estimate": 5}},
		{"Medium Priority Done", "done", "medium", "archived", map[string]interface{}{"assignee": "alice", "estimate": 3}},
		{"Low Priority Active", "active", "low", "active", map[string]interface{}{"assignee": "charlie", "estimate": 2}},
		{"Medium Priority Active", "active", "medium", "active", map[string]interface{}{"assignee": "alice", "estimate": 6}},
		{"Low Priority Deleted", "done", "low", "deleted", map[string]interface{}{"assignee": "bob", "estimate": 1}},
		{"High Priority Archived", "done", "high", "archived", map[string]interface{}{"assignee": "charlie", "estimate": 10}},
		{"Pending Medium Task", "pending", "medium", "active", map[string]interface{}{"assignee": "alice", "estimate": 4}},
	}

	// Create all test documents
	for _, data := range testData {
		_, err := store.AddRaw(data.title, map[string]interface{}{
			"status":         data.status,
			"priority":       data.priority,
			"activity":       data.activity,
			"_data.Assignee": data.data["assignee"],
			"_data.Estimate": data.data["estimate"],
		})
		if err != nil {
			t.Fatalf("failed to create test document %q: %v", data.title, err)
		}
	}

	t.Run("SingleDimensionFiltering", func(t *testing.T) {
		// Test each dimension individually
		activeTasks, err := store.Query().Status("active").Find()
		if err != nil {
			t.Fatalf("failed to query active tasks: %v", err)
		}

		expectedActiveCount := 3 // High Priority Active, Low Priority Active, Medium Priority Active
		if len(activeTasks) != expectedActiveCount {
			t.Errorf("expected %d active tasks, got %d", expectedActiveCount, len(activeTasks))
		}

		highPriorityTasks, err := store.Query().Priority("high").Find()
		if err != nil {
			t.Fatalf("failed to query high priority tasks: %v", err)
		}

		expectedHighCount := 3 // High Priority Active, High Priority Pending, High Priority Archived
		if len(highPriorityTasks) != expectedHighCount {
			t.Errorf("expected %d high priority tasks, got %d", expectedHighCount, len(highPriorityTasks))
		}

		archivedTasks, err := store.Query().Activity("archived").Find()
		if err != nil {
			t.Fatalf("failed to query archived tasks: %v", err)
		}

		expectedArchivedCount := 2 // Medium Priority Done, High Priority Archived
		if len(archivedTasks) != expectedArchivedCount {
			t.Errorf("expected %d archived tasks, got %d", expectedArchivedCount, len(archivedTasks))
		}
	})

	t.Run("MultiDimensionFiltering", func(t *testing.T) {
		// Test combinations of dimensions
		activeHighTasks, err := store.Query().Status("active").Priority("high").Find()
		if err != nil {
			t.Fatalf("failed to query active high priority tasks: %v", err)
		}

		if len(activeHighTasks) != 1 {
			t.Errorf("expected 1 active high priority task, got %d", len(activeHighTasks))
		}

		doneArchivedTasks, err := store.Query().Status("done").Activity("archived").Find()
		if err != nil {
			t.Fatalf("failed to query done archived tasks: %v", err)
		}

		if len(doneArchivedTasks) != 2 {
			t.Errorf("expected 2 done archived tasks, got %d", len(doneArchivedTasks))
		}

		activeMediumTasks, err := store.Query().Status("active").Priority("medium").Activity("active").Find()
		if err != nil {
			t.Fatalf("failed to query active medium priority active tasks: %v", err)
		}

		if len(activeMediumTasks) != 1 {
			t.Errorf("expected 1 active medium priority active task, got %d", len(activeMediumTasks))
		}
	})

	t.Run("ORFiltering", func(t *testing.T) {
		// Test IN operations (OR logic)
		highOrLowTasks, err := store.Query().PriorityIn("high", "low").Find()
		if err != nil {
			t.Fatalf("failed to query high or low priority tasks: %v", err)
		}

		expectedCount := 5 // 3 high + 2 low priority tasks
		if len(highOrLowTasks) != expectedCount {
			t.Errorf("expected %d high or low priority tasks, got %d", expectedCount, len(highOrLowTasks))
		}

		pendingOrDoneTasks, err := store.Query().StatusIn("pending", "done").Find()
		if err != nil {
			t.Fatalf("failed to query pending or done tasks: %v", err)
		}

		expectedPendingDoneCount := 5 // 2 pending + 3 done
		if len(pendingOrDoneTasks) != expectedPendingDoneCount {
			t.Errorf("expected %d pending or done tasks, got %d", expectedPendingDoneCount, len(pendingOrDoneTasks))
		}
	})

	t.Run("NOTFiltering", func(t *testing.T) {
		// Test NOT operations
		notHighTasks, err := store.Query().PriorityNot("high").Find()
		if err != nil {
			t.Fatalf("failed to query non-high priority tasks: %v", err)
		}

		expectedNotHighCount := 5 // Total 8 - 3 high priority
		if len(notHighTasks) != expectedNotHighCount {
			t.Errorf("expected %d non-high priority tasks, got %d", expectedNotHighCount, len(notHighTasks))
		}

		notActiveTasks, err := store.Query().StatusNot("active").Find()
		if err != nil {
			t.Fatalf("failed to query non-active tasks: %v", err)
		}

		expectedNotActiveCount := 5 // Total 8 - 3 active
		if len(notActiveTasks) != expectedNotActiveCount {
			t.Errorf("expected %d non-active tasks, got %d", expectedNotActiveCount, len(notActiveTasks))
		}

		notDeletedTasks, err := store.Query().ActivityNot("deleted").Find()
		if err != nil {
			t.Fatalf("failed to query non-deleted tasks: %v", err)
		}

		expectedNotDeletedCount := 7 // Total 8 - 1 deleted
		if len(notDeletedTasks) != expectedNotDeletedCount {
			t.Errorf("expected %d non-deleted tasks, got %d", expectedNotDeletedCount, len(notDeletedTasks))
		}
	})

	t.Run("DataFieldFiltering", func(t *testing.T) {
		// Test custom data field filtering
		aliceTasks, err := store.Query().Data("Assignee", "alice").Find()
		if err != nil {
			t.Fatalf("failed to query Alice's tasks: %v", err)
		}

		expectedAliceCount := 4 // Alice appears in 4 tasks
		if len(aliceTasks) != expectedAliceCount {
			t.Errorf("expected %d tasks assigned to Alice, got %d", expectedAliceCount, len(aliceTasks))
		}

		lowEstimateTasks, err := store.Query().DataIn("Estimate", 1, 2, 3).Find()
		if err != nil {
			t.Fatalf("failed to query low estimate tasks: %v", err)
		}

		expectedLowEstimateCount := 3 // estimates 1, 2, 3
		if len(lowEstimateTasks) != expectedLowEstimateCount {
			t.Errorf("expected %d low estimate tasks, got %d", expectedLowEstimateCount, len(lowEstimateTasks))
		}
	})

	t.Run("ComplexCombinedFiltering", func(t *testing.T) {
		// Test complex combinations
		complexQuery, err := store.Query().
			StatusIn("active", "pending").
			PriorityNot("low").
			Activity("active").
			Data("Assignee", "alice").
			Find()
		if err != nil {
			t.Fatalf("failed to execute complex query: %v", err)
		}

		// Let's analyze what we should get:
		// Alice's tasks: "High Priority Active", "Medium Priority Done", "Medium Priority Active", "Pending Medium Task"
		// Filter: StatusIn("active", "pending") -> "High Priority Active", "Medium Priority Active", "Pending Medium Task"
		// Filter: PriorityNot("low") -> same (none of Alice's active/pending tasks are low priority)
		// Filter: Activity("active") -> "High Priority Active", "Medium Priority Active", "Pending Medium Task"
		// So we should get 3 tasks
		expectedComplexCount := 3
		if len(complexQuery) != expectedComplexCount {
			t.Errorf("expected %d tasks from complex query, got %d", expectedComplexCount, len(complexQuery))
			for i, task := range complexQuery {
				t.Logf("Result %d: %s (status: %s, priority: %s, activity: %s)",
					i, task.Title, task.Status, task.Priority, task.Activity)
			}
		}
	})
}

func TestOrderingWithMultipleCriteria(t *testing.T) {
	store, cleanup := createFreshStore(t)
	defer cleanup()

	// Create documents with controlled timestamps for ordering tests
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	testDocs := []struct {
		title    string
		status   string
		priority string
		offset   time.Duration
		estimate int
	}{
		{"Task A", "active", "high", 0 * time.Hour, 5},
		{"Task B", "pending", "medium", 1 * time.Hour, 3},
		{"Task C", "done", "low", 2 * time.Hour, 8},
		{"Task D", "active", "high", 3 * time.Hour, 2},
		{"Task E", "pending", "low", 4 * time.Hour, 6},
	}

	// Create documents with specific timestamps
	for _, doc := range testDocs {
		// Set specific time for this document
		docTime := baseTime.Add(doc.offset)
		err := store.SetTimeFunc(func() time.Time { return docTime })
		if err != nil {
			t.Fatalf("failed to set time function: %v", err)
		}

		_, err = store.AddRaw(doc.title, map[string]interface{}{
			"status":         doc.status,
			"priority":       doc.priority,
			"activity":       "active",
			"_data.Estimate": doc.estimate,
		})
		if err != nil {
			t.Fatalf("failed to create document %s: %v", doc.title, err)
		}
	}

	t.Run("SingleFieldOrdering", func(t *testing.T) {
		// Test ordering by creation time (ascending)
		chronologicalTasks, err := store.Query().OrderBy("created_at").Find()
		if err != nil {
			t.Fatalf("failed to query chronologically ordered tasks: %v", err)
		}

		expectedOrder := []string{"Task A", "Task B", "Task C", "Task D", "Task E"}
		for i, task := range chronologicalTasks {
			if task.Title != expectedOrder[i] {
				t.Errorf("chronological order position %d: expected %s, got %s",
					i, expectedOrder[i], task.Title)
			}
		}

		// Test ordering by creation time (descending)
		reverseChronologicalTasks, err := store.Query().OrderByDesc("created_at").Find()
		if err != nil {
			t.Fatalf("failed to query reverse chronologically ordered tasks: %v", err)
		}

		expectedReverseOrder := []string{"Task E", "Task D", "Task C", "Task B", "Task A"}
		for i, task := range reverseChronologicalTasks {
			if task.Title != expectedReverseOrder[i] {
				t.Errorf("reverse chronological order position %d: expected %s, got %s",
					i, expectedReverseOrder[i], task.Title)
			}
		}
	})

	t.Run("MultipleFieldOrdering", func(t *testing.T) {
		// Test ordering by priority then by creation time
		priorityThenTimeTasks, err := store.Query().
			OrderBy("priority").
			OrderBy("created_at").
			Find()
		if err != nil {
			t.Fatalf("failed to query priority+time ordered tasks: %v", err)
		}

		// Priority order: high, low, medium (alphabetical within nanostore)
		// Within each priority, chronological order
		t.Logf("Priority+Time ordered tasks:")
		for i, task := range priorityThenTimeTasks {
			t.Logf("  %d: %s (priority: %s)", i, task.Title, task.Priority)
		}
	})

	t.Run("DataFieldOrdering", func(t *testing.T) {
		// Test ordering by custom data fields
		estimateOrderedTasks, err := store.Query().OrderByData("Estimate").Find()
		if err != nil {
			t.Fatalf("failed to query estimate ordered tasks: %v", err)
		}

		// Should be ordered by estimate: 2, 3, 5, 6, 8
		expectedEstimateOrder := []string{"Task D", "Task B", "Task A", "Task E", "Task C"}
		for i, task := range estimateOrderedTasks {
			if task.Title != expectedEstimateOrder[i] {
				t.Errorf("estimate order position %d: expected %s, got %s",
					i, expectedEstimateOrder[i], task.Title)
			}
		}

		// Test descending order
		estimateDescTasks, err := store.Query().OrderByDataDesc("Estimate").Find()
		if err != nil {
			t.Fatalf("failed to query estimate desc ordered tasks: %v", err)
		}

		// Should be ordered by estimate desc: 8, 6, 5, 3, 2
		expectedEstimateDescOrder := []string{"Task C", "Task E", "Task A", "Task B", "Task D"}
		for i, task := range estimateDescTasks {
			if task.Title != expectedEstimateDescOrder[i] {
				t.Errorf("estimate desc order position %d: expected %s, got %s",
					i, expectedEstimateDescOrder[i], task.Title)
			}
		}
	})
}

func TestPaginationWithLargeDatasets(t *testing.T) {
	store, cleanup := createFreshStore(t)
	defer cleanup()

	// Create a larger dataset for pagination testing
	const totalDocs = 50

	for i := 0; i < totalDocs; i++ {
		title := fmt.Sprintf("Task %03d", i)
		status := []string{"pending", "active", "done"}[i%3]
		priority := []string{"low", "medium", "high"}[i%3]

		_, err := store.Create(title, &TodoItem{
			Status:   status,
			Priority: priority,
			Activity: "active",
		})
		if err != nil {
			t.Fatalf("failed to create task %d: %v", i, err)
		}
	}

	t.Run("BasicPagination", func(t *testing.T) {
		// Test different page sizes
		pageSizes := []int{5, 10, 15, 20}

		for _, pageSize := range pageSizes {
			t.Run(fmt.Sprintf("PageSize%d", pageSize), func(t *testing.T) {
				var allPaginatedTasks []TodoItem
				offset := 0

				for {
					page, err := store.Query().
						OrderBy("title"). // Ensure consistent ordering
						Limit(pageSize).
						Offset(offset).
						Find()
					if err != nil {
						t.Fatalf("failed to get page at offset %d: %v", offset, err)
					}

					if len(page) == 0 {
						break
					}

					allPaginatedTasks = append(allPaginatedTasks, page...)
					offset += pageSize

					// Prevent infinite loop
					if offset > totalDocs+pageSize {
						break
					}
				}

				if len(allPaginatedTasks) != totalDocs {
					t.Errorf("pagination with page size %d: expected %d total tasks, got %d",
						pageSize, totalDocs, len(allPaginatedTasks))
				}

				// Verify ordering consistency
				for i := 1; i < len(allPaginatedTasks); i++ {
					if allPaginatedTasks[i].Title <= allPaginatedTasks[i-1].Title {
						t.Errorf("pagination ordering inconsistency at position %d: %s <= %s",
							i, allPaginatedTasks[i].Title, allPaginatedTasks[i-1].Title)
					}
				}
			})
		}
	})

	t.Run("PaginationWithFilters", func(t *testing.T) {
		// Test pagination combined with filtering
		activeTasks, err := store.Query().Status("active").OrderBy("title").Find()
		if err != nil {
			t.Fatalf("failed to get all active tasks: %v", err)
		}

		// Paginate through active tasks
		pageSize := 7
		var paginatedActive []TodoItem

		for offset := 0; offset < len(activeTasks); offset += pageSize {
			page, err := store.Query().
				Status("active").
				OrderBy("title").
				Limit(pageSize).
				Offset(offset).
				Find()
			if err != nil {
				t.Fatalf("failed to get active tasks page at offset %d: %v", offset, err)
			}

			paginatedActive = append(paginatedActive, page...)
		}

		if len(paginatedActive) != len(activeTasks) {
			t.Errorf("filtered pagination: expected %d active tasks, got %d",
				len(activeTasks), len(paginatedActive))
		}

		// Verify same tasks in same order
		for i, task := range activeTasks {
			if i >= len(paginatedActive) || task.UUID != paginatedActive[i].UUID {
				t.Errorf("filtered pagination mismatch at position %d", i)
			}
		}
	})
}

func TestEdgeCasesAndErrorConditions(t *testing.T) {
	store, cleanup := createFreshStore(t)
	defer cleanup()

	t.Run("EmptyStore", func(t *testing.T) {
		// Test operations on empty store
		allTasks, err := store.Query().Find()
		if err != nil {
			t.Fatalf("failed to query empty store: %v", err)
		}

		if len(allTasks) != 0 {
			t.Errorf("expected empty result from empty store, got %d tasks", len(allTasks))
		}

		// Test First() on empty store
		_, err = store.Query().First()
		if err == nil {
			t.Error("expected error when calling First() on empty store")
		}

		// Test Count() on empty store
		count, err := store.Query().Count()
		if err != nil {
			t.Fatalf("failed to count empty store: %v", err)
		}

		if count != 0 {
			t.Errorf("expected count 0 for empty store, got %d", count)
		}

		// Test Exists() on empty store
		exists, err := store.Query().Exists()
		if err != nil {
			t.Fatalf("failed to check existence in empty store: %v", err)
		}

		if exists {
			t.Error("expected false from Exists() on empty store")
		}
	})

	t.Run("InvalidFilters", func(t *testing.T) {
		// Create one document for testing
		_, err := store.Create("Test Task", &TodoItem{
			Status:   "active",
			Priority: "medium",
			Activity: "active",
		})
		if err != nil {
			t.Fatalf("failed to create test document: %v", err)
		}

		// Test filtering with non-existent values
		nonExistentStatus, err := store.Query().Status("nonexistent").Find()
		if err != nil {
			t.Fatalf("failed to query with non-existent status: %v", err)
		}

		if len(nonExistentStatus) != 0 {
			t.Errorf("expected no results for non-existent status, got %d", len(nonExistentStatus))
		}

		// Test empty IN filter
		emptyIn, err := store.Query().StatusIn().Find()
		if err != nil {
			t.Fatalf("failed to query with empty StatusIn: %v", err)
		}

		if len(emptyIn) != 0 {
			t.Errorf("expected no results for empty StatusIn, got %d", len(emptyIn))
		}
	})

	t.Run("LargeOffsets", func(t *testing.T) {
		// Test with offset larger than dataset
		largeOffset, err := store.Query().Offset(1000).Find()
		if err != nil {
			t.Fatalf("failed to query with large offset: %v", err)
		}

		if len(largeOffset) != 0 {
			t.Errorf("expected no results with large offset, got %d", len(largeOffset))
		}
	})

	t.Run("ZeroAndNegativeLimits", func(t *testing.T) {
		// Test with zero limit - behavior may vary by implementation
		zeroLimit, err := store.Query().Limit(0).Find()
		if err != nil {
			t.Fatalf("failed to query with zero limit: %v", err)
		}

		// Zero limit behavior is implementation-dependent
		// Some implementations return no results, others ignore the limit
		t.Logf("Query with zero limit returned %d results", len(zeroLimit))

		// Test with negative limit (should be handled gracefully)
		negativeLimit, err := store.Query().Limit(-1).Find()
		if err != nil {
			t.Fatalf("failed to query with negative limit: %v", err)
		}

		// Behavior with negative limit may vary, but shouldn't crash
		t.Logf("Query with negative limit returned %d results", len(negativeLimit))

		// The important thing is that these don't crash - the exact behavior may vary
	})
}
