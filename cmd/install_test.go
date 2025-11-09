package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"ramp/internal/config"
	"ramp/internal/git"
)

// TestInstallBasic tests basic repository cloning
func TestInstallBasic(t *testing.T) {
	tp := NewTestProject(t)

	// Create a remote repo but don't clone it yet
	remoteDir := filepath.Join(t.TempDir(), "repo1-remote")
	runGitCmd(t, remoteDir, "init", "--bare")

	// Add repo to config with remote URL
	tp.Config.Repos = append(tp.Config.Repos, &config.Repo{
		Path: "repos",
		Git:  remoteDir, // Use local remote path
	})
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Run install
	err := runInstall()
	if err != nil {
		t.Fatalf("runInstall() error = %v", err)
	}

	// Verify repo was cloned
	repoPath := filepath.Join(tp.ReposDir, "repo1-remote")
	if !git.IsGitRepo(repoPath) {
		t.Error("repository was not cloned")
	}
}

// TestInstallSkipsExisting tests that already cloned repos are skipped
func TestInstallSkipsExisting(t *testing.T) {
	tp := NewTestProject(t)

	// Initialize a repo (it's already cloned)
	repo1 := tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Run install again - should skip
	err := runInstall()
	if err != nil {
		t.Fatalf("runInstall() error = %v", err)
	}

	// Verify repo still exists and wasn't re-cloned
	if !git.IsGitRepo(repo1.SourceDir) {
		t.Error("existing repository should still exist")
	}
}

// TestInstallMultipleRepos tests cloning multiple repositories
func TestInstallMultipleRepos(t *testing.T) {
	tp := NewTestProject(t)

	// Create multiple remote repos
	repos := []string{"repo1", "repo2", "repo3"}
	for _, name := range repos {
		remoteDir := filepath.Join(t.TempDir(), name+"-remote")
		runGitCmd(t, remoteDir, "init", "--bare")

		autoRefresh := true
		tp.Config.Repos = append(tp.Config.Repos, &config.Repo{
			Path:        "repos",
			Git:         remoteDir,
			AutoRefresh: &autoRefresh,
		})
	}

	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Run install
	err := runInstall()
	if err != nil {
		t.Fatalf("runInstall() error = %v", err)
	}

	// Verify all repos were cloned
	for _, name := range repos {
		repoPath := filepath.Join(tp.ReposDir, name+"-remote")
		if !git.IsGitRepo(repoPath) {
			t.Errorf("repository %s was not cloned", name)
		}
	}
}

// TestInstallPartialClone tests when some repos exist and some don't
func TestInstallPartialClone(t *testing.T) {
	tp := NewTestProject(t)

	// Create repo1 (already cloned)
	repo1 := tp.InitRepo("repo1")

	// Create repo2 remote but don't clone
	remoteDir2 := filepath.Join(t.TempDir(), "repo2-remote")
	runGitCmd(t, remoteDir2, "init", "--bare")

	autoRefresh := true
	tp.Config.Repos = append(tp.Config.Repos, &config.Repo{
		Path:        "repos",
		Git:         remoteDir2,
		AutoRefresh: &autoRefresh,
	})

	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Run install
	err := runInstall()
	if err != nil {
		t.Fatalf("runInstall() error = %v", err)
	}

	// Verify repo1 still exists (not re-cloned)
	if !git.IsGitRepo(repo1.SourceDir) {
		t.Error("existing repo1 should still exist")
	}

	// Verify repo2 was cloned
	repo2Path := filepath.Join(tp.ReposDir, "repo2-remote")
	if !git.IsGitRepo(repo2Path) {
		t.Error("repo2 should have been cloned")
	}
}

