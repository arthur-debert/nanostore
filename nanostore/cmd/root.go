package main

import (
	"context"
	"fmt"
	"os"
	"strings"

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

	// Add the 'next' command for the POC
	nextCmd := &cobra.Command{
		Use:   "next",
		Short: "POC for the new query syntax parser",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, ok := fromContext(cmd.Context())
			if !ok {
				// This should not happen if preParse is correct
				return fmt.Errorf("query object not found in context")
			}

			fmt.Println("--- Next Command (POC) ---")
			fmt.Printf("Parsed Query Object: %+v\n", query)
			fmt.Println("--------------------------")
			fmt.Println("Received positional args:", args)
			fmt.Println("Flag --x-db:", cmd.Flag("x-db").Value)
			fmt.Println("Flag --x-type:", cmd.Flag("x-type").Value)
			return nil
		},
	}
	// Add flags with the 'x-' prefix for the POC
	nextCmd.Flags().String("x-db", "", "Database file path")
	nextCmd.Flags().String("x-type", "", "Document type")
	rootCmd.AddCommand(nextCmd)
}

// preParse separates CLI arguments into cobra flags, filter flags, and positional args.
func preParse(args []string) (cobraArgs, filterArgs, positionalArgs []string) {
	if len(args) < 2 || args[1] != "next" {
		return args, nil, nil // Not the 'next' command, do nothing
	}

	cobraArgs = []string{args[0], args[1]} // Keep program name and command

	knownFlags := map[string]bool{
		"--db":     true,
		"--type":   true,
		"--format": true,
	}

	for i := 2; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			positionalArgs = append(positionalArgs, arg)
			continue
		}

		flag := strings.SplitN(arg, "=", 2)[0]
		if knownFlags[flag] {
			cobraArgs = append(cobraArgs, "--x-"+strings.TrimPrefix(arg, "--"))
		} else {
			filterArgs = append(filterArgs, arg)
		}
	}
	return cobraArgs, filterArgs, positionalArgs
}

func main() {
	// Check for Viper CLI mode
	if len(os.Args) > 1 && os.Args[1] == "--use-viper" {
		// Remove the --use-viper flag and run Viper CLI
		os.Args = append(os.Args[:1], os.Args[2:]...)
		mainViper()
		return
	}

	cobraArgs, filterArgs, positionalArgs := preParse(os.Args)
	query := parseFilters(filterArgs)

	// Pass the query object via context
	ctx := rootCmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	rootCmd.SetContext(withQuery(ctx, query))

	os.Args = append(cobraArgs, positionalArgs...) // Pass only cobra args and positionals to Execute

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
