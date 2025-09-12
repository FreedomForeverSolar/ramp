package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/git"
	"ramp/internal/ui"
)

var renamePrefixFlag string

var renameCmd = &cobra.Command{
	Use:   "rename <old-feature-name> <new-feature-name>",
	Short: "Rename a feature by renaming branches and moving worktrees",
	Long: `Rename a feature by:
1. Renaming git branches across all repositories
2. Moving worktree directories from trees/<old-feature>/ to trees/<new-feature>/
3. Running setup script in the new location (if configured)

This preserves all git history and uncommitted changes in the worktrees.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		oldFeatureName := args[0]
		newFeatureName := args[1]
		if err := runRename(oldFeatureName, newFeatureName, renamePrefixFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
	renameCmd.Flags().StringVar(&renamePrefixFlag, "prefix", "", "Override the branch prefix (defaults to config default_branch_prefix)")
}

func runRename(oldFeatureName, newFeatureName, prefix string) error {
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

	oldTreesDir := filepath.Join(projectDir, "trees", oldFeatureName)
	newTreesDir := filepath.Join(projectDir, "trees", newFeatureName)

	// Validate old feature exists
	if _, err := os.Stat(oldTreesDir); os.IsNotExist(err) {
		return fmt.Errorf("feature '%s' not found (trees directory does not exist)", oldFeatureName)
	}

	// Validate new feature doesn't exist
	if _, err := os.Stat(newTreesDir); err == nil {
		return fmt.Errorf("feature '%s' already exists (trees directory exists)", newFeatureName)
	}

	progress := ui.NewProgress()
	progress.Start(fmt.Sprintf("Renaming feature '%s' to '%s' for project '%s'", oldFeatureName, newFeatureName, cfg.Name))
	progress.Success(fmt.Sprintf("Renaming feature '%s' to '%s' for project '%s'", oldFeatureName, newFeatureName, cfg.Name))

	// Check for uncommitted changes first
	hasUncommitted, err := checkForUncommittedChanges(cfg, oldTreesDir)
	if err != nil {
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	if hasUncommitted {
		progress.Warning("Uncommitted changes found - rename will preserve all changes")
	}

	// Rename branches and move worktrees for each repository
	repos := cfg.GetRepos()
	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		oldWorktreeDir := filepath.Join(oldTreesDir, name)
		newWorktreeDir := filepath.Join(newTreesDir, name)

		if git.IsGitRepo(repoDir) {
			var oldBranchName, newBranchName string

			// Try to detect the actual branch name from the worktree
			if _, err := os.Stat(oldWorktreeDir); err == nil {
				if detectedBranch, err := git.GetWorktreeBranch(oldWorktreeDir); err == nil {
					oldBranchName = detectedBranch
					// Construct new branch name by replacing the old feature name with new one
					if strings.HasSuffix(detectedBranch, oldFeatureName) && strings.HasPrefix(detectedBranch, effectivePrefix) {
						newBranchName = effectivePrefix + newFeatureName
					} else {
						// Fallback: assume it's a simple replacement
						newBranchName = strings.Replace(detectedBranch, oldFeatureName, newFeatureName, 1)
					}
					progress.Info(fmt.Sprintf("%s: detected branch %s, will rename to %s", name, oldBranchName, newBranchName))
				} else {
					// Fallback to constructed branch names
					oldBranchName = effectivePrefix + oldFeatureName
					newBranchName = effectivePrefix + newFeatureName
					progress.Info(fmt.Sprintf("%s: could not detect branch, using fallback %s -> %s", name, oldBranchName, newBranchName))
				}

				// Rename the branch first
				progress.Info(fmt.Sprintf("%s: renaming branch", name))
				if err := git.RenameBranch(repoDir, oldBranchName, newBranchName); err != nil {
					progress.Error(fmt.Sprintf("Failed to rename branch for %s", name))
					return fmt.Errorf("failed to rename branch for %s: %w", name, err)
				}

				// Move the worktree
				progress.Info(fmt.Sprintf("%s: moving worktree", name))
				if err := git.MoveWorktree(repoDir, oldWorktreeDir, newWorktreeDir); err != nil {
					progress.Error(fmt.Sprintf("Failed to move worktree for %s", name))
					return fmt.Errorf("failed to move worktree for %s: %w", name, err)
				}
			} else {
				progress.Warning(fmt.Sprintf("%s: no worktree found at %s", name, oldWorktreeDir))
			}
		} else {
			progress.Warning(fmt.Sprintf("%s: not a git repository at %s", name, repoDir))
		}
	}

	progress.Success("Renamed branches and moved worktrees")

	// The worktree moves above should have moved the individual repo directories,
	// but we may need to handle the case where the parent trees directory structure changed
	// In most cases, git worktree move should handle the directory structure properly

	// Run setup script in new location if configured
	if cfg.Setup != "" {
		if err := runSetupScriptWithProgress(projectDir, newTreesDir, cfg.Setup, progress); err != nil {
			progress.Warning(fmt.Sprintf("Setup script failed: %v", err))
		}
	}

	progress.Success(fmt.Sprintf("Feature '%s' renamed to '%s' successfully!", oldFeatureName, newFeatureName))
	progress.Info(fmt.Sprintf("📁 Worktrees are now located in: %s", newTreesDir))
	return nil
}