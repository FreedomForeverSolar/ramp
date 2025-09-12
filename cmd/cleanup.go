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

var cleanupPrefixFlag string

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
		if err := runCleanup(featureName, cleanupPrefixFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.Flags().StringVar(&cleanupPrefixFlag, "prefix", "", "Override the branch prefix (defaults to config default_branch_prefix)")
}

func runCleanup(featureName, prefix string) error {
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

	// Determine effective prefix - flag takes precedence, then config, then empty
	effectivePrefix := prefix
	if effectivePrefix == "" {
		effectivePrefix = cfg.GetBranchPrefix()
	}

	sourceDir := filepath.Join(projectDir, "source")
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
	branchName := effectivePrefix + featureName
	repos := cfg.GetRepos()
	for name := range repos {
		repoDir := filepath.Join(sourceDir, name)
		worktreeDir := filepath.Join(treesDir, name)

		if git.IsGitRepo(repoDir) {
			// Remove worktree
			if _, err := os.Stat(worktreeDir); err == nil {
				fmt.Printf("  %s: removing worktree\n", name)
				if err := git.RemoveWorktree(repoDir, worktreeDir); err != nil {
					fmt.Printf("    Warning: failed to remove worktree: %v\n", err)
				}
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
	sourceDir := filepath.Join(projectDir, "source")

	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = treesDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Set up environment variables that the cleanup script expects
	cmd.Env = append(os.Environ(), fmt.Sprintf("RAMP_PROJECT_DIR=%s", projectDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_TREES_DIR=%s", treesDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_WORKTREE_NAME=%s", featureName))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_ROOT_SOURCE_PATH=%s", sourceDir))

	return cmd.Run()
}