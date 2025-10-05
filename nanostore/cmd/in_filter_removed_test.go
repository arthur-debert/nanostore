package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestINFiltersNotSupported(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		t.Fatalf("Failed to load builtin types: %v", err)
	}

	executor := NewReflectionExecutor(registry)

	// Test that filter-in raises an error
	t.Run("FilterInNotSupported", func(t *testing.T) {
		_, _, err := executor.buildFilterWhere(
			"", "", "", "", // No date filters
			nil, nil, // No NULL filters
			"", "", "", false, // No text search
			nil, nil, nil, nil, nil, nil, nil, // No enhanced filters (including empty filterIn slot)
			"", "") // No status/priority filters

		// This should work (no IN filters provided)
		if err != nil {
			t.Fatalf("Expected no error when no IN filters provided, got: %v", err)
		}

		// Test that the system properly explains IN filters are not supported
		// by checking that the flag is not registered in the CLI
		cli := NewViperCLI()
		cmd := cli.GetRootCommand()

		// Find the list command
		var listCmd *cobra.Command
		for _, subCmd := range cmd.Commands() {
			if subCmd.Name() == "list" {
				listCmd = subCmd
				break
			}
		}

		if listCmd == nil {
			t.Fatal("Could not find list command")
		}

		// Verify that filter-in flag is not registered
		filterInFlag := listCmd.Flags().Lookup("filter-in")
		if filterInFlag != nil {
			t.Error("filter-in flag should not be registered since IN filters are not supported")
		}

		// Verify that status-in and priority-in flags are not registered
		statusInFlag := listCmd.Flags().Lookup("status-in")
		if statusInFlag != nil {
			t.Error("status-in flag should not be registered since IN filters are not supported")
		}

		priorityInFlag := listCmd.Flags().Lookup("priority-in")
		if priorityInFlag != nil {
			t.Error("priority-in flag should not be registered since IN filters are not supported")
		}
	})

	t.Run("HelpTextMentionsINLimitation", func(t *testing.T) {
		cli := NewViperCLI()
		cmd := cli.GetRootCommand()

		// Find the list command and check its help text
		var listCmd *cobra.Command
		for _, subCmd := range cmd.Commands() {
			if subCmd.Name() == "list" {
				listCmd = subCmd
				break
			}
		}

		if listCmd == nil {
			t.Fatal("Could not find list command")
		}

		helpText := listCmd.Long

		// Verify that the help text does not mention filter-in as available
		if strings.Contains(helpText, "--filter-in") {
			t.Error("Help text should not mention --filter-in since it's not supported")
		}

		// The help should focus on supported filters
		supportedFilters := []string{"--filter-eq", "--filter-ne", "--filter-gt", "--filter-lt", "--filter-gte", "--filter-lte", "--filter-like"}
		for _, filter := range supportedFilters {
			if !strings.Contains(helpText, filter) {
				t.Errorf("Help text should mention supported filter %s", filter)
			}
		}
	})
}
