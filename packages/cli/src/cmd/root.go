/*
Copyright © 2026 system-cli
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "system",
	Short: "System CLI application (utilities tools)",
	Long:  `The system CLI application is a comprehensive backend utility belonging to the utilities suite of tools.

Use this root executable to manage configuring, running, and interacting with all system-related operations securely and efficiently from your terminal.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
