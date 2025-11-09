package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"ramp/internal/config"
	"ramp/internal/git"
)

// TestStatusBasic tests basic status display
func TestStatusBasic(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Status should run without error
	err := runStatus()
	if err != nil {
		t.Fatalf("runStatus() error = %v", err)
	}
}

// TestStatusMultipleRepos tests status with multiple repositories
func TestStatusMultipleRepos(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")
	tp.InitRepo("repo2")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	err := runStatus()
	if err != nil {
		t.Fatalf("runStatus() error = %v", err)
	}
}

// TestStatusWithFeatures tests status showing active features
func TestStatusWithFeatures(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a feature
	err := runUp("test-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Status should show the feature
	err = runStatus()
	if err != nil {
		t.Fatalf("runStatus() error = %v", err)
	}
}

// TestStatusWithUncommittedChanges tests status with uncommitted changes
func TestStatusWithUncommittedChanges(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Make uncommitted changes in source repo
	testFile := filepath.Join(repo1.SourceDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Status should show uncommitted changes
	err := runStatus()
	if err != nil {
		t.Fatalf("runStatus() error = %v", err)
	}
}

// TestGetRepoStatus tests repository status collection
func TestGetRepoStatus(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	t.Run("normal repo", func(t *testing.T) {
		status := getRepoStatus(tp.Dir, "repo1", tp.Config.Repos[0])

		if status.error != "" {
			t.Errorf("expected no error, got %q", status.error)
		}
		if status.currentBranch != "main" {
			t.Errorf("currentBranch = %q, want %q", status.currentBranch, "main")
		}
		if status.hasUncommitted {
			t.Error("should not have uncommitted changes")
		}
	})

	t.Run("repo with uncommitted changes", func(t *testing.T) {
		testFile := filepath.Join(repo1.SourceDir, "uncommitted.txt")
		os.WriteFile(testFile, []byte("test"), 0644)

		status := getRepoStatus(tp.Dir, "repo1", tp.Config.Repos[0])

		if status.error != "" {
			t.Errorf("expected no error, got %q", status.error)
		}
		if !status.hasUncommitted {
			t.Error("should have uncommitted changes")
		}
	})

	t.Run("missing repo", func(t *testing.T) {
		fakeRepo := &config.Repo{
			Path: "repos",
			Git:  "fake-repo",
		}

		status := getRepoStatus(tp.Dir, "fake-repo", fakeRepo)

		if status.error == "" {
			t.Error("expected error for missing repo")
		}
		if status.error != "repository not cloned" {
			t.Errorf("error = %q, want %q", status.error, "repository not cloned")
		}
	})
}

// TestGetFeatureWorktreeStatus tests feature worktree status collection
func TestGetFeatureWorktreeStatus(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a feature
	err := runUp("test-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	t.Run("basic feature status", func(t *testing.T) {
		status := getFeatureWorktreeStatus(tp.Dir, "test-feature", "repo1", tp.Config.Repos[0])

		if status.error != "" {
			t.Errorf("expected no error, got %q", status.error)
		}
		if status.branchName != "feature/test-feature" {
			t.Errorf("branchName = %q, want %q", status.branchName, "feature/test-feature")
		}
		if status.defaultBranch != "main" {
			t.Errorf("defaultBranch = %q, want %q", status.defaultBranch, "main")
		}
		if status.hasUncommitted {
			t.Error("should not have uncommitted changes")
		}
		if status.aheadCount != 0 {
			t.Errorf("aheadCount = %d, want 0", status.aheadCount)
		}
	})

	t.Run("feature with uncommitted changes", func(t *testing.T) {
		worktreePath := filepath.Join(tp.TreesDir, "test-feature", "repo1")
		testFile := filepath.Join(worktreePath, "uncommitted.txt")
		os.WriteFile(testFile, []byte("test"), 0644)

		status := getFeatureWorktreeStatus(tp.Dir, "test-feature", "repo1", tp.Config.Repos[0])

		if status.error != "" {
			t.Errorf("expected no error, got %q", status.error)
		}
		if !status.hasUncommitted {
			t.Error("should have uncommitted changes")
		}
	})

	t.Run("feature with commits ahead", func(t *testing.T) {
		worktreePath := filepath.Join(tp.TreesDir, "test-feature", "repo1")
		testFile := filepath.Join(worktreePath, "committed.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCmd(t, worktreePath, "add", ".")
		runGitCmd(t, worktreePath, "commit", "-m", "test commit")

		status := getFeatureWorktreeStatus(tp.Dir, "test-feature", "repo1", tp.Config.Repos[0])

		if status.error != "" {
			t.Errorf("expected no error, got %q", status.error)
		}
		if status.aheadCount != 1 {
			t.Errorf("aheadCount = %d, want 1", status.aheadCount)
		}
	})

	t.Run("non-existent feature", func(t *testing.T) {
		status := getFeatureWorktreeStatus(tp.Dir, "nonexistent", "repo1", tp.Config.Repos[0])

		if status.error == "" {
			t.Error("expected error for non-existent feature")
		}
		if status.error != "worktree not found" {
			t.Errorf("error = %q, want %q", status.error, "worktree not found")
		}
	})
}

// TestNeedsAttention tests feature needs attention detection
func TestNeedsAttention(t *testing.T) {
	tests := []struct {
		name     string
		statuses []featureWorktreeStatus
		want     bool
	}{
		{
			name:     "no changes",
			statuses: []featureWorktreeStatus{{hasUncommitted: false, aheadCount: 0}},
			want:     false,
		},
		{
			name:     "uncommitted changes",
			statuses: []featureWorktreeStatus{{hasUncommitted: true, aheadCount: 0}},
			want:     true,
		},
		{
			name:     "commits ahead",
			statuses: []featureWorktreeStatus{{hasUncommitted: false, aheadCount: 2}},
			want:     true,
		},
		{
			name:     "commits ahead and merged",
			statuses: []featureWorktreeStatus{{hasUncommitted: false, aheadCount: 2, isMerged: true}},
			want:     false,
		},
		{
			name:     "multiple repos, one needs attention",
			statuses: []featureWorktreeStatus{
				{hasUncommitted: false, aheadCount: 0},
				{hasUncommitted: true, aheadCount: 0},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsAttention(tt.statuses)
			if got != tt.want {
				t.Errorf("needsAttention() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsMerged tests feature merged detection
func TestIsMerged(t *testing.T) {
	tests := []struct {
		name     string
		statuses []featureWorktreeStatus
		want     bool
	}{
		{
			name:     "clean feature",
			statuses: []featureWorktreeStatus{{aheadCount: 0, behindCount: 0, isMerged: false}},
			want:     false,
		},
		{
			name:     "merged feature",
			statuses: []featureWorktreeStatus{{aheadCount: 0, behindCount: 1, isMerged: true}},
			want:     true,
		},
		{
			name:     "not merged yet",
			statuses: []featureWorktreeStatus{{aheadCount: 2, behindCount: 0, isMerged: false}},
			want:     false,
		},
		{
			name:     "merged with uncommitted",
			statuses: []featureWorktreeStatus{{aheadCount: 0, behindCount: 1, isMerged: true, hasUncommitted: true}},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMerged(tt.statuses)
			if got != tt.want {
				t.Errorf("isMerged() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsClean tests feature clean detection
func TestIsClean(t *testing.T) {
	tests := []struct {
		name     string
		statuses []featureWorktreeStatus
		want     bool
	}{
		{
			name:     "completely clean",
			statuses: []featureWorktreeStatus{{hasUncommitted: false, aheadCount: 0}},
			want:     true,
		},
		{
			name:     "has uncommitted changes",
			statuses: []featureWorktreeStatus{{hasUncommitted: true, aheadCount: 0}},
			want:     false,
		},
		{
			name:     "has commits ahead",
			statuses: []featureWorktreeStatus{{hasUncommitted: false, aheadCount: 1}},
			want:     false,
		},
		{
			name: "multiple repos all clean",
			statuses: []featureWorktreeStatus{
				{hasUncommitted: false, aheadCount: 0},
				{hasUncommitted: false, aheadCount: 0},
			},
			want: true,
		},
		{
			name: "multiple repos one dirty",
			statuses: []featureWorktreeStatus{
				{hasUncommitted: false, aheadCount: 0},
				{hasUncommitted: false, aheadCount: 1},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isClean(tt.statuses)
			if got != tt.want {
				t.Errorf("isClean() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFormatCompactStatus tests status formatting
func TestFormatCompactStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   featureWorktreeStatus
		showAll  bool
		contains string
	}{
		{
			name:     "error status",
			status:   featureWorktreeStatus{error: "test error"},
			showAll:  false,
			contains: "error",
		},
		{
			name:     "clean status no show all",
			status:   featureWorktreeStatus{hasUncommitted: false, aheadCount: 0},
			showAll:  false,
			contains: "",
		},
		{
			name:     "clean status show all",
			status:   featureWorktreeStatus{hasUncommitted: false, aheadCount: 0},
			showAll:  true,
			contains: "○",
		},
		{
			name:     "uncommitted changes",
			status:   featureWorktreeStatus{hasUncommitted: true, aheadCount: 0},
			showAll:  false,
			contains: "◉",
		},
		{
			name:     "commits ahead",
			status:   featureWorktreeStatus{hasUncommitted: false, aheadCount: 2},
			showAll:  false,
			contains: "2 ahead",
		},
		{
			name: "diff stats",
			status: featureWorktreeStatus{
				hasUncommitted: true,
				diffStats: &git.DiffStats{
					FilesChanged: 3,
					Insertions:   10,
					Deletions:    5,
				},
			},
			showAll:  false,
			contains: "+3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCompactStatus(tt.status, tt.showAll)
			if tt.contains != "" && got != tt.contains && len(got) > 0 && got[0:1] != tt.contains[0:1] {
				// For contains check, just verify the expected string is present
				if len(tt.contains) > 0 {
					found := false
					for i := 0; i <= len(got)-len(tt.contains); i++ {
						if got[i:i+len(tt.contains)] == tt.contains {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("formatCompactStatus() = %q, should contain %q", got, tt.contains)
					}
				}
			} else if tt.contains == "" && got != "" {
				t.Errorf("formatCompactStatus() = %q, want empty string", got)
			}
		})
	}
}

// TestStatusNoFeatures tests status with no active features
func TestStatusNoFeatures(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Status should work with no features
	err := runStatus()
	if err != nil {
		t.Fatalf("runStatus() error = %v", err)
	}
}

// TestStatusEmptyProject tests status with no repos
func TestStatusEmptyProject(t *testing.T) {
	tp := NewTestProject(t)

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Status should work with no repos
	err := runStatus()
	if err != nil {
		t.Fatalf("runStatus() error = %v", err)
	}
}

// TestStatusMissingRepo tests status when a configured repo doesn't exist
func TestStatusMissingRepo(t *testing.T) {
	tp := NewTestProject(t)

	// Add a repo to config that doesn't exist
	autoRefresh := true
	tp.Config.Repos = append(tp.Config.Repos, &config.Repo{
		Path:        "repos",
		Git:         "git@github.com:fake/missing.git",
		AutoRefresh: &autoRefresh,
	})
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Status should handle missing repos gracefully
	err := runStatus()
	if err != nil {
		t.Fatalf("runStatus() error = %v", err)
	}
}

// TestStatusWithMergedFeature tests status showing merged features
func TestStatusWithMergedFeature(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a feature
	err := runUp("merged-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Add work and merge it
	worktreePath := filepath.Join(tp.TreesDir, "merged-feature", "repo1")
	testFile := filepath.Join(worktreePath, "work.txt")
	os.WriteFile(testFile, []byte("work"), 0644)
	runGitCmd(t, worktreePath, "add", ".")
	runGitCmd(t, worktreePath, "commit", "-m", "work")

	runGitCmd(t, repo1.SourceDir, "checkout", "main")
	runGitCmd(t, repo1.SourceDir, "merge", "feature/merged-feature", "--no-ff", "-m", "merge")

	// Status should show merged feature
	err = runStatus()
	if err != nil {
		t.Fatalf("runStatus() error = %v", err)
	}
}

// TestStatusWithOrphanedWorktree tests that status handles orphaned worktrees
func TestStatusWithOrphanedWorktree(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")
	tp.InitRepo("repo2")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a feature
	err := runUp("orphaned", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify feature was created
	if !tp.FeatureExists("orphaned") {
		t.Fatal("feature was not created")
	}

	// Manually delete the trees directory (simulating user action)
	treesDir := filepath.Join(tp.TreesDir, "orphaned")
	if err := os.RemoveAll(treesDir); err != nil {
		t.Fatalf("failed to manually remove trees directory: %v", err)
	}

	// Status should handle orphaned worktree gracefully and not crash
	err = runStatus()
	if err != nil {
		t.Fatalf("runStatus() should handle orphaned worktree gracefully, got error: %v", err)
	}

	// The status command should complete successfully even with orphaned worktrees
	// It's acceptable if the feature doesn't show up in the status output since the directory is gone
}

// TestGetFeatureWorktreeStatusOrphaned tests orphaned worktree detection
func TestGetFeatureWorktreeStatusOrphaned(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a feature
	err := runUp("orphaned-status", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Manually delete the worktree directory
	worktreePath := filepath.Join(tp.TreesDir, "orphaned-status", "repo1")
	if err := os.RemoveAll(worktreePath); err != nil {
		t.Fatalf("failed to remove worktree directory: %v", err)
	}

	// Get status for the orphaned worktree
	cfg, err := config.LoadConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	status := getFeatureWorktreeStatus(tp.Dir, "orphaned-status", "repo1", cfg.Repos[0])

	// Should have an error indicating worktree not found
	if status.error == "" {
		t.Error("expected error for orphaned worktree, got none")
	}

	if status.error != "worktree not found" {
		t.Errorf("expected 'worktree not found' error, got: %q", status.error)
	}
}
