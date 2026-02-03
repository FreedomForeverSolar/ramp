package operations

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"ramp/internal/config"
	"ramp/internal/scaffold"
)

// MockProgressReporter captures progress messages for testing
type MockProgressReporter struct {
	Messages []string
}

func (m *MockProgressReporter) Start(message string)                        { m.Messages = append(m.Messages, "start: "+message) }
func (m *MockProgressReporter) Update(message string)                       { m.Messages = append(m.Messages, "update: "+message) }
func (m *MockProgressReporter) UpdateWithProgress(message string, pct int)  { m.Messages = append(m.Messages, "progress: "+message) }
func (m *MockProgressReporter) Stop()                                       { m.Messages = append(m.Messages, "stop") }
func (m *MockProgressReporter) Success(message string)                      { m.Messages = append(m.Messages, "success: "+message) }
func (m *MockProgressReporter) Error(message string)                        { m.Messages = append(m.Messages, "error: "+message) }
func (m *MockProgressReporter) Warning(message string)                      { m.Messages = append(m.Messages, "warning: "+message) }
func (m *MockProgressReporter) Info(message string)                         { m.Messages = append(m.Messages, "info: "+message) }
func (m *MockProgressReporter) Complete(message string)                     { m.Messages = append(m.Messages, "complete: "+message) }

// TestProject represents a complete ramp project setup for testing
type TestProject struct {
	Dir       string
	ReposDir  string
	TreesDir  string
	RampDir   string
	Repos     map[string]*TestRepo
	Config    *config.Config
	t         *testing.T
}

// TestRepo represents a git repository in the test project
type TestRepo struct {
	Name      string
	SourceDir string
	RemoteDir string
}

// NewTestProject creates a fully functional ramp project for integration testing
func NewTestProject(t *testing.T) *TestProject {
	t.Helper()

	// Disable user config to prevent personal hooks/commands from interfering with tests
	t.Setenv("RAMP_USER_CONFIG_DIR", "")

	projectDir := t.TempDir()

	// Resolve symlinks to ensure canonical path (important on macOS where /var -> /private/var)
	canonicalDir, err := filepath.EvalSymlinks(projectDir)
	if err != nil {
		canonicalDir = projectDir
	}

	// Create project structure with NO repos initially
	data := scaffold.ProjectData{
		Name:           "test-project",
		BranchPrefix:   "feature/",
		IncludeSetup:   false,
		IncludeCleanup: false,
		EnablePorts:    true,
		BasePort:       3000,
		Repos:          []scaffold.RepoData{},
	}

	if err := scaffold.CreateProject(canonicalDir, data); err != nil {
		t.Fatalf("failed to create test project: %v", err)
	}

	cfg, err := config.LoadConfig(canonicalDir)
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}

	tp := &TestProject{
		Dir:      canonicalDir,
		ReposDir: filepath.Join(canonicalDir, "repos"),
		TreesDir: filepath.Join(canonicalDir, "trees"),
		RampDir:  filepath.Join(canonicalDir, ".ramp"),
		Repos:    make(map[string]*TestRepo),
		Config:   cfg,
		t:        t,
	}

	return tp
}

// InitRepo initializes a git repository with a remote and updates the config
func (tp *TestProject) InitRepo(name string) *TestRepo {
	tp.t.Helper()

	// Create bare remote repository
	remoteDir := filepath.Join(tp.t.TempDir(), name+"-remote")
	runGitCmd(tp.t, remoteDir, "init", "--bare")

	// Create source repository
	sourceDir := filepath.Join(tp.ReposDir, name)
	runGitCmd(tp.t, sourceDir, "init")
	runGitCmd(tp.t, sourceDir, "config", "user.email", "test@example.com")
	runGitCmd(tp.t, sourceDir, "config", "user.name", "Test User")
	runGitCmd(tp.t, sourceDir, "config", "commit.gpgsign", "false")

	// Create initial commit
	readmeFile := filepath.Join(sourceDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# "+name), 0644); err != nil {
		tp.t.Fatalf("failed to create README: %v", err)
	}
	runGitCmd(tp.t, sourceDir, "add", "README.md")
	runGitCmd(tp.t, sourceDir, "commit", "-m", "initial commit")

	// Rename to main
	runGitCmd(tp.t, sourceDir, "branch", "-M", "main")

	// Add remote and push
	runGitCmd(tp.t, sourceDir, "remote", "add", "origin", remoteDir)
	runGitCmd(tp.t, sourceDir, "push", "-u", "origin", "main")

	repo := &TestRepo{
		Name:      name,
		SourceDir: sourceDir,
		RemoteDir: remoteDir,
	}

	tp.Repos[name] = repo

	// Update config to include this repo
	autoRefresh := true
	tp.Config.Repos = append(tp.Config.Repos, &config.Repo{
		Path:        "repos",
		Git:         sourceDir,
		AutoRefresh: &autoRefresh,
	})

	// Save updated config
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		tp.t.Fatalf("failed to save updated config: %v", err)
	}

	return repo
}

