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
	Short: "System monitor CLI",
	Long:  `A terminal-based system monitor showing CPU, RAM, disk, network and processes.`,
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
