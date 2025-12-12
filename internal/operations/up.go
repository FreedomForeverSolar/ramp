package operations

import (
	"fmt"
	"os"
	"path/filepath"

	"ramp/internal/config"
	"ramp/internal/envfile"
	"ramp/internal/git"
	"ramp/internal/ports"
)

// UpOptions configures the feature creation operation.
type UpOptions struct {
	// Required
	FeatureName string
	ProjectDir  string
	Config      *config.Config
	Progress    ProgressReporter

	// Optional - output streaming for setup script
	Output OutputStreamer // For streaming setup script stdout/stderr

	// Optional - branch configuration
	Prefix   string // Branch prefix override (empty = use config default)
	NoPrefix bool   // Explicitly disable prefix
	Target   string // Source branch/feature to create from

	// Optional - pre-operation behavior
	AutoInstall bool // Auto-install repos if not present (default: false)

	// Optional - refresh behavior
	// By default, operations.Up() respects each repo's auto_refresh config.
	// Use these flags to override:
	ForceRefresh bool // Force refresh ALL repos regardless of per-repo config
	SkipRefresh  bool // Skip refresh for ALL repos regardless of per-repo config
}

// UpResult contains the results of feature creation.
type UpResult struct {
	FeatureName    string
	TreesDir       string
	BranchName     string
	Repos          []string
	AllocatedPorts []int
}

// upState tracks state for rollback purposes.
type upState struct {
	repoName        string
	worktreeCreated bool
	worktreeDir     string
	branchName      string
	treesDirCreated bool
	portAllocated   bool
	setupRan        bool
}

// Up creates a new feature with worktrees for all repositories.
// This is the core business logic used by both CLI and UI.
func Up(opts UpOptions) (*UpResult, error) {
	projectDir := opts.ProjectDir
	cfg := opts.Config
	progress := opts.Progress
	featureName := opts.FeatureName

	// Phase 0a: Auto-install if requested and needed
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

	// Phase 0b: Auto-refresh based on per-repo config (unless SkipRefresh is set)
	if !opts.SkipRefresh {
		repos := cfg.GetRepos()

		// Build filter for repos that should be refreshed
		repoFilter := make(map[string]bool)
		for name, repo := range repos {
			shouldRefresh := false
			if opts.ForceRefresh {
				// --refresh flag: refresh ALL repos
				shouldRefresh = true
			} else {
				// Default: respect per-repo auto_refresh config
				shouldRefresh = repo.ShouldAutoRefresh()
			}

			if shouldRefresh {
				repoFilter[name] = true
			}
		}

		// Only show refresh UI if there are repos to refresh
		if len(repoFilter) > 0 {
			progress.Start("Auto-refreshing repositories before creating feature")

			// Log skipped repos
			for name := range repos {
				if !repoFilter[name] {
					progress.Info(fmt.Sprintf("%s: auto-refresh disabled, skipping", name))
				}
			}

			RefreshRepositories(RefreshOptions{
				ProjectDir: projectDir,
				Config:     cfg,
				Progress:   progress,
				RepoFilter: repoFilter,
			})

			progress.Success("Auto-refresh completed")
		}
	}

	progress.Start(fmt.Sprintf("Creating feature '%s' for project '%s'", featureName, cfg.Name))

	// Determine effective prefix
	var effectivePrefix string
	if opts.NoPrefix {
		effectivePrefix = ""
	} else if opts.Prefix != "" {
		effectivePrefix = opts.Prefix
	} else {
		effectivePrefix = cfg.GetBranchPrefix()
	}

	branchName := effectivePrefix + featureName
	treesDir := filepath.Join(projectDir, "trees", featureName)
	repos := cfg.GetRepos()

	// Resolve target branch for each repository if target is specified
	var sourceBranches map[string]string
	if opts.Target != "" {
		progress.Update("Resolving target branch across repositories")
		sourceBranches = make(map[string]string)
		for name, repo := range repos {
			repoDir := repo.GetRepoPath(projectDir)
			sourceBranch, err := git.ResolveSourceBranch(repoDir, opts.Target, effectivePrefix)
			if err != nil {
				progress.Warning(fmt.Sprintf("%s: target '%s' not found, will use default branch", name, opts.Target))
				sourceBranches[name] = ""
			} else {
				sourceBranches[name] = sourceBranch
				progress.Info(fmt.Sprintf("%s: resolved target '%s' to source branch '%s'", name, opts.Target, sourceBranch))
			}
		}
		progress.Success("Target branch resolution completed")
	}

	// Phase 1: Validation
	progress.Start("Validating repositories and checking for conflicts")
	states := make(map[string]*upState)

	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		worktreeDir := filepath.Join(treesDir, name)

		if !git.IsGitRepo(repoDir) {
			progress.Error(fmt.Sprintf("Source repo not found at %s", repoDir))
			return nil, fmt.Errorf("source repo not found at %s", repoDir)
		}

		// Prune stale worktree entries before checking for conflicts
		// This cleans up orphaned worktrees where the directory was removed but git still has a reference
		_ = git.PruneWorktrees(repoDir)

		if _, err := os.Stat(worktreeDir); err == nil {
			progress.Error(fmt.Sprintf("Worktree directory already exists: %s", worktreeDir))
			return nil, fmt.Errorf("worktree directory already exists: %s", worktreeDir)
		}

		localExists, err := git.LocalBranchExists(repoDir, branchName)
		if err != nil {
			progress.Error(fmt.Sprintf("Failed to check local branch for %s", name))
			return nil, fmt.Errorf("failed to check local branch for %s: %w", name, err)
		}

		remoteExists, err := git.RemoteBranchExists(repoDir, branchName)
		if err != nil {
			progress.Error(fmt.Sprintf("Failed to check remote branch for %s", name))
			return nil, fmt.Errorf("failed to check remote branch for %s: %w", name, err)
		}

		// When using a target, existing branches are conflicts
		if opts.Target != "" && sourceBranches[name] != "" {
			if localExists {
				progress.Error(fmt.Sprintf("Branch %s already exists locally in %s", branchName, name))
				return nil, fmt.Errorf("branch %s already exists locally in repository %s", branchName, name)
			}
			progress.Info(fmt.Sprintf("%s: will create worktree with new branch %s from %s", name, branchName, sourceBranches[name]))
		} else if opts.Target != "" && sourceBranches[name] == "" {
			if localExists {
				progress.Info(fmt.Sprintf("%s: will create worktree with existing local branch %s", name, branchName))
			} else if remoteExists {
				progress.Info(fmt.Sprintf("%s: will create worktree with existing remote branch %s", name, branchName))
			} else {
				progress.Info(fmt.Sprintf("%s: will create worktree with new branch %s from default branch", name, branchName))
			}
		} else {
			if localExists {
				progress.Info(fmt.Sprintf("%s: will create worktree with existing local branch %s", name, branchName))
			} else if remoteExists {
				progress.Info(fmt.Sprintf("%s: will create worktree with existing remote branch %s", name, branchName))
			} else {
				progress.Info(fmt.Sprintf("%s: will create worktree with new branch %s", name, branchName))
			}
		}

		states[name] = &upState{
			repoName:        name,
			worktreeCreated: false,
			worktreeDir:     worktreeDir,
			branchName:      branchName,
			treesDirCreated: false,
			portAllocated:   false,
			setupRan:        false,
		}
	}

	progress.Success("Validation completed successfully")

	// Phase 2: Create trees directory
	progress.Start("Creating trees directory")
	if err := os.MkdirAll(treesDir, 0755); err != nil {
		progress.Error("Failed to create trees directory")
		return nil, fmt.Errorf("failed to create trees directory: %w", err)
	}

	for _, state := range states {
		state.treesDirCreated = true
	}
	progress.Success("Trees directory created")

	// Phase 3: Create worktrees
	repoNames := []string{}
	total := len(repos)
	i := 0

	for name, repo := range repos {
		repoNames = append(repoNames, name)
		state := states[name]
		repoDir := repo.GetRepoPath(projectDir)

		progress.UpdateWithProgress(fmt.Sprintf("Creating worktree for %s...", name), (i+1)*50/total)

		var err error
		if opts.Target != "" && sourceBranches[name] != "" {
			err = git.CreateWorktreeFromSourceQuiet(repoDir, state.worktreeDir, state.branchName, sourceBranches[name], name)
		} else {
			err = git.CreateWorktreeQuiet(repoDir, state.worktreeDir, state.branchName, name)
		}

		if err != nil {
			progress.Error(fmt.Sprintf("Failed to create worktree for %s", name))
			rollbackUp(projectDir, treesDir, featureName, states, cfg, progress)
			return nil, fmt.Errorf("failed to create worktree for %s: %w", name, err)
		}

		state.worktreeCreated = true
		i++
	}

	var worktreesMessage string
	if len(repos) == 1 {
		for name := range repos {
			worktreesMessage = fmt.Sprintf("Created worktree: %s", name)
		}
	} else {
		worktreesMessage = fmt.Sprintf("Created %d worktrees", len(repos))
	}
	progress.Success(worktreesMessage)

	// Phase 4: Allocate ports
	var allocatedPorts []int
	if cfg.HasPortConfig() {
		progress.UpdateWithProgress("Allocating ports...", 55)

		portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
		if err != nil {
			progress.Error("Failed to initialize port allocations")
			rollbackUp(projectDir, treesDir, featureName, states, cfg, progress)
			return nil, fmt.Errorf("failed to initialize port allocations: %w", err)
		}

		allocatedPorts, err = portAllocations.AllocatePort(featureName, cfg.GetPortsPerFeature())
		if err != nil {
			progress.Error("Failed to allocate ports")
			rollbackUp(projectDir, treesDir, featureName, states, cfg, progress)
			return nil, fmt.Errorf("failed to allocate ports for feature: %w", err)
		}

		for _, state := range states {
			state.portAllocated = true
		}

		if len(allocatedPorts) == 1 {
			progress.Success(fmt.Sprintf("Allocated port %d", allocatedPorts[0]))
		} else {
			progress.Success(fmt.Sprintf("Allocated ports %d-%d", allocatedPorts[0], allocatedPorts[len(allocatedPorts)-1]))
		}
	}

	// Phase 5: Process env files
	if HasEnvFiles(repos) {
		progress.UpdateWithProgress("Processing environment files...", 65)

		envVars := BuildEnvVars(projectDir, treesDir, featureName, allocatedPorts, cfg, repos)

		for name, repo := range repos {
			if len(repo.EnvFiles) > 0 {
				state := states[name]
				sourceRepoDir := repo.GetRepoPath(projectDir)

				// Determine refresh behavior for env scripts
				shouldRefresh := false
				if opts.ForceRefresh {
					shouldRefresh = true
				} else if !opts.SkipRefresh {
					shouldRefresh = repo.ShouldAutoRefresh()
				}

				if err := envfile.ProcessEnvFiles(name, repo.EnvFiles, sourceRepoDir, state.worktreeDir, envVars, shouldRefresh); err != nil {
					progress.Error(fmt.Sprintf("Failed to process env files for %s", name))
					rollbackUp(projectDir, treesDir, featureName, states, cfg, progress)
					return nil, fmt.Errorf("failed to process env files for %s: %w", name, err)
				}
			}
		}
		progress.Success("Environment files processed")
	}

	// Phase 6: Run setup script
	if cfg.Setup != "" {
		progress.UpdateWithProgress("Running setup script...", 80)

		if err := RunSetupScript(projectDir, treesDir, featureName, cfg, allocatedPorts, repos, progress, opts.Output); err != nil {
			progress.Error("Setup script failed")
			for _, state := range states {
				state.setupRan = true
			}
			rollbackUp(projectDir, treesDir, featureName, states, cfg, progress)
			return nil, fmt.Errorf("setup script failed: %w", err)
		}

		for _, state := range states {
			state.setupRan = true
		}
		progress.Success("Ran setup script")
	}

	progress.Complete(fmt.Sprintf("Feature '%s' created successfully", featureName))

	return &UpResult{
		FeatureName:    featureName,
		TreesDir:       treesDir,
		BranchName:     branchName,
		Repos:          repoNames,
		AllocatedPorts: allocatedPorts,
	}, nil
}

