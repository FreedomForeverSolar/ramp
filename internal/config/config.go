package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Repo struct {
	Path          string `yaml:"path"`
	Git           string `yaml:"git"`
	DefaultBranch string `yaml:"default_branch"`
}

type Command struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
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
			return dir, nil
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