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
// Returns ~/.config/ramp/ramp.yaml
func GetUserConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "ramp", "ramp.yaml"), nil
}

// GetUserConfigDir returns the directory containing user-level ramp config.
// Returns ~/.config/ramp
func GetUserConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "ramp"), nil
}

// LoadUserConfig loads the user-level configuration.
// Returns nil if the file doesn't exist (not an error).
func LoadUserConfig() (*UserConfig, error) {
	userPath, err := GetUserConfigPath()
	if err != nil {
		return nil, nil // Can't determine path, treat as not existing
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
