package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nano-db",
	Short: "Nano-DB CLI - Document and ID store management",
	Long: `A modern, ergonomic CLI for Nano-DB.

Examples:
  # List all active tasks
  nano-db list --x-type=Task --status=active

  # Get a specific document
  nano-db get --x-type=Task 1`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logging before any command runs
		if err := initLogging(x_logLevel, x_logQueries, x_logResults); err != nil {
			return fmt.Errorf("failed to initialize logging: %w", err)
		}
		return nil
	},
}

// Global flag variables
var (
	x_typeName   string
	x_dbPath     string
	x_format     string
	x_noColor    bool
	x_quiet      bool
	x_dryRun     bool
	x_logLevel   string
	x_logQueries bool
	x_logResults bool
)

func init() {
	// Universal flags for all commands, now prefixed with 'x-'
	rootCmd.PersistentFlags().StringVar(&x_typeName, "x-type", os.Getenv("NANOSTORE_TYPE"), "Type definition (required)")
	rootCmd.PersistentFlags().StringVar(&x_dbPath, "x-db", os.Getenv("NANOSTORE_DB"), "Database file path (required)")
	rootCmd.PersistentFlags().StringVar(&x_format, "x-format", getEnvOrDefault("NANOSTORE_FORMAT", "table"), "Output format: table|json|yaml|csv")
	rootCmd.PersistentFlags().BoolVar(&x_noColor, "x-no-color", getEnvBool("NANOSTORE_NO_COLOR"), "Disable colors")
	rootCmd.PersistentFlags().BoolVar(&x_quiet, "x-quiet", getEnvBool("NANOSTORE_QUIET"), "Suppress headers")
	rootCmd.PersistentFlags().BoolVar(&x_dryRun, "x-dry-run", getEnvBool("NANOSTORE_DRY_RUN"), "Show what would happen without executing")
	rootCmd.PersistentFlags().StringVar(&x_logLevel, "x-log-level", getEnvOrDefault("NANOSTORE_LOG_LEVEL", "warn"), "Log level (debug|info|warn|error)")
	rootCmd.PersistentFlags().BoolVar(&x_logQueries, "x-log-queries", getEnvBool("NANOSTORE_LOG_QUERIES"), "Log SQL queries to stdout")
	rootCmd.PersistentFlags().BoolVar(&x_logResults, "x-log-results", getEnvBool("NANOSTORE_LOG_RESULTS"), "Log query results to stdout")

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
			// ... (types command logic remains the same)
			return nil
		},
	}
	rootCmd.AddCommand(typesCmd)

	// Add commands to root, organized by category
	for _, cmds := range commandsByCategory {
		for _, cmd := range cmds {
			cobraCmd := cmd.ToCobraCommand(generator)
			rootCmd.AddCommand(cobraCmd)
		}
	}
}

// preParse separates CLI arguments into cobra flags, filter flags, and positional args.
// This logic now applies to all commands.
func preParse(args []string) (cobraArgs, filterArgs, positionalArgs []string) {
	if len(args) < 2 {
		return args, nil, nil
	}

	cobraArgs = []string{args[0]} // Keep program name
	var command string

	// Find the command
	for i := 1; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "-") {
			command = args[i]
			cobraArgs = append(cobraArgs, command)
			// The rest of the args start after the command
			args = args[i+1:]
			break
		}
		// If it's a root flag before the command
		cobraArgs = append(cobraArgs, args[i])
	}

	// These commands do not support filter flags
	nonFilterCommands := map[string]bool{
		"types": true,
		"help":  true,
	}

	if nonFilterCommands[command] {
		positionalArgs = args
		return
	}

	// Separate remaining args into flags and positionals
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			// It's a flag. Check if it's a command flag or a filter flag.
			// Command flags: --x-*, --filter, --args, --sort, --limit, --cascade
			if strings.HasPrefix(arg, "--x-") || arg == "--filter" || arg == "--args" || arg == "--sort" || arg == "--limit" || arg == "--cascade" {
				cobraArgs = append(cobraArgs, arg)
				// If this is --filter, also add the next argument if it exists and doesn't start with -
				if arg == "--filter" && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					cobraArgs = append(cobraArgs, args[i+1])
					// Skip the next argument in the main loop
					continue
				}
			} else {
				filterArgs = append(filterArgs, arg)
			}
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}
	return
}

// Helper functions for environment variables
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string) bool {
	value := os.Getenv(key)
	if value == "" {
		return false
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return b
}

func main() {
	cobraArgs, filterArgs, positionalArgs := preParse(os.Args)
	query := parseFilters(filterArgs)

	// Pass the query object via context
	ctx := rootCmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	rootCmd.SetContext(withQuery(ctx, query))

	// Reconstruct os.Args for Cobra
	os.Args = append(cobraArgs, positionalArgs...)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
