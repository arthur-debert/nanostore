package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	dataFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nanonotes",
	Short: "A simple note-taking application using nanostore",
	Long: `Nanonotes is a sample application demonstrating nanostore's capabilities
for managing notes with soft delete and pinning features.

Notes have:
- A mandatory title
- An optional body
- A status (active or deleted)
- A pinned flag

The canonical view shows all non-deleted notes.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&dataFile, "file", "f", "notes.json", "data file to use")
}

// getDataFile returns the data file path
func getDataFile() string {
	if dataFile != "" {
		return dataFile
	}
	// Default to notes.json in current directory
	return "notes.json"
}
