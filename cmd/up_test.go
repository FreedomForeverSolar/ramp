package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ramp/internal/config"
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

// TestUpRejectsSlashInFeatureName tests that slashes in feature names are rejected
func TestUpRejectsSlashInFeatureName(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Try to create feature with slash in name - should fail
	err := runUp("epic/sub-feature", "", "")
	if err == nil {
		t.Fatal("runUp() should return error for feature name with slash")
	}

	if !strings.Contains(err.Error(), "slash") {
		t.Errorf("error should mention 'slash', got: %v", err)
	}

	// Verify feature directory was NOT created
	if tp.FeatureExists("epic/sub-feature") {
		t.Error("feature directory should not have been created")
	}

	// Verify nested directory was NOT created
	if tp.FeatureExists("epic") {
		t.Error("nested feature directory should not have been created")
	}
}

// TestUpWithNestedBranchViaPrefix tests that prefix can create nested branch names
func TestUpWithNestedBranchViaPrefix(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create feature using prefix to achieve nested branch name
	// Feature name is just "sub-feature", prefix creates the nesting
	err := runUp("sub-feature", "epic/", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify feature directory uses simple name (no slash)
	if !tp.FeatureExists("sub-feature") {
		t.Error("feature directory should be 'sub-feature'")
	}

	// Verify nested feature directory was NOT created
	if tp.FeatureExists("epic/sub-feature") {
		t.Error("nested feature directory should not exist")
	}

	// Verify branch has nested path via prefix
	repo1 := tp.Repos["repo1"]
	if !repo1.BranchExists(t, "epic/sub-feature") {
		t.Error("branch 'epic/sub-feature' should exist via prefix")
	}
}

func TestUpWithSetupScript(t *testing.T) {
	tp := NewTestProject(t)
	_ = tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a setup script that writes to a marker file
	scriptPath := filepath.Join(tp.Dir, ".ramp", "scripts", "setup.sh")
	os.MkdirAll(filepath.Dir(scriptPath), 0755)
	scriptContent := `#!/bin/bash
echo "setup executed" > "$RAMP_TREES_DIR/setup-marker.txt"
echo "project_dir=$RAMP_PROJECT_DIR" >> "$RAMP_TREES_DIR/setup-marker.txt"
echo "worktree_name=$RAMP_WORKTREE_NAME" >> "$RAMP_TREES_DIR/setup-marker.txt"
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	// Update config to include setup script
	cfg, _ := config.LoadConfig(tp.Dir)
	cfg.Setup = "scripts/setup.sh"
	config.SaveConfig(cfg, tp.Dir)

	err := runUp("test-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify setup script was executed
	markerFile := filepath.Join(tp.Dir, "trees", "test-feature", "setup-marker.txt")
	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		t.Fatal("setup script was not executed - marker file not found")
	}

	// Verify environment variables were set correctly
	content, _ := os.ReadFile(markerFile)
	contentStr := string(content)

	if !strings.Contains(contentStr, "setup executed") {
		t.Error("setup script did not write expected content")
	}
	if !strings.Contains(contentStr, "worktree_name=test-feature") {
		t.Error("RAMP_WORKTREE_NAME environment variable not set correctly")
	}
	if !strings.Contains(contentStr, fmt.Sprintf("project_dir=%s", tp.Dir)) {
		t.Error("RAMP_PROJECT_DIR environment variable not set correctly")
	}
}

func TestUpSetupScriptFailure(t *testing.T) {
	tp := NewTestProject(t)
	_ = tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a setup script that fails
	scriptPath := filepath.Join(tp.Dir, ".ramp", "scripts", "setup.sh")
	os.MkdirAll(filepath.Dir(scriptPath), 0755)
	scriptContent := `#!/bin/bash
exit 1
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	// Update config to include setup script
	cfg, _ := config.LoadConfig(tp.Dir)
	cfg.Setup = "scripts/setup.sh"
	config.SaveConfig(cfg, tp.Dir)

	err := runUp("test-feature", "", "")
	if err == nil {
		t.Error("runUp() should fail when setup script fails")
	}

	// Verify error message mentions setup script
	if !strings.Contains(err.Error(), "setup script") {
		t.Errorf("error should mention setup script, got: %v", err)
	}
}

func TestUpSetupScriptWithPort(t *testing.T) {
	tp := NewTestProject(t)
	_ = tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a setup script that captures the port
	scriptPath := filepath.Join(tp.Dir, ".ramp", "scripts", "setup.sh")
	os.MkdirAll(filepath.Dir(scriptPath), 0755)
	scriptContent := `#!/bin/bash
echo "port=$RAMP_PORT" > "$RAMP_TREES_DIR/port-marker.txt"
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	// Update config to include setup script and port allocation
	cfg, _ := config.LoadConfig(tp.Dir)
	cfg.Setup = "scripts/setup.sh"
	cfg.BasePort = 3000
	cfg.MaxPorts = 100
	config.SaveConfig(cfg, tp.Dir)

	err := runUp("test-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify port environment variable was set
	markerFile := filepath.Join(tp.Dir, "trees", "test-feature", "port-marker.txt")
	content, _ := os.ReadFile(markerFile)
	contentStr := string(content)

	if !strings.Contains(contentStr, "port=3000") {
		t.Errorf("RAMP_PORT not set correctly, got: %s", contentStr)
	}
}

// TestUpWithEnvFilesSimple tests simple env file copy with auto-replace
func TestUpWithEnvFilesSimple(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("app")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create source .env file with RAMP variables
	envContent := `PORT=${RAMP_PORT}
API_PORT=${RAMP_PORT}1
APP_NAME=myapp-${RAMP_WORKTREE_NAME}
`
	os.WriteFile(filepath.Join(repo1.SourceDir, ".env"), []byte(envContent), 0644)

	// Update config to include env_files
	cfg, _ := config.LoadConfig(tp.Dir)
	cfg.BasePort = 4000
	cfg.Repos[0].EnvFiles = []config.EnvFile{
		{Source: ".env", Dest: ".env"},
	}
	config.SaveConfig(cfg, tp.Dir)

	// Create feature
	err := runUp("my-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify .env was copied and variables replaced
	destEnv := filepath.Join(tp.Dir, "trees", "my-feature", "app", ".env")
	content, err := os.ReadFile(destEnv)
	if err != nil {
		t.Fatalf("failed to read destination .env: %v", err)
	}

	expected := `PORT=4000
API_PORT=40001
APP_NAME=myapp-my-feature
`
	if string(content) != expected {
		t.Errorf("env file content mismatch\ngot:\n%s\nwant:\n%s", string(content), expected)
	}
}

// TestUpWithEnvFilesCrossRepo tests cross-repo env file copying
func TestUpWithEnvFilesCrossRepo(t *testing.T) {
	tp := NewTestProject(t)
	configsRepo := tp.InitRepo("configs")
	_ = tp.InitRepo("app")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create env file in configs repo
	os.MkdirAll(filepath.Join(configsRepo.SourceDir, "app"), 0755)
	envContent := `PORT=3000
API_URL=http://localhost:3000
`
	os.WriteFile(filepath.Join(configsRepo.SourceDir, "app", "prod.env"), []byte(envContent), 0644)

	// Commit to configs repo so worktree can be created
	runGitCmd(t, configsRepo.SourceDir, "add", ".")
	runGitCmd(t, configsRepo.SourceDir, "commit", "-m", "Add env file")

	// Update config to include cross-repo env_files
	cfg, _ := config.LoadConfig(tp.Dir)
	cfg.BasePort = 4000
	cfg.Repos[1].EnvFiles = []config.EnvFile{
		{
			Source: "../configs/app/prod.env",
			Dest:   ".env",
			Replace: map[string]string{
				"PORT":    "${RAMP_PORT}",
				"API_URL": "http://localhost:${RAMP_PORT}",
			},
		},
	}
	config.SaveConfig(cfg, tp.Dir)

	// Create feature
	err := runUp("my-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify .env was copied from configs to app with replacements
	destEnv := filepath.Join(tp.Dir, "trees", "my-feature", "app", ".env")
	content, err := os.ReadFile(destEnv)
	if err != nil {
		t.Fatalf("failed to read destination .env: %v", err)
	}

	expected := `PORT=4000
API_URL=http://localhost:4000
`
	if string(content) != expected {
		t.Errorf("env file content mismatch\ngot:\n%s\nwant:\n%s", string(content), expected)
	}
}

// TestUpWithEnvFilesCustomReplacements tests custom replacements only
func TestUpWithEnvFilesCustomReplacements(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("app")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create source .env with both custom keys and RAMP vars
	envContent := `PORT=3000
API_PORT=3001
UNUSED_VAR=${RAMP_PORT}
APP_NAME=default
`
	os.WriteFile(filepath.Join(repo1.SourceDir, ".env"), []byte(envContent), 0644)

	// Update config with explicit replacements
	cfg, _ := config.LoadConfig(tp.Dir)
	cfg.BasePort = 4000
	cfg.Repos[0].EnvFiles = []config.EnvFile{
		{
			Source: ".env",
			Dest:   ".env",
			Replace: map[string]string{
				"PORT":     "${RAMP_PORT}",
				"API_PORT": "${RAMP_PORT}1",
			},
		},
	}
	config.SaveConfig(cfg, tp.Dir)

	// Create feature
	err := runUp("my-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify only specified keys were replaced
	destEnv := filepath.Join(tp.Dir, "trees", "my-feature", "app", ".env")
	content, err := os.ReadFile(destEnv)
	if err != nil {
		t.Fatalf("failed to read destination .env: %v", err)
	}

	expected := `PORT=4000
API_PORT=40001
UNUSED_VAR=${RAMP_PORT}
APP_NAME=default
`
	if string(content) != expected {
		t.Errorf("env file content mismatch\ngot:\n%s\nwant:\n%s", string(content), expected)
	}
}

// TestUpWithEnvFilesMultiple tests multiple env files
func TestUpWithEnvFilesMultiple(t *testing.T) {
	tp := NewTestProject(t)
	repo1 := tp.InitRepo("app")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create multiple source files
	os.WriteFile(filepath.Join(repo1.SourceDir, ".env"), []byte("PORT=${RAMP_PORT}\n"), 0644)
	os.WriteFile(filepath.Join(repo1.SourceDir, ".env.local"), []byte("DEBUG=true\n"), 0644)

	// Update config with multiple env_files
	cfg, _ := config.LoadConfig(tp.Dir)
	cfg.BasePort = 4000
	cfg.Repos[0].EnvFiles = []config.EnvFile{
		{Source: ".env", Dest: ".env"},
		{Source: ".env.local", Dest: ".env.local"},
	}
	config.SaveConfig(cfg, tp.Dir)

	// Create feature
	err := runUp("my-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Verify both files were copied
	destEnv1 := filepath.Join(tp.Dir, "trees", "my-feature", "app", ".env")
	content1, err := os.ReadFile(destEnv1)
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}
	if string(content1) != "PORT=4000\n" {
		t.Errorf(".env content = %q, want %q", string(content1), "PORT=4000\n")
	}

	destEnv2 := filepath.Join(tp.Dir, "trees", "my-feature", "app", ".env.local")
	content2, err := os.ReadFile(destEnv2)
	if err != nil {
		t.Fatalf("failed to read .env.local: %v", err)
	}
	if string(content2) != "DEBUG=true\n" {
		t.Errorf(".env.local content = %q, want %q", string(content2), "DEBUG=true\n")
	}
}

// TestUpWithEnvFilesMissing tests graceful handling of missing source files
func TestUpWithEnvFilesMissing(t *testing.T) {
	tp := NewTestProject(t)
	_ = tp.InitRepo("app")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Update config with env_files but don't create source file
	cfg, _ := config.LoadConfig(tp.Dir)
	cfg.Repos[0].EnvFiles = []config.EnvFile{
		{Source: ".env", Dest: ".env"},
	}
	config.SaveConfig(cfg, tp.Dir)

	// Create feature - should succeed with warning
	err := runUp("my-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() should not error on missing env file, got: %v", err)
	}

	// Verify destination file was not created
	destEnv := filepath.Join(tp.Dir, "trees", "my-feature", "app", ".env")
	_, err = os.ReadFile(destEnv)
	if err == nil {
		t.Error("destination .env should not exist when source is missing")
	}
}
