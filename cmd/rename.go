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

	// Create the new trees directory before moving worktrees
	if err := os.MkdirAll(newTreesDir, 0755); err != nil {
		return fmt.Errorf("failed to create new trees directory: %w", err)
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

	// Move any remaining files from old feature directory to new one
	// This handles build artifacts, generated files, or other non-git files
	if err := moveRemainingFiles(oldTreesDir, newTreesDir, repos, progress); err != nil {
		progress.Warning(fmt.Sprintf("Failed to move remaining files: %v", err))
	}

	progress.Success(fmt.Sprintf("Feature '%s' renamed to '%s' successfully!", oldFeatureName, newFeatureName))
	progress.Info(fmt.Sprintf("📁 Worktrees are now located in: %s", newTreesDir))
	return nil
}

func moveRemainingFiles(oldTreesDir, newTreesDir string, repos map[string]*config.Repo, progress *ui.ProgressUI) error {
	// Check if old directory exists and has any remaining files/directories
	oldEntries, err := os.ReadDir(oldTreesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to move
		}
		return fmt.Errorf("failed to read old trees directory: %w", err)
	}

	// Get list of repository names to skip (these were moved by git worktree)
	repoNames := make(map[string]bool)
	for name := range repos {
		repoNames[name] = true
	}

	// Move any files/directories that aren't repository worktrees
	for _, entry := range oldEntries {
		if repoNames[entry.Name()] {
			// Skip repository directories - these were moved by git worktree
			continue
		}

		oldPath := filepath.Join(oldTreesDir, entry.Name())
		newPath := filepath.Join(newTreesDir, entry.Name())

		progress.Info(fmt.Sprintf("Moving additional file: %s", entry.Name()))
		
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to move %s: %w", entry.Name(), err)
		}
	}

	// Remove the old trees directory if it's now empty
	if isEmpty, err := isDirEmpty(oldTreesDir); err == nil && isEmpty {
		progress.Info("Removing empty old feature directory")
		if err := os.Remove(oldTreesDir); err != nil {
			progress.Warning(fmt.Sprintf("Failed to remove old directory: %v", err))
		}
	}

	return nil
}

func isDirEmpty(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}