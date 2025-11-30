package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags (available to all subcommands)
	workspacePath string
)

// rootCmd represents the base command: `thandie`
var rootCmd = &cobra.Command{
	Use:   "thandie",
	Short: "Thandie monitors local workspaces and syncs their state",
	Long: `Thandie is a CLI tool for monitoring your local development workspaces
and syncing their state with a remote service.`,
	// If you want `thandie` to do something when called with no subcommand,
	// add a Run: func(cmd, args) {...} here. For now, we'll leave it empty.
}

// Execute is called by main.main()
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global/persistent flags available to ALL subcommands
	rootCmd.PersistentFlags().StringVarP(
		&workspacePath,
		"workspace",
		"w",
		".",
		"Path to the workspace directory",
	)

	// If you want local (non-persistent) flags for the root, use rootCmd.Flags().
}
