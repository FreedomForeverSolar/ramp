package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"ramp/internal/config"
	"ramp/internal/scaffold"
)

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

	projectDir := t.TempDir()

	// Create project structure with NO repos initially
	// Repos will be added via InitRepo()
	data := scaffold.ProjectData{
		Name:           "test-project",
		BranchPrefix:   "feature/",
		IncludeSetup:   false,
		IncludeCleanup: false,
		EnablePorts:    true,
		BasePort:       3000,
		Repos:          []scaffold.RepoData{}, // Empty - repos added via InitRepo
	}

	if err := scaffold.CreateProject(projectDir, data); err != nil {
		t.Fatalf("failed to create test project: %v", err)
	}

	cfg, err := config.LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}

	tp := &TestProject{
		Dir:      projectDir,
		ReposDir: filepath.Join(projectDir, "repos"),
		TreesDir: filepath.Join(projectDir, "trees"),
		RampDir:  filepath.Join(projectDir, ".ramp"),
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
		Git:         sourceDir, // Use local path instead of SSH URL
		AutoRefresh: &autoRefresh,
	})

	// Save updated config
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		tp.t.Fatalf("failed to save updated config: %v", err)
	}

	return repo
}

// CreateBranch creates a branch in the source repository
func (tr *TestRepo) CreateBranch(t *testing.T, branchName string) {
	t.Helper()
	runGitCmd(t, tr.SourceDir, "checkout", "-b", branchName)
	runGitCmd(t, tr.SourceDir, "commit", "--allow-empty", "-m", "branch commit")
	runGitCmd(t, tr.SourceDir, "push", "-u", "origin", branchName)
	runGitCmd(t, tr.SourceDir, "checkout", "main")
}

// AddCommit adds a commit to the current branch
func (tr *TestRepo) AddCommit(t *testing.T, message string) {
	t.Helper()
	testFile := filepath.Join(tr.SourceDir, "test-"+message+".txt")
	if err := os.WriteFile(testFile, []byte(message), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	runGitCmd(t, tr.SourceDir, "add", ".")
	runGitCmd(t, tr.SourceDir, "commit", "-m", message)
}

// ModifyFile modifies a file without committing
func (tr *TestRepo) ModifyFile(t *testing.T, filename, content string) {
	t.Helper()
	filePath := filepath.Join(tr.SourceDir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}
}

// GetCurrentBranch returns the current branch name
func (tr *TestRepo) GetCurrentBranch(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = tr.SourceDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	return string(output)
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

// GetWorktreeBranch returns the branch name of a worktree
func (tp *TestProject) GetWorktreeBranch(featureName, repoName string) (string, error) {
	worktreeDir := filepath.Join(tp.TreesDir, featureName, repoName)
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = worktreeDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// BranchExists checks if a branch exists in a repo
func (tr *TestRepo) BranchExists(t *testing.T, branchName string) bool {
	t.Helper()
	cmd := exec.Command("git", "branch", "--list", branchName)
	cmd.Dir = tr.SourceDir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}

// Helper function to run git commands in tests
func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()

	// Create directory if it doesn't exist
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

// ChangeToProjectDir changes the working directory to the project dir
// Returns a cleanup function to restore the original directory
func (tp *TestProject) ChangeToProjectDir() func() {
	tp.t.Helper()

	originalDir, err := os.Getwd()
	if err != nil {
		tp.t.Fatalf("failed to get current directory: %v", err)
	}

	if err := os.Chdir(tp.Dir); err != nil {
		tp.t.Fatalf("failed to change to project directory: %v", err)
	}

	return func() {
		if err := os.Chdir(originalDir); err != nil {
			tp.t.Fatalf("failed to restore directory: %v", err)
		}
	}
}
