// This is the main entry point for the nanostore CLI.
// The migrate commands are subcommands of the main nanostore binary.
// Build with: go build -o bin/nanostore ./cmd/migrate
// Usage: nanostore migrate <command> [options]
package main

import (
	"os"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
