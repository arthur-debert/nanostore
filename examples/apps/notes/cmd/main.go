// Command-line interface for the notes app
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arthur-debert/nanostore/examples/apps/notes"
)

func main() {
	// Get database path from home directory
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(home, ".notes.db")

	// Parse command
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	// Create app instance
	app, err := notes.New(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating notes app: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()

	switch cmd {
	case "list":
		cmdList(app, os.Args[2:])
	case "add":
		cmdAdd(app, os.Args[2:])
	case "archive":
		cmdArchive(app, os.Args[2:])
	case "unarchive":
		cmdUnarchive(app, os.Args[2:])
	case "delete":
		cmdDelete(app, os.Args[2:])
	case "tag":
		cmdTag(app, os.Args[2:])
	case "search":
		cmdSearch(app, os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: notes <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  list [--archived] [--deleted]  List notes")
	fmt.Println("  add <title> [-c content] [-t tags]  Add a new note")
	fmt.Println("  archive <id>                   Archive a note")
	fmt.Println("  unarchive <id>                 Unarchive a note")
	fmt.Println("  delete <id>                    Soft-delete a note")
	fmt.Println("  tag <id> <tags>                Update note tags")
	fmt.Println("  search <query> [--archived]    Search notes")
}

func cmdList(app *notes.Notes, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	showArchived := fs.Bool("archived", false, "Show archived notes")
	showDeleted := fs.Bool("deleted", false, "Show deleted notes")
	fs.Parse(args)

	notesList, err := app.List(notes.ListOptions{
		ShowArchived: *showArchived,
		ShowDeleted:  *showDeleted,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing notes: %v\n", err)
		os.Exit(1)
	}

	output := notes.FormatList(notesList, *showArchived || *showDeleted)
	fmt.Print(output)
}

func cmdAdd(app *notes.Notes, args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	content := fs.String("c", "", "Note content")
	tagsStr := fs.String("t", "", "Comma-separated tags")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: title required\n")
		os.Exit(1)
	}

	title := strings.Join(fs.Args(), " ")

	var tags []string
	if *tagsStr != "" {
		tags = strings.Split(*tagsStr, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	id, err := app.Add(title, *content, tags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding note: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added: %s\n", id)
}

func cmdArchive(app *notes.Notes, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: ID required\n")
		os.Exit(1)
	}

	err := app.Archive(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error archiving note: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Archived: %s\n", args[0])
}

func cmdUnarchive(app *notes.Notes, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: ID required\n")
		os.Exit(1)
	}

	err := app.Unarchive(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unarchiving note: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Unarchived: %s\n", args[0])
}

func cmdDelete(app *notes.Notes, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: ID required\n")
		os.Exit(1)
	}

	err := app.Delete(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting note: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Deleted: %s\n", args[0])
}

func cmdTag(app *notes.Notes, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: ID and tags required\n")
		os.Exit(1)
	}

	id := args[0]
	tags := strings.Split(args[1], ",")
	for i := range tags {
		tags[i] = strings.TrimSpace(tags[i])
	}

	err := app.UpdateTags(id, tags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating tags: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated tags for %s\n", id)
}

func cmdSearch(app *notes.Notes, args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	showArchived := fs.Bool("archived", false, "Include archived notes")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: search query required\n")
		os.Exit(1)
	}

	query := strings.Join(fs.Args(), " ")

	notesList, err := app.Search(query, *showArchived)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching: %v\n", err)
		os.Exit(1)
	}

	output := notes.FormatList(notesList, false)
	fmt.Print(output)
}
