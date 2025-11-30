package main

import (
	"fmt"
	"os"

	"github.com/ThandieOps/thandie-agent/internal/logger"
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
			fmt.Fprintf(os.Stderr, "Error: workspace path is empty. Use --workspace or -w to specify it.\n")
			os.Exit(1)
		}

		// Get scanner config from global config
		ignoreDirs := []string{".git", "node_modules", "vendor"} // default
		includeHidden := false                                   // default
		if cfg != nil {
			ignoreDirs = cfg.Scanner.IgnoreDirs
			includeHidden = cfg.Scanner.IncludeHidden
		}

		// Log configuration to file only (not stdout/stderr)
		// These messages will appear in the log file if logging is enabled
		if cfg != nil {
			logPath, pathErr := logger.GetLogFilePath()
			if pathErr == nil {
				logger.Info("logging configuration",
					"level", cfg.Logging.Level,
					"to_file", cfg.Logging.ToFile,
					"json", cfg.Logging.JSON,
					"log_path", logPath)
			}
			logger.Info("scanning workspace", "path", wsPath)
			logger.Info("scanner configuration",
				"ignore_dirs", ignoreDirs,
				"include_hidden", includeHidden)
		}

		// Scan directories with metadata collection using TUI
		dirInfos, err := runScanWithTUI(wsPath, ignoreDirs, includeHidden)
		if err != nil {
			logger.Error("failed to scan workspace", "error", err, "path", wsPath)
			fmt.Fprintf(os.Stderr, "Error: failed to scan workspace: %v\n", err)
			os.Exit(1)
		}

		logger.Info("scan completed", "directories_found", len(dirInfos))

		// Display summary instead of full directory listing
		if len(dirInfos) == 0 {
			fmt.Printf("Scan complete: No top-level directories found in %s\n", wsPath)
			return
		}

		// Count git repositories
		gitRepos := 0
		uncommittedRepos := 0
		for _, info := range dirInfos {
			if info.GitMetadata != nil && info.GitMetadata.IsGitRepo {
				gitRepos++
				if info.GitMetadata.HasUncommitted {
					uncommittedRepos++
				}
			}
		}

		fmt.Printf("\nScan complete: Found %d directories", len(dirInfos))
		if gitRepos > 0 {
			fmt.Printf(" (%d git repositories", gitRepos)
			if uncommittedRepos > 0 {
				fmt.Printf(", %d with uncommitted changes", uncommittedRepos)
			}
			fmt.Printf(")")
		}
		fmt.Printf("\n")
	},
}

func init() {
	// Attach the `scan` command to the root: thandie scan
	rootCmd.AddCommand(scanCmd)

	// If you want flags specific to scan, add them here:
	// scanCmd.Flags().Bool("json", false, "Output results as JSON")
}
