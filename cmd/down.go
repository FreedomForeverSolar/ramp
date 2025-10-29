package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/git"
	"ramp/internal/ports"
	"ramp/internal/ui"
)

var downCmd = &cobra.Command{
	Use:   "down <feature-name>",
	Short: "Clean up a feature branch by removing worktrees and branches",
	Long: `Clean up a feature branch by:
1. Running the cleanup script (if configured)
2. Removing worktree directories from trees/<feature-name>/
3. Removing the feature branches that were created
4. Prompting for confirmation if there are uncommitted changes`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		featureName := strings.TrimRight(args[0], "/")
		if err := runDown(featureName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}

func runDown(featureName string) error {
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

	// Auto-install if needed
	if err := AutoInstallIfNeeded(projectDir, cfg); err != nil {
		return fmt.Errorf("auto-installation failed: %w", err)
	}

	// Get config prefix for fallback when branch detection fails
	configPrefix := cfg.GetBranchPrefix()

	treesDir := filepath.Join(projectDir, "trees", featureName)

	// Check if trees directory exists
	if _, err := os.Stat(treesDir); os.IsNotExist(err) {
		return fmt.Errorf("feature '%s' not found (trees directory does not exist)", featureName)
	}

	progress := ui.NewProgress()
	progress.Start(fmt.Sprintf("Cleaning up feature '%s' for project '%s'", featureName, cfg.Name))
	progress.Success(fmt.Sprintf("Cleaning up feature '%s' for project '%s'", featureName, cfg.Name))

	// Check for uncommitted changes first
	hasUncommitted, err := checkForUncommittedChanges(cfg, treesDir)
	if err != nil {
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	if hasUncommitted {
		if !confirmDeletion(featureName) {
			fmt.Println("Cleanup cancelled.")
			return nil
		}
	}

	// Run cleanup script if configured
	if cfg.Cleanup != "" {
		if err := runCleanupScriptWithProgress(projectDir, treesDir, cfg.Cleanup, progress); err != nil {
			progress.Warning(fmt.Sprintf("Cleanup script failed: %v", err))
		}
	}

	// Remove git worktrees and branches
	repos := cfg.GetRepos()
	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		worktreeDir := filepath.Join(treesDir, name)

		if git.IsGitRepo(repoDir) {
			var branchName string

			// Try to detect the actual branch name from the worktree
			if _, err := os.Stat(worktreeDir); err == nil {
				if detectedBranch, err := git.GetWorktreeBranch(worktreeDir); err == nil {
					branchName = detectedBranch
					progress.Info(fmt.Sprintf("%s: detected branch %s", name, branchName))
				} else {
					// Fallback to constructed branch name
					branchName = configPrefix + featureName
					progress.Info(fmt.Sprintf("%s: could not detect branch, using fallback %s", name, branchName))
				}

				// Remove worktree
				progress.Info(fmt.Sprintf("%s: removing worktree", name))
				if err := git.RemoveWorktree(repoDir, worktreeDir); err != nil {
					progress.Warning(fmt.Sprintf("Failed to remove worktree for %s: %v", name, err))
				}
			} else {
				// No worktree exists, use fallback branch name
				branchName = configPrefix + featureName
				progress.Info(fmt.Sprintf("%s: no worktree found, using fallback branch %s", name, branchName))
			}

			// Delete branch
			progress.Info(fmt.Sprintf("%s: deleting branch %s", name, branchName))
			if err := git.DeleteBranch(repoDir, branchName); err != nil {
				progress.Warning(fmt.Sprintf("Failed to delete branch for %s: %v", name, err))
			}

			// Prune stale remote tracking branches
			if err := git.FetchPrune(repoDir); err != nil {
				progress.Warning(fmt.Sprintf("Failed to prune remote tracking branches for %s: %v", name, err))
			}
		}
	}

	// Release allocated port
	progress.Info("Releasing allocated port")
	portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
	if err != nil {
		progress.Warning(fmt.Sprintf("Failed to initialize port allocations for cleanup: %v", err))
	} else {
		if err := portAllocations.ReleasePort(featureName); err != nil {
			progress.Warning(fmt.Sprintf("Failed to release port: %v", err))
		} else {
			progress.Info("Port released successfully")
		}
	}

	// Remove trees directory
	progress.Info(fmt.Sprintf("Removing trees directory: %s", treesDir))
	if err := os.RemoveAll(treesDir); err != nil {
		progress.Error(fmt.Sprintf("Failed to remove trees directory: %s", treesDir))
		return fmt.Errorf("failed to remove trees directory: %w", err)
	}

	progress.Success(fmt.Sprintf("Feature '%s' cleaned up successfully!", featureName))
	return nil
}

func checkForUncommittedChanges(cfg *config.Config, treesDir string) (bool, error) {
	repos := cfg.GetRepos()
	for name := range repos {
		worktreeDir := filepath.Join(treesDir, name)
		if _, err := os.Stat(worktreeDir); err == nil {
			if git.IsGitRepo(worktreeDir) {
				hasChanges, err := git.HasUncommittedChanges(worktreeDir)
				if err != nil {
					return false, fmt.Errorf("failed to check uncommitted changes in %s: %w", name, err)
				}
				if hasChanges {
					progress := ui.NewProgress()
					progress.Warning(fmt.Sprintf("Uncommitted changes found in %s", name))
					progress.Stop()
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func confirmDeletion(featureName string) bool {
	fmt.Printf("\nThere are uncommitted changes in one or more repositories.\n")
	fmt.Printf("Are you sure you want to delete feature '%s'? This will permanently lose uncommitted changes. (y/N): ", featureName)

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "y" || input == "yes"
}

func runCleanupScript(projectDir, treesDir, cleanupScript string) error {
	scriptPath := filepath.Join(projectDir, ".ramp", cleanupScript)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("cleanup script not found: %s", scriptPath)
	}

	// Extract feature name from treesDir path
	featureName := filepath.Base(treesDir)

	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = treesDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up environment variables that the cleanup script expects
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

	return cmd.Run()
}

func runCleanupScriptWithProgress(projectDir, treesDir, cleanupScript string, progress *ui.ProgressUI) error {
	scriptPath := filepath.Join(projectDir, ".ramp", cleanupScript)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("cleanup script not found: %s", scriptPath)
	}

	// Extract feature name from treesDir path
	featureName := filepath.Base(treesDir)

	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = treesDir

	// Set up environment variables that the cleanup script expects
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

	message := fmt.Sprintf("Running cleanup script: %s", cleanupScript)
	return ui.RunCommandWithProgress(cmd, message)
}
