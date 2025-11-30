package main

import (
	"fmt"
	"log"

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
			log.Fatal("workspace path is empty; use --workspace or -w to specify it")
		}

		dirs, err := scanner.ListTopLevelDirs(wsPath)
		if err != nil {
			log.Fatalf("failed to scan workspace: %v", err)
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
