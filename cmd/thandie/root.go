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
	// Remove duplicate help commands that Cobra may add during execution
	// We keep only our custom help command (identified by the "custom" annotation)
	cmds := rootCmd.Commands()
	for _, c := range cmds {
		if c.Use == "help [command]" || c.Use == "help" {
			// Check if this is our custom help command by looking for the annotation
			if c.Annotations == nil || c.Annotations["custom"] != "true" {
				rootCmd.RemoveCommand(c)
			}
		}
	}

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
