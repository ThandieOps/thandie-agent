package config

// Config represents the application configuration structure
type Config struct {
	Version   int             `mapstructure:"version" yaml:"version"`
	Workspace WorkspaceConfig `mapstructure:"workspace" yaml:"workspace"`
	Scanner   ScannerConfig   `mapstructure:"scanner" yaml:"scanner"`
	Logging   LoggingConfig   `mapstructure:"logging" yaml:"logging"`
}

// WorkspaceConfig holds workspace-related settings
type WorkspaceConfig struct {
	Default  string             `mapstructure:"default" yaml:"default"`
	Profiles []WorkspaceProfile `mapstructure:"profiles" yaml:"profiles"`
}

// WorkspaceProfile represents a named workspace profile (for future use)
type WorkspaceProfile struct {
	Name string   `mapstructure:"name" yaml:"name"`
	Path string   `mapstructure:"path" yaml:"path"`
	Tags []string `mapstructure:"tags" yaml:"tags,omitempty"`
}

// ScannerConfig holds scanner-related settings
type ScannerConfig struct {
	IncludeHidden bool     `mapstructure:"include_hidden" yaml:"include_hidden"`
	IgnoreDirs    []string `mapstructure:"ignore_dirs" yaml:"ignore_dirs"`
	MaxDepth      int      `mapstructure:"max_depth" yaml:"max_depth"`
}

// LoggingConfig holds logging-related settings
type LoggingConfig struct {
	Level string `mapstructure:"level" yaml:"level"`
}
