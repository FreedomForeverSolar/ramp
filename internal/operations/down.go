package operations

import (
	"fmt"
	"os"
	"path/filepath"

	"ramp/internal/config"
	"ramp/internal/features"
	"ramp/internal/git"
	"ramp/internal/ports"
)

// DownOptions configures the feature deletion operation.
type DownOptions struct {
	// Required
	FeatureName string
	ProjectDir  string
	Config      *config.Config
	Progress    ProgressReporter

	// Optional
	Force       bool // Skip uncommitted changes check (already confirmed by caller)
	AutoInstall bool // Auto-install repos if not present (default: false)
}

// DownResult contains the results of feature deletion.
type DownResult struct {
	FeatureName      string
	RemovedWorktrees []string
	DeletedBranches  []string
	ReleasedPort     bool
}

// CheckForUncommittedChanges checks if any worktree has uncommitted changes.
// Returns the list of repos with uncommitted changes.
func CheckForUncommittedChanges(cfg *config.Config, treesDir string) ([]string, error) {
	repos := cfg.GetRepos()
	reposWithChanges := []string{}

	for name := range repos {
		worktreeDir := filepath.Join(treesDir, name)
		if _, err := os.Stat(worktreeDir); err == nil {
			if git.IsGitRepo(worktreeDir) {
				hasChanges, err := git.HasUncommittedChanges(worktreeDir)
				if err != nil {
					return nil, fmt.Errorf("failed to check uncommitted changes in %s: %w", name, err)
				}
				if hasChanges {
					reposWithChanges = append(reposWithChanges, name)
				}
			}
		}
	}

	return reposWithChanges, nil
}

