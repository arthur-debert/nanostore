// Command-line interface for the todo app
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arthur-debert/nanostore/examples/apps/todo"
)

func main() {
	// Get database path from home directory
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(home, ".todo.db")

	// Parse command
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	// Create app instance
	app, err := todo.New(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating todo app: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()

	switch cmd {
	case "list":
		cmdList(app, os.Args[2:])
	case "add":
		cmdAdd(app, os.Args[2:])
	case "complete":
		cmdComplete(app, os.Args[2:])
	case "reopen":
		cmdReopen(app, os.Args[2:])
	case "search":
		cmdSearch(app, os.Args[2:])
	case "move":
		cmdMove(app, os.Args[2:])
	case "delete":
		cmdDelete(app, os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: todo <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  list [--all]                List todos")
	fmt.Println("  add <title> [-p parent]     Add a new todo")
	fmt.Println("  complete <id> [<id>...]     Mark todo(s) as completed")
	fmt.Println("  reopen <id>                 Reopen completed todo")
	fmt.Println("  search <query> [--all]      Search todos")
	fmt.Println("  move <id> <new-parent>      Move todo to new parent")
	fmt.Println("  delete <id> [--cascade]     Delete todo")
}

func cmdList(app *todo.Todo, args []string) {
	var showAll bool

	fs := flag.NewFlagSet("list", flag.ExitOnError)
	fs.BoolVar(&showAll, "all", false, "Show completed items")
	fs.Parse(args)

	items, err := app.List(todo.ListOptions{ShowAll: showAll})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing todos: %v\n", err)
		os.Exit(1)
	}

	output := todo.FormatTree(items, "", showAll)
	fmt.Print(output)
}

func cmdAdd(app *todo.Todo, args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	parent := fs.String("p", "", "Parent ID")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: title required\n")
		os.Exit(1)
	}

	title := strings.Join(fs.Args(), " ")

	var parentID *string
	if *parent != "" {
		parentID = parent
	}

	id, err := app.Add(title, parentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding todo: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added: %s\n", id)
}

func cmdComplete(app *todo.Todo, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: ID(s) required\n")
		os.Exit(1)
	}

	// Support multiple IDs for batch completion
	if len(args) > 1 {
		err := app.CompleteMultiple(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error completing todos: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Completed: %s\n", strings.Join(args, ", "))
	} else {
		// Single ID completion
		err := app.Complete(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error completing todo: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Completed: %s\n", args[0])
	}
}

func cmdReopen(app *todo.Todo, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: ID required\n")
		os.Exit(1)
	}

	err := app.Reopen(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reopening todo: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Reopened: %s\n", args[0])
}

func cmdSearch(app *todo.Todo, args []string) {
	var showAll bool

	fs := flag.NewFlagSet("search", flag.ExitOnError)
	fs.BoolVar(&showAll, "all", false, "Show completed items")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: search query required\n")
		os.Exit(1)
	}

	query := strings.Join(fs.Args(), " ")

	items, err := app.Search(query, showAll)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching: %v\n", err)
		os.Exit(1)
	}

	output := todo.FormatTree(items, "", showAll)
	fmt.Print(output)
}

func cmdMove(app *todo.Todo, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: ID and new parent required\n")
		os.Exit(1)
	}

	id := args[0]
	newParent := args[1]

	var newParentID *string
	if newParent != "root" && newParent != "" {
		newParentID = &newParent
	}

	err := app.Move(id, newParentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error moving todo: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Moved %s to %s\n", id, newParent)
}

func cmdDelete(app *todo.Todo, args []string) {
	var cascade bool

	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	fs.BoolVar(&cascade, "cascade", false, "Delete children too")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: ID required\n")
		os.Exit(1)
	}

	err := app.Delete(fs.Arg(0), cascade)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting todo: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Deleted: %s\n", fs.Arg(0))
}
