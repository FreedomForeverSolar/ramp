package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/git"
	"ramp/internal/ui"
)

type RebaseState struct {
	RepoName       string
	OriginalBranch string
	Stashed        bool
	Success        bool
	BranchExists   bool
}

var rebaseCmd = &cobra.Command{
	Use:   "rebase <branch-name>",
	Short: "Switch all source repositories to the specified branch",
	Long: `Switch all source repositories in the project to the specified branch.
The branch can exist locally, remotely, or both. If the branch doesn't exist 
in any repository, the command will fail.

The operation is atomic - if any repository fails to switch, all repositories 
will be reverted to their original branches.

If there are uncommitted changes, you will be prompted to stash them.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		branchName := args[0]
		if err := runRebase(branchName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(rebaseCmd)
}

func runRebase(branchName string) error {
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

	progress := ui.NewProgress()
	progress.Start(fmt.Sprintf("Rebasing project '%s' to branch '%s'", cfg.Name, branchName))
	progress.Success(fmt.Sprintf("Rebasing project '%s' to branch '%s'", cfg.Name, branchName))

	repos := cfg.GetRepos()
	switchedCount := 0
	skippedCount := 0
	
	// Phase 1: Validation - check branch availability and collect current state
	progress.Start("Checking branch availability in repositories")
	states := make(map[string]*RebaseState)
	reposWithBranch := 0
	
	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)

		if !git.IsGitRepo(repoDir) {
			progress.Error(fmt.Sprintf("Repository %s not found at %s", name, repoDir))
			return fmt.Errorf("repository %s not found at %s", name, repoDir)
		}

		// Get current branch for rollback purposes
		currentBranch, err := git.GetCurrentBranch(repoDir)
		if err != nil {
			progress.Error(fmt.Sprintf("Failed to get current branch for %s", name))
			return fmt.Errorf("failed to get current branch for %s: %w", name, err)
		}

		// Check if target branch exists
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

		branchExists := localExists || remoteExists
		if branchExists {
			reposWithBranch++
		}

		states[name] = &RebaseState{
			RepoName:       name,
			OriginalBranch: currentBranch,
			Stashed:        false,
			Success:        false,
			BranchExists:   branchExists,
		}

		if localExists {
			progress.Info(fmt.Sprintf("%s: branch %s exists locally", name, branchName))
		} else if remoteExists {
			progress.Info(fmt.Sprintf("%s: branch %s exists remotely", name, branchName))
		} else {
			progress.Info(fmt.Sprintf("%s: branch %s does not exist, will keep current branch", name, branchName))
		}
	}
	
	if reposWithBranch == 0 {
		progress.Error(fmt.Sprintf("Branch '%s' does not exist in any repository", branchName))
		return fmt.Errorf("branch '%s' does not exist in any repository", branchName)
	}
	
	progress.Success(fmt.Sprintf("Branch validation completed (%d repos have branch, %d will be skipped)", reposWithBranch, len(repos)-reposWithBranch))

	// Phase 2: Check for uncommitted changes and handle them
	progress.Start("Checking for uncommitted changes")
	hasUncommitted := false
	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		
		hasChanges, err := git.HasUncommittedChanges(repoDir)
		if err != nil {
			progress.Error(fmt.Sprintf("Failed to check uncommitted changes in %s", name))
			return fmt.Errorf("failed to check uncommitted changes in %s: %w", name, err)
		}
		
		if hasChanges {
			progress.Warning(fmt.Sprintf("Uncommitted changes found in %s", name))
			hasUncommitted = true
		}
	}

	if hasUncommitted {
		if !confirmStashChanges() {
			progress.Info("Operation cancelled by user")
			return nil
		}

		progress.Start("Stashing uncommitted changes")
		for name, repo := range repos {
			repoDir := repo.GetRepoPath(projectDir)
			stashed, err := git.StashChanges(repoDir)
			if err != nil {
				progress.Error(fmt.Sprintf("Failed to stash changes in %s", name))
				return fmt.Errorf("failed to stash changes in %s: %w", name, err)
			}
			states[name].Stashed = stashed
			if stashed {
				progress.Info(fmt.Sprintf("%s: changes stashed", name))
			}
		}
		progress.Success("Uncommitted changes stashed")
	} else {
		progress.Success("No uncommitted changes found")
	}

	// Phase 3: Execute checkout operations
	progress.Start("Switching repositories to target branch")
	
	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		state := states[name]

		// Skip if branch doesn't exist in this repo
		if !state.BranchExists {
			progress.Info(fmt.Sprintf("%s: skipped (branch doesn't exist)", name))
			skippedCount++
			continue
		}

		// Skip if already on target branch
		if state.OriginalBranch == branchName {
			progress.Info(fmt.Sprintf("%s: already on branch %s", name, branchName))
			state.Success = true
			switchedCount++
			continue
		}

		localExists, _ := git.LocalBranchExists(repoDir, branchName)
		
		var err error
		if localExists {
			// Checkout existing local branch
			progress.Info(fmt.Sprintf("%s: checking out local branch %s", name, branchName))
			err = git.Checkout(repoDir, branchName)
		} else {
			// Create and checkout remote branch
			progress.Info(fmt.Sprintf("%s: checking out remote branch %s", name, branchName))
			err = git.CheckoutRemoteBranch(repoDir, branchName)
		}

		if err != nil {
			progress.Error(fmt.Sprintf("Failed to checkout branch in %s", name))
			// Rollback all successful operations
			if rollbackErr := rollbackRebase(projectDir, repos, states, progress); rollbackErr != nil {
				return fmt.Errorf("checkout failed for %s (%v) and rollback failed: %w", name, err, rollbackErr)
			}
			return fmt.Errorf("failed to checkout branch in %s: %w", name, err)
		}
		
		state.Success = true
		switchedCount++
		progress.Info(fmt.Sprintf("%s: successfully switched to %s", name, branchName))
	}

	progress.Success(fmt.Sprintf("Repository switching completed (%d switched, %d skipped)", switchedCount, skippedCount))

	// Phase 4: Restore stashed changes
	if hasUncommitted {
		progress.Start("Restoring stashed changes")
		for name, repo := range repos {
			state := states[name]
			if state.Stashed {
				repoDir := repo.GetRepoPath(projectDir)
				if err := git.PopStash(repoDir); err != nil {
					progress.Warning(fmt.Sprintf("Failed to restore stashed changes in %s: %v", name, err))
					progress.Warning("You may need to manually restore stashed changes with 'git stash pop'")
				} else {
					progress.Info(fmt.Sprintf("%s: stashed changes restored", name))
				}
			}
		}
		progress.Success("Stashed changes restored")
	}

	// Final summary
	if skippedCount > 0 {
		progress.Success(fmt.Sprintf("Successfully rebased project to branch '%s'! (%d switched, %d skipped)", branchName, switchedCount, skippedCount))
	} else {
		progress.Success(fmt.Sprintf("Successfully rebased project to branch '%s'! (all %d repos switched)", branchName, switchedCount))
	}
	return nil
}

func rollbackRebase(projectDir string, repos map[string]*config.Repo, states map[string]*RebaseState, progress *ui.ProgressUI) error {
	progress.Warning("Rolling back changes due to failure")
	
	for name, repo := range repos {
		state := states[name]
		if state.Success {
			repoDir := repo.GetRepoPath(projectDir)
			progress.Info(fmt.Sprintf("%s: rolling back to %s", name, state.OriginalBranch))
			if err := git.Checkout(repoDir, state.OriginalBranch); err != nil {
				progress.Error(fmt.Sprintf("Failed to rollback %s: %v", name, err))
			}
		}
		
		// Restore stashed changes if any
		if state.Stashed {
			repoDir := repo.GetRepoPath(projectDir)
			if err := git.PopStash(repoDir); err != nil {
				progress.Warning(fmt.Sprintf("Failed to restore stashed changes in %s during rollback: %v", name, err))
			}
		}
	}
	
	progress.Info("Rollback completed")
	return nil
}

func confirmStashChanges() bool {
	fmt.Printf("\nThere are uncommitted changes in one or more repositories.\n")
	fmt.Printf("Do you want to stash these changes and continue? (y/N): ")
	
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	return input == "y" || input == "yes"
}