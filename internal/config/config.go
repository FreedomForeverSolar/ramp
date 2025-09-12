package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Repo struct {
	Path          string `yaml:"path"`
	DefaultBranch string `yaml:"default_branch"`
}

type Config struct {
	Name  string  `yaml:"name"`
	Repos []*Repo `yaml:"repos"`
	Setup string  `yaml:"setup,omitempty"`
}

func (c *Config) GetRepos() map[string]*Repo {
	result := make(map[string]*Repo)
	for _, repo := range c.Repos {
		name := extractRepoName(repo.Path)
		result[name] = repo
	}
	return result
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