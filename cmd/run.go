package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/ui"
)

var runCmd = &cobra.Command{
	Use:   "run <command-name> [feature-name]",
	Short: "Run a custom command defined in the configuration",
	Long: `Run a custom command defined in the ramp.yaml configuration.

Commands are executed from within the feature's trees directory with access
to the same environment variables as setup and cleanup scripts.

If no feature name is provided, attempts to detect the current feature
from the working directory if you're inside a trees directory.

Example:
  ramp run open my-feature    # Run 'open' command for 'my-feature'
  ramp run deploy             # Run 'deploy' command for current feature`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		commandName := args[0]
		var featureName string
		if len(args) > 1 {
			featureName = args[1]
		}
		
		if err := runCustomCommand(commandName, featureName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runCustomCommand(commandName, featureName string) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	projectDir, err := config.FindRampProject(wd)
	if err != nil {
		return err
	}

	cfg, err := config.LoadConfig(projectDir)
	if err != nil {
		return err
	}

	// Find the command in the configuration
	command := cfg.GetCommand(commandName)
	if command == nil {
		return fmt.Errorf("command '%s' not found in configuration", commandName)
	}

	// Auto-initialize if needed
	if err := autoInitializeIfNeeded(projectDir, cfg); err != nil {
		return fmt.Errorf("auto-initialization failed: %w", err)
	}

	// If no feature name provided, try to detect from current directory
	if featureName == "" {
		if detectedFeature, err := detectFeatureFromWorkingDir(wd, projectDir); err == nil {
			featureName = detectedFeature
		} else {
			return fmt.Errorf("no feature name provided and could not detect from current directory: %w", err)
		}
	}

	treesDir := filepath.Join(projectDir, "trees", featureName)

	// Check if trees directory exists
	if _, err := os.Stat(treesDir); os.IsNotExist(err) {
		return fmt.Errorf("feature '%s' not found (trees directory does not exist)", featureName)
	}

	progress := ui.NewProgress()
	progress.Start(fmt.Sprintf("Running command '%s' for feature '%s'", commandName, featureName))
	
	if err := runCommandWithEnv(projectDir, treesDir, command.Command, progress); err != nil {
		progress.Error(fmt.Sprintf("Command '%s' failed", commandName))
		return fmt.Errorf("command '%s' failed: %w", commandName, err)
	}

	progress.Success(fmt.Sprintf("Command '%s' completed successfully!", commandName))
	return nil
}

func detectFeatureFromWorkingDir(wd, projectDir string) (string, error) {
	// Check if we're inside a trees directory
	treesDir := filepath.Join(projectDir, "trees")
	if !strings.HasPrefix(wd, treesDir) {
		return "", fmt.Errorf("not inside a trees directory")
	}

	// Extract feature name from path
	relPath, err := filepath.Rel(treesDir, wd)
	if err != nil {
		return "", fmt.Errorf("failed to determine relative path: %w", err)
	}

	// The feature name is the first directory component
	pathParts := strings.Split(relPath, string(filepath.Separator))
	if len(pathParts) == 0 || pathParts[0] == "." {
		return "", fmt.Errorf("could not extract feature name from path")
	}

	return pathParts[0], nil
}

func runCommandWithEnv(projectDir, treesDir, commandScript string, progress *ui.ProgressUI) error {
	scriptPath := filepath.Join(projectDir, ".ramp", commandScript)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("command script not found: %s", scriptPath)
	}

	// Extract feature name from treesDir path
	featureName := filepath.Base(treesDir)

	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = treesDir
	
	// Set up environment variables that the command script expects
	cmd.Env = append(os.Environ(), fmt.Sprintf("RAMP_PROJECT_DIR=%s", projectDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_TREES_DIR=%s", treesDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_WORKTREE_NAME=%s", featureName))

	// Add dynamic repository path environment variables
	cfg, err := config.LoadConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load config for env vars: %w", err)
	}
	
	repos := cfg.GetRepos()
	for name, repo := range repos {
		envVarName := config.GenerateEnvVarName(name)
		repoPath := repo.GetRepoPath(projectDir)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVarName, repoPath))
	}

	message := fmt.Sprintf("Running command script: %s", commandScript)
	return ui.RunCommandWithProgress(cmd, message)
}