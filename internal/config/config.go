package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type EnvFile struct {
	Source  string            `yaml:"source"`
	Dest    string            `yaml:"dest"`
	Cache   string            `yaml:"cache,omitempty"`
	Replace map[string]string `yaml:"replace,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling to support both simple string
// syntax (e.g., "- .env") and full object syntax (e.g., "- source: .env")
func (e *EnvFile) UnmarshalYAML(node *yaml.Node) error {
	// Try to unmarshal as a string first (simple syntax)
	var simpleStr string
	if err := node.Decode(&simpleStr); err == nil {
		// Simple string syntax: use same value for both source and dest
		e.Source = simpleStr
		e.Dest = simpleStr
		e.Replace = nil
		return nil
	}

	// If not a string, try to unmarshal as an object (full syntax)
	type envFileAlias EnvFile // Prevent recursion
	var alias envFileAlias
	if err := node.Decode(&alias); err != nil {
		return err
	}

	*e = EnvFile(alias)
	return nil
}

type Repo struct {
	Path        string    `yaml:"path"`
	Git         string    `yaml:"git"`
	AutoRefresh *bool     `yaml:"auto_refresh,omitempty"`
	EnvFiles    []EnvFile `yaml:"env_files,omitempty"`
}

type Command struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
}

type PromptOption struct {
	Value string `yaml:"value"`
	Label string `yaml:"label"`
}

type Prompt struct {
	Name     string          `yaml:"name"`
	Question string          `yaml:"question"`
	Options  []*PromptOption `yaml:"options"`
	Default  string          `yaml:"default,omitempty"`
}

type Config struct {
	Name                string     `yaml:"name"`
	Repos               []*Repo    `yaml:"repos"`
	Setup               string     `yaml:"setup,omitempty"`
	Cleanup             string     `yaml:"cleanup,omitempty"`
	DefaultBranchPrefix string     `yaml:"default-branch-prefix,omitempty"`
	Commands            []*Command `yaml:"commands,omitempty"`
	BasePort            int        `yaml:"base_port,omitempty"`
	MaxPorts            int        `yaml:"max_ports,omitempty"`
	Prompts             []*Prompt  `yaml:"prompts,omitempty"`
}

type LocalConfig struct {
	Preferences map[string]string `yaml:"preferences"`
}

func (c *Config) GetRepos() map[string]*Repo {
	result := make(map[string]*Repo)
	for _, repo := range c.Repos {
		name := extractRepoName(repo.Git)
		result[name] = repo
	}
	return result
}

func (c *Config) GetBranchPrefix() string {
	return c.DefaultBranchPrefix
}

func (c *Config) GetCommand(name string) *Command {
	for _, cmd := range c.Commands {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}

func (c *Config) GetBasePort() int {
	if c.BasePort <= 0 {
		return 3000 // Default base port
	}
	return c.BasePort
}

func (c *Config) GetMaxPorts() int {
	if c.MaxPorts <= 0 {
		return 100 // Default max ports
	}
	return c.MaxPorts
}

func (c *Config) HasPortConfig() bool {
	return c.BasePort > 0 || c.MaxPorts > 0
}

func (c *Config) HasPrompts() bool {
	return len(c.Prompts) > 0
}

func extractRepoName(repoPath string) string {
	// Handle git@github.com:owner/repo.git format
	if strings.Contains(repoPath, ":") {
		parts := strings.Split(repoPath, ":")
		if len(parts) > 1 {
			repoPath = parts[1]
		}
	}
	
	// Remove .git suffix
	repoPath = strings.TrimSuffix(repoPath, ".git")
	
	// Extract repo name from owner/repo format
	parts := strings.Split(repoPath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	
	return repoPath
}

func LoadConfig(projectDir string) (*Config, error) {
	configPath := filepath.Join(projectDir, ".ramp", "ramp.yaml")
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	return &config, nil
}

func FindRampProject(startDir string) (string, error) {
	dir := startDir

	for {
		rampDir := filepath.Join(dir, ".ramp")
		configFile := filepath.Join(rampDir, "ramp.yaml")

		if _, err := os.Stat(configFile); err == nil {
			// Resolve symlinks to ensure canonical path
			// This is important on macOS where /var is a symlink to /private/var
			canonicalDir, err := filepath.EvalSymlinks(dir)
			if err != nil {
				// If we can't resolve symlinks, return the original path
				return dir, nil
			}
			return canonicalDir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no ramp project found (looking for .ramp/ramp.yaml)")
}

// GetRepoPath returns the absolute path where a repository should be located
func (r *Repo) GetRepoPath(projectDir string) string {
	repoName := extractRepoName(r.Git)
	return filepath.Join(projectDir, r.Path, repoName)
}

// GetGitURL returns the git URL for cloning
func (r *Repo) GetGitURL() string {
	return r.Git
}

// ShouldAutoRefresh returns true if this repository should be auto-refreshed.
// Defaults to true if not explicitly set to false.
func (r *Repo) ShouldAutoRefresh() bool {
	if r.AutoRefresh == nil {
		return true // Default to true
	}
	return *r.AutoRefresh
}

// GenerateEnvVarName generates an environment variable name from a repo name
func GenerateEnvVarName(repoName string) string {
	// Convert to uppercase and replace hyphens with underscores
	re := regexp.MustCompile(`[^A-Za-z0-9_]`)
	cleaned := re.ReplaceAllString(repoName, "_")
	cleaned = strings.ToUpper(cleaned)

	// Remove multiple consecutive underscores
	re = regexp.MustCompile(`_{2,}`)
	cleaned = re.ReplaceAllString(cleaned, "_")

	// Trim leading/trailing underscores
	cleaned = strings.Trim(cleaned, "_")

	return "RAMP_REPO_PATH_" + cleaned
}

// SaveConfig writes a Config structure to ramp.yaml with nice formatting
func SaveConfig(cfg *Config, projectDir string) error {
	configPath := filepath.Join(projectDir, ".ramp", "ramp.yaml")

	// Ensure .ramp directory exists
	rampDir := filepath.Join(projectDir, ".ramp")
	if err := os.MkdirAll(rampDir, 0755); err != nil {
		return fmt.Errorf("failed to create .ramp directory: %w", err)
	}

	// Build YAML manually for better formatting
	var yamlBuilder strings.Builder

	// Project name
	yamlBuilder.WriteString(fmt.Sprintf("name: %s\n", cfg.Name))

	// Repos section
	if len(cfg.Repos) > 0 {
		yamlBuilder.WriteString("repos:\n")
		for _, repo := range cfg.Repos {
			yamlBuilder.WriteString(fmt.Sprintf("  - path: %s\n", repo.Path))
			yamlBuilder.WriteString(fmt.Sprintf("    git: %s\n", repo.Git))
			if repo.AutoRefresh != nil {
				yamlBuilder.WriteString(fmt.Sprintf("    auto_refresh: %t\n", *repo.AutoRefresh))
			}
			if len(repo.EnvFiles) > 0 {
				yamlBuilder.WriteString("    env_files:\n")
				for _, envFile := range repo.EnvFiles {
					// Simple syntax if source and dest are the same, no cache, and no replacements
					if envFile.Source == envFile.Dest && envFile.Cache == "" && len(envFile.Replace) == 0 {
						yamlBuilder.WriteString(fmt.Sprintf("      - %s\n", envFile.Source))
					} else {
						// Full object syntax
						yamlBuilder.WriteString(fmt.Sprintf("      - source: %s\n", envFile.Source))
						yamlBuilder.WriteString(fmt.Sprintf("        dest: %s\n", envFile.Dest))
						if envFile.Cache != "" {
							yamlBuilder.WriteString(fmt.Sprintf("        cache: %s\n", envFile.Cache))
						}
						if len(envFile.Replace) > 0 {
							yamlBuilder.WriteString("        replace:\n")
							for key, value := range envFile.Replace {
								yamlBuilder.WriteString(fmt.Sprintf("          %s: %q\n", key, value))
							}
						}
					}
				}
			}
		}
		yamlBuilder.WriteString("\n")
	}

	// Branch prefix
	if cfg.DefaultBranchPrefix != "" {
		yamlBuilder.WriteString(fmt.Sprintf("default-branch-prefix: %s\n", cfg.DefaultBranchPrefix))
	}

	// Port configuration
	if cfg.BasePort > 0 {
		yamlBuilder.WriteString(fmt.Sprintf("base_port: %d\n", cfg.BasePort))
	}
	if cfg.MaxPorts > 0 {
		yamlBuilder.WriteString(fmt.Sprintf("max_ports: %d\n", cfg.MaxPorts))
	}

	// Setup and cleanup scripts
	if cfg.Setup != "" {
		yamlBuilder.WriteString(fmt.Sprintf("setup: %s\n", cfg.Setup))
	}
	if cfg.Cleanup != "" {
		yamlBuilder.WriteString(fmt.Sprintf("cleanup: %s\n", cfg.Cleanup))
	}

	// Commands section
	if len(cfg.Commands) > 0 {
		yamlBuilder.WriteString("\ncommands:\n")
		for _, cmd := range cfg.Commands {
			yamlBuilder.WriteString(fmt.Sprintf("  - name: %s\n", cmd.Name))
			yamlBuilder.WriteString(fmt.Sprintf("    command: %s\n", cmd.Command))
		}
	}

	// Prompts section
	if len(cfg.Prompts) > 0 {
		yamlBuilder.WriteString("\nprompts:\n")
		for _, prompt := range cfg.Prompts {
			yamlBuilder.WriteString(fmt.Sprintf("  - name: %s\n", prompt.Name))
			yamlBuilder.WriteString(fmt.Sprintf("    question: %q\n", prompt.Question))
			yamlBuilder.WriteString("    options:\n")
			for _, opt := range prompt.Options {
				yamlBuilder.WriteString(fmt.Sprintf("      - value: %s\n", opt.Value))
				yamlBuilder.WriteString(fmt.Sprintf("        label: %s\n", opt.Label))
			}
			if prompt.Default != "" {
				yamlBuilder.WriteString(fmt.Sprintf("    default: %s\n", prompt.Default))
			}
		}
	}

	// Write to file
	if err := os.WriteFile(configPath, []byte(yamlBuilder.String()), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadLocalConfig loads the local.yaml configuration file.
// Returns nil if the file doesn't exist (not an error).
func LoadLocalConfig(projectDir string) (*LocalConfig, error) {
	localPath := filepath.Join(projectDir, ".ramp", "local.yaml")

	// If file doesn't exist, return nil (not an error)
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read local config file %s: %w", localPath, err)
	}

	var localCfg LocalConfig
	if err := yaml.Unmarshal(data, &localCfg); err != nil {
		return nil, fmt.Errorf("failed to parse local config file %s: %w", localPath, err)
	}

	return &localCfg, nil
}

// SaveLocalConfig writes a LocalConfig structure to local.yaml
func SaveLocalConfig(localCfg *LocalConfig, projectDir string) error {
	localPath := filepath.Join(projectDir, ".ramp", "local.yaml")

	// Ensure .ramp directory exists
	rampDir := filepath.Join(projectDir, ".ramp")
	if err := os.MkdirAll(rampDir, 0755); err != nil {
		return fmt.Errorf("failed to create .ramp directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(localCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal local config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(localPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write local config file: %w", err)
	}

	return nil
}