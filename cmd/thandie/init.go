package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ThandieOps/thandie-agent/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// initCmd represents: `thandie init`
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Thandie configuration file",
	Long: `Initialize Thandie by creating a configuration file with your preferences.
This command will prompt you for configuration values with sensible defaults.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInit(); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Attach the `init` command to the root: thandie init
	rootCmd.AddCommand(initCmd)
}

// runInit handles the interactive initialization process
func runInit() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Default config file location
	defaultConfigPath := filepath.Join(homeDir, ".config", "thandie", "config.yml")

	// Default workspace path
	defaultWorkspace := filepath.Join(homeDir, "Workspace")

	reader := bufio.NewReader(os.Stdin)

	// Prompt for config file location
	fmt.Printf("Config file location [%s]: ", defaultConfigPath)
	configPathInput, _ := reader.ReadString('\n')
	configPathInput = strings.TrimSpace(configPathInput)
	if configPathInput == "" {
		configPathInput = defaultConfigPath
	}

	// Expand ~ to home directory if present
	if strings.HasPrefix(configPathInput, "~/") {
		configPathInput = filepath.Join(homeDir, configPathInput[2:])
	}

	// Prompt for workspace path
	fmt.Printf("Default workspace path [%s]: ", defaultWorkspace)
	workspaceInput, _ := reader.ReadString('\n')
	workspaceInput = strings.TrimSpace(workspaceInput)
	if workspaceInput == "" {
		workspaceInput = defaultWorkspace
	}

	// Expand ~ to home directory if present
	if strings.HasPrefix(workspaceInput, "~/") {
		workspaceInput = filepath.Join(homeDir, workspaceInput[2:])
	}

	// Create config struct with user values and defaults
	cfg := &config.Config{
		Version: 1,
		Workspace: config.WorkspaceConfig{
			Default:  workspaceInput,
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

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPathInput)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if config file already exists
	if _, err := os.Stat(configPathInput); err == nil {
		fmt.Printf("\nConfig file already exists at %s\n", configPathInput)
		fmt.Print("Overwrite? (y/N): ")
		overwriteInput, _ := reader.ReadString('\n')
		overwriteInput = strings.TrimSpace(strings.ToLower(overwriteInput))
		if overwriteInput != "y" && overwriteInput != "yes" {
			fmt.Println("Initialization cancelled.")
			return nil
		}
	}

	// Marshal config to YAML
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPathInput, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("\n✓ Configuration file created successfully at %s\n", configPathInput)
	fmt.Printf("  Default workspace: %s\n", workspaceInput)
	return nil
}
