package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/git"
	"ramp/internal/ports"
	"ramp/internal/ui"
)

var prefixFlag string

var upCmd = &cobra.Command{
	Use:   "up <feature-name>",
	Short: "Create a new feature branch with git worktrees for all repositories",
	Long: `Create a new feature branch by creating git worktrees for all repositories
from their configured locations. This creates isolated working directories for each repo
in the trees/<feature-name>/ directory.

After creating worktrees, runs any setup script specified in the configuration.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		featureName := args[0]
		if err := runUp(featureName, prefixFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVar(&prefixFlag, "prefix", "", "Override the branch prefix (defaults to config default_branch_prefix)")
}

func runUp(featureName, prefix string) error {
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

	// Auto-initialize if needed
	if err := autoInitializeIfNeeded(projectDir, cfg); err != nil {
		return fmt.Errorf("auto-initialization failed: %w", err)
	}

	// Determine effective prefix - flag takes precedence, then config, then empty
	effectivePrefix := prefix
	if effectivePrefix == "" {
		effectivePrefix = cfg.GetBranchPrefix()
	}

	treesDir := filepath.Join(projectDir, "trees", featureName)

	if err := os.MkdirAll(treesDir, 0755); err != nil {
		return fmt.Errorf("failed to create trees directory: %w", err)
	}

	progress := ui.NewProgress()
	progress.Start(fmt.Sprintf("Creating feature '%s' for project '%s'", featureName, cfg.Name))
	progress.Success(fmt.Sprintf("Creating feature '%s' for project '%s'", featureName, cfg.Name))
	
	progress.Start(fmt.Sprintf("Creating worktrees in %s", treesDir))

	repos := cfg.GetRepos()
	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		worktreeDir := filepath.Join(treesDir, name)

		if !git.IsGitRepo(repoDir) {
			progress.Error(fmt.Sprintf("Source repo not found at %s even after auto-initialization", repoDir))
			return fmt.Errorf("source repo not found at %s even after auto-initialization", repoDir)
		}

		branchName := effectivePrefix + featureName
		
		// Check branch status to provide informative message
		localExists, err := git.LocalBranchExists(repoDir, branchName)
		if err != nil {
			progress.Error(fmt.Sprintf("Failed to check local branch for %s", name))
			return fmt.Errorf("failed to check local branch for %s: %w", name, err)
		}
		
		remoteExists, err := git.RemoteBranchExists(repoDir, branchName)
		if err != nil {
			progress.Error(fmt.Sprintf("Failed to check remote branch for %s", name))
			return fmt.Errorf("failed to check remote branch for %s: %w", name, err)
		}

		// Show detailed branch info in verbose mode or as info messages
		if localExists {
			progress.Info(fmt.Sprintf("%s: creating worktree with existing local branch %s", name, branchName))
		} else if remoteExists {
			progress.Info(fmt.Sprintf("%s: creating worktree with existing remote branch %s", name, branchName))
		} else {
			progress.Info(fmt.Sprintf("%s: creating worktree with new branch %s", name, branchName))
		}

		if err := git.CreateWorktree(repoDir, worktreeDir, branchName); err != nil {
			progress.Error(fmt.Sprintf("Failed to create worktree for %s", name))
			return fmt.Errorf("failed to create worktree for %s: %w", name, err)
		}
	}

	progress.Success("Creating worktrees")

	// Allocate port for this feature only if port configuration is present
	var allocatedPort int
	if cfg.HasPortConfig() {
		progress.Start("Allocating port for feature")
		portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
		if err != nil {
			progress.Error("Failed to initialize port allocations")
			return fmt.Errorf("failed to initialize port allocations: %w", err)
		}

		allocatedPort, err = portAllocations.AllocatePort(featureName)
		if err != nil {
			progress.Error("Failed to allocate port")
			return fmt.Errorf("failed to allocate port for feature: %w", err)
		}
		progress.Success(fmt.Sprintf("Allocated port %d for feature", allocatedPort))
	}

	if cfg.Setup != "" {
		if err := runSetupScriptWithProgress(projectDir, treesDir, cfg.Setup, progress); err != nil {
			progress.Error("Setup script failed")
			return fmt.Errorf("setup script failed: %w", err)
		}
	}

	progress.Success(fmt.Sprintf("Feature '%s' created successfully!", featureName))
	progress.Info(fmt.Sprintf("📁 Worktrees are located in: %s", treesDir))
	return nil
}

func runSetupScript(projectDir, treesDir, setupScript string) error {
	scriptPath := filepath.Join(projectDir, ".ramp", setupScript)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("setup script not found: %s", scriptPath)
	}

	// Extract feature name from treesDir path
	featureName := filepath.Base(treesDir)

	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = treesDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up environment variables that the setup script expects
	cmd.Env = append(os.Environ(), fmt.Sprintf("RAMP_PROJECT_DIR=%s", projectDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_TREES_DIR=%s", treesDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_WORKTREE_NAME=%s", featureName))

	// Add RAMP_PORT environment variable only if port configuration exists
	cfg, err := config.LoadConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load config for env vars: %w", err)
	}

	if cfg.HasPortConfig() {
		portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
		if err != nil {
			return fmt.Errorf("failed to initialize port allocations for env vars: %w", err)
		}

		if port, exists := portAllocations.GetPort(featureName); exists {
			cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_PORT=%d", port))
		}
	}
	
	repos := cfg.GetRepos()
	for name, repo := range repos {
		envVarName := config.GenerateEnvVarName(name)
		repoPath := repo.GetRepoPath(projectDir)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVarName, repoPath))
	}

	return cmd.Run()
}

func runSetupScriptWithProgress(projectDir, treesDir, setupScript string, progress *ui.ProgressUI) error {
	scriptPath := filepath.Join(projectDir, ".ramp", setupScript)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("setup script not found: %s", scriptPath)
	}

	// Extract feature name from treesDir path
	featureName := filepath.Base(treesDir)

	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = treesDir
	
	// Set up environment variables that the setup script expects
	cmd.Env = append(os.Environ(), fmt.Sprintf("RAMP_PROJECT_DIR=%s", projectDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_TREES_DIR=%s", treesDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_WORKTREE_NAME=%s", featureName))

	// Add RAMP_PORT environment variable only if port configuration exists
	cfg, err := config.LoadConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load config for env vars: %w", err)
	}

	if cfg.HasPortConfig() {
		portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
		if err != nil {
			return fmt.Errorf("failed to initialize port allocations for env vars: %w", err)
		}

		if port, exists := portAllocations.GetPort(featureName); exists {
			cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_PORT=%d", port))
		}
	}
	
	repos := cfg.GetRepos()
	for name, repo := range repos {
		envVarName := config.GenerateEnvVarName(name)
		repoPath := repo.GetRepoPath(projectDir)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVarName, repoPath))
	}

	message := fmt.Sprintf("Running setup script: %s", setupScript)
	return ui.RunCommandWithProgress(cmd, message)
}
