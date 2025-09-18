package search

import (
	"time"

	"github.com/arthur-debert/nanostore/types"
)

// MockDocumentProvider implements DocumentProvider for testing
type MockDocumentProvider struct {
	documents []types.Document
	err       error
}

// NewMockDocumentProvider creates a new mock with the given documents
func NewMockDocumentProvider(documents []types.Document) *MockDocumentProvider {
	return &MockDocumentProvider{
		documents: documents,
	}
}

// SetError configures the mock to return an error
func (m *MockDocumentProvider) SetError(err error) {
	m.err = err
}

// GetDocuments returns the mock documents or error
func (m *MockDocumentProvider) GetDocuments(filters map[string]interface{}) ([]types.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.documents, nil
}

// SampleDocuments provides sample documents for testing
func SampleDocuments() []types.Document {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	return []types.Document{
		{
			UUID:     "1",
			SimpleID: "1",
			Title:    "Important Meeting",
			Body:     "Discuss quarterly budget and planning",
			Dimensions: map[string]interface{}{
				"status":            "pending",
				"priority":          "high",
				"_data.assigned_to": "alice",
			},
			CreatedAt: baseTime,
			UpdatedAt: baseTime,
		},
		{
			UUID:     "2",
			SimpleID: "2",
			Title:    "Budget Review",
			Body:     "Review the meeting notes from last quarter",
			Dimensions: map[string]interface{}{
				"status":            "active",
				"priority":          "medium",
				"_data.assigned_to": "bob",
			},
			CreatedAt: baseTime.Add(time.Hour),
			UpdatedAt: baseTime.Add(time.Hour),
		},
		{
			UUID:     "3",
			SimpleID: "3",
			Title:    "Team Standup",
			Body:     "Daily standup meeting for development team",
			Dimensions: map[string]interface{}{
				"status":            "done",
				"priority":          "low",
				"_data.assigned_to": "alice",
			},
			CreatedAt: baseTime.Add(2 * time.Hour),
			UpdatedAt: baseTime.Add(2 * time.Hour),
		},
		{
			UUID:     "4",
			SimpleID: "4",
			Title:    "MEETING",
			Body:     "All caps meeting title for testing",
			Dimensions: map[string]interface{}{
				"status":   "pending",
				"priority": "high",
			},
			CreatedAt: baseTime.Add(3 * time.Hour),
			UpdatedAt: baseTime.Add(3 * time.Hour),
		},
	}
}
