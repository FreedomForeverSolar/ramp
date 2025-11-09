package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestExtractRepoName tests extracting repository names from various git URL formats
func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		name     string
		repoPath string
		want     string
	}{
		{
			name:     "SSH format with .git",
			repoPath: "git@github.com:owner/repo.git",
			want:     "repo",
		},
		{
			name:     "SSH format without .git",
			repoPath: "git@github.com:owner/repo",
			want:     "repo",
		},
		{
			name:     "HTTPS format with .git",
			repoPath: "https://github.com/owner/repo.git",
			want:     "repo",
		},
		{
			name:     "HTTPS format without .git",
			repoPath: "https://github.com/owner/repo",
			want:     "repo",
		},
		{
			name:     "nested path",
			repoPath: "git@gitlab.com:org/team/project.git",
			want:     "project",
		},
		{
			name:     "simple name",
			repoPath: "myrepo",
			want:     "myrepo",
		},
		{
			name:     "local path",
			repoPath: "/path/to/repo.git",
			want:     "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRepoName(tt.repoPath)
			if got != tt.want {
				t.Errorf("extractRepoName(%q) = %q, want %q", tt.repoPath, got, tt.want)
			}
		})
	}
}

// TestGenerateEnvVarName tests environment variable name generation from repo names
func TestGenerateEnvVarName(t *testing.T) {
	tests := []struct {
		name     string
		repoName string
		want     string
	}{
		{
			name:     "simple name",
			repoName: "myrepo",
			want:     "RAMP_REPO_PATH_MYREPO",
		},
		{
			name:     "hyphenated name",
			repoName: "my-repo",
			want:     "RAMP_REPO_PATH_MY_REPO",
		},
		{
			name:     "dotted name",
			repoName: "my.repo.name",
			want:     "RAMP_REPO_PATH_MY_REPO_NAME",
		},
		{
			name:     "mixed separators",
			repoName: "my-repo.name",
			want:     "RAMP_REPO_PATH_MY_REPO_NAME",
		},
		{
			name:     "multiple consecutive hyphens",
			repoName: "my--repo",
			want:     "RAMP_REPO_PATH_MY_REPO",
		},
		{
			name:     "special characters",
			repoName: "my@repo#123",
			want:     "RAMP_REPO_PATH_MY_REPO_123",
		},
		{
			name:     "leading/trailing underscores",
			repoName: "_repo_",
			want:     "RAMP_REPO_PATH_REPO",
		},
		{
			name:     "numbers",
			repoName: "repo123",
			want:     "RAMP_REPO_PATH_REPO123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateEnvVarName(tt.repoName)
			if got != tt.want {
				t.Errorf("GenerateEnvVarName(%q) = %q, want %q", tt.repoName, got, tt.want)
			}
		})
	}
}

// TestConfigDefaults tests default value handling for config fields
func TestConfigDefaults(t *testing.T) {
	t.Run("GetBasePort default", func(t *testing.T) {
		cfg := &Config{}
		if got := cfg.GetBasePort(); got != 3000 {
			t.Errorf("GetBasePort() = %d, want 3000", got)
		}
	})

	t.Run("GetBasePort custom", func(t *testing.T) {
		cfg := &Config{BasePort: 8000}
		if got := cfg.GetBasePort(); got != 8000 {
			t.Errorf("GetBasePort() = %d, want 8000", got)
		}
	})

	t.Run("GetMaxPorts default", func(t *testing.T) {
		cfg := &Config{}
		if got := cfg.GetMaxPorts(); got != 100 {
			t.Errorf("GetMaxPorts() = %d, want 100", got)
		}
	})

	t.Run("GetMaxPorts custom", func(t *testing.T) {
		cfg := &Config{MaxPorts: 50}
		if got := cfg.GetMaxPorts(); got != 50 {
			t.Errorf("GetMaxPorts() = %d, want 50", got)
		}
	})

	t.Run("HasPortConfig false by default", func(t *testing.T) {
		cfg := &Config{}
		if got := cfg.HasPortConfig(); got != false {
			t.Errorf("HasPortConfig() = %v, want false", got)
		}
	})

	t.Run("HasPortConfig true with base_port", func(t *testing.T) {
		cfg := &Config{BasePort: 3000}
		if got := cfg.HasPortConfig(); got != true {
			t.Errorf("HasPortConfig() = %v, want true", got)
		}
	})

	t.Run("HasPortConfig true with max_ports", func(t *testing.T) {
		cfg := &Config{MaxPorts: 100}
		if got := cfg.HasPortConfig(); got != true {
			t.Errorf("HasPortConfig() = %v, want true", got)
		}
	})
}

