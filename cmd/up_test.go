package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUpBasic tests creating a basic feature
func TestUpBasic(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")
	tp.InitRepo("repo2")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature
	err := runUp("my-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify feature directory was created
	if !tp.FeatureExists("my-feature") {
		t.Error("feature directory was not created")
	}

	// Verify worktrees were created for both repos
	if !tp.WorktreeExists("my-feature", "repo1") {
		t.Error("worktree for repo1 was not created")
	}

	if !tp.WorktreeExists("my-feature", "repo2") {
		t.Error("worktree for repo2 was not created")
	}

	// Verify branches were created with correct prefix
	repo1 := tp.Repos["repo1"]
	if !repo1.BranchExists(t, "feature/my-feature") {
		t.Error("branch feature/my-feature was not created in repo1")
	}

	repo2 := tp.Repos["repo2"]
	if !repo2.BranchExists(t, "feature/my-feature") {
		t.Error("branch feature/my-feature was not created in repo2")
	}
}

// TestUpWithCustomPrefix tests creating a feature with custom prefix
func TestUpWithCustomPrefix(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature with custom prefix
	err := runUp("test-feature", "bugfix/", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify branch was created with custom prefix
	repo1 := tp.Repos["repo1"]
	if !repo1.BranchExists(t, "bugfix/test-feature") {
		t.Error("branch bugfix/test-feature was not created")
	}

	// Should NOT create with default prefix
	if repo1.BranchExists(t, "feature/test-feature") {
		t.Error("branch with default prefix should not exist")
	}
}

// TestUpWithNoPrefix tests creating a feature without prefix
func TestUpWithNoPrefix(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Set no-prefix flag
	noPrefixFlag = true
	defer func() { noPrefixFlag = false }()

	// Create feature without prefix
	err := runUp("plain-branch", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify branch was created without prefix
	repo1 := tp.Repos["repo1"]
	if !repo1.BranchExists(t, "plain-branch") {
		t.Error("branch plain-branch was not created")
	}

	// Should NOT have prefix
	if repo1.BranchExists(t, "feature/plain-branch") {
		t.Error("branch should not have prefix")
	}
}

// TestUpDuplicateFeature tests error when feature already exists
func TestUpDuplicateFeature(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature first time
	err := runUp("duplicate", "", "")
	if err != nil {
		t.Fatalf("first runUp() error = %v", err)
	}

	// Try to create again - should fail
	err = runUp("duplicate", "", "")
	if err == nil {
		t.Error("runUp() should return error for duplicate feature")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}
}

// TestUpWithTarget tests creating feature from existing branch
func TestUpWithTarget(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	// Create a source branch with a commit
	repo1.CreateBranch(t, "feature/source")
	runGitCmd(t, repo1.SourceDir, "checkout", "feature/source")
	repo1.AddCommit(t, "source commit")
	runGitCmd(t, repo1.SourceDir, "checkout", "main")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create new feature from source
	err := runUp("derived", "", "source")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify the new branch exists
	if !repo1.BranchExists(t, "feature/derived") {
		t.Error("derived branch was not created")
	}

	// Verify worktree was created
	if !tp.WorktreeExists("derived", "repo1") {
		t.Error("worktree was not created")
	}
}

// TestUpWithTargetRemoteBranch tests creating from remote branch
func TestUpWithTargetRemoteBranch(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	// Create and push a remote branch
	repo1.CreateBranch(t, "feature/remote-source")

	// Delete local branch to ensure we're using remote
	runGitCmd(t, repo1.SourceDir, "branch", "-D", "feature/remote-source")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create from remote branch
	err := runUp("from-remote", "", "origin/feature/remote-source")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify branch was created
	if !repo1.BranchExists(t, "feature/from-remote") {
		t.Error("branch was not created from remote source")
	}
}

// TestUpRollbackOnFailure tests that partial work is cleaned up on failure

// TestUpPortAllocation tests that ports are allocated correctly
func TestUpPortAllocation(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create first feature
	err := runUp("feature1", "", "")
	if err != nil {
		t.Fatalf("runUp() feature1 error = %v", err)
	}

	// Create second feature
	err = runUp("feature2", "", "")
	if err != nil {
		t.Fatalf("runUp() feature2 error = %v", err)
	}

	// Verify port allocations file exists
	portFile := filepath.Join(tp.RampDir, "port_allocations.json")
	if _, err := os.Stat(portFile); os.IsNotExist(err) {
		t.Error("port allocations file was not created")
	}

	// Note: We don't check exact port numbers here because the port package
	// is already tested separately. We just verify the file exists.
}

// TestUpCreatesTreesDirectory tests that trees directory is created
func TestUpCreatesTreesDirectory(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	// Remove trees directory
	os.RemoveAll(tp.TreesDir)

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	err := runUp("test", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify trees directory was created
	if _, err := os.Stat(tp.TreesDir); os.IsNotExist(err) {
		t.Error("trees directory was not created")
	}
}

// TestUpInvalidFeatureName tests error handling for invalid names
func TestUpInvalidFeatureName(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Test empty name (should be caught by cobra, but test the function)
	err := runUp("", "", "")
	if err == nil {
		t.Error("runUp() should fail with empty feature name")
	}
}

// TestUpWithNonExistentTarget tests error when target doesn't exist

// TestUpMultipleRepos tests that all repos get worktrees
func TestUpMultipleRepos(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")
	tp.InitRepo("repo2")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	err := runUp("multi", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify both repos have worktrees
	repos := []string{"repo1", "repo2"}
	for _, repoName := range repos {
		if !tp.WorktreeExists("multi", repoName) {
			t.Errorf("worktree for %s was not created", repoName)
		}

		// Verify branch exists
		repo := tp.Repos[repoName]
		if !repo.BranchExists(t, "feature/multi") {
			t.Errorf("branch was not created in %s", repoName)
		}
	}
}

// TestUpWithExistingLocalBranch tests using existing local branch
func TestUpWithExistingLocalBranch(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	// Pre-create the branch
	repo1.CreateBranch(t, "feature/existing")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature - should use existing branch
	err := runUp("existing", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify worktree uses the existing branch
	if !tp.WorktreeExists("existing", "repo1") {
		t.Error("worktree was not created")
	}

	// The branch should still exist (not recreated)
	if !repo1.BranchExists(t, "feature/existing") {
		t.Error("existing branch should still exist")
	}
}

// TestUpWithExistingRemoteBranch tests tracking remote branch
func TestUpWithExistingRemoteBranch(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	// Create and push remote branch
	repo1.CreateBranch(t, "feature/remote")

	// Delete local branch
	runGitCmd(t, repo1.SourceDir, "branch", "-D", "feature/remote")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature - should track remote branch
	err := runUp("remote", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify worktree was created
	if !tp.WorktreeExists("remote", "repo1") {
		t.Error("worktree was not created")
	}

	// Verify local branch now exists (tracking remote)
	if !repo1.BranchExists(t, "feature/remote") {
		t.Error("local tracking branch was not created")
	}
}

// TestUpPreservesSlashInFeatureName tests feature names with slashes
func TestUpPreservesSlashInFeatureName(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature with slash in name
	err := runUp("epic/sub-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify directory uses the full name
	if !tp.FeatureExists("epic/sub-feature") {
		t.Error("feature directory with slash was not created")
	}

	// Verify branch includes prefix + full name
	repo1 := tp.Repos["repo1"]
	if !repo1.BranchExists(t, "feature/epic/sub-feature") {
		t.Error("branch with nested path was not created")
	}
}
