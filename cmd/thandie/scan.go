package main

import (
	"fmt"
	"os"

	"github.com/ThandieOps/thandie-agent/internal/cache"
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
		// Log logging configuration status
		if cfg != nil {
			logPath, pathErr := logger.GetLogFilePath()
			if pathErr == nil {
				logger.Info("logging configuration",
					"level", cfg.Logging.Level,
					"to_file", cfg.Logging.ToFile,
					"json", cfg.Logging.JSON,
					"log_path", logPath)
			} else {
				logger.Info("logging configuration",
					"level", cfg.Logging.Level,
					"to_file", cfg.Logging.ToFile,
					"json", cfg.Logging.JSON)
			}
		}

		// Resolve workspace path using precedence: flag > env > config > default
		wsPath := getWorkspacePath()
		if wsPath == "" {
			logger.Error("workspace path is empty", "hint", "use --workspace or -w to specify it")
			os.Exit(1)
		}

		logger.Info("scanning workspace", "path", wsPath)
		logger.Debug("scanning workspace", "path", wsPath)

		// Get scanner config from global config
		ignoreDirs := []string{".git", "node_modules", "vendor"} // default
		includeHidden := false                                   // default
		if cfg != nil {
			ignoreDirs = cfg.Scanner.IgnoreDirs
			includeHidden = cfg.Scanner.IncludeHidden
		}

		logger.Info("scanner configuration",
			"ignore_dirs", ignoreDirs,
			"include_hidden", includeHidden)

		dirs, err := scanner.ListTopLevelDirs(wsPath, ignoreDirs, includeHidden)
		if err != nil {
			logger.Error("failed to scan workspace", "error", err, "path", wsPath)
			os.Exit(1)
		}

		logger.Info("scan completed", "directories_found", len(dirs))

		// Save scan results to cache
		cacheInstance, err := cache.New()
		if err != nil {
			logger.Warn("failed to initialize cache", "error", err)
		} else {
			if err := cacheInstance.SaveScanResult(wsPath, dirs); err != nil {
				logger.Warn("failed to save scan results to cache", "error", err)
			} else {
				logger.Info("scan results cached", "count", len(dirs), "cache_dir", cacheInstance.GetCacheDir())
				logger.Debug("scan results cached", "count", len(dirs), "cache_dir", cacheInstance.GetCacheDir())
			}
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
