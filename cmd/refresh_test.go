package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ramp/internal/ui"
)

// TestRefreshBasic tests basic repository refresh
func TestRefreshBasic(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Make a commit in the remote
	testFile := filepath.Join(repo1.RemoteDir, "..", "temp-work", "file.txt")
	os.MkdirAll(filepath.Dir(testFile), 0755)
	os.WriteFile(testFile, []byte("remote work"), 0644)

	// Clone the remote to a temp location, make changes, and push
	tempClone := filepath.Join(t.TempDir(), "temp-clone")
	runGitCmd(t, tempClone, "clone", repo1.RemoteDir, ".")
	runGitCmd(t, tempClone, "config", "user.email", "test@test.com")
	runGitCmd(t, tempClone, "config", "user.name", "Test")
	runGitCmd(t, tempClone, "config", "commit.gpgsign", "false")
	// Ensure we're on the main branch (clone should have checked it out automatically)
	runGitCmd(t, tempClone, "checkout", "main")

	updateFile := filepath.Join(tempClone, "update.txt")
	os.WriteFile(updateFile, []byte("update"), 0644)
	runGitCmd(t, tempClone, "add", ".")
	runGitCmd(t, tempClone, "commit", "-m", "remote update")
	runGitCmd(t, tempClone, "push", "origin", "main")

	// Now refresh should pull these changes
	err := runRefresh()
	if err != nil {
		t.Fatalf("runRefresh() error = %v", err)
	}

	// Verify the changes were pulled
	pulledFile := filepath.Join(repo1.SourceDir, "update.txt")
	if _, err := os.Stat(pulledFile); os.IsNotExist(err) {
		t.Error("refresh should have pulled remote changes")
	}
}

// TestRefreshMultipleRepos tests refreshing multiple repositories
func TestRefreshMultipleRepos(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")
	repo2 := tp.InitRepo("repo2")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Push updates to both remotes
	for i, repo := range []*TestRepo{repo1, repo2} {
		tempClone := filepath.Join(t.TempDir(), repo.Name+"-clone")
		runGitCmd(t, tempClone, "clone", repo.RemoteDir, ".")
		runGitCmd(t, tempClone, "config", "user.email", "test@test.com")
		runGitCmd(t, tempClone, "config", "user.name", "Test")
		runGitCmd(t, tempClone, "config", "commit.gpgsign", "false")
		runGitCmd(t, tempClone, "checkout", "main")

		updateFile := filepath.Join(tempClone, "update.txt")
		os.WriteFile(updateFile, []byte("update"), 0644)
		runGitCmd(t, tempClone, "add", ".")
		runGitCmd(t, tempClone, "commit", "-m", "update")
		runGitCmd(t, tempClone, "push", "origin", "main")

		// Verify update exists in remote
		t.Logf("Pushed update to repo%d", i+1)
	}

	// Refresh all repos
	err := runRefresh()
	if err != nil {
		t.Fatalf("runRefresh() error = %v", err)
	}

	// Verify both repos were updated
	for _, repo := range []*TestRepo{repo1, repo2} {
		pulledFile := filepath.Join(repo.SourceDir, "update.txt")
		if _, err := os.Stat(pulledFile); os.IsNotExist(err) {
			t.Errorf("repo %s should have pulled remote changes", repo.Name)
		}
	}
}

// TestRefreshRepositoryFunction tests the RefreshRepository helper
func TestRefreshRepositoryFunction(t *testing.T) {
	t.Run("successful refresh", func(t *testing.T) {
		tp := NewTestProject(t)
		repo1 := tp.InitRepo("repo1")

		// Push an update to remote
		tempClone := filepath.Join(t.TempDir(), "clone")
		runGitCmd(t, tempClone, "clone", repo1.RemoteDir, ".")
		runGitCmd(t, tempClone, "config", "user.email", "test@test.com")
		runGitCmd(t, tempClone, "config", "user.name", "Test")
		runGitCmd(t, tempClone, "config", "commit.gpgsign", "false")
		runGitCmd(t, tempClone, "checkout", "main")

		updateFile := filepath.Join(tempClone, "update.txt")
		os.WriteFile(updateFile, []byte("update"), 0644)
		runGitCmd(t, tempClone, "add", ".")
		runGitCmd(t, tempClone, "commit", "-m", "update")
		runGitCmd(t, tempClone, "push", "origin", "main")

		// Refresh the repository
		progress := ui.NewProgress()
		err := RefreshRepository(repo1.SourceDir, "repo1", progress)
		if err != nil {
			t.Fatalf("RefreshRepository() error = %v", err)
		}

		// Verify changes were pulled
		pulledFile := filepath.Join(repo1.SourceDir, "update.txt")
		if _, err := os.Stat(pulledFile); os.IsNotExist(err) {
			t.Error("RefreshRepository should have pulled remote changes")
		}
	})

	t.Run("non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		// Refresh non-git directory should not error
		progress := ui.NewProgress()
		err := RefreshRepository(tempDir, "fake-repo", progress)
		if err != nil {
			t.Errorf("RefreshRepository() with non-git dir should not error, got: %v", err)
		}
	})
}