// TestRepoAutoRefreshDefault tests the critical backwards compatibility behavior
// that auto_refresh defaults to true when not specified
func TestRepoAutoRefreshDefault(t *testing.T) {
	t.Run("defaults to true when nil", func(t *testing.T) {
		repo := &Repo{
			Path: "repos",
			Git:  "git@github.com:owner/repo.git",
		}
		if !repo.ShouldAutoRefresh() {
			t.Error("ShouldAutoRefresh() = false, want true (default)")
		}
	})

	t.Run("respects explicit true", func(t *testing.T) {
		trueVal := true
		repo := &Repo{
			Path:        "repos",
			Git:         "git@github.com:owner/repo.git",
			AutoRefresh: &trueVal,
		}
		if !repo.ShouldAutoRefresh() {
			t.Error("ShouldAutoRefresh() = false, want true")
		}
	})

	t.Run("respects explicit false", func(t *testing.T) {
		falseVal := false
		repo := &Repo{
			Path:        "repos",
			Git:         "git@github.com:owner/repo.git",
			AutoRefresh: &falseVal,
		}
		if repo.ShouldAutoRefresh() {
			t.Error("ShouldAutoRefresh() = true, want false")
		}
	})
}

// TestGetRepoPath tests absolute path construction
func TestGetRepoPath(t *testing.T) {
	tests := []struct {
		name       string
		repo       *Repo
		projectDir string
		want       string
	}{
		{
			name: "standard path",
			repo: &Repo{
				Path: "repos",
				Git:  "git@github.com:owner/myrepo.git",
			},
			projectDir: "/home/user/project",
			want:       "/home/user/project/repos/myrepo",
		},
		{
			name: "nested path",
			repo: &Repo{
				Path: "external/sources",
				Git:  "https://github.com/org/tool.git",
			},
			projectDir: "/projects/main",
			want:       "/projects/main/external/sources/tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.repo.GetRepoPath(tt.projectDir)
			if got != tt.want {
				t.Errorf("GetRepoPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestGetRepos tests the map generation from repos list
func TestGetRepos(t *testing.T) {
	cfg := &Config{
		Repos: []*Repo{
			{Path: "repos", Git: "git@github.com:owner/repo1.git"},
			{Path: "repos", Git: "git@github.com:owner/repo2.git"},
			{Path: "repos", Git: "https://github.com/owner/repo3.git"},
		},
	}

	repos := cfg.GetRepos()

	if len(repos) != 3 {
		t.Fatalf("GetRepos() returned %d repos, want 3", len(repos))
	}

	expectedNames := []string{"repo1", "repo2", "repo3"}
	for _, name := range expectedNames {
		if _, exists := repos[name]; !exists {
			t.Errorf("GetRepos() missing expected repo %q", name)
		}
	}

	// Verify the repos point to the correct config entries
	if repos["repo1"].Git != "git@github.com:owner/repo1.git" {
		t.Errorf("repo1 has wrong git URL")
	}
}

// TestGetCommand tests command lookup
func TestGetCommand(t *testing.T) {
	cfg := &Config{
		Commands: []*Command{
			{Name: "test", Command: "scripts/test.sh"},
			{Name: "deploy", Command: "scripts/deploy.sh"},
		},
	}

	t.Run("existing command", func(t *testing.T) {
		cmd := cfg.GetCommand("test")
		if cmd == nil {
			t.Fatal("GetCommand(\"test\") returned nil, want command")
		}
		if cmd.Command != "scripts/test.sh" {
			t.Errorf("GetCommand(\"test\").Command = %q, want %q", cmd.Command, "scripts/test.sh")
		}
	})

	t.Run("non-existing command", func(t *testing.T) {
		cmd := cfg.GetCommand("nonexistent")
		if cmd != nil {
			t.Errorf("GetCommand(\"nonexistent\") = %v, want nil", cmd)
		}
	})

	t.Run("empty commands list", func(t *testing.T) {
		emptyCfg := &Config{}
		cmd := emptyCfg.GetCommand("test")
		if cmd != nil {
			t.Errorf("GetCommand on empty config = %v, want nil", cmd)
		}
	})
}

// TestGetBranchPrefix tests branch prefix retrieval
func TestGetBranchPrefix(t *testing.T) {
	t.Run("custom prefix", func(t *testing.T) {
		cfg := &Config{DefaultBranchPrefix: "feature/"}
		if got := cfg.GetBranchPrefix(); got != "feature/" {
			t.Errorf("GetBranchPrefix() = %q, want %q", got, "feature/")
		}
	})

	t.Run("empty prefix", func(t *testing.T) {
		cfg := &Config{}
		if got := cfg.GetBranchPrefix(); got != "" {
			t.Errorf("GetBranchPrefix() = %q, want empty string", got)
		}
	})
}

// TestSaveAndLoadConfig tests the critical round-trip behavior
func TestSaveAndLoadConfig(t *testing.T) {
	tempDir := t.TempDir()

	trueVal := true
	falseVal := false

	original := &Config{
		Name: "test-project",
		Repos: []*Repo{
			{
				Path:        "repos",
				Git:         "git@github.com:owner/repo1.git",
				AutoRefresh: &trueVal,
			},
			{
				Path:        "repos",
				Git:         "https://github.com/owner/repo2.git",
				AutoRefresh: &falseVal,
			},
		},
		Setup:               "scripts/setup.sh",
		Cleanup:             "scripts/cleanup.sh",
		DefaultBranchPrefix: "feature/",
		BasePort:            3000,
		MaxPorts:            50,
		Commands: []*Command{
			{Name: "test", Command: "scripts/test.sh"},
			{Name: "deploy", Command: "scripts/deploy.sh"},
		},
	}

	// Save
	if err := SaveConfig(original, tempDir); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tempDir, ".ramp", "ramp.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("config file was not created at %s", configPath)
	}

	// Load
	loaded, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Compare
	if loaded.Name != original.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, original.Name)
	}

	if len(loaded.Repos) != len(original.Repos) {
		t.Fatalf("Repos length = %d, want %d", len(loaded.Repos), len(original.Repos))
	}

	for i, repo := range loaded.Repos {
		origRepo := original.Repos[i]
		if repo.Path != origRepo.Path {
			t.Errorf("Repo[%d].Path = %q, want %q", i, repo.Path, origRepo.Path)
		}
		if repo.Git != origRepo.Git {
			t.Errorf("Repo[%d].Git = %q, want %q", i, repo.Git, origRepo.Git)
		}
		if (repo.AutoRefresh == nil) != (origRepo.AutoRefresh == nil) {
			t.Errorf("Repo[%d].AutoRefresh nil mismatch", i)
		} else if repo.AutoRefresh != nil && *repo.AutoRefresh != *origRepo.AutoRefresh {
			t.Errorf("Repo[%d].AutoRefresh = %v, want %v", i, *repo.AutoRefresh, *origRepo.AutoRefresh)
		}
	}

	if loaded.Setup != original.Setup {
		t.Errorf("Setup = %q, want %q", loaded.Setup, original.Setup)
	}

	if loaded.Cleanup != original.Cleanup {
		t.Errorf("Cleanup = %q, want %q", loaded.Cleanup, original.Cleanup)
	}

	if loaded.DefaultBranchPrefix != original.DefaultBranchPrefix {
		t.Errorf("DefaultBranchPrefix = %q, want %q", loaded.DefaultBranchPrefix, original.DefaultBranchPrefix)
	}

	if loaded.BasePort != original.BasePort {
		t.Errorf("BasePort = %d, want %d", loaded.BasePort, original.BasePort)
	}

	if loaded.MaxPorts != original.MaxPorts {
		t.Errorf("MaxPorts = %d, want %d", loaded.MaxPorts, original.MaxPorts)
	}

	if len(loaded.Commands) != len(original.Commands) {
		t.Fatalf("Commands length = %d, want %d", len(loaded.Commands), len(original.Commands))
	}

	for i, cmd := range loaded.Commands {
		origCmd := original.Commands[i]
		if cmd.Name != origCmd.Name {
			t.Errorf("Command[%d].Name = %q, want %q", i, cmd.Name, origCmd.Name)
		}
		if cmd.Command != origCmd.Command {
			t.Errorf("Command[%d].Command = %q, want %q", i, cmd.Command, origCmd.Command)
		}
	}
}

// TestLoadConfigAutoRefreshBackwardsCompatibility ensures that configs without
// auto_refresh fields default to true (critical backwards compatibility)
func TestLoadConfigAutoRefreshBackwardsCompatibility(t *testing.T) {
	tempDir := t.TempDir()

	// Create a config file WITHOUT auto_refresh fields (simulating old config)
	configContent := `name: legacy-project
repos:
  - path: repos
    git: git@github.com:owner/repo.git
`

	rampDir := filepath.Join(tempDir, ".ramp")
	if err := os.MkdirAll(rampDir, 0755); err != nil {
		t.Fatalf("failed to create .ramp dir: %v", err)
	}

	configPath := filepath.Join(rampDir, "ramp.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load the config
	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// CRITICAL: auto_refresh should default to true
	if len(cfg.Repos) == 0 {
		t.Fatal("no repos loaded")
	}

	repo := cfg.Repos[0]
	if !repo.ShouldAutoRefresh() {
		t.Error("repo.ShouldAutoRefresh() = false, want true for backwards compatibility")
	}

	// The AutoRefresh field should be nil (not explicitly set)
	if repo.AutoRefresh != nil {
		t.Errorf("repo.AutoRefresh = %v, want nil (not explicitly set)", *repo.AutoRefresh)
	}
}

// TestLoadConfigErrors tests error handling
func TestLoadConfigErrors(t *testing.T) {
	t.Run("non-existent directory", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/path")
		if err == nil {
			t.Error("LoadConfig() with non-existent path should return error")
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		tempDir := t.TempDir()
		rampDir := filepath.Join(tempDir, ".ramp")
		if err := os.MkdirAll(rampDir, 0755); err != nil {
			t.Fatalf("failed to create .ramp dir: %v", err)
		}

		configPath := filepath.Join(rampDir, "ramp.yaml")
		invalidYAML := "name: test\nrepos:\n  - invalid yaml content here: [[[{"
		if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
			t.Fatalf("failed to write invalid config: %v", err)
		}

		_, err := LoadConfig(tempDir)
		if err == nil {
			t.Error("LoadConfig() with invalid YAML should return error")
		}
	})
}

// TestFindRampProject tests directory tree walking
func TestFindRampProject(t *testing.T) {
	tempDir := t.TempDir()

	// Resolve symlinks to ensure canonical path (important on macOS where /var -> /private/var)
	canonicalTempDir, err := filepath.EvalSymlinks(tempDir)
	if err != nil {
		// If we can't resolve symlinks, use the original path
		canonicalTempDir = tempDir
	}

	// Create project structure:
	// tempDir/
	//   .ramp/
	//     ramp.yaml
	//   subdir1/
	//     subdir2/
	projectRoot := canonicalTempDir
	rampDir := filepath.Join(projectRoot, ".ramp")
	if err := os.MkdirAll(rampDir, 0755); err != nil {
		t.Fatalf("failed to create .ramp: %v", err)
	}

	configPath := filepath.Join(rampDir, "ramp.yaml")
	if err := os.WriteFile(configPath, []byte("name: test\n"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	subdir1 := filepath.Join(projectRoot, "subdir1")
	subdir2 := filepath.Join(subdir1, "subdir2")
	if err := os.MkdirAll(subdir2, 0755); err != nil {
		t.Fatalf("failed to create subdirs: %v", err)
	}

	t.Run("find from project root", func(t *testing.T) {
		found, err := FindRampProject(projectRoot)
		if err != nil {
			t.Fatalf("FindRampProject() error = %v", err)
		}
		if found != projectRoot {
			t.Errorf("FindRampProject() = %q, want %q", found, projectRoot)
		}
	})

	t.Run("find from subdir1", func(t *testing.T) {
		found, err := FindRampProject(subdir1)
		if err != nil {
			t.Fatalf("FindRampProject() error = %v", err)
		}
		if found != projectRoot {
			t.Errorf("FindRampProject() = %q, want %q", found, projectRoot)
		}
	})

	t.Run("find from subdir2 (nested)", func(t *testing.T) {
		found, err := FindRampProject(subdir2)
		if err != nil {
			t.Fatalf("FindRampProject() error = %v", err)
		}
		if found != projectRoot {
			t.Errorf("FindRampProject() = %q, want %q", found, projectRoot)
		}
	})

	t.Run("not found in unrelated directory", func(t *testing.T) {
		unrelatedDir := t.TempDir()
		_, err := FindRampProject(unrelatedDir)
		if err == nil {
			t.Error("FindRampProject() in unrelated dir should return error")
		}
	})
}

// TestSaveConfigFormatting tests that SaveConfig produces readable YAML
func TestSaveConfigFormatting(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &Config{
		Name: "my-project",
		Repos: []*Repo{
			{Path: "repos", Git: "git@github.com:owner/repo.git"},
		},
		Setup:               "scripts/setup.sh",
		DefaultBranchPrefix: "feature/",
	}

	if err := SaveConfig(cfg, tempDir); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Read the file and check formatting
	configPath := filepath.Join(tempDir, ".ramp", "ramp.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}

	contentStr := string(content)

	// Check that it contains expected sections
	expectedSections := []string{
		"name: my-project",
		"repos:",
		"  - path: repos",
		"    git: git@github.com:owner/repo.git",
		"default-branch-prefix: feature/",
		"setup: scripts/setup.sh",
	}

	for _, expected := range expectedSections {
		if !contains(contentStr, expected) {
			t.Errorf("saved config missing expected section: %q\nGot:\n%s", expected, contentStr)
		}
	}
}

// TestSaveConfigOmitsEmptyFields tests that optional fields are omitted when empty
func TestSaveConfigOmitsEmptyFields(t *testing.T) {
	tempDir := t.TempDir()

	// Minimal config
	cfg := &Config{
		Name: "minimal",
		Repos: []*Repo{
			{Path: "repos", Git: "git@github.com:owner/repo.git"},
		},
	}

	if err := SaveConfig(cfg, tempDir); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	configPath := filepath.Join(tempDir, ".ramp", "ramp.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}

	contentStr := string(content)

	// These should NOT appear in minimal config
	unexpectedSections := []string{
		"setup:",
		"cleanup:",
		"base_port:",
		"commands:",
	}

	for _, unexpected := range unexpectedSections {
		if contains(contentStr, unexpected) {
			t.Errorf("saved config should not contain %q for minimal config\nGot:\n%s", unexpected, contentStr)
		}
	}
}

// TestMinimalConfig ensures minimal valid configs work
func TestMinimalConfig(t *testing.T) {
	tempDir := t.TempDir()

	minimalYAML := `name: minimal
repos:
  - path: repos
    git: git@github.com:owner/repo.git
`

	rampDir := filepath.Join(tempDir, ".ramp")
	if err := os.MkdirAll(rampDir, 0755); err != nil {
		t.Fatalf("failed to create .ramp: %v", err)
	}

	configPath := filepath.Join(rampDir, "ramp.yaml")
	if err := os.WriteFile(configPath, []byte(minimalYAML), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Name != "minimal" {
		t.Errorf("Name = %q, want %q", cfg.Name, "minimal")
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("Repos length = %d, want 1", len(cfg.Repos))
	}

	// Verify defaults
	if cfg.GetBasePort() != 3000 {
		t.Errorf("GetBasePort() = %d, want 3000 (default)", cfg.GetBasePort())
	}

	if cfg.GetMaxPorts() != 100 {
		t.Errorf("GetMaxPorts() = %d, want 100 (default)", cfg.GetMaxPorts())
	}

	if cfg.Repos[0].ShouldAutoRefresh() != true {
		t.Error("ShouldAutoRefresh() = false, want true (default)")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestGetGitURL tests the simple getter
func TestGetGitURL(t *testing.T) {
	repo := &Repo{
		Path: "repos",
		Git:  "git@github.com:owner/repo.git",
	}

	if got := repo.GetGitURL(); got != "git@github.com:owner/repo.git" {
		t.Errorf("GetGitURL() = %q, want %q", got, "git@github.com:owner/repo.git")
	}
}

// Benchmark tests for performance-critical functions
func BenchmarkExtractRepoName(b *testing.B) {
	url := "git@github.com:owner/repository-name.git"
	for i := 0; i < b.N; i++ {
		extractRepoName(url)
	}
}

func BenchmarkGenerateEnvVarName(b *testing.B) {
	repoName := "my-complex-repo-name.with.dots"
	for i := 0; i < b.N; i++ {
		GenerateEnvVarName(repoName)
	}
}

func BenchmarkFindRampProject(b *testing.B) {
	// Create a test directory structure
	tempDir := b.TempDir()
	rampDir := filepath.Join(tempDir, ".ramp")
	os.MkdirAll(rampDir, 0755)
	configPath := filepath.Join(rampDir, "ramp.yaml")
	os.WriteFile(configPath, []byte("name: test\n"), 0644)

	deepDir := filepath.Join(tempDir, "a", "b", "c", "d", "e")
	os.MkdirAll(deepDir, 0755)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FindRampProject(deepDir)
	}
}
