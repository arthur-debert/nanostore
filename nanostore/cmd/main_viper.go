package main

import (
	"fmt"
	"os"
)

// mainViper is the entry point for the Viper-powered CLI
func mainViper() {
	cli := NewViperCLI()
	
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}