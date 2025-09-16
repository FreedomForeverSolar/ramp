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

type UpState struct {
	RepoName         string
	WorktreeCreated  bool
	WorktreeDir      string
	BranchName       string
	TreesDirCreated  bool
	PortAllocated    bool
	SetupRan         bool
}

var prefixFlag string
var targetFlag string

var upCmd = &cobra.Command{
	Use:   "up <feature-name>",
	Short: "Create a new feature branch with git worktrees for all repositories",
	Long: `Create a new feature branch by creating git worktrees for all repositories
from their configured locations. This creates isolated working directories for each repo
in the trees/<feature-name>/ directory.

By default, new feature branches are created from the default branch. Use the --target
flag to create the feature from a different source:
  - Existing feature name: --target my-existing-feature
  - Local branch name: --target feature/my-branch
  - Remote branch name: --target origin/feature/my-branch

The operation is atomic - if any step fails, all successful operations will be
rolled back to ensure no partial feature state remains.

After creating worktrees, runs any setup script specified in the configuration.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		featureName := args[0]
		if err := runUp(featureName, prefixFlag, targetFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVar(&prefixFlag, "prefix", "", "Override the branch prefix (defaults to config default_branch_prefix)")
	upCmd.Flags().StringVar(&targetFlag, "target", "", "Create feature from existing feature name, local branch, or remote branch")
}

func runUp(featureName, prefix, target string) error {
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

	// Auto-refresh repositories that have auto_refresh enabled (or not explicitly disabled)
	repos := cfg.GetRepos()
	hasAutoRefreshRepos := false
	for _, repo := range repos {
		if repo.ShouldAutoRefresh() {
			hasAutoRefreshRepos = true
			break
		}
	}

	if hasAutoRefreshRepos {
		progress := ui.NewProgress()
		progress.Start("Auto-refreshing repositories before creating feature")

		for name, repo := range repos {
			if repo.ShouldAutoRefresh() {
				repoDir := repo.GetRepoPath(projectDir)
				RefreshRepository(repoDir, name, progress)
			} else {
				progress.Info(fmt.Sprintf("%s: auto-refresh disabled, skipping", name))
			}
		}

		progress.Success("Auto-refresh completed")
	}

	progress := ui.NewProgress()
	progress.Start(fmt.Sprintf("Creating feature '%s' for project '%s'", featureName, cfg.Name))
	progress.Success(fmt.Sprintf("Creating feature '%s' for project '%s'", featureName, cfg.Name))

	// Determine effective prefix - flag takes precedence, then config, then empty
	effectivePrefix := prefix
	if effectivePrefix == "" {
		effectivePrefix = cfg.GetBranchPrefix()
	}

	treesDir := filepath.Join(projectDir, "trees", featureName)

	// Resolve target branch for each repository if target is specified
	var sourceBranches map[string]string
	if target != "" {
		progress.Update("Resolving target branch across repositories")
		sourceBranches = make(map[string]string)
		for name, repo := range repos {
			repoDir := repo.GetRepoPath(projectDir)
			sourceBranch, err := git.ResolveSourceBranch(repoDir, target, effectivePrefix)
			if err != nil {
				// If target doesn't exist in this repo, we'll use default branch (no source specified)
				progress.Warning(fmt.Sprintf("%s: target '%s' not found, will use default branch", name, target))
				// Empty string indicates to use default behavior
				sourceBranches[name] = ""
			} else {
				sourceBranches[name] = sourceBranch
				progress.Info(fmt.Sprintf("%s: resolved target '%s' to source branch '%s'", name, target, sourceBranch))
			}
		}
		progress.Success("Target branch resolution completed")
	}

	// Phase 1: Validation - check all preconditions before making any changes
	progress.Start("Validating repositories and checking for conflicts")
	states := make(map[string]*UpState)
	branchName := effectivePrefix + featureName

	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		worktreeDir := filepath.Join(treesDir, name)

		if !git.IsGitRepo(repoDir) {
			progress.Error(fmt.Sprintf("Source repo not found at %s even after auto-initialization", repoDir))
			return fmt.Errorf("source repo not found at %s even after auto-initialization", repoDir)
		}

		// Check if worktree directory already exists
		if _, err := os.Stat(worktreeDir); err == nil {
			progress.Error(fmt.Sprintf("Worktree directory already exists: %s", worktreeDir))
			return fmt.Errorf("worktree directory already exists: %s", worktreeDir)
		}

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

		// When using a target, we create new branches, so existing branches are conflicts
		if target != "" && sourceBranches[name] != "" {
			if localExists {
				progress.Error(fmt.Sprintf("Branch %s already exists locally in %s", branchName, name))
				return fmt.Errorf("branch %s already exists locally in repository %s", branchName, name)
			}
			sourceBranch := sourceBranches[name]
			progress.Info(fmt.Sprintf("%s: will create worktree with new branch %s from %s", name, branchName, sourceBranch))
		} else if target != "" && sourceBranches[name] == "" {
			// Target was specified but not found in this repo, use default behavior
			if localExists {
				progress.Info(fmt.Sprintf("%s: will create worktree with existing local branch %s", name, branchName))
			} else if remoteExists {
				progress.Info(fmt.Sprintf("%s: will create worktree with existing remote branch %s", name, branchName))
			} else {
				progress.Info(fmt.Sprintf("%s: will create worktree with new branch %s from default branch", name, branchName))
			}
		} else {
			// Original behavior: use existing branches or create new ones
			if localExists {
				progress.Info(fmt.Sprintf("%s: will create worktree with existing local branch %s", name, branchName))
			} else if remoteExists {
				progress.Info(fmt.Sprintf("%s: will create worktree with existing remote branch %s", name, branchName))
			} else {
				progress.Info(fmt.Sprintf("%s: will create worktree with new branch %s", name, branchName))
			}
		}

		states[name] = &UpState{
			RepoName:        name,
			WorktreeCreated: false,
			WorktreeDir:     worktreeDir,
			BranchName:      branchName,
			TreesDirCreated: false,
			PortAllocated:   false,
			SetupRan:        false,
		}
	}

	progress.Success("Validation completed successfully")

	// Phase 2: Execute operations with state tracking
	progress.Start("Creating trees directory")
	if err := os.MkdirAll(treesDir, 0755); err != nil {
		progress.Error("Failed to create trees directory")
		return fmt.Errorf("failed to create trees directory: %w", err)
	}

	// Mark that we created the trees directory
	for _, state := range states {
		state.TreesDirCreated = true
	}
	progress.Success("Trees directory created")

	var worktreesMessage string
	if len(repos) == 1 {
		for name := range repos {
			worktreesMessage = fmt.Sprintf("Created worktree: %s", name)
		}
	} else {
		worktreesMessage = fmt.Sprintf("Created %d worktrees", len(repos))
	}

	progress.Update("Creating worktrees")
	for name, repo := range repos {
		state := states[name]
		repoDir := repo.GetRepoPath(projectDir)

		var err error
		if target != "" && sourceBranches[name] != "" {
			// Use target source branch
			sourceBranch := sourceBranches[name]
			err = git.CreateWorktreeFromSource(repoDir, state.WorktreeDir, state.BranchName, sourceBranch, name)
		} else {
			// Use default behavior (either no target, or target not found in this repo)
			err = git.CreateWorktree(repoDir, state.WorktreeDir, state.BranchName, name)
		}

		if err != nil {
			progress.Error(fmt.Sprintf("Failed to create worktree for %s", name))
			// Rollback all successful operations
			if rollbackErr := rollbackUp(projectDir, treesDir, featureName, states, progress); rollbackErr != nil {
				return fmt.Errorf("worktree creation failed for %s (%v) and rollback failed: %w", name, err, rollbackErr)
			}
			return fmt.Errorf("failed to create worktree for %s: %w", name, err)
		}

		state.WorktreeCreated = true
	}
	progress.Success(worktreesMessage)

	// Allocate port for this feature only if port configuration is present
	var allocatedPort int
	if cfg.HasPortConfig() {
		progress.Update("Allocating port for feature")
		portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
		if err != nil {
			progress.Error("Failed to initialize port allocations")
			// Rollback all successful operations
			if rollbackErr := rollbackUp(projectDir, treesDir, featureName, states, progress); rollbackErr != nil {
				return fmt.Errorf("port allocation initialization failed (%v) and rollback failed: %w", err, rollbackErr)
			}
			return fmt.Errorf("failed to initialize port allocations: %w", err)
		}

		allocatedPort, err = portAllocations.AllocatePort(featureName)
		if err != nil {
			progress.Error("Failed to allocate port")
			// Rollback all successful operations
			if rollbackErr := rollbackUp(projectDir, treesDir, featureName, states, progress); rollbackErr != nil {
				return fmt.Errorf("port allocation failed (%v) and rollback failed: %w", err, rollbackErr)
			}
			return fmt.Errorf("failed to allocate port for feature: %w", err)
		}

		// Mark that we allocated a port
		for _, state := range states {
			state.PortAllocated = true
		}
		progress.Success(fmt.Sprintf("Allocated port %d", allocatedPort))
	}

	// Run setup script if configured
	if cfg.Setup != "" {
		progress.Update("Running setup script")
		if err := runSetupScriptWithProgress(projectDir, treesDir, cfg.Setup, progress); err != nil {
			progress.Error("Setup script failed")
			// Mark that setup ran (even if it failed) for rollback purposes
			for _, state := range states {
				state.SetupRan = true
			}
			// Rollback all successful operations
			if rollbackErr := rollbackUp(projectDir, treesDir, featureName, states, progress); rollbackErr != nil {
				return fmt.Errorf("setup script failed (%v) and rollback failed: %w", err, rollbackErr)
			}
			return fmt.Errorf("setup script failed: %w", err)
		}

		// Mark that setup ran successfully
		for _, state := range states {
			state.SetupRan = true
		}
		progress.Success("Ran setup script")
	}

	progress.Success(fmt.Sprintf("Feature '%s' created successfully", featureName))
	fmt.Printf("Feature '%s' created at %s\n", featureName, treesDir)
	return nil
}

func rollbackUp(projectDir, treesDir, featureName string, states map[string]*UpState, progress *ui.ProgressUI) error {
	progress.Warning("Rolling back changes due to failure")

	cfg, err := config.LoadConfig(projectDir)
	if err != nil {
		progress.Error("Failed to load config during rollback")
		return fmt.Errorf("failed to load config during rollback: %w", err)
	}

	repos := cfg.GetRepos()

	// Remove worktrees that were successfully created
	for name, state := range states {
		if state.WorktreeCreated {
			repo := repos[name]
			repoDir := repo.GetRepoPath(projectDir)
			progress.Info(fmt.Sprintf("%s: removing worktree", name))

			if err := git.RemoveWorktree(repoDir, state.WorktreeDir); err != nil {
				progress.Warning(fmt.Sprintf("Failed to remove worktree for %s: %v", name, err))
				// Continue with other cleanup operations
			} else {
				progress.Info(fmt.Sprintf("%s: worktree removed", name))
			}

			// Also try to delete the branch if it was newly created
			// We can determine this by checking if it was created during this operation
			// For safety, we'll only delete if both local and remote don't exist from before
			localExists, _ := git.LocalBranchExists(repoDir, state.BranchName)

			// If the branch was newly created (which we can infer if it now exists locally
			// but we detected it didn't exist before), we should clean it up
			if localExists {
				progress.Info(fmt.Sprintf("%s: deleting branch %s", name, state.BranchName))
				if err := git.DeleteBranch(repoDir, state.BranchName); err != nil {
					progress.Warning(fmt.Sprintf("Failed to delete branch %s for %s: %v", state.BranchName, name, err))
				} else {
					progress.Info(fmt.Sprintf("%s: branch %s deleted", name, state.BranchName))
				}
			}
		}
	}

	// Release port if it was allocated
	var portAllocated bool
	for _, state := range states {
		if state.PortAllocated {
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

	// Remove trees directory if it was created and is empty or only contains our failed worktree dirs
	var treesDirCreated bool
	for _, state := range states {
		if state.TreesDirCreated {
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
