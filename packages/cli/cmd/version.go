/*
Copyright © 2026 system-cli
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "0.0.1"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("system-cli v%s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
