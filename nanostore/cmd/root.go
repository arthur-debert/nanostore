package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nanostore",
	Short: "Nanostore CLI - Document and ID store management",
	Long: `Nanostore is a document and ID store library that uses SQLite storage
to manage document storage and dynamically generate user-facing, hierarchical IDs.

This CLI provides generic operations that map directly to the Go API methods.
Use --type to specify the document type and --db for the database path.

Examples:
  # Create a new task document
  nanostore --type Task --db tasks.db create "New Task" --status pending --priority high
  
  # List all active tasks  
  nanostore --type Task --db tasks.db list --filter status=active
  
  # Get a specific document
  nanostore --type Task --db tasks.db get 1`,
}

var (
	// Global flags that apply to all commands
	typeName string
	dbPath   string
	format   string
	noColor  bool
	quiet    bool
	dryRun   bool
)

func init() {
	// Universal flags for all commands
	rootCmd.PersistentFlags().StringVarP(&typeName, "type", "t", "", "Type definition (required)")
	rootCmd.PersistentFlags().StringVarP(&dbPath, "db", "d", "", "Database file path (required)")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "table", "Output format: table|json|yaml|csv")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colors")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress headers")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without executing")

	// Most commands require type and db, but we'll validate per-command instead
	// of marking them as globally required

	// Generate and add all API commands
	generator := NewCommandGenerator()
	commands := generator.GenerateCommands()

	// Group commands by category
	commandsByCategory := make(map[CommandCategory][]Command)
	for _, cmd := range commands {
		commandsByCategory[cmd.Category] = append(commandsByCategory[cmd.Category], cmd)
	}

	// Add a types command to list available types
	typesCmd := &cobra.Command{
		Use:   "types",
		Short: "List available document types",
		RunE: func(cmd *cobra.Command, args []string) error {
			types := generator.registry.ListTypes()
			if len(types) == 0 {
				fmt.Println("No types registered. You can register types using JSON schema files.")
				return nil
			}

			fmt.Println("Available types:")
			for _, typeName := range types {
				fmt.Printf("  - %s\n", typeName)
			}

			// Show schema for specific type if requested
			if len(args) > 0 {
				typeName := args[0]
				schema, err := generator.registry.GetSchemaJSON(typeName)
				if err != nil {
					return fmt.Errorf("failed to get schema for type %s: %w", typeName, err)
				}
				fmt.Printf("\nSchema for %s:\n%s\n", typeName, schema)
			}

			return nil
		},
	}
	rootCmd.AddCommand(typesCmd)

	// Add commands to root, organized by category
	for _, cmds := range commandsByCategory {
		// Add commands directly to root for convenience
		for _, cmd := range cmds {
			cobraCmd := cmd.ToCobraCommand(generator)
			rootCmd.AddCommand(cobraCmd)
		}
	}
}

func main() {
	// Check for demo mode
	if len(os.Args) > 1 && os.Args[1] == "--demo-viper" {
		runViperDemo()
		return
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
