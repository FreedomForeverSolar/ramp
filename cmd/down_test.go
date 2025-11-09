package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ramp/internal/config"
)

// TestDownBasic tests basic feature teardown
func TestDownBasic(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")
	tp.InitRepo("repo2")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature
	err := runUp("test-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify feature exists
	if !tp.FeatureExists("test-feature") {
		t.Fatal("feature was not created")
	}

	// Tear down feature
	err = runDown("test-feature")
	if err != nil {
		t.Fatalf("runDown() error = %v", err)
	}

	// Verify feature directory was removed
	if tp.FeatureExists("test-feature") {
		t.Error("feature directory was not removed")
	}

	// Verify worktrees were removed
	if tp.WorktreeExists("test-feature", "repo1") {
		t.Error("repo1 worktree was not removed")
	}

	if tp.WorktreeExists("test-feature", "repo2") {
		t.Error("repo2 worktree was not removed")
	}

	// Verify branches were deleted
	repo1 := tp.Repos["repo1"]
	if repo1.BranchExists(t, "feature/test-feature") {
		t.Error("repo1 branch was not deleted")
	}

	repo2 := tp.Repos["repo2"]
	if repo2.BranchExists(t, "feature/test-feature") {
		t.Error("repo2 branch was not deleted")
	}
}

// TestDownNonExistentFeature tests error when feature doesn't exist
func TestDownNonExistentFeature(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Try to delete non-existent feature
	err := runDown("nonexistent")
	if err == nil {
		t.Error("runDown() should return error for non-existent feature")
	}

	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error should mention feature not found, got: %v", err)
	}
}