// TestInstallCreatesDirectories tests directory creation
func TestInstallCreatesDirectories(t *testing.T) {
	tp := NewTestProject(t)

	// Remove repos directory
	os.RemoveAll(tp.ReposDir)

	// Create remote repo
	remoteDir := filepath.Join(t.TempDir(), "repo1-remote")
	runGitCmd(t, remoteDir, "init", "--bare")

	tp.Config.Repos = append(tp.Config.Repos, &config.Repo{
		Path: "repos",
		Git:  remoteDir,
	})
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Run install
	err := runInstall()
	if err != nil {
		t.Fatalf("runInstall() error = %v", err)
	}

	// Verify repos directory was created
	if _, err := os.Stat(tp.ReposDir); os.IsNotExist(err) {
		t.Error("repos directory was not created")
	}
}

// TestIsProjectInstalled tests the installation check
func TestIsProjectInstalled(t *testing.T) {
	tp := NewTestProject(t)

	t.Run("empty project is considered installed", func(t *testing.T) {
		// No repos configured - nothing to install, so it's "installed" by default
		installed := isProjectInstalled(tp.Config, tp.Dir)
		if !installed {
			t.Error("project with no repos should be considered installed (nothing to install)")
		}
	})

	t.Run("project with cloned repos is installed", func(t *testing.T) {
		// Initialize repos
		tp.InitRepo("repo1")
		tp.InitRepo("repo2")

		// Should be installed now
		installed := isProjectInstalled(tp.Config, tp.Dir)
		if !installed {
			t.Error("project with all repos cloned should be installed")
		}
	})

	t.Run("project with missing repo not installed", func(t *testing.T) {
		// Add a third repo to config that doesn't exist
		autoRefresh := true
		tp.Config.Repos = append(tp.Config.Repos, &config.Repo{
			Path:        "repos",
			Git:         "git@github.com:fake/repo3.git",
			AutoRefresh: &autoRefresh,
		})

		// Should not be installed (repo3 missing)
		installed := isProjectInstalled(tp.Config, tp.Dir)
		if installed {
			t.Error("project with missing repo should not be installed")
		}
	})
}

// TestAutoInstallIfNeeded tests automatic installation
func TestAutoInstallIfNeeded(t *testing.T) {
	tp := NewTestProject(t)

	t.Run("auto-install when needed", func(t *testing.T) {
		// Create remote repo
		remoteDir := filepath.Join(t.TempDir(), "repo1-remote")
		runGitCmd(t, remoteDir, "init", "--bare")

		tp.Config.Repos = append(tp.Config.Repos, &config.Repo{
			Path: "repos",
			Git:  remoteDir,
		})
		if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}

		// Should auto-install
		err := AutoInstallIfNeeded(tp.Dir, tp.Config)
		if err != nil {
			t.Fatalf("AutoInstallIfNeeded() error = %v", err)
		}

		// Verify repo was cloned
		repoPath := filepath.Join(tp.ReposDir, "repo1-remote")
		if !git.IsGitRepo(repoPath) {
			t.Error("repo should have been auto-installed")
		}
	})

	t.Run("skip when already installed", func(t *testing.T) {
		// Repo already exists from previous test

		// Should skip (no error)
		err := AutoInstallIfNeeded(tp.Dir, tp.Config)
		if err != nil {
			t.Fatalf("AutoInstallIfNeeded() error = %v", err)
		}

		// Repo should still exist
		repoPath := filepath.Join(tp.ReposDir, "repo1-remote")
		if !git.IsGitRepo(repoPath) {
			t.Error("repo should still exist")
		}
	})
}

// TestInstallWithNestedPath tests repos in nested directories
func TestInstallWithNestedPath(t *testing.T) {
	tp := NewTestProject(t)

	// Create remote repo
	remoteDir := filepath.Join(t.TempDir(), "myrepo-remote")
	runGitCmd(t, remoteDir, "init", "--bare")

	// Configure with nested path
	tp.Config.Repos = append(tp.Config.Repos, &config.Repo{
		Path: "external/sources", // Nested path
		Git:  remoteDir,
	})
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Run install
	err := runInstall()
	if err != nil {
		t.Fatalf("runInstall() error = %v", err)
	}

	// Verify repo was cloned in nested path
	repoPath := filepath.Join(tp.Dir, "external", "sources", "myrepo-remote")
	if !git.IsGitRepo(repoPath) {
		t.Error("repository was not cloned in nested path")
	}
}