// TestRefreshWithUncommittedChanges tests refresh behavior with local changes
func TestRefreshWithUncommittedChanges(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	// Make local uncommitted changes
	localFile := filepath.Join(repo1.SourceDir, "local-change.txt")
	os.WriteFile(localFile, []byte("local work"), 0644)

	// Push remote change
	tempClone := filepath.Join(t.TempDir(), "clone")
	runGitCmd(t, tempClone, "clone", repo1.RemoteDir, ".")
	runGitCmd(t, tempClone, "config", "user.email", "test@test.com")
	runGitCmd(t, tempClone, "config", "user.name", "Test")
	runGitCmd(t, tempClone, "config", "commit.gpgsign", "false")
	runGitCmd(t, tempClone, "checkout", "main")

	updateFile := filepath.Join(tempClone, "remote-change.txt")
	os.WriteFile(updateFile, []byte("remote"), 0644)
	runGitCmd(t, tempClone, "add", ".")
	runGitCmd(t, tempClone, "commit", "-m", "remote change")
	runGitCmd(t, tempClone, "push", "origin", "main")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Refresh - may fail or succeed depending on merge conflicts
	// We just verify it doesn't panic
	err := runRefresh()
	// It's OK if it fails due to uncommitted changes
	if err != nil {
		t.Logf("Refresh with uncommitted changes failed (expected): %v", err)
	}

	// Local change should still exist
	if _, err := os.Stat(localFile); os.IsNotExist(err) {
		t.Error("local changes should still exist after refresh")
	}
}

// TestRefreshBranchWithoutRemoteTracking tests refresh on detached or local-only branches
func TestRefreshBranchWithoutRemoteTracking(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	// Create a local-only branch
	runGitCmd(t, repo1.SourceDir, "checkout", "-b", "local-only-branch")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Refresh should handle this gracefully (skip pull)
	err := runRefresh()
	if err != nil {
		t.Fatalf("runRefresh() should handle local-only branch, got error: %v", err)
	}

	// Should still be on local-only-branch
	currentBranch := runGitCmdOutput(t, repo1.SourceDir, "symbolic-ref", "--short", "HEAD")
	if currentBranch != "local-only-branch" {
		t.Errorf("should still be on local-only-branch, got: %s", currentBranch)
	}
}

// TestRefreshEmptyProject tests refresh with no repos
func TestRefreshEmptyProject(t *testing.T) {
	tp := NewTestProject(t)
	// Don't initialize any repos

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Refresh should handle empty project
	err := runRefresh()
	if err != nil {
		t.Fatalf("runRefresh() with empty project should not error: %v", err)
	}
}

// TestRefreshAfterRepoDeleted tests refresh when repo directory is missing
func TestRefreshAfterRepoDeleted(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	// Delete the repo directory
	os.RemoveAll(repo1.SourceDir)

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Refresh should handle missing repo gracefully
	err := runRefresh()
	// May error due to auto-install trying to clone, but shouldn't panic
	if err != nil {
		t.Logf("Refresh with missing repo returned error (may be expected): %v", err)
	}
}

