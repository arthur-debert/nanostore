package nanostore

import (
	"testing"
)

func TestTypedQuery(t *testing.T) {
	// Define a test document type
	type ProjectDoc struct {
		Document
		Status   string `values:"planning,active,completed,archived" default:"planning"`
		Priority string `values:"low,medium,high,critical" default:"medium"`
		TeamID   string `dimension:"team_id,ref"`
	}

	// Helper to create a test store with sample data
	setupStore := func(t *testing.T) *TypedStore[ProjectDoc] {
		store, err := NewFromType[ProjectDoc](":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}

		// Create sample projects
		projects := []struct {
			title    string
			status   string
			priority string
			teamID   string
		}{
			{"Project Alpha", "active", "high", ""},
			{"Project Beta", "planning", "medium", ""},
			{"Project Gamma", "completed", "low", ""},
			{"Project Delta", "active", "critical", ""},
			{"Project Epsilon", "archived", "medium", ""},
			{"Project Zeta", "active", "high", ""},
		}

		teamUUID, _ := store.Create("Team A", &ProjectDoc{})

		for _, p := range projects {
			doc := &ProjectDoc{
				Status:   p.status,
				Priority: p.priority,
			}
			if p.title == "Project Beta" || p.title == "Project Gamma" {
				doc.TeamID = teamUUID
			}

			_, err := store.Create(p.title, doc)
			if err != nil {
				t.Fatalf("failed to create project %s: %v", p.title, err)
			}
		}

		return store
	}

	t.Run("Find", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		results, err := store.Query().Find()
		if err != nil {
			t.Fatalf("failed to find all: %v", err)
		}

		// Should have 7 documents (6 projects + 1 team)
		if len(results) != 7 {
			t.Errorf("expected 7 documents, got %d", len(results))
		}
	})

	t.Run("Status filter", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		results, err := store.Query().Status("active").Find()
		if err != nil {
			t.Fatalf("failed to find active projects: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("expected 3 active projects, got %d", len(results))
		}

		for _, r := range results {
			if r.Status != "active" {
				t.Errorf("expected status 'active', got %q", r.Status)
			}
		}
	})

	t.Run("StatusNot filter", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		results, err := store.Query().StatusNot("active").Find()
		if err != nil {
			t.Fatalf("failed to find non-active projects: %v", err)
		}

		// Should have 4 projects (planning, completed, archived) + 1 team
		if len(results) != 4 {
			t.Errorf("expected 4 non-active documents, got %d", len(results))
		}

		for _, r := range results {
			if r.Status == "active" {
				t.Errorf("expected status not to be 'active', but got 'active'")
			}
		}
	})

	t.Run("StatusIn filter", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		results, err := store.Query().StatusIn("active", "completed").Find()
		if err != nil {
			t.Fatalf("failed to find projects: %v", err)
		}

		if len(results) != 4 {
			t.Errorf("expected 4 projects, got %d", len(results))
		}

		for _, r := range results {
			if r.Status != "active" && r.Status != "completed" {
				t.Errorf("expected status to be 'active' or 'completed', got %q", r.Status)
			}
		}
	})

	t.Run("Multiple filters", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		results, err := store.Query().
			Status("active").
			Priority("high").
			Find()
		if err != nil {
			t.Fatalf("failed to find projects: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 high priority active projects, got %d", len(results))
		}

		for _, r := range results {
			if r.Status != "active" {
				t.Errorf("expected status 'active', got %q", r.Status)
			}
			if r.Priority != "high" {
				t.Errorf("expected priority 'high', got %q", r.Priority)
			}
		}
	})

	t.Run("Hierarchical filters", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		// Find projects with a team
		results, err := store.Query().Call("team_idExists").Find()
		if err != nil {
			t.Fatalf("failed to find projects with team: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 projects with team, got %d", len(results))
		}

		// Find projects without a team
		results, err = store.Query().Call("team_idNotExists").Find()
		if err != nil {
			t.Fatalf("failed to find projects without team: %v", err)
		}

		// Should be 4 projects + 1 team document
		if len(results) != 5 {
			t.Errorf("expected 5 documents without team, got %d", len(results))
		}
	})

	t.Run("First", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		result, err := store.Query().Status("active").First()
		if err != nil {
			t.Fatalf("failed to get first active project: %v", err)
		}

		if result.Status != "active" {
			t.Errorf("expected status 'active', got %q", result.Status)
		}
	})

	t.Run("First with no results", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		_, err := store.Query().Status("nonexistent").First()
		if err == nil {
			t.Error("expected error for no results, got nil")
		}
	})

	t.Run("Get", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		result, err := store.Query().
			Status("completed").
			Priority("low").
			Get()
		if err != nil {
			t.Fatalf("failed to get single result: %v", err)
		}

		if result.Title != "Project Gamma" {
			t.Errorf("expected 'Project Gamma', got %q", result.Title)
		}
	})

	t.Run("Get with multiple results", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		_, err := store.Query().Status("active").Get()
		if err == nil {
			t.Error("expected error for multiple results, got nil")
		}
		if err != nil && !contains(err.Error(), "expected exactly one") {
			t.Errorf("expected error about multiple results, got: %v", err)
		}
	})

	t.Run("Count", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		count, err := store.Query().Status("active").Count()
		if err != nil {
			t.Fatalf("failed to count: %v", err)
		}

		if count != 3 {
			t.Errorf("expected 3 active projects, got %d", count)
		}
	})

	t.Run("Exists", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		exists, err := store.Query().Status("archived").Exists()
		if err != nil {
			t.Fatalf("failed to check exists: %v", err)
		}

		if !exists {
			t.Error("expected archived projects to exist")
		}

		exists, err = store.Query().Status("cancelled").Exists()
		if err != nil {
			t.Fatalf("failed to check exists: %v", err)
		}

		if exists {
			t.Error("expected no cancelled projects")
		}
	})

	t.Run("Limit and Offset", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		// Get first 2 results
		results, err := store.Query().Limit(2).Find()
		if err != nil {
			t.Fatalf("failed to find with limit: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 results with limit, got %d", len(results))
		}

		// Skip first 3 and get next 2
		results, err = store.Query().Offset(3).Limit(2).Find()
		if err != nil {
			t.Fatalf("failed to find with offset and limit: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 results with offset and limit, got %d", len(results))
		}
	})

	t.Run("Chaining", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		results, err := store.Query().
			StatusIn("active", "planning").
			PriorityIn("high", "critical").
			Limit(10).
			Find()

		if err != nil {
			t.Fatalf("failed to find with chained filters: %v", err)
		}

		// Should find Project Alpha (active, high), Project Delta (active, critical), and Project Zeta (active, high)
		if len(results) != 3 {
			t.Errorf("expected 3 results, got %d", len(results))
		}
	})

	t.Run("WithFilter", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		results, err := store.Query().
			WithFilter("status", "active").
			WithFilter("priority", "high").
			Find()

		if err != nil {
			t.Fatalf("failed to find with filters: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
	})

	t.Run("Dynamic method calls", func(t *testing.T) {
		store := setupStore(t)
		defer func() { _ = store.Close() }()

		// Test using Call method directly
		results, err := store.Query().
			Call("status", "active").
			Call("priority", "high").
			Find()

		if err != nil {
			t.Fatalf("failed to find with dynamic calls: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}
