package cmd

import (
	"fmt"
	"samples-nanonotes/app"
	"time"
)

// Import Note and NoteApp types from app package
type Note = app.Note
type NoteApp = app.NoteApp

// getApp creates and returns a NoteApp instance
func getApp() (*NoteApp, error) {
	appInstance, err := app.NewNoteApp(getDataFile())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize app: %w", err)
	}
	return appInstance, nil
}

// formatTime formats a time for display
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