// Down removes a feature with worktrees for all repositories.
// This is the core business logic used by both CLI and UI.
func Down(opts DownOptions) (*DownResult, error) {
	projectDir := opts.ProjectDir
	cfg := opts.Config
	progress := opts.Progress
	featureName := opts.FeatureName

	// Auto-install if requested and needed
	if opts.AutoInstall && !IsProjectInstalled(cfg, projectDir) {
		progress.Start("Repositories not installed, running auto-installation...")
		_, err := Install(InstallOptions{
			ProjectDir: projectDir,
			Config:     cfg,
			Progress:   progress,
		})
		if err != nil {
			return nil, fmt.Errorf("auto-installation failed: %w", err)
		}
	}

	configPrefix := cfg.GetBranchPrefix()
	treesDir := filepath.Join(projectDir, "trees", featureName)

	// Check if trees directory exists
	treesDirExists := true
	if _, err := os.Stat(treesDir); os.IsNotExist(err) {
		treesDirExists = false

		// Check if any worktrees or branches exist for this feature
		repos := cfg.GetRepos()
		featureExists := false
		for name, repo := range repos {
			repoDir := repo.GetRepoPath(projectDir)
			worktreeDir := filepath.Join(treesDir, name)

			if git.IsGitRepo(repoDir) {
				if git.WorktreeRegistered(repoDir, worktreeDir) {
					featureExists = true
					break
				}

				branchName := configPrefix + featureName
				if exists, _ := git.LocalBranchExists(repoDir, branchName); exists {
					featureExists = true
					break
				}
			}
		}

		if !featureExists {
			return nil, fmt.Errorf("feature '%s' not found (trees directory does not exist)", featureName)
		}
	}

	progress.Start(fmt.Sprintf("Cleaning up feature '%s' for project '%s'", featureName, cfg.Name))

	if !treesDirExists {
		progress.Warning(fmt.Sprintf("Trees directory for feature '%s' not found - cleaning up orphaned worktrees", featureName))
	}

	// Check for uncommitted changes if not forced
	if treesDirExists && !opts.Force {
		reposWithChanges, err := CheckForUncommittedChanges(cfg, treesDir)
		if err != nil {
			return nil, err
		}
		for _, name := range reposWithChanges {
			progress.Warning(fmt.Sprintf("Uncommitted changes found in %s", name))
		}
	}

	// Run cleanup script if configured and directory exists
	if cfg.Cleanup != "" && treesDirExists {
		if err := RunCleanupScript(projectDir, treesDir, featureName, cfg, progress); err != nil {
			progress.Warning(fmt.Sprintf("Cleanup script failed: %v", err))
		}
	}

	result := &DownResult{
		FeatureName:      featureName,
		RemovedWorktrees: []string{},
		DeletedBranches:  []string{},
	}

	// Remove git worktrees and branches
	repos := cfg.GetRepos()
	total := len(repos)
	i := 0

	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		worktreeDir := filepath.Join(treesDir, name)

		progress.UpdateWithProgress(fmt.Sprintf("Removing worktree for %s...", name), (i+1)*70/total)

		if git.IsGitRepo(repoDir) {
			var branchName string

			// Try to detect the actual branch name from the worktree
			if _, err := os.Stat(worktreeDir); err == nil {
				if detectedBranch, err := git.GetWorktreeBranch(worktreeDir); err == nil {
					branchName = detectedBranch
					progress.Info(fmt.Sprintf("%s: detected branch %s", name, branchName))
				} else {
					branchName = configPrefix + featureName
					progress.Info(fmt.Sprintf("%s: could not detect branch, using fallback %s", name, branchName))
				}
			} else {
				branchName = configPrefix + featureName
				progress.Info(fmt.Sprintf("%s: worktree directory not found, using fallback branch %s", name, branchName))
			}

			// Remove worktree
			progress.Info(fmt.Sprintf("%s: removing worktree registration", name))
			if err := git.RemoveWorktreeQuiet(repoDir, worktreeDir); err != nil {
				progress.Warning(fmt.Sprintf("Failed to remove worktree for %s: %v", name, err))
				_ = git.PruneWorktrees(repoDir)
			} else {
				result.RemovedWorktrees = append(result.RemovedWorktrees, name)
			}

			// Delete branch
			progress.Info(fmt.Sprintf("%s: deleting branch %s", name, branchName))
			if err := git.DeleteBranchQuiet(repoDir, branchName); err != nil {
				progress.Warning(fmt.Sprintf("Failed to delete branch for %s: %v", name, err))
			} else {
				result.DeletedBranches = append(result.DeletedBranches, branchName)
			}

			// Prune stale remote tracking branches
			if err := git.FetchPruneQuiet(repoDir); err != nil {
				progress.Warning(fmt.Sprintf("Failed to prune remote tracking branches for %s: %v", name, err))
			}
		}
		i++
	}

	// Release allocated port
	progress.UpdateWithProgress("Releasing allocated port...", 80)
	portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
	if err != nil {
		progress.Warning(fmt.Sprintf("Failed to initialize port allocations for cleanup: %v", err))
	} else {
		if err := portAllocations.ReleasePort(featureName); err != nil {
			progress.Warning(fmt.Sprintf("Failed to release port: %v", err))
		} else {
			progress.Info("Port released successfully")
			result.ReleasedPort = true
		}
	}

	// Remove feature metadata (display name, etc.)
	metadataStore, err := features.NewMetadataStore(projectDir)
	if err != nil {
		progress.Warning(fmt.Sprintf("Failed to initialize metadata store for cleanup: %v", err))
	} else {
		if err := metadataStore.RemoveFeature(featureName); err != nil {
			progress.Warning(fmt.Sprintf("Failed to remove feature metadata: %v", err))
		}
	}

	// Remove trees directory if it exists
	if treesDirExists {
		progress.UpdateWithProgress("Removing trees directory...", 90)
		if err := os.RemoveAll(treesDir); err != nil {
			progress.Error(fmt.Sprintf("Failed to remove trees directory: %s", treesDir))
			return nil, fmt.Errorf("failed to remove trees directory: %w", err)
		}
	} else {
		progress.Info("Trees directory already removed (orphaned worktree)")
	}

	progress.Complete(fmt.Sprintf("Feature '%s' cleaned up successfully!", featureName))

	return result, nil
}
