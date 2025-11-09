package cmd

import (
	"path/filepath"
	"testing"

	"ramp/internal/config"
	"ramp/internal/git"
	"ramp/internal/ui"
)

// TestRebaseToExistingLocalBranch tests rebasing to a branch that exists locally
func TestRebaseToExistingLocalBranch(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a new branch in source repo
	runGitCmd(t, repo1.SourceDir, "checkout", "-b", "feature-branch")
	runGitCmd(t, repo1.SourceDir, "checkout", "main")

	// Rebase to the feature branch
	err := runRebase("feature-branch")
	if err != nil {
		t.Fatalf("runRebase() error = %v", err)
	}

	// Verify we're now on feature-branch
	currentBranch, _ := git.GetCurrentBranch(repo1.SourceDir)
	if currentBranch != "feature-branch" {
		t.Errorf("current branch = %q, want %q", currentBranch, "feature-branch")
	}
}

// TestRebaseToRemoteBranch tests rebasing to a branch that only exists remotely
func TestRebaseToRemoteBranch(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a branch and push to remote
	runGitCmd(t, repo1.SourceDir, "checkout", "-b", "remote-feature")
	runGitCmd(t, repo1.SourceDir, "push", "-u", "origin", "remote-feature")
	runGitCmd(t, repo1.SourceDir, "checkout", "main")

	// Delete local branch but keep remote
	runGitCmd(t, repo1.SourceDir, "branch", "-D", "remote-feature")

	// Fetch to get remote branch
	runGitCmd(t, repo1.SourceDir, "fetch", "origin")

	// Rebase to the remote branch
	err := runRebase("remote-feature")
	if err != nil {
		t.Fatalf("runRebase() error = %v", err)
	}

	// Verify we're now on remote-feature
	currentBranch, _ := git.GetCurrentBranch(repo1.SourceDir)
	if currentBranch != "remote-feature" {
		t.Errorf("current branch = %q, want %q", currentBranch, "remote-feature")
	}
}

// TestRebaseMultipleRepos tests rebasing multiple repositories
func TestRebaseMultipleRepos(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")
	repo2 := tp.InitRepo("repo2")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create the same branch in both repos
	runGitCmd(t, repo1.SourceDir, "checkout", "-b", "shared-branch")
	runGitCmd(t, repo1.SourceDir, "checkout", "main")

	runGitCmd(t, repo2.SourceDir, "checkout", "-b", "shared-branch")
	runGitCmd(t, repo2.SourceDir, "checkout", "main")

	// Rebase both repos to shared-branch
	err := runRebase("shared-branch")
	if err != nil {
		t.Fatalf("runRebase() error = %v", err)
	}

	// Verify both repos switched
	branch1, _ := git.GetCurrentBranch(repo1.SourceDir)
	if branch1 != "shared-branch" {
		t.Errorf("repo1 current branch = %q, want %q", branch1, "shared-branch")
	}

	branch2, _ := git.GetCurrentBranch(repo2.SourceDir)
	if branch2 != "shared-branch" {
		t.Errorf("repo2 current branch = %q, want %q", branch2, "shared-branch")
	}
}

// TestRebaseSkipRepoWithoutBranch tests that repos without the branch are skipped
func TestRebaseSkipRepoWithoutBranch(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")
	repo2 := tp.InitRepo("repo2")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create branch only in repo1
	runGitCmd(t, repo1.SourceDir, "checkout", "-b", "partial-branch")
	runGitCmd(t, repo1.SourceDir, "checkout", "main")

	// Rebase to partial-branch (should succeed, repo2 skipped)
	err := runRebase("partial-branch")
	if err != nil {
		t.Fatalf("runRebase() error = %v", err)
	}

	// Verify repo1 switched
	branch1, _ := git.GetCurrentBranch(repo1.SourceDir)
	if branch1 != "partial-branch" {
		t.Errorf("repo1 current branch = %q, want %q", branch1, "partial-branch")
	}

	// Verify repo2 stayed on main
	branch2, _ := git.GetCurrentBranch(repo2.SourceDir)
	if branch2 != "main" {
		t.Errorf("repo2 current branch = %q, want %q (should be skipped)", branch2, "main")
	}
}

// TestRebaseNonExistentBranch tests error when branch doesn't exist anywhere
func TestRebaseNonExistentBranch(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Try to rebase to a branch that doesn't exist
	err := runRebase("nonexistent-branch")
	if err == nil {
		t.Fatal("runRebase() should fail for non-existent branch")
	}

	expectedMsg := "branch 'nonexistent-branch' does not exist in any repository"
	if err.Error() != expectedMsg {
		t.Errorf("error = %q, want %q", err.Error(), expectedMsg)
	}
}

// TestRebaseAlreadyOnBranch tests rebasing when already on target branch
func TestRebaseAlreadyOnBranch(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create and checkout a branch
	runGitCmd(t, repo1.SourceDir, "checkout", "-b", "current-branch")

	// Rebase to the same branch (should succeed, no-op)
	err := runRebase("current-branch")
	if err != nil {
		t.Fatalf("runRebase() error = %v", err)
	}

	// Verify still on current-branch
	currentBranch, _ := git.GetCurrentBranch(repo1.SourceDir)
	if currentBranch != "current-branch" {
		t.Errorf("current branch = %q, want %q", currentBranch, "current-branch")
	}
}