// FeatureExists checks if a feature directory exists
func (tp *TestProject) FeatureExists(featureName string) bool {
	featureDir := filepath.Join(tp.TreesDir, featureName)
	_, err := os.Stat(featureDir)
	return err == nil
}

// WorktreeExists checks if a worktree exists for a specific repo in a feature
func (tp *TestProject) WorktreeExists(featureName, repoName string) bool {
	worktreeDir := filepath.Join(tp.TreesDir, featureName, repoName)
	_, err := os.Stat(worktreeDir)
	return err == nil
}

// Helper function to run git commands in tests
func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %v\nOutput: %s", args, dir, err, string(output))
	}
}

// === UP OPERATION TESTS ===

func TestUpBasic(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	progress := &MockProgressReporter{}

	result, err := Up(UpOptions{
		FeatureName: "test-feature",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		SkipRefresh: true,
	})

	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	if result.FeatureName != "test-feature" {
		t.Errorf("FeatureName = %q, want %q", result.FeatureName, "test-feature")
	}

	if result.BranchName != "feature/test-feature" {
		t.Errorf("BranchName = %q, want %q", result.BranchName, "feature/test-feature")
	}

	if !tp.FeatureExists("test-feature") {
		t.Error("Feature directory should exist")
	}

	if !tp.WorktreeExists("test-feature", "repo1") {
		t.Error("Worktree should exist for repo1")
	}
}

func TestUpMultipleRepos(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")
	tp.InitRepo("repo2")

	progress := &MockProgressReporter{}

	result, err := Up(UpOptions{
		FeatureName: "multi-repo",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		SkipRefresh: true,
	})

	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	if len(result.Repos) != 2 {
		t.Errorf("Repos count = %d, want 2", len(result.Repos))
	}

	if !tp.WorktreeExists("multi-repo", "repo1") {
		t.Error("Worktree should exist for repo1")
	}

	if !tp.WorktreeExists("multi-repo", "repo2") {
		t.Error("Worktree should exist for repo2")
	}
}

func TestUpWithNoPrefix(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	progress := &MockProgressReporter{}

	result, err := Up(UpOptions{
		FeatureName: "plain-branch",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		NoPrefix:    true,
		SkipRefresh: true,
	})

	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	if result.BranchName != "plain-branch" {
		t.Errorf("BranchName = %q, want %q (no prefix)", result.BranchName, "plain-branch")
	}
}

func TestUpWithCustomPrefix(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	progress := &MockProgressReporter{}

	result, err := Up(UpOptions{
		FeatureName: "my-feature",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		Prefix:      "bugfix/",
		SkipRefresh: true,
	})

	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	if result.BranchName != "bugfix/my-feature" {
		t.Errorf("BranchName = %q, want %q", result.BranchName, "bugfix/my-feature")
	}
}

func TestUpAllocatesPorts(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	progress := &MockProgressReporter{}

	result, err := Up(UpOptions{
		FeatureName: "with-port",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		SkipRefresh: true,
	})

	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	if len(result.AllocatedPorts) == 0 {
		t.Error("Should have allocated ports")
	}

	if result.AllocatedPorts[0] != 3000 {
		t.Errorf("First port = %d, want 3000", result.AllocatedPorts[0])
	}
}

func TestUpDuplicateFeature(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	progress := &MockProgressReporter{}

	// Create first feature
	_, err := Up(UpOptions{
		FeatureName: "duplicate",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		SkipRefresh: true,
	})
	if err != nil {
		t.Fatalf("First Up() error = %v", err)
	}

	// Try to create duplicate
	_, err = Up(UpOptions{
		FeatureName: "duplicate",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		SkipRefresh: true,
	})

	if err == nil {
		t.Error("Second Up() should fail for duplicate feature")
	}
}

func TestUpMissingSourceRepo(t *testing.T) {
	tp := NewTestProject(t)
	// Add repo to config without actually creating it
	autoRefresh := false
	tp.Config.Repos = append(tp.Config.Repos, &config.Repo{
		Path:        "repos",
		Git:         "/nonexistent/repo",
		AutoRefresh: &autoRefresh,
	})

	progress := &MockProgressReporter{}

	_, err := Up(UpOptions{
		FeatureName: "missing-repo",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		SkipRefresh: true,
	})

	if err == nil {
		t.Error("Up() should fail when source repo is missing")
	}
}

// === DOWN OPERATION TESTS ===

