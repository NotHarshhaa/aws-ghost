package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "aws-ghost",
	Short: "Scan your AWS account for forgotten, idle, and wasteful resources",
	Long: `aws-ghost scans your AWS account for forgotten, idle, and wasteful resources
and shows you exactly what they're costing you.

It is read-only, safe, and honest. It tells you what's wasting money so you can
decide what to do with it.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(securityCmd)
}
