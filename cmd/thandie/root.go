package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ThandieOps/thandie-agent/internal/config"
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
	// Map THANDIE_WORKSPACE to workspace key
	viper.BindEnv("workspace", "THANDIE_WORKSPACE")

	// Set defaults
	viper.SetDefault("version", 1)
	viper.SetDefault("workspace.default", "")
	viper.SetDefault("scanner.include_hidden", false)
	viper.SetDefault("scanner.ignore_dirs", []string{".git", "node_modules", "vendor"})
	viper.SetDefault("scanner.max_depth", 1)
	viper.SetDefault("logging.level", "info")

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
		// If unmarshaling fails, create a default config
		cfg = &config.Config{
			Version: 1,
			Workspace: config.WorkspaceConfig{
				Default:  "",
				Profiles: []config.WorkspaceProfile{},
			},
			Scanner: config.ScannerConfig{
				IncludeHidden: false,
				IgnoreDirs:    []string{".git", "node_modules", "vendor"},
				MaxDepth:      1,
			},
			Logging: config.LoggingConfig{
				Level: "info",
			},
		}
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