func TestDownBasic(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	progress := &MockProgressReporter{}

	// Create feature first
	_, err := Up(UpOptions{
		FeatureName: "to-delete",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		SkipRefresh: true,
	})
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	// Verify feature exists
	if !tp.FeatureExists("to-delete") {
		t.Fatal("Feature should exist before deletion")
	}

	// Delete feature
	result, err := Down(DownOptions{
		FeatureName: "to-delete",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		Force:       true,
	})

	if err != nil {
		t.Fatalf("Down() error = %v", err)
	}

	if result.FeatureName != "to-delete" {
		t.Errorf("FeatureName = %q, want %q", result.FeatureName, "to-delete")
	}

	if !tp.FeatureExists("to-delete") == false {
		// Feature directory should be gone
	}

	if tp.FeatureExists("to-delete") {
		t.Error("Feature directory should be removed")
	}
}

func TestDownNonExistent(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	progress := &MockProgressReporter{}

	_, err := Down(DownOptions{
		FeatureName: "nonexistent",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		Force:       true,
	})

	if err == nil {
		t.Error("Down() should fail for nonexistent feature")
	}
}

func TestDownReleasesPort(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	progress := &MockProgressReporter{}

	// Create feature with port
	upResult, err := Up(UpOptions{
		FeatureName: "with-port",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		SkipRefresh: true,
	})
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	if len(upResult.AllocatedPorts) == 0 {
		t.Fatal("Should have allocated port")
	}

	// Delete feature
	downResult, err := Down(DownOptions{
		FeatureName: "with-port",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		Force:       true,
	})

	if err != nil {
		t.Fatalf("Down() error = %v", err)
	}

	if !downResult.ReleasedPort {
		t.Error("Should have released port")
	}
}

func TestDownMultipleRepos(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")
	tp.InitRepo("repo2")

	progress := &MockProgressReporter{}

	// Create feature
	_, err := Up(UpOptions{
		FeatureName: "multi",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		SkipRefresh: true,
	})
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	// Delete feature
	result, err := Down(DownOptions{
		FeatureName: "multi",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		Force:       true,
	})

	if err != nil {
		t.Fatalf("Down() error = %v", err)
	}

	if len(result.RemovedWorktrees) != 2 {
		t.Errorf("RemovedWorktrees = %d, want 2", len(result.RemovedWorktrees))
	}

	if !tp.WorktreeExists("multi", "repo1") == false {
		// Worktrees should be gone
	}

	if tp.WorktreeExists("multi", "repo1") {
		t.Error("Worktree for repo1 should be removed")
	}

	if tp.WorktreeExists("multi", "repo2") {
		t.Error("Worktree for repo2 should be removed")
	}
}

func TestCheckForUncommittedChanges(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	progress := &MockProgressReporter{}

	// Create feature
	_, err := Up(UpOptions{
		FeatureName: "dirty-check",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		SkipRefresh: true,
	})
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	treesDir := filepath.Join(tp.Dir, "trees", "dirty-check")

	// Check with clean worktree
	reposWithChanges, err := CheckForUncommittedChanges(tp.Config, treesDir)
	if err != nil {
		t.Fatalf("CheckForUncommittedChanges() error = %v", err)
	}

	if len(reposWithChanges) != 0 {
		t.Errorf("Should have no uncommitted changes, got %v", reposWithChanges)
	}

	// Create uncommitted change
	worktreeDir := filepath.Join(treesDir, "repo1")
	testFile := filepath.Join(worktreeDir, "dirty.txt")
	if err := os.WriteFile(testFile, []byte("dirty"), 0644); err != nil {
		t.Fatalf("Failed to create dirty file: %v", err)
	}

	// Check with dirty worktree
	reposWithChanges, err = CheckForUncommittedChanges(tp.Config, treesDir)
	if err != nil {
		t.Fatalf("CheckForUncommittedChanges() error = %v", err)
	}

	if len(reposWithChanges) != 1 {
		t.Errorf("Should have 1 repo with uncommitted changes, got %d", len(reposWithChanges))
	}
}

// === INSTALL OPERATION TESTS ===

func TestIsProjectInstalled(t *testing.T) {
	tp := NewTestProject(t)

	// With no repos configured, project is considered installed (vacuously true)
	if !IsProjectInstalled(tp.Config, tp.Dir) {
		t.Error("Should be installed with no repos (nothing to install)")
	}

	// Add a repo to config but don't create it
	autoRefresh := false
	tp.Config.Repos = append(tp.Config.Repos, &config.Repo{
		Path:        "repos",
		Git:         "/nonexistent/repo",
		AutoRefresh: &autoRefresh,
	})

	// Now should not be installed (repo configured but missing)
	if IsProjectInstalled(tp.Config, tp.Dir) {
		t.Error("Should not be installed when configured repo is missing")
	}

	// Reset config and properly initialize a repo
	tp.Config.Repos = []*config.Repo{}
	tp.InitRepo("repo1")

	// After repo initialized, should be installed
	if !IsProjectInstalled(tp.Config, tp.Dir) {
		t.Error("Should be installed after repo added")
	}
}