// TestRefreshWithDifferentBranch tests refresh when not on main
func TestRefreshWithDifferentBranch(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	// Create and checkout a feature branch
	runGitCmd(t, repo1.SourceDir, "checkout", "-b", "feature/test")
	runGitCmd(t, repo1.SourceDir, "push", "-u", "origin", "feature/test")

	// Push an update to the feature branch
	tempClone := filepath.Join(t.TempDir(), "clone")
	runGitCmd(t, tempClone, "clone", repo1.RemoteDir, ".")
	runGitCmd(t, tempClone, "config", "user.email", "test@test.com")
	runGitCmd(t, tempClone, "config", "user.name", "Test")
	runGitCmd(t, tempClone, "config", "commit.gpgsign", "false")
	runGitCmd(t, tempClone, "checkout", "feature/test")

	updateFile := filepath.Join(tempClone, "feature-update.txt")
	os.WriteFile(updateFile, []byte("feature work"), 0644)
	runGitCmd(t, tempClone, "add", ".")
	runGitCmd(t, tempClone, "commit", "-m", "feature update")
	runGitCmd(t, tempClone, "push", "origin", "feature/test")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Refresh should pull the feature branch updates
	err := runRefresh()
	if err != nil {
		t.Fatalf("runRefresh() error = %v", err)
	}

	// Verify the changes were pulled
	pulledFile := filepath.Join(repo1.SourceDir, "feature-update.txt")
	if _, err := os.Stat(pulledFile); os.IsNotExist(err) {
		t.Error("refresh should have pulled feature branch changes")
	}
}

