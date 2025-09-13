package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/ports"
	"ramp/internal/ui"
)

var runCmd = &cobra.Command{
	Use:   "run <command-name> [feature-name]",
	Short: "Run a custom command defined in the configuration",
	Long: `Run a custom command defined in the ramp.yaml configuration.

If a feature name is provided, the command is executed from within that
feature's trees directory with access to feature-specific environment variables.

If no feature name is provided, the command is executed from the source
directory with access to source repository paths.

Example:
  ramp run open my-feature    # Run 'open' command for 'my-feature'
  ramp run deploy             # Run 'deploy' command against source repos`,
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

	// If no feature name provided, run against source directory
	if featureName == "" {
		progress := ui.NewProgress()
		progress.Start(fmt.Sprintf("Running command '%s' against source repositories", commandName))
		
		if err := runCommandInSource(projectDir, command.Command, progress); err != nil {
			progress.Error(fmt.Sprintf("Command '%s' failed", commandName))
			return fmt.Errorf("command '%s' failed: %w", commandName, err)
		}

		progress.Success(fmt.Sprintf("Command '%s' completed successfully!", commandName))
		return nil
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

	// Add RAMP_PORT environment variable
	cfg, err := config.LoadConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load config for env vars: %w", err)
	}

	portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
	if err != nil {
		return fmt.Errorf("failed to initialize port allocations for env vars: %w", err)
	}

	if port, exists := portAllocations.GetPort(featureName); exists {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_PORT=%d", port))
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

func runCommandInSource(projectDir, commandScript string, progress *ui.ProgressUI) error {
	scriptPath := filepath.Join(projectDir, ".ramp", commandScript)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("command script not found: %s", scriptPath)
	}

	// Run from the project directory (where source repos are located)
	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = projectDir
	
	// Set up environment variables for source directory execution
	cmd.Env = append(os.Environ(), fmt.Sprintf("RAMP_PROJECT_DIR=%s", projectDir))
	
	// Load config to get repository paths
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