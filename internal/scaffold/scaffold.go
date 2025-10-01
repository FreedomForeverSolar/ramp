package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"ramp/internal/config"
)

// RepoData holds information about a repository to be configured
type RepoData struct {
	GitURL string
	Path   string
}

// ProjectData holds all information collected during interactive init
type ProjectData struct {
	Name             string
	Repos            []RepoData
	IncludeSetup     bool
	IncludeCleanup   bool
	EnablePorts      bool
	BasePort         int
	BranchPrefix     string
	SampleCommands   []string
}

// CreateProject orchestrates the creation of a new ramp project
func CreateProject(projectDir string, data ProjectData) error {
	if err := CreateDirectoryStructure(projectDir); err != nil {
		return fmt.Errorf("failed to create directory structure: %w", err)
	}

	if err := GenerateConfigFile(projectDir, data); err != nil {
		return fmt.Errorf("failed to generate config file: %w", err)
	}

	if data.IncludeSetup {
		if err := GenerateSetupScript(projectDir, data.Repos); err != nil {
			return fmt.Errorf("failed to generate setup script: %w", err)
		}
	}

	if data.IncludeCleanup {
		if err := GenerateCleanupScript(projectDir, data.Repos); err != nil {
			return fmt.Errorf("failed to generate cleanup script: %w", err)
		}
	}

	for _, cmdName := range data.SampleCommands {
		if err := GenerateSampleCommand(projectDir, cmdName, data.Repos); err != nil {
			return fmt.Errorf("failed to generate sample command %s: %w", cmdName, err)
		}
	}

	return nil
}

// CreateDirectoryStructure creates the basic ramp project structure
func CreateDirectoryStructure(projectDir string) error {
	dirs := []string{
		filepath.Join(projectDir, ".ramp", "scripts"),
		filepath.Join(projectDir, "repos"),
		filepath.Join(projectDir, "trees"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// GenerateConfigFile creates the ramp.yaml configuration file
func GenerateConfigFile(projectDir string, data ProjectData) error {
	cfg := &config.Config{
		Name:                data.Name,
		DefaultBranchPrefix: data.BranchPrefix,
	}

	// Add repositories
	for _, repo := range data.Repos {
		autoRefresh := true
		cfg.Repos = append(cfg.Repos, &config.Repo{
			Path:        repo.Path,
			Git:         repo.GitURL,
			AutoRefresh: &autoRefresh,
		})
	}

	// Add optional features
	if data.IncludeSetup {
		cfg.Setup = "scripts/setup.sh"
	}

	if data.IncludeCleanup {
		cfg.Cleanup = "scripts/cleanup.sh"
	}

	if data.EnablePorts {
		cfg.BasePort = data.BasePort
		cfg.MaxPorts = 100 // Default max ports
	}

	// Add sample commands
	for _, cmdName := range data.SampleCommands {
		cfg.Commands = append(cfg.Commands, &config.Command{
			Name:    cmdName,
			Command: fmt.Sprintf("scripts/%s.sh", cmdName),
		})
	}

	return config.SaveConfig(cfg, projectDir)
}

// GenerateSetupScript creates a sample setup script
func GenerateSetupScript(projectDir string, repos []RepoData) error {
	scriptPath := filepath.Join(projectDir, ".ramp", "scripts", "setup.sh")
	content := setupScriptTemplate(repos)
	return writeExecutableScript(scriptPath, content)
}

// GenerateCleanupScript creates a sample cleanup script
func GenerateCleanupScript(projectDir string, repos []RepoData) error {
	scriptPath := filepath.Join(projectDir, ".ramp", "scripts", "cleanup.sh")
	content := cleanupScriptTemplate(repos)
	return writeExecutableScript(scriptPath, content)
}

// GenerateSampleCommand creates a sample custom command script
func GenerateSampleCommand(projectDir, commandName string, repos []RepoData) error {
	scriptPath := filepath.Join(projectDir, ".ramp", "scripts", fmt.Sprintf("%s.sh", commandName))
	content := sampleCommandTemplate(commandName, repos)
	return writeExecutableScript(scriptPath, content)
}

// writeExecutableScript writes a script file with executable permissions
func writeExecutableScript(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write script %s: %w", path, err)
	}
	return nil
}
