package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Print the version number of aws-ghost.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("aws-ghost v0.1.0")
	},
}
