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
	Long: `A modern, ergonomic CLI for NanoStore.

Examples:
  # List all active tasks
  nanostore list --x-type=Task --status=active

  # Get a specific document
  nanostore get --x-type=Task 1`,
}

// Global flag variables
var (
	x_typeName string
	x_dbPath   string
	x_format   string
	x_noColor  bool
	x_quiet    bool
	x_dryRun   bool
)

func init() {
	// Universal flags for all commands, now prefixed with 'x-'
	rootCmd.PersistentFlags().StringVar(&x_typeName, "x-type", "", "Type definition (required)")
	rootCmd.PersistentFlags().StringVar(&x_dbPath, "x-db", "", "Database file path (required)")
	rootCmd.PersistentFlags().StringVar(&x_format, "x-format", "table", "Output format: table|json|yaml|csv")
	rootCmd.PersistentFlags().BoolVar(&x_noColor, "x-no-color", false, "Disable colors")
	rootCmd.PersistentFlags().BoolVar(&x_quiet, "x-quiet", false, "Suppress headers")
	rootCmd.PersistentFlags().BoolVar(&x_dryRun, "x-dry-run", false, "Show what would happen without executing")

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
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			// It's a flag. Check if it's a command flag or a filter flag.
			// For simplicity, we assume all flags starting with --x- are command flags.
			if strings.HasPrefix(arg, "--x-") {
				cobraArgs = append(cobraArgs, arg)
			} else {
				filterArgs = append(filterArgs, arg)
			}
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}
	return
}

func main() {
	// Check for Viper CLI mode (keeping for compatibility if needed)
	if len(os.Args) > 1 && os.Args[1] == "--use-viper" {
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

	// Reconstruct os.Args for Cobra
	os.Args = append(cobraArgs, positionalArgs...)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
