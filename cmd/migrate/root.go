// Part of the nanostore CLI - this file implements the 'nanostore migrate <command>' subcommand.
// Build the CLI with: scripts/build
// This creates bin/nanostore which includes all migration commands.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/migration"
	"github.com/arthur-debert/nanostore/nanostore/store"
	"github.com/spf13/cobra"
)

var (
	storePath string
	dryRun    bool
	verbose   bool
)

var rootCmd = &cobra.Command{
	Use:   "nanostore",
	Short: "Nanostore CLI",
	Long:  "Nanostore is a document and ID store library with SQLite backend.",
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Schema migration tools",
	Long:  "Perform field operations on nanostore documents including rename, remove, add, and transform.",
}

func init() {
	// Add migrate as a subcommand of root
	rootCmd.AddCommand(migrateCmd)

	// Add flags to the migrate command
	migrateCmd.PersistentFlags().StringVarP(&storePath, "store", "s", "", "path to store file (required)")
	migrateCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false, "preview changes without applying them")
	migrateCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show detailed output")
	_ = migrateCmd.MarkPersistentFlagRequired("store")

	// Add subcommands to migrate
	migrateCmd.AddCommand(renameFieldCmd)
	migrateCmd.AddCommand(removeFieldCmd)
	migrateCmd.AddCommand(addFieldCmd)
	migrateCmd.AddCommand(transformFieldCmd)
	migrateCmd.AddCommand(validateCmd)
}

// loadStore loads the nanostore from the specified path
func loadStore() (store.Store, error) {
	if storePath == "" {
		return nil, fmt.Errorf("store path is required")
	}

	// Ensure absolute path
	absPath, err := filepath.Abs(storePath)
	if err != nil {
		return nil, fmt.Errorf("invalid store path: %w", err)
	}

	// Load config (for now, use empty config)
	// TODO: Load config from adjacent config file if exists
	config := nanostore.Config{}

	return store.New(absPath, &config)
}

// printMessage prints a message with appropriate formatting based on level
func printMessage(msg migration.Message) {
	prefix := ""
	switch msg.Level {
	case migration.LevelError:
		prefix = "ERROR: "
		fmt.Fprintf(os.Stderr, "\033[31m%s%s\033[0m\n", prefix, msg.Text)
	case migration.LevelWarning:
		prefix = "WARN: "
		fmt.Fprintf(os.Stderr, "\033[33m%s%s\033[0m\n", prefix, msg.Text)
	case migration.LevelInfo:
		fmt.Printf("%s\n", msg.Text)
	case migration.LevelDebug:
		if verbose {
			fmt.Printf("DEBUG: %s\n", msg.Text)
		}
	}

	// Print details if verbose and present
	if verbose && msg.Details != nil {
		for k, v := range msg.Details {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
}

// handleResult processes the migration result and exits with appropriate code
func handleResult(result *migration.Result) {
	// Print all messages
	for _, msg := range result.Messages {
		printMessage(msg)
	}

	// Print summary
	fmt.Println()
	if result.Success {
		fmt.Printf("Migration completed successfully\n")
		if result.Stats.TotalDocs > 0 {
			fmt.Printf("  Modified: %d/%d documents\n", result.Stats.ModifiedDocs, result.Stats.TotalDocs)
			fmt.Printf("  Duration: %v\n", result.Stats.Duration)
		}
		if dryRun {
			fmt.Printf("  (DRY RUN - no changes applied)\n")
		}
	} else {
		fmt.Printf("Migration failed\n")
	}

	os.Exit(result.Code)
}
