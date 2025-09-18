package search

import (
	"errors"
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/types"
)

func TestEngine_Search_EmptyQuery(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	results, err := engine.Search(SearchOptions{Query: ""}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty query, got %d", len(results))
	}
}

func TestEngine_Search_ProviderError(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	provider.SetError(errors.New("database error"))
	engine := NewEngine(provider)

	_, err := engine.Search(SearchOptions{Query: "test"}, nil)

	if err == nil {
		t.Error("Expected error when provider fails")
	}
	if !strings.Contains(err.Error(), "failed to get documents") {
		t.Errorf("Expected error to mention document retrieval, got: %v", err)
	}
}

func TestEngine_Search_CaseSensitive(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	// Case sensitive search should not match different case
	results, err := engine.Search(SearchOptions{
		Query:         "meeting",
		CaseSensitive: true,
		Fields:        []string{"title", "body"},
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 results for case-sensitive 'meeting', got %d", len(results))
	}

	// Should find "Budget Review" (body), "Team Standup" (body), and "MEETING" (body), but not "Important Meeting" (title has capital M)
	foundTitles := make(map[string]bool)
	for _, result := range results {
		foundTitles[result.Document.Title] = true
	}

	if foundTitles["Important Meeting"] {
		t.Error("Should not find 'Important Meeting' (title has capital M)")
	}
	if !foundTitles["Budget Review"] {
		t.Error("Expected to find 'Budget Review' (body match)")
	}
	if !foundTitles["Team Standup"] {
		t.Error("Expected to find 'Team Standup' (body match)")
	}
	if !foundTitles["MEETING"] {
		t.Error("Expected to find 'MEETING' (body contains lowercase 'meeting')")
	}
}

func TestEngine_Search_CaseInsensitive(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	results, err := engine.Search(SearchOptions{
		Query:         "MEETING",
		CaseSensitive: false,
		Fields:        []string{"title", "body"},
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 4 {
		t.Errorf("Expected 4 results for case-insensitive 'MEETING', got %d", len(results))
	}
}

func TestEngine_Search_ExactMatch(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	results, err := engine.Search(SearchOptions{
		Query:      "Important Meeting",
		ExactMatch: true,
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for exact match 'Important Meeting', got %d", len(results))
	}
	if len(results) > 0 && results[0].Document.Title != "Important Meeting" {
		t.Errorf("Expected exact match for 'Important Meeting', got %s", results[0].Document.Title)
	}
}

func TestEngine_Search_PartialMatch(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	results, err := engine.Search(SearchOptions{
		Query:      "budget",
		ExactMatch: false,
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results for partial match 'budget', got %d", len(results))
	}

	// Should find "Budget Review" (title) and "Important Meeting" (body)
	foundTitles := make(map[string]bool)
	for _, result := range results {
		foundTitles[result.Document.Title] = true
	}

	if !foundTitles["Budget Review"] {
		t.Error("Expected to find 'Budget Review'")
	}
	if !foundTitles["Important Meeting"] {
		t.Error("Expected to find 'Important Meeting' (body contains 'budget')")
	}
}

func TestEngine_Search_FieldSpecific(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	// Search only in title field
	results, err := engine.Search(SearchOptions{
		Query:  "meeting",
		Fields: []string{"title"},
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results for title-only search 'meeting', got %d", len(results))
	}

	// Should find "Important Meeting" and "MEETING", but not "Team Standup" (meeting only in body)
	foundTitles := make(map[string]bool)
	for _, result := range results {
		foundTitles[result.Document.Title] = true
	}

	if !foundTitles["Important Meeting"] {
		t.Error("Expected to find 'Important Meeting'")
	}
	if !foundTitles["MEETING"] {
		t.Error("Expected to find 'MEETING'")
	}
	if foundTitles["Team Standup"] {
		t.Error("Should not find 'Team Standup' when searching title only")
	}
}

func TestEngine_Search_DimensionFields(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	// Search in status dimension
	results, err := engine.Search(SearchOptions{
		Query:  "pending",
		Fields: []string{"status"},
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results for status 'pending', got %d", len(results))
	}

	for _, result := range results {
		if result.MatchType != MatchDimension {
			t.Errorf("Expected MatchDimension, got %s", result.MatchType)
		}
		if !contains(result.MatchedFields, "status") {
			t.Error("Expected status in matched fields")
		}
	}
}

func TestEngine_Search_CustomDataFields(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	// Search in custom data field
	results, err := engine.Search(SearchOptions{
		Query:  "alice",
		Fields: []string{"_data.assigned_to"},
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results for assigned_to 'alice', got %d", len(results))
	}

	for _, result := range results {
		if result.MatchType != MatchCustomData {
			t.Errorf("Expected MatchCustomData, got %s", result.MatchType)
		}
	}
}

func TestEngine_Search_WithHighlights(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	results, err := engine.Search(SearchOptions{
		Query:           "Meeting",
		EnableHighlight: true,
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// Check that highlights are present
	found := false
	for _, result := range results {
		if len(result.Highlights) > 0 {
			found = true
			// Check that highlights contain markers
			for field, highlight := range result.Highlights {
				if strings.Contains(highlight, "**") {
					t.Logf("Found highlight in %s: %s", field, highlight)
				}
			}
		}
	}

	if !found {
		t.Error("Expected highlights to be present when EnableHighlight is true")
	}
}

func TestEngine_Search_WithoutHighlights(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	results, err := engine.Search(SearchOptions{
		Query:           "Meeting",
		EnableHighlight: false,
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// Check that highlights are not present
	for _, result := range results {
		if len(result.Highlights) > 0 {
			t.Error("Expected no highlights when EnableHighlight is false")
		}
	}
}

func TestEngine_Search_MaxResults(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	maxResults := 1
	results, err := engine.Search(SearchOptions{
		Query:      "e", // Should match multiple documents
		MaxResults: &maxResults,
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result with MaxResults=1, got %d", len(results))
	}
}

func TestEngine_Search_Scoring(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	results, err := engine.Search(SearchOptions{
		Query: "meeting",
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) < 2 {
		t.Fatal("Expected at least 2 results for scoring test")
	}

	// Results should be sorted by score (highest first)
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("Results not properly sorted by score: %f > %f",
				results[i].Score, results[i-1].Score)
		}
	}

	// Title matches should generally score higher than body matches
	var titleMatch, bodyMatch *SearchResult
	for i := range results {
		switch results[i].MatchType {
		case MatchPartialTitle:
			titleMatch = &results[i]
		case MatchPartialBody:
			bodyMatch = &results[i]
		}
	}

	if titleMatch != nil && bodyMatch != nil {
		if titleMatch.Score <= bodyMatch.Score {
			t.Errorf("Expected title match score (%f) > body match score (%f)",
				titleMatch.Score, bodyMatch.Score)
		}
	}
}

func TestEngine_Search_MatchTypes(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	// Test exact title match
	results, err := engine.Search(SearchOptions{
		Query:      "Important Meeting",
		ExactMatch: true,
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) > 0 && results[0].MatchType != MatchExactTitle {
		t.Errorf("Expected MatchExactTitle, got %s", results[0].MatchType)
	}

	// Test partial title match
	results, err = engine.Search(SearchOptions{
		Query:      "Meeting",
		ExactMatch: false,
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	foundPartialTitle := false
	for _, result := range results {
		if result.MatchType == MatchPartialTitle {
			foundPartialTitle = true
			break
		}
	}
	if !foundPartialTitle {
		t.Error("Expected to find MatchPartialTitle")
	}
}

func TestEngine_Search_CustomHighlightMarkers(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	results, err := engine.Search(SearchOptions{
		Query:                "Meeting",
		EnableHighlight:      true,
		HighlightStartMarker: "<mark>",
		HighlightEndMarker:   "</mark>",
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// Check for custom markers in highlights
	found := false
	for _, result := range results {
		for field, highlight := range result.Highlights {
			if strings.Contains(highlight, "<mark>") && strings.Contains(highlight, "</mark>") {
				t.Logf("Found custom highlight in %s: %s", field, highlight)
				found = true
			}
		}
	}

	if !found {
		t.Error("Expected custom highlight markers to be present")
	}
}

func TestEngine_Search_StructuredMatchData(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	results, err := engine.Search(SearchOptions{
		Query:               "meeting",
		IncludeMatchDetails: true,
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// Check that FieldMatches are populated
	result := results[0]
	if len(result.FieldMatches) == 0 {
		t.Error("Expected FieldMatches to be populated when IncludeMatchDetails is true")
	}

	// Verify match details structure
	for _, fieldMatch := range result.FieldMatches {
		if fieldMatch.FieldName == "" {
			t.Error("Expected FieldName to be set")
		}
		if fieldMatch.OriginalText == "" {
			t.Error("Expected OriginalText to be set")
		}
		if len(fieldMatch.Matches) == 0 {
			t.Error("Expected at least one match in Matches slice")
		}
		if fieldMatch.FieldScore <= 0 {
			t.Error("Expected FieldScore to be greater than 0")
		}

		// Verify individual match info
		for _, match := range fieldMatch.Matches {
			if match.Start < 0 || match.End <= match.Start {
				t.Errorf("Invalid match positions: Start=%d, End=%d", match.Start, match.End)
			}
			if match.Text == "" {
				t.Error("Expected match Text to be set")
			}
			if match.Score <= 0 {
				t.Error("Expected match Score to be greater than 0")
			}
			if match.MatchType == "" {
				t.Error("Expected MatchType to be set")
			}

			// Verify match text is correct
			expectedText := fieldMatch.OriginalText[match.Start:match.End]
			if !strings.EqualFold(match.Text, expectedText) {
				t.Errorf("Match text mismatch: got %q, expected %q", match.Text, expectedText)
			}
		}
	}
}

func TestEngine_Search_MatchPositions(t *testing.T) {
	provider := NewMockDocumentProvider([]types.Document{
		{
			UUID:     "test",
			SimpleID: "1",
			Title:    "test word test",
			Body:     "another test here",
		},
	})
	engine := NewEngine(provider)

	results, err := engine.Search(SearchOptions{
		Query:               "test",
		IncludeMatchDetails: true,
		Fields:              []string{"title"},
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	fieldMatch := results[0].FieldMatches[0]
	if len(fieldMatch.Matches) != 2 {
		t.Fatalf("Expected 2 matches in title 'test word test', got %d", len(fieldMatch.Matches))
	}

	// Verify first match
	match1 := fieldMatch.Matches[0]
	if match1.Start != 0 || match1.End != 4 {
		t.Errorf("First match: expected positions 0-4, got %d-%d", match1.Start, match1.End)
	}
	if match1.Text != "test" {
		t.Errorf("First match: expected text 'test', got %q", match1.Text)
	}

	// Verify second match
	match2 := fieldMatch.Matches[1]
	if match2.Start != 10 || match2.End != 14 {
		t.Errorf("Second match: expected positions 10-14, got %d-%d", match2.Start, match2.End)
	}
	if match2.Text != "test" {
		t.Errorf("Second match: expected text 'test', got %q", match2.Text)
	}
}

func TestEngine_Search_HighlightingMultipleMatches(t *testing.T) {
	provider := NewMockDocumentProvider([]types.Document{
		{
			UUID:     "test",
			SimpleID: "1",
			Title:    "test word test word test",
			Body:     "test test test",
		},
	})
	engine := NewEngine(provider)

	results, err := engine.Search(SearchOptions{
		Query:           "test",
		EnableHighlight: true,
		Fields:          []string{"title", "body"},
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]

	// Check title highlighting (should have 3 highlighted "test" occurrences)
	titleHighlight := result.Highlights["title"]
	expectedTitle := "**test** word **test** word **test**"
	if titleHighlight != expectedTitle {
		t.Errorf("Title highlight mismatch.\nExpected: %q\nGot:      %q", expectedTitle, titleHighlight)
	}

	// Check body highlighting (should have 3 highlighted "test" occurrences)
	bodyHighlight := result.Highlights["body"]
	expectedBody := "**test** **test** **test**"
	if bodyHighlight != expectedBody {
		t.Errorf("Body highlight mismatch.\nExpected: %q\nGot:      %q", expectedBody, bodyHighlight)
	}
}

func TestEngine_Search_BackwardCompatibility(t *testing.T) {
	provider := NewMockDocumentProvider(SampleDocuments())
	engine := NewEngine(provider)

	// Test that old API still works without new options
	results, err := engine.Search(SearchOptions{
		Query:           "Meeting",
		EnableHighlight: true,
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// Should still use default ** markers
	result := results[0]
	if len(result.Highlights) == 0 {
		t.Error("Expected legacy Highlights to be populated")
	}

	for _, highlight := range result.Highlights {
		if strings.Contains(highlight, "**") {
			// Good - default markers are used
			break
		}
	}

	// FieldMatches should be empty when IncludeMatchDetails is false
	if len(result.FieldMatches) != 0 {
		t.Error("Expected FieldMatches to be empty when IncludeMatchDetails is false")
	}
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