// rollbackUp cleans up on failure.
func rollbackUp(projectDir, treesDir, featureName string, states map[string]*upState, cfg *config.Config, progress ProgressReporter) {
	progress.Warning("Rolling back changes due to failure")

	repos := cfg.GetRepos()

	// Remove worktrees that were created
	for name, state := range states {
		if state.worktreeCreated {
			repo := repos[name]
			repoDir := repo.GetRepoPath(projectDir)
			progress.Info(fmt.Sprintf("%s: removing worktree", name))

			if err := git.RemoveWorktreeQuiet(repoDir, state.worktreeDir); err != nil {
				progress.Warning(fmt.Sprintf("Failed to remove worktree for %s: %v", name, err))
			} else {
				progress.Info(fmt.Sprintf("%s: worktree removed", name))
			}

			// Delete branch if it exists
			localExists, _ := git.LocalBranchExists(repoDir, state.branchName)
			if localExists {
				progress.Info(fmt.Sprintf("%s: deleting branch %s", name, state.branchName))
				if err := git.DeleteBranchQuiet(repoDir, state.branchName); err != nil {
					progress.Warning(fmt.Sprintf("Failed to delete branch %s for %s: %v", state.branchName, name, err))
				} else {
					progress.Info(fmt.Sprintf("%s: branch %s deleted", name, state.branchName))
				}
			}
		}
	}

	// Release port if allocated
	var portAllocated bool
	for _, state := range states {
		if state.portAllocated {
			portAllocated = true
			break
		}
	}

	if portAllocated {
		progress.Info("Releasing allocated port")
		portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
		if err != nil {
			progress.Warning(fmt.Sprintf("Failed to initialize port allocations during rollback: %v", err))
		} else {
			if err := portAllocations.ReleasePort(featureName); err != nil {
				progress.Warning(fmt.Sprintf("Failed to release port: %v", err))
			} else {
				progress.Info("Port released successfully")
			}
		}
	}

	// Remove trees directory
	var treesDirCreated bool
	for _, state := range states {
		if state.treesDirCreated {
			treesDirCreated = true
			break
		}
	}

	if treesDirCreated {
		progress.Info("Removing trees directory")
		if err := os.RemoveAll(treesDir); err != nil {
			progress.Warning(fmt.Sprintf("Failed to remove trees directory: %v", err))
		} else {
			progress.Info("Trees directory removed")
		}
	}

	progress.Info("Rollback completed")
}
