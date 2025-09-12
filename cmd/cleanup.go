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
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup <feature-name>",
	Short: "Clean up a feature branch by removing worktrees and branches",
	Long: `Clean up a feature branch by:
1. Running the cleanup script (if configured)
2. Removing worktree directories from trees/<feature-name>/
3. Removing the feature branches that were created
4. Prompting for confirmation if there are uncommitted changes`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		featureName := args[0]
		if err := runCleanup(featureName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
}

func runCleanup(featureName string) error {
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

	// Get config prefix for fallback when branch detection fails
	configPrefix := cfg.GetBranchPrefix()

	treesDir := filepath.Join(projectDir, "trees", featureName)

	// Check if trees directory exists
	if _, err := os.Stat(treesDir); os.IsNotExist(err) {
		return fmt.Errorf("feature '%s' not found (trees directory does not exist)", featureName)
	}

	fmt.Printf("Cleaning up feature '%s' for project '%s'\n", featureName, cfg.Name)

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
		fmt.Printf("Running cleanup script: %s\n", cfg.Cleanup)
		if err := runCleanupScript(projectDir, treesDir, cfg.Cleanup); err != nil {
			fmt.Printf("Warning: cleanup script failed: %v\n", err)
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
					fmt.Printf("  %s: detected branch %s\n", name, branchName)
				} else {
					// Fallback to constructed branch name
					branchName = configPrefix + featureName
					fmt.Printf("  %s: could not detect branch, using fallback %s\n", name, branchName)
				}
				
				// Remove worktree
				fmt.Printf("  %s: removing worktree\n", name)
				if err := git.RemoveWorktree(repoDir, worktreeDir); err != nil {
					fmt.Printf("    Warning: failed to remove worktree: %v\n", err)
				}
			} else {
				// No worktree exists, use fallback branch name
				branchName = configPrefix + featureName
				fmt.Printf("  %s: no worktree found, using fallback branch %s\n", name, branchName)
			}

			// Delete branch
			fmt.Printf("  %s: deleting branch %s\n", name, branchName)
			if err := git.DeleteBranch(repoDir, branchName); err != nil {
				fmt.Printf("    Warning: failed to delete branch: %v\n", err)
			}
		}
	}

	// Remove trees directory
	fmt.Printf("Removing trees directory: %s\n", treesDir)
	if err := os.RemoveAll(treesDir); err != nil {
		return fmt.Errorf("failed to remove trees directory: %w", err)
	}

	fmt.Printf("✅ Feature '%s' cleaned up successfully!\n", featureName)
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
					fmt.Printf("⚠️  Uncommitted changes found in %s\n", name)
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

	return cmd.Run()
}