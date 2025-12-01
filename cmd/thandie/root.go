package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ThandieOps/thandie-agent/internal/config"
	"github.com/ThandieOps/thandie-agent/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Global flags (available to all subcommands)
	workspacePath string

	// Global config instance
	cfg *config.Config
)

// rootCmd represents the base command: `thandie`
var rootCmd = &cobra.Command{
	Use:   "thandie",
	Short: "Thandie monitors local workspaces and syncs their state",
	Long: `Thandie is a CLI tool for monitoring your local development workspaces
and syncing their state with a remote service.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Launch TUI when no subcommand is provided
		wsPath := getWorkspacePath()
		if wsPath == "" {
			logger.Error("workspace path is empty", "hint", "use --workspace or -w to specify it")
			os.Exit(1)
		}

		tui := NewTUIApp(wsPath)
		if err := tui.Run(); err != nil {
			logger.Error("TUI error", "error", err)
			os.Exit(1)
		}
	},
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
	// Initialize Viper configuration
	initConfig()

	// Global/persistent flags available to ALL subcommands
	rootCmd.PersistentFlags().StringVarP(
		&workspacePath,
		"workspace",
		"w",
		"",
		"Path to the workspace directory (overrides THANDIE_WORKSPACE env var and config file)",
	)

	// Bind the flag to Viper (this allows Viper to read the flag value)
	if err := viper.BindPFlag("workspace", rootCmd.PersistentFlags().Lookup("workspace")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding workspace flag: %v\n", err)
	}

	// If you want local (non-persistent) flags for the root, use rootCmd.Flags().
}

// initConfig initializes Viper to read from config file, environment variables, and flags
func initConfig() {
	// Set config name and type
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Set config path: ~/.config/thandie/
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home dir, continue without config file
		// Environment variables and flags will still work
		return
	}

	configDir := filepath.Join(homeDir, ".config", "thandie")
	viper.AddConfigPath(configDir)

	// Set environment variable prefix
	viper.SetEnvPrefix("THANDIE")
	viper.AutomaticEnv() // Automatically read environment variables with THANDIE_ prefix
	// Map THANDIE_WORKSPACE to workspace.default (not workspace itself, to avoid conflict with nested structure)
	viper.BindEnv("workspace.default", "THANDIE_WORKSPACE")

	// Set defaults
	viper.SetDefault("version", 1)
	viper.SetDefault("workspace.default", "")
	viper.SetDefault("scanner.include_hidden", false)
	viper.SetDefault("scanner.ignore_dirs", []string{".git", "node_modules", "vendor"})
	viper.SetDefault("scanner.max_depth", 1)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.to_file", false)
	viper.SetDefault("logging.json", false)

	// Read config file (if it exists)
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is okay - we'll use defaults/env/flags
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Other errors (like parse errors) are more serious, but we'll continue
			// The user can still use flags and env vars
		}
	}

	// Unmarshal config into struct
	cfg = &config.Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		// If unmarshaling fails, try to read values directly from Viper
		fmt.Fprintf(os.Stderr, "Warning: failed to unmarshal config: %v\n", err)
		fmt.Fprintf(os.Stderr, "Attempting to read config values directly from Viper...\n")

		// Build config from Viper values directly
		cfg = &config.Config{
			Version: viper.GetInt("version"),
			Workspace: config.WorkspaceConfig{
				Default:  viper.GetString("workspace.default"),
				Profiles: []config.WorkspaceProfile{}, // Profiles parsing might be complex, skip for now
			},
			Scanner: config.ScannerConfig{
				IncludeHidden: viper.GetBool("scanner.include_hidden"),
				IgnoreDirs:    viper.GetStringSlice("scanner.ignore_dirs"),
				MaxDepth:      viper.GetInt("scanner.max_depth"),
			},
			Logging: config.LoggingConfig{
				Level:  viper.GetString("logging.level"),
				ToFile: viper.GetBool("logging.to_file"),
				JSON:   viper.GetBool("logging.json"),
			},
		}
		fmt.Fprintf(os.Stderr, "Config loaded from Viper directly - Logging.ToFile=%v\n", cfg.Logging.ToFile)
	}

	// Debug: Print config values to stderr before logger init (for debugging)
	// This helps verify config is being read correctly
	if cfg != nil {
		fmt.Fprintf(os.Stderr, "DEBUG: Config loaded - Logging.ToFile=%v, Logging.Level=%s\n", cfg.Logging.ToFile, cfg.Logging.Level)
		// Also check what Viper has directly
		fmt.Fprintf(os.Stderr, "DEBUG: Viper logging.to_file=%v\n", viper.GetBool("logging.to_file"))
	}

	// Initialize logger from config
	if cfg != nil {
		if err := logger.Init(cfg.Logging.Level, cfg.Logging.JSON, cfg.Logging.ToFile); err != nil {
			// Log error but don't fail - continue with stderr logging
			logPath, pathErr := logger.GetLogFilePath()
			if pathErr == nil {
				fmt.Fprintf(os.Stderr, "ERROR: failed to initialize file logging (log path: %s): %v\n", logPath, err)
			} else {
				fmt.Fprintf(os.Stderr, "ERROR: failed to initialize file logging: %v\n", err)
			}
			logger.Init(cfg.Logging.Level, cfg.Logging.JSON, false) // Fallback to stderr only
		} else if cfg.Logging.ToFile {
			// Log successful file logging initialization (only if enabled)
			logPath, err := logger.GetLogFilePath()
			if err == nil {
				fmt.Fprintf(os.Stderr, "INFO: File logging enabled - log path: %s\n", logPath)
				// Now that logger is initialized, also log it
				logger.Info("file logging enabled", "path", logPath)
			}
		} else {
			fmt.Fprintf(os.Stderr, "DEBUG: File logging is disabled (to_file=false)\n")
		}
	} else {
		logger.Init("info", false, false) // default: info level, text format, no file
	}
}

// getWorkspacePath returns the workspace path following the precedence order:
// 1. CLI flag (--workspace)
// 2. Environment variable (THANDIE_WORKSPACE)
// 3. Config file (workspace.default)
// 4. Default ($HOME/Workspace)
func getWorkspacePath() string {
	// 1. Check CLI flag (highest precedence)
	if workspacePath != "" {
		return workspacePath
	}

	// 2. Check environment variable directly (explicit precedence)
	if envPath := os.Getenv("THANDIE_WORKSPACE"); envPath != "" {
		return envPath
	}

	// 3. Check config file
	if cfg != nil && cfg.Workspace.Default != "" {
		return cfg.Workspace.Default
	}

	// 4. Default fallback
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "." // Last resort: current directory
	}
	return filepath.Join(homeDir, "Workspace")
}