// TestRebaseState tests the RebaseState struct
func TestRebaseState(t *testing.T) {
	state := &RebaseState{
		RepoName:       "test-repo",
		OriginalBranch: "main",
		Stashed:        true,
		Success:        true,
		BranchExists:   true,
	}

	if state.RepoName != "test-repo" {
		t.Errorf("RepoName = %q, want %q", state.RepoName, "test-repo")
	}
	if state.OriginalBranch != "main" {
		t.Errorf("OriginalBranch = %q, want %q", state.OriginalBranch, "main")
	}
	if !state.Stashed {
		t.Error("Stashed should be true")
	}
	if !state.Success {
		t.Error("Success should be true")
	}
	if !state.BranchExists {
		t.Error("BranchExists should be true")
	}
}

// TestRollbackRebase tests the rollback functionality
func TestRollbackRebase(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	// Create a feature branch and switch to it
	runGitCmd(t, repo1.SourceDir, "checkout", "-b", "feature")
	runGitCmd(t, repo1.SourceDir, "checkout", "main")

	// Manually switch to feature
	runGitCmd(t, repo1.SourceDir, "checkout", "feature")

	// Create state indicating successful switch
	states := map[string]*RebaseState{
		"repo1": {
			RepoName:       "repo1",
			OriginalBranch: "main",
			Stashed:        false,
			Success:        true,
			BranchExists:   true,
		},
	}

	progress := ui.NewProgress()

	// Rollback should switch back to main
	err := rollbackRebase(tp.Dir, tp.Config.GetRepos(), states, progress)
	if err != nil {
		t.Fatalf("rollbackRebase() error = %v", err)
	}

	// Verify we're back on main
	currentBranch, _ := git.GetCurrentBranch(repo1.SourceDir)
	if currentBranch != "main" {
		t.Errorf("current branch = %q, want %q (should be rolled back)", currentBranch, "main")
	}
}

// TestRebaseWithMissingRepo tests error when configured repo doesn't exist
func TestRebaseWithMissingRepo(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	// Add another repo to config but don't clone it
	autoRefresh := true
	missingPath := filepath.Join(tp.ReposDir, "missing-repo")
	tp.Config.Repos = append(tp.Config.Repos, &config.Repo{
		Path:        "repos",
		Git:         missingPath, // Use local path so auto-install tries to clone
		AutoRefresh: &autoRefresh,
	})
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Try to rebase - auto-install will try to clone the missing repo and fail
	err := runRebase("main")
	if err == nil {
		t.Fatal("runRebase() should fail when auto-installation fails")
	}

	// Error should mention auto-installation failure
	if err.Error()[:len("auto-installation")] != "auto-installation" {
		t.Errorf("error should mention auto-installation, got %q", err.Error())
	}
}

// TestRebaseToMain tests rebasing back to main branch
func TestRebaseToMain(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create and switch to a feature branch
	runGitCmd(t, repo1.SourceDir, "checkout", "-b", "feature")

	// Rebase back to main
	err := runRebase("main")
	if err != nil {
		t.Fatalf("runRebase() error = %v", err)
	}

	// Verify we're back on main
	currentBranch, _ := git.GetCurrentBranch(repo1.SourceDir)
	if currentBranch != "main" {
		t.Errorf("current branch = %q, want %q", currentBranch, "main")
	}
}

// TestRebasePartialSuccess tests behavior when some repos switch successfully
func TestRebasePartialSuccess(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")
	repo2 := tp.InitRepo("repo2")
	repo3 := tp.InitRepo("repo3")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create branch in repo1 and repo2, but not repo3
	runGitCmd(t, repo1.SourceDir, "checkout", "-b", "partial-branch")
	runGitCmd(t, repo1.SourceDir, "checkout", "main")

	runGitCmd(t, repo2.SourceDir, "checkout", "-b", "partial-branch")
	runGitCmd(t, repo2.SourceDir, "checkout", "main")

	// Rebase should succeed (repo3 skipped)
	err := runRebase("partial-branch")
	if err != nil {
		t.Fatalf("runRebase() error = %v", err)
	}

	// Verify repo1 and repo2 switched
	branch1, _ := git.GetCurrentBranch(repo1.SourceDir)
	if branch1 != "partial-branch" {
		t.Errorf("repo1 current branch = %q, want %q", branch1, "partial-branch")
	}

	branch2, _ := git.GetCurrentBranch(repo2.SourceDir)
	if branch2 != "partial-branch" {
		t.Errorf("repo2 current branch = %q, want %q", branch2, "partial-branch")
	}

	// Verify repo3 stayed on main
	branch3, _ := git.GetCurrentBranch(repo3.SourceDir)
	if branch3 != "main" {
		t.Errorf("repo3 current branch = %q, want %q (should be skipped)", branch3, "main")
	}
}
