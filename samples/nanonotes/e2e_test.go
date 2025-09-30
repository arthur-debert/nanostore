package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ECLI(t *testing.T) {
	// Build the binary
	tempDir := t.TempDir()
	binary := filepath.Join(tempDir, "nanonotes")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	// Set up data file
	dataFile := filepath.Join(tempDir, "test-notes.json")

	// Helper function to run commands
	runCmd := func(args ...string) (string, error) {
		fullArgs := append([]string{"-f", dataFile}, args...)
		cmd := exec.Command(binary, fullArgs...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		return out.String(), err
	}

	t.Run("add notes", func(t *testing.T) {
		output, err := runCmd("add", "First note", "--body", "This is the body")
		if err != nil {
			t.Fatalf("failed to add note: %v, output: %s", err, output)
		}

		if !strings.Contains(output, "Note added successfully") {
			t.Errorf("unexpected output: %s", output)
		}

		// Add more notes
		runCmd("add", "Shopping list", "--body", "Buy milk and bread", "--tags", "personal,todo")
		runCmd("add", "Meeting notes", "--body", "Discuss project timeline")
		runCmd("add", "Important task", "--body", "Complete by Friday")
	})

	t.Run("list notes", func(t *testing.T) {
		output, err := runCmd("list")
		if err != nil {
			t.Fatalf("failed to list notes: %v, output: %s", err, output)
		}

		// Should show all 4 notes
		if !strings.Contains(output, "Showing 4 note(s)") {
			t.Errorf("expected 4 notes, got: %s", output)
		}

		// Check for note titles
		if !strings.Contains(output, "First note") {
			t.Error("missing 'First note' in list")
		}
		if !strings.Contains(output, "Shopping list") {
			t.Error("missing 'Shopping list' in list")
		}
	})

	t.Run("pin note", func(t *testing.T) {
		output, err := runCmd("pin", "4")
		if err != nil {
			t.Fatalf("failed to pin note: %v, output: %s", err, output)
		}

		if !strings.Contains(output, "pinned successfully") {
			t.Errorf("unexpected output: %s", output)
		}

		// List pinned notes
		output, err = runCmd("list", "--pinned")
		if err != nil {
			t.Fatalf("failed to list pinned notes: %v", err)
		}

		if !strings.Contains(output, "Important task") {
			t.Error("pinned note not shown in pinned list")
		}
		if !strings.Contains(output, "Showing 1 note(s)") {
			t.Error("should show only 1 pinned note")
		}
	})

	t.Run("search notes", func(t *testing.T) {
		output, err := runCmd("search", "meeting")
		if err != nil {
			t.Fatalf("failed to search notes: %v, output: %s", err, output)
		}

		if !strings.Contains(output, "Meeting notes") {
			t.Error("search should find 'Meeting notes'")
		}
		if !strings.Contains(output, "Found 1 note(s)") {
			t.Error("should find exactly 1 note")
		}
	})

	t.Run("delete note", func(t *testing.T) {
		output, err := runCmd("delete", "2")
		if err != nil {
			t.Fatalf("failed to delete note: %v, output: %s", err, output)
		}

		if !strings.Contains(output, "marked as deleted") {
			t.Errorf("unexpected output: %s", output)
		}

		// List should not show deleted note
		output, err = runCmd("list")
		if err != nil {
			t.Fatalf("failed to list notes: %v", err)
		}

		if strings.Contains(output, "Shopping list") {
			t.Error("deleted note should not appear in default list")
		}
		if !strings.Contains(output, "Showing 3 note(s)") {
			t.Error("should show 3 active notes")
		}

		// List with --all should show deleted note
		output, err = runCmd("list", "--all")
		if err != nil {
			t.Fatalf("failed to list all notes: %v", err)
		}

		if !strings.Contains(output, "Shopping list") {
			t.Error("deleted note should appear in --all list")
		}
		if !strings.Contains(output, "deleted") {
			t.Error("deleted note should show deleted status")
		}
	})

	t.Run("stats", func(t *testing.T) {
		output, err := runCmd("stats")
		if err != nil {
			t.Fatalf("failed to get stats: %v, output: %s", err, output)
		}

		if !strings.Contains(output, "Total notes:    4") {
			t.Error("should show 4 total notes")
		}
		if !strings.Contains(output, "Active notes:   3") {
			t.Error("should show 3 active notes")
		}
		if !strings.Contains(output, "Deleted notes:  1") {
			t.Error("should show 1 deleted note")
		}
		if !strings.Contains(output, "Pinned notes:   1") {
			t.Error("should show 1 pinned note")
		}
	})

	t.Run("clean", func(t *testing.T) {
		// Use --force to skip confirmation
		output, err := runCmd("clean", "--force")
		if err != nil {
			t.Fatalf("failed to clean notes: %v, output: %s", err, output)
		}

		if !strings.Contains(output, "Successfully cleaned 1 deleted note(s)") {
			t.Errorf("unexpected output: %s", output)
		}

		// Verify deleted note is gone
		output, err = runCmd("list", "--all")
		if err != nil {
			t.Fatalf("failed to list all notes: %v", err)
		}

		if strings.Contains(output, "Shopping list") {
			t.Error("cleaned note should not appear even with --all")
		}
		if !strings.Contains(output, "Showing 3 note(s)") {
			t.Error("should show only 3 notes after clean")
		}
	})

	t.Run("unpin note", func(t *testing.T) {
		// After clean, note 4 becomes note 3 since one note was deleted
		output, err := runCmd("unpin", "3")
		if err != nil {
			t.Fatalf("failed to unpin note: %v, output: %s", err, output)
		}

		if !strings.Contains(output, "unpinned successfully") {
			t.Errorf("unexpected output: %s", output)
		}

		// Verify no pinned notes
		output, err = runCmd("list", "--pinned")
		if err != nil {
			t.Fatalf("failed to list pinned notes: %v", err)
		}

		if !strings.Contains(output, "No notes found") {
			t.Error("should show no pinned notes")
		}
	})
}

func TestCLIEdgeCases(t *testing.T) {
	// Build the binary
	tempDir := t.TempDir()
	binary := filepath.Join(tempDir, "nanonotes")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	dataFile := filepath.Join(tempDir, "edge-test.json")

	runCmd := func(args ...string) (string, error) {
		fullArgs := append([]string{"-f", dataFile}, args...)
		cmd := exec.Command(binary, fullArgs...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		return out.String(), err
	}

	t.Run("delete non-existent note", func(t *testing.T) {
		_, err := runCmd("delete", "999")
		if err == nil {
			t.Error("deleting non-existent note should fail")
		}
	})

	t.Run("pin already pinned note", func(t *testing.T) {
		runCmd("add", "Test note")
		runCmd("pin", "1")

		output, err := runCmd("pin", "1")
		if err == nil {
			t.Error("pinning already pinned note should fail")
		}
		if !strings.Contains(output, "already pinned") {
			t.Errorf("unexpected error message: %s", output)
		}
	})

	t.Run("clean with no deleted notes", func(t *testing.T) {
		// Start fresh
		os.Remove(dataFile)

		runCmd("add", "Active note")

		output, err := runCmd("clean", "--force")
		if err != nil {
			t.Fatalf("clean should not fail: %v", err)
		}

		if !strings.Contains(output, "No deleted notes to clean") {
			t.Errorf("unexpected output: %s", output)
		}
	})

	t.Run("empty search query", func(t *testing.T) {
		_, err := runCmd("search")
		if err == nil {
			t.Error("search without query should fail")
		}
	})
}