// TestRefreshWithStashInWorktree tests that stashes from worktrees don't leak into source repos
// This is a regression test for: "I had a stash in a feature tree then called ramp refresh
// and the repo inside my .repos folder had pulled in the stash after pulling in changes from main"
func TestRefreshWithStashInWorktree(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a feature with worktree
	err := runUp("test-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Make changes in the worktree and create a stash
	worktreeDir := filepath.Join(tp.Dir, "trees", "test-feature", "repo1")
	stashFile := filepath.Join(worktreeDir, "stashed-change.txt")
	os.WriteFile(stashFile, []byte("This should be stashed"), 0644)

	// Create a stash in the worktree
	runGitCmd(t, worktreeDir, "add", ".")
	runGitCmd(t, worktreeDir, "stash", "push", "-m", "worktree stash")

	// Verify the stash exists
	stashList := runGitCmdOutput(t, worktreeDir, "stash", "list")
	if !strings.Contains(stashList, "worktree stash") {
		t.Fatalf("stash should exist in worktree, got: %s", stashList)
	}

	// Push a remote change to main
	tempClone := filepath.Join(t.TempDir(), "clone")
	runGitCmd(t, tempClone, "clone", repo1.RemoteDir, ".")
	runGitCmd(t, tempClone, "config", "user.email", "test@test.com")
	runGitCmd(t, tempClone, "config", "user.name", "Test")
	runGitCmd(t, tempClone, "config", "commit.gpgsign", "false")
	runGitCmd(t, tempClone, "checkout", "main")

	updateFile := filepath.Join(tempClone, "remote-update.txt")
	os.WriteFile(updateFile, []byte("remote update"), 0644)
	runGitCmd(t, tempClone, "add", ".")
	runGitCmd(t, tempClone, "commit", "-m", "remote update")
	runGitCmd(t, tempClone, "push", "origin", "main")

	// Now run refresh
	err = runRefresh()
	if err != nil {
		t.Fatalf("runRefresh() error = %v", err)
	}

	// Check source repo status - it should be clean (no uncommitted changes from stash)
	statusOutput := runGitCmdOutput(t, repo1.SourceDir, "status", "--porcelain")
	if statusOutput != "" {
		t.Errorf("source repo should be clean after refresh, but got uncommitted changes:\n%s", statusOutput)
	}

	// Verify the stashed file is NOT in the source repo working tree
	sourceStashFile := filepath.Join(repo1.SourceDir, "stashed-change.txt")
	if _, err := os.Stat(sourceStashFile); err == nil {
		t.Error("stashed file should NOT appear in source repo working tree after refresh")
	}

	// Verify remote update was pulled successfully
	pulledFile := filepath.Join(repo1.SourceDir, "remote-update.txt")
	if _, err := os.Stat(pulledFile); os.IsNotExist(err) {
		t.Error("refresh should have pulled remote changes")
	}

	// The stash should still exist (it's shared across worktrees)
	// but should not be automatically applied
	stashListAfter := runGitCmdOutput(t, repo1.SourceDir, "stash", "list")
	if !strings.Contains(stashListAfter, "worktree stash") {
		t.Logf("Note: stash list after refresh: %s", stashListAfter)
	}
}

// TestRefreshWithStashAndAutoStashConfig tests the edge case where rebase.autoStash is enabled
// This can cause stashes to be automatically applied during git pull
func TestRefreshWithStashAndAutoStashConfig(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Enable pull.rebase and rebase.autoStash - this can cause stashes to auto-apply during pull
	runGitCmd(t, repo1.SourceDir, "config", "pull.rebase", "true")
	runGitCmd(t, repo1.SourceDir, "config", "rebase.autoStash", "true")

	// Create a feature with worktree
	err := runUp("test-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Make changes in the worktree and create a stash
	worktreeDir := filepath.Join(tp.Dir, "trees", "test-feature", "repo1")
	stashFile := filepath.Join(worktreeDir, "stashed-change.txt")
	os.WriteFile(stashFile, []byte("This should be stashed"), 0644)

	// Create a stash in the worktree
	runGitCmd(t, worktreeDir, "add", ".")
	runGitCmd(t, worktreeDir, "stash", "push", "-m", "worktree stash")

	// Add uncommitted changes to the source repo BEFORE pushing remote changes
	// This simulates the user working in the source repo while having stashes from worktrees
	sourceUncommittedFile := filepath.Join(repo1.SourceDir, "source-uncommitted.txt")
	os.WriteFile(sourceUncommittedFile, []byte("uncommitted work in source"), 0644)

	// Push a remote change to main that would require a rebase
	tempClone := filepath.Join(t.TempDir(), "clone")
	runGitCmd(t, tempClone, "clone", repo1.RemoteDir, ".")
	runGitCmd(t, tempClone, "config", "user.email", "test@test.com")
	runGitCmd(t, tempClone, "config", "user.name", "Test")
	runGitCmd(t, tempClone, "config", "commit.gpgsign", "false")
	runGitCmd(t, tempClone, "checkout", "main")

	updateFile := filepath.Join(tempClone, "remote-update.txt")
	os.WriteFile(updateFile, []byte("remote update"), 0644)
	runGitCmd(t, tempClone, "add", ".")
	runGitCmd(t, tempClone, "commit", "-m", "remote update")
	runGitCmd(t, tempClone, "push", "origin", "main")

	// Now run refresh - with autostash, it should stash source changes, pull, then pop
	err = runRefresh()
	if err != nil {
		t.Fatalf("runRefresh() error = %v", err)
	}

	// Check source repo status
	statusOutput := runGitCmdOutput(t, repo1.SourceDir, "status", "--porcelain")

	// The source repo should still have its own uncommitted file
	if _, err := os.Stat(sourceUncommittedFile); os.IsNotExist(err) {
		t.Error("source repo's own uncommitted file should still exist")
	}

	// Check if the stashed file from worktree appeared in source repo
	sourceStashFile := filepath.Join(repo1.SourceDir, "stashed-change.txt")
	if _, err := os.Stat(sourceStashFile); err == nil {
		t.Error("CONFIRMED BUG: stashed file from worktree appeared in source repo working tree!")
		t.Errorf("Source repo status:\n%s", statusOutput)
	}
}

// TestStashesAreSharedAcrossWorktrees documents that git stashes are shared across all worktrees
// This is important context for understanding potential stash-related issues
func TestStashesAreSharedAcrossWorktrees(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a feature with worktree
	err := runUp("test-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	worktreeDir := filepath.Join(tp.Dir, "trees", "test-feature", "repo1")

	// Create a stash in the WORKTREE
	stashFile := filepath.Join(worktreeDir, "worktree-change.txt")
	os.WriteFile(stashFile, []byte("change in worktree"), 0644)
	runGitCmd(t, worktreeDir, "add", ".")
	runGitCmd(t, worktreeDir, "stash", "push", "-m", "stash from worktree")

	// Check stash list in worktree
	worktreeStashList := runGitCmdOutput(t, worktreeDir, "stash", "list")
	t.Logf("Stash list from worktree:\n%s", worktreeStashList)

	// Check stash list in SOURCE REPO
	sourceStashList := runGitCmdOutput(t, repo1.SourceDir, "stash", "list")
	t.Logf("Stash list from source repo:\n%s", sourceStashList)

	// VERIFY: The stash should be visible in BOTH locations
	if !strings.Contains(sourceStashList, "stash from worktree") {
		t.Error("Stashes should be shared: stash from worktree should be visible in source repo")
	}

	if worktreeStashList != sourceStashList {
		t.Error("Stash lists should be identical across worktrees and source repo")
	}

	t.Log("✓ CONFIRMED: Git stashes are shared across all worktrees")
	t.Log("  This means a 'git stash pop' in source repo would apply worktree's stash!")
}

// Helper function to run git and get output (for tests)
func runGitCmdOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return strings.TrimSpace(string(output))
}
