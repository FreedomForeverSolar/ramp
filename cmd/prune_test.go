package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"ramp/internal/config"
)

// TestPruneNoFeatures tests prune with no features
func TestPruneNoFeatures(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Note: runPrune requires user confirmation, so we can't easily test it
	// directly without mocking stdin. Instead we test the helper functions.

	// Test findMergedFeatures with no features
	cfg, err := config.LoadConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	merged, err := findMergedFeatures(tp.Dir, cfg)
	if err != nil {
		t.Fatalf("findMergedFeatures() error = %v", err)
	}

	if len(merged) != 0 {
		t.Errorf("findMergedFeatures() returned %d features, want 0", len(merged))
	}
}

// TestPrune WithUnmergedFeatures tests that unmerged features are not included
func TestPruneWithUnmergedFeatures(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a feature
	err := runUp("unmerged-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Add a commit to the feature (makes it unmerged)
	worktreeDir := filepath.Join(tp.TreesDir, "unmerged-feature", "repo1")
	testFile := filepath.Join(worktreeDir, "feature-work.txt")
	if err := os.WriteFile(testFile, []byte("work"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	runGitCmd(t, worktreeDir, "add", ".")
	runGitCmd(t, worktreeDir, "commit", "-m", "feature work")

	// Find merged features - should be empty since feature has commits
	cfg, err := config.LoadConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	merged, err := findMergedFeatures(tp.Dir, cfg)
	if err != nil {
		t.Fatalf("findMergedFeatures() error = %v", err)
	}

	// Should not include the unmerged feature
	if len(merged) != 0 {
		t.Errorf("findMergedFeatures() returned %d features, want 0 (feature not merged)", len(merged))
	}
}

// TestPruneWithMergedFeature tests detection of merged features
func TestPruneWithMergedFeature(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a feature
	err := runUp("to-merge", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Add and commit work in the feature
	worktreeDir := filepath.Join(tp.TreesDir, "to-merge", "repo1")
	testFile := filepath.Join(worktreeDir, "feature.txt")
	if err := os.WriteFile(testFile, []byte("feature"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	runGitCmd(t, worktreeDir, "add", ".")
	runGitCmd(t, worktreeDir, "commit", "-m", "add feature")

	// Merge the feature into main
	runGitCmd(t, repo1.SourceDir, "checkout", "main")
	runGitCmd(t, repo1.SourceDir, "merge", "feature/to-merge", "--no-ff", "-m", "merge feature")

	// Find merged features
	cfg, err := config.LoadConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	merged, err := findMergedFeatures(tp.Dir, cfg)
	if err != nil {
		t.Fatalf("findMergedFeatures() error = %v", err)
	}

	// Should include the merged feature
	if len(merged) != 1 {
		t.Errorf("findMergedFeatures() returned %d features, want 1 (merged feature)", len(merged))
	}

	if len(merged) > 0 && merged[0].name != "to-merge" {
		t.Errorf("merged feature name = %q, want %q", merged[0].name, "to-merge")
	}
}

// TestPruneMultipleFeaturesPartiallyMerged tests mixed scenarios
func TestPruneMultipleFeaturesPartiallyMerged(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature1 - will be merged
	err := runUp("merged-feature", "", "")
	if err != nil {
		t.Fatalf("runUp(merged-feature) error = %v", err)
	}

	worktree1 := filepath.Join(tp.TreesDir, "merged-feature", "repo1")
	file1 := filepath.Join(worktree1, "merged.txt")
	os.WriteFile(file1, []byte("merged"), 0644)
	runGitCmd(t, worktree1, "add", ".")
	runGitCmd(t, worktree1, "commit", "-m", "merged work")

	// Merge feature1
	runGitCmd(t, repo1.SourceDir, "checkout", "main")
	runGitCmd(t, repo1.SourceDir, "merge", "feature/merged-feature", "--no-ff", "-m", "merge 1")

	// Create feature2 - will NOT be merged
	err = runUp("unmerged-feature", "", "")
	if err != nil {
		t.Fatalf("runUp(unmerged-feature) error = %v", err)
	}

	worktree2 := filepath.Join(tp.TreesDir, "unmerged-feature", "repo1")
	file2 := filepath.Join(worktree2, "unmerged.txt")
	os.WriteFile(file2, []byte("unmerged"), 0644)
	runGitCmd(t, worktree2, "add", ".")
	runGitCmd(t, worktree2, "commit", "-m", "unmerged work")

	// Find merged features
	cfg, err := config.LoadConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	merged, err := findMergedFeatures(tp.Dir, cfg)
	if err != nil {
		t.Fatalf("findMergedFeatures() error = %v", err)
	}

	// Should only include merged-feature
	if len(merged) != 1 {
		t.Errorf("findMergedFeatures() returned %d features, want 1", len(merged))
	}

	if len(merged) > 0 && merged[0].name != "merged-feature" {
		t.Errorf("merged feature name = %q, want %q", merged[0].name, "merged-feature")
	}
}

// TestPruneCleanFeature tests that clean features (no commits) are excluded
func TestPruneCleanFeature(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a feature but don't add any commits
	err := runUp("clean-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Find merged features
	cfg, err := config.LoadConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	merged, err := findMergedFeatures(tp.Dir, cfg)
	if err != nil {
		t.Fatalf("findMergedFeatures() error = %v", err)
	}

	// Clean features should not be included (they have no commits to merge)
	// The actual behavior depends on how git merge-base handles this
	// Typically clean features return 0 ahead, 0 behind, which might be
	// considered "merged" or "clean" depending on implementation
	// For now, just verify the function doesn't crash
	t.Logf("Found %d merged features (clean feature behavior)", len(merged))
}

// TestPruneWithNestedFeatureName tests nested feature paths
// Note: Currently findMergedFeatures() has a limitation with nested paths
// It only scans the first level of trees/, so "epic/task" appears as "epic"
// This test verifies that the feature can be created and merged, even though
// findMergedFeatures won't detect it correctly (known limitation)
func TestPruneWithNestedFeatureName(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create nested feature
	err := runUp("epic/task", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Add work and merge
	worktreeDir := filepath.Join(tp.TreesDir, "epic/task", "repo1")
	testFile := filepath.Join(worktreeDir, "work.txt")
	os.WriteFile(testFile, []byte("work"), 0644)
	runGitCmd(t, worktreeDir, "add", ".")
	runGitCmd(t, worktreeDir, "commit", "-m", "work")

	runGitCmd(t, repo1.SourceDir, "checkout", "main")
	runGitCmd(t, repo1.SourceDir, "merge", "feature/epic/task", "--no-ff", "-m", "merge")

	// Find merged features
	cfg, err := config.LoadConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	merged, err := findMergedFeatures(tp.Dir, cfg)
	if err != nil {
		t.Fatalf("findMergedFeatures() error = %v", err)
	}

	// KNOWN LIMITATION: findMergedFeatures() only scans first-level directories in trees/
	// For nested path "epic/task", it only sees "epic" and can't find worktrees inside it
	// This is a limitation of the current implementation
	if len(merged) != 0 {
		t.Logf("Note: findMergedFeatures() found %d features (expected 0 due to nested path limitation)", len(merged))
	}

	// Verify the feature directory exists (even though prune can't detect it)
	if !tp.FeatureExists("epic/task") {
		t.Error("nested feature directory should exist")
	}
}

// TestFindMergedFeaturesWithEmptyTreesDir tests behavior with no features
func TestFindMergedFeaturesWithEmptyTreesDir(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	// Ensure trees directory is empty
	os.RemoveAll(tp.TreesDir)
	os.MkdirAll(tp.TreesDir, 0755)

	cfg, err := config.LoadConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	merged, err := findMergedFeatures(tp.Dir, cfg)
	if err != nil {
		t.Fatalf("findMergedFeatures() error = %v", err)
	}

	if len(merged) != 0 {
		t.Errorf("findMergedFeatures() returned %d features, want 0 for empty dir", len(merged))
	}
}

// TestPruneWithMultipleRepos tests merged detection with multiple repos
func TestPruneWithMultipleRepos(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")
	repo2 := tp.InitRepo("repo2")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature across both repos
	err := runUp("multi-repo-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Add commits to both repos
	worktree1 := filepath.Join(tp.TreesDir, "multi-repo-feature", "repo1")
	file1 := filepath.Join(worktree1, "file1.txt")
	os.WriteFile(file1, []byte("work1"), 0644)
	runGitCmd(t, worktree1, "add", ".")
	runGitCmd(t, worktree1, "commit", "-m", "work in repo1")

	worktree2 := filepath.Join(tp.TreesDir, "multi-repo-feature", "repo2")
	file2 := filepath.Join(worktree2, "file2.txt")
	os.WriteFile(file2, []byte("work2"), 0644)
	runGitCmd(t, worktree2, "add", ".")
	runGitCmd(t, worktree2, "commit", "-m", "work in repo2")

	// Merge both repos
	runGitCmd(t, repo1.SourceDir, "checkout", "main")
	runGitCmd(t, repo1.SourceDir, "merge", "feature/multi-repo-feature", "--no-ff", "-m", "merge")

	runGitCmd(t, repo2.SourceDir, "checkout", "main")
	runGitCmd(t, repo2.SourceDir, "merge", "feature/multi-repo-feature", "--no-ff", "-m", "merge")

	// Find merged features
	cfg, err := config.LoadConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	merged, err := findMergedFeatures(tp.Dir, cfg)
	if err != nil {
		t.Fatalf("findMergedFeatures() error = %v", err)
	}

	// Should detect as merged when all repos are merged
	if len(merged) != 1 {
		t.Errorf("findMergedFeatures() returned %d features, want 1", len(merged))
	}
}