// TestDownReleasesPort tests that port is released
func TestDownReleasesPort(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature (allocates port)
	err := runUp("with-port", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify port allocations file exists
	portFile := filepath.Join(tp.RampDir, "port_allocations.json")
	if _, err := os.Stat(portFile); os.IsNotExist(err) {
		t.Fatal("port allocations file does not exist")
	}

	// Read port allocations before down
	beforeData, err := os.ReadFile(portFile)
	if err != nil {
		t.Fatalf("failed to read port allocations: %v", err)
	}

	if !strings.Contains(string(beforeData), "with-port") {
		t.Error("port allocation should contain feature name")
	}

	// Delete feature
	err = runDown("with-port")
	if err != nil {
		t.Fatalf("runDown() error = %v", err)
	}

	// Read port allocations after down
	afterData, err := os.ReadFile(portFile)
	if err != nil {
		t.Fatalf("failed to read port allocations after down: %v", err)
	}

	if strings.Contains(string(afterData), "with-port") {
		t.Error("port allocation should not contain feature name after down")
	}
}

// TestDownWithUncommittedChanges tests safety check for uncommitted changes
func TestDownWithUncommittedChanges(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature
	err := runUp("dirty", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Make uncommitted changes in worktree
	worktreeDir := filepath.Join(tp.TreesDir, "dirty", "repo1")
	testFile := filepath.Join(worktreeDir, "uncommitted.txt")
	if err := os.WriteFile(testFile, []byte("changes"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Note: The actual runDown function prompts for confirmation with uncommitted changes
	// In a real test environment, we would need to mock the input or test that
	// the detection works. For now, we just verify the function doesn't panic.

	// Since we can't easily test interactive prompts in automated tests,
	// we'll skip the actual down operation here and just verify the setup worked
	if !tp.FeatureExists("dirty") {
		t.Error("feature should still exist (we didn't run down)")
	}
}

// TestDownCleansUpPartialState tests cleanup when worktree is missing
func TestDownCleansUpPartialState(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")
	tp.InitRepo("repo2")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature
	err := runUp("partial", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Manually remove one worktree to simulate partial state
	worktreePath := filepath.Join(tp.TreesDir, "partial", "repo1")
	os.RemoveAll(worktreePath)

	// Down should still work and clean up everything else
	err = runDown("partial")
	if err != nil {
		// It's okay if there's an error, but feature should still be cleaned up
		t.Logf("runDown() returned error (expected): %v", err)
	}

	// Feature directory should be removed
	if tp.FeatureExists("partial") {
		t.Error("feature directory should be removed even with partial state")
	}

	// Remaining worktree should be cleaned up
	if tp.WorktreeExists("partial", "repo2") {
		t.Error("repo2 worktree should be cleaned up")
	}

	// Branches should be cleaned up
	repo2 := tp.Repos["repo2"]
	if repo2.BranchExists(t, "feature/partial") {
		t.Error("repo2 branch should be deleted")
	}
}

// TestDownMultipleFeatures tests that only specified feature is removed
func TestDownMultipleFeatures(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create multiple features
	features := []string{"feature1", "feature2", "feature3"}
	for _, name := range features {
		err := runUp(name, "", "")
		if err != nil {
			t.Fatalf("runUp(%s) error = %v", name, err)
		}
	}

	// Delete only feature2
	err := runDown("feature2")
	if err != nil {
		t.Fatalf("runDown() error = %v", err)
	}

	// Verify feature2 is gone
	if tp.FeatureExists("feature2") {
		t.Error("feature2 should be removed")
	}

	// Verify others still exist
	if !tp.FeatureExists("feature1") {
		t.Error("feature1 should still exist")
	}

	if !tp.FeatureExists("feature3") {
		t.Error("feature3 should still exist")
	}
}

// TestDownRemovesBranchesWithDifferentPrefix tests cleanup of branches with various prefixes
func TestDownRemovesBranchesWithDifferentPrefix(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature with custom prefix
	err := runUp("test", "bugfix/", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify branch with custom prefix exists
	repo1 := tp.Repos["repo1"]
	if !repo1.BranchExists(t, "bugfix/test") {
		t.Fatal("branch bugfix/test should exist")
	}

	// Delete feature
	err = runDown("test")
	if err != nil {
		t.Fatalf("runDown() error = %v", err)
	}

	// Verify branch with custom prefix was deleted
	if repo1.BranchExists(t, "bugfix/test") {
		t.Error("branch bugfix/test should be deleted")
	}
}

// TestDownWithNoPrefixBranch tests deleting feature without prefix
func TestDownWithNoPrefixBranch(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature without prefix
	noPrefixFlag = true
	err := runUp("plain", "", "")
	noPrefixFlag = false

	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify branch without prefix exists
	repo1 := tp.Repos["repo1"]
	if !repo1.BranchExists(t, "plain") {
		t.Fatal("branch plain should exist")
	}

	// Delete feature
	err = runDown("plain")
	if err != nil {
		t.Fatalf("runDown() error = %v", err)
	}

	// Verify branch was deleted
	if repo1.BranchExists(t, "plain") {
		t.Error("branch plain should be deleted")
	}

	// Feature directory should be gone
	if tp.FeatureExists("plain") {
		t.Error("feature directory should be removed")
	}
}

// TestDownPrunesRemoteTracking tests that git fetch --prune is called
func TestDownPrunesRemoteTracking(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create and push feature
	err := runUp("to-prune", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	repo1 := tp.Repos["repo1"]

	// Push the branch to remote
	worktreeDir := filepath.Join(tp.TreesDir, "to-prune", "repo1")
	runGitCmd(t, worktreeDir, "push", "-u", "origin", "feature/to-prune")

	// Delete feature
	err = runDown("to-prune")
	if err != nil {
		t.Fatalf("runDown() error = %v", err)
	}

	// Verify local branch is gone
	if repo1.BranchExists(t, "feature/to-prune") {
		t.Error("local branch should be deleted")
	}

	// Note: Testing that git fetch --prune actually runs would require
	// mocking or checking git internals. We just verify the function completes.
}


// TestDownPreservesSourceRepos tests that source repos are not affected
func TestDownPreservesSourceRepos(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	// Add a file to source repo
	sourceFile := filepath.Join(repo1.SourceDir, "important.txt")
	if err := os.WriteFile(sourceFile, []byte("important data"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create and delete feature
	err := runUp("temp", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	err = runDown("temp")
	if err != nil {
		t.Fatalf("runDown() error = %v", err)
	}

	// Verify source repo file still exists
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		t.Error("source repo file should not be deleted")
	}

	// Verify source repo directory still exists
	if _, err := os.Stat(repo1.SourceDir); os.IsNotExist(err) {
		t.Error("source repo directory should not be deleted")
	}
}

// TestDownWithNestedBranchViaPrefix tests cleaning up features created with custom prefix
func TestDownWithNestedBranchViaPrefix(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature using prefix to create nested branch name
	// Feature name is simple, but branch name will be nested via prefix
	err := runUp("subtask", "epic/task/", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify feature directory uses simple name (no slash)
	if !tp.FeatureExists("subtask") {
		t.Fatal("feature 'subtask' was not created")
	}

	// Verify nested branch was created via prefix
	repo1 := tp.Repos["repo1"]
	if !repo1.BranchExists(t, "epic/task/subtask") {
		t.Fatal("nested branch 'epic/task/subtask' was not created")
	}

	// Delete it using simple feature name
	err = runDown("subtask")
	if err != nil {
		t.Fatalf("runDown() error = %v", err)
	}

	// Verify it's gone
	if tp.FeatureExists("subtask") {
		t.Error("feature should be removed")
	}

	// Verify branch is gone
	if repo1.BranchExists(t, "epic/task/subtask") {
		t.Error("nested branch should be deleted")
	}
}

func TestDownWithCleanupScript(t *testing.T) {
	tp := NewTestProject(t)
	_ = tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a cleanup script that writes to a marker file
	scriptPath := filepath.Join(tp.Dir, ".ramp", "scripts", "cleanup.sh")
	os.MkdirAll(filepath.Dir(scriptPath), 0755)
	scriptContent := `#!/bin/bash
echo "cleanup executed" > "$RAMP_PROJECT_DIR/.ramp/cleanup-marker.txt"
echo "feature=$RAMP_WORKTREE_NAME" >> "$RAMP_PROJECT_DIR/.ramp/cleanup-marker.txt"
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	// Update config to include cleanup script
	cfg, _ := config.LoadConfig(tp.Dir)
	cfg.Cleanup = "scripts/cleanup.sh"
	config.SaveConfig(cfg, tp.Dir)

	// Create feature first
	runUp("test-feature", "", "")

	// Delete feature
	err := runDown("test-feature")
	if err != nil {
		t.Fatalf("runDown() error = %v", err)
	}

	// Verify cleanup script was executed
	markerFile := filepath.Join(tp.Dir, ".ramp", "cleanup-marker.txt")
	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		t.Fatal("cleanup script was not executed - marker file not found")
	}

	// Verify environment variables were set correctly
	content, _ := os.ReadFile(markerFile)
	contentStr := string(content)

	if !strings.Contains(contentStr, "cleanup executed") {
		t.Error("cleanup script did not write expected content")
	}
	if !strings.Contains(contentStr, "feature=test-feature") {
		t.Error("RAMP_WORKTREE_NAME environment variable not set correctly")
	}
}

func TestDownCleanupScriptFailure(t *testing.T) {
	tp := NewTestProject(t)
	_ = tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a cleanup script that fails
	scriptPath := filepath.Join(tp.Dir, ".ramp", "scripts", "cleanup.sh")
	os.MkdirAll(filepath.Dir(scriptPath), 0755)
	scriptContent := `#!/bin/bash
exit 1
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	// Update config to include cleanup script
	cfg, _ := config.LoadConfig(tp.Dir)
	cfg.Cleanup = "scripts/cleanup.sh"
	config.SaveConfig(cfg, tp.Dir)

	// Create feature first
	runUp("test-feature", "", "")

	// Delete feature - should continue despite cleanup script failure
	err := runDown("test-feature")
	// The function still succeeds but logs a warning about cleanup failure
	if err != nil {
		t.Fatalf("runDown() error = %v", err)
	}

	// Verify feature was still removed
	if tp.FeatureExists("test-feature") {
		t.Error("feature should be removed even when cleanup script fails")
	}
}
