package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	storePath string
	verbose   bool
)

var rootCmd = &cobra.Command{
	Use:   "todos",
	Short: "A hierarchical todo application built with nanostore",
	Long: `Todos is a command-line todo application that demonstrates nanostore's
hierarchical document management and ID generation capabilities.

Features:
- Hierarchical todo items with subtasks
- Priority levels (low, medium, high) with ID prefixes
- Status tracking (pending, active, done) 
- Search and filtering capabilities
- Export functionality

Examples:
  todos add "Buy groceries"
  todos add --parent 1 "Milk" 
  todos list
  todos complete 1.1
  todos export`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&storePath, "store", "s", "todos.json", "Path to the todos database file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
