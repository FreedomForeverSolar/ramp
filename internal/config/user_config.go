package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// UserConfig represents user-level configuration that applies across all projects.
// Only commands and hooks are allowed - project-specific settings like repos
// must be defined in project config.
type UserConfig struct {
	Commands []*Command `yaml:"commands,omitempty"`
	Hooks    []*Hook    `yaml:"hooks,omitempty"`
}

// GetUserConfigPath returns the path to user-level ramp config.
// Returns ~/.config/ramp/ramp.yaml, or uses RAMP_USER_CONFIG_DIR if set.
// If RAMP_USER_CONFIG_DIR is set to empty string, returns empty to disable user config.
func GetUserConfigPath() (string, error) {
	dir, err := GetUserConfigDir()
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", nil // User config disabled
	}
	return filepath.Join(dir, "ramp.yaml"), nil
}

// GetUserConfigDir returns the directory containing user-level ramp config.
// Returns ~/.config/ramp by default, or RAMP_USER_CONFIG_DIR if set.
// If RAMP_USER_CONFIG_DIR is set to empty string, returns empty to disable user config.
func GetUserConfigDir() (string, error) {
	// Check for override (useful for testing)
	if envDir, ok := os.LookupEnv("RAMP_USER_CONFIG_DIR"); ok {
		return envDir, nil // May be empty to disable user config
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "ramp"), nil
}

// LoadUserConfig loads the user-level configuration.
// Returns nil if the file doesn't exist (not an error).
// Returns nil if user config is disabled via RAMP_USER_CONFIG_DIR="".
func LoadUserConfig() (*UserConfig, error) {
	userPath, err := GetUserConfigPath()
	if err != nil {
		return nil, nil // Can't determine path, treat as not existing
	}
	if userPath == "" {
		return nil, nil // User config disabled
	}

	if _, err := os.Stat(userPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(userPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read user config: %w", err)
	}

	var userCfg UserConfig
	if err := yaml.Unmarshal(data, &userCfg); err != nil {
		return nil, fmt.Errorf("failed to parse user config: %w", err)
	}

	return &userCfg, nil
}
