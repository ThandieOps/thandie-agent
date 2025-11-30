package main

import (
	"fmt"
	"os"

	"github.com/ThandieOps/thandie-agent/internal/logger"
	"github.com/ThandieOps/thandie-agent/internal/scanner"
	"github.com/spf13/cobra"
)

// scanCmd represents: `thandie scan`
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan the workspace and list top-level directories",
	Long: `Scan the configured workspace directory and display the
top-level project folders found there.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Resolve workspace path using precedence: flag > env > config > default
		wsPath := getWorkspacePath()
		if wsPath == "" {
			logger.Error("workspace path is empty", "hint", "use --workspace or -w to specify it")
			os.Exit(1)
		}

		logger.Debug("scanning workspace", "path", wsPath)
		dirs, err := scanner.ListTopLevelDirs(wsPath)
		if err != nil {
			logger.Error("failed to scan workspace", "error", err, "path", wsPath)
			os.Exit(1)
		}

		if len(dirs) == 0 {
			fmt.Printf("No top-level directories found in %s\n", wsPath)
			return
		}

		fmt.Printf("Top-level directories in %s:\n", wsPath)
		for _, d := range dirs {
			fmt.Println(" -", d)
		}
	},
}

func init() {
	// Attach the `scan` command to the root: thandie scan
	rootCmd.AddCommand(scanCmd)

	// If you want flags specific to scan, add them here:
	// scanCmd.Flags().Bool("json", false, "Output results as JSON")
}
