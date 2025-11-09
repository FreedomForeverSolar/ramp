package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ramp/internal/config"
)

// TestCreateDirectoryStructure tests directory creation
func TestCreateDirectoryStructure(t *testing.T) {
	tempDir := t.TempDir()

	err := CreateDirectoryStructure(tempDir)
	if err != nil {
		t.Fatalf("CreateDirectoryStructure() error = %v", err)
	}

	// Check that all required directories exist
	requiredDirs := []string{
		filepath.Join(tempDir, ".ramp", "scripts"),
		filepath.Join(tempDir, "repos"),
		filepath.Join(tempDir, "trees"),
	}

	for _, dir := range requiredDirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("directory %s does not exist: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}
}

// TestExtractRepoName tests repository name extraction
func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		name   string
		gitURL string
		want   string
	}{
		{
			name:   "SSH format with .git",
			gitURL: "git@github.com:owner/repo.git",
			want:   "repo",
		},
		{
			name:   "HTTPS format with .git",
			gitURL: "https://github.com/owner/repo.git",
			want:   "repo",
		},
		{
			name:   "SSH format without .git",
			gitURL: "git@github.com:owner/repo",
			want:   "repo",
		},
		{
			name:   "nested path",
			gitURL: "git@gitlab.com:org/team/project.git",
			want:   "project",
		},
		{
			name:   "simple name",
			gitURL: "myrepo",
			want:   "myrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRepoName(tt.gitURL)
			if got != tt.want {
				t.Errorf("extractRepoName(%q) = %q, want %q", tt.gitURL, got, tt.want)
			}
		})
	}
}

// TestGenerateConfigFile tests config file generation
func TestGenerateConfigFile(t *testing.T) {
	t.Run("minimal config", func(t *testing.T) {
		tempDir := t.TempDir()
		CreateDirectoryStructure(tempDir)

		data := ProjectData{
			Name:         "test-project",
			BranchPrefix: "feature/",
			Repos: []RepoData{
				{GitURL: "git@github.com:owner/repo.git", Path: "repos"},
			},
		}

		err := GenerateConfigFile(tempDir, data)
		if err != nil {
			t.Fatalf("GenerateConfigFile() error = %v", err)
		}

		// Verify config file exists
		configPath := filepath.Join(tempDir, ".ramp", "ramp.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatalf("config file was not created")
		}

		// Load and verify config
		cfg, err := config.LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("failed to load generated config: %v", err)
		}

		if cfg.Name != "test-project" {
			t.Errorf("config.Name = %q, want %q", cfg.Name, "test-project")
		}

		if cfg.DefaultBranchPrefix != "feature/" {
			t.Errorf("config.DefaultBranchPrefix = %q, want %q", cfg.DefaultBranchPrefix, "feature/")
		}

		if len(cfg.Repos) != 1 {
			t.Fatalf("config has %d repos, want 1", len(cfg.Repos))
		}

		// CRITICAL: auto_refresh should default to true
		if !cfg.Repos[0].ShouldAutoRefresh() {
			t.Error("config.Repos[0].ShouldAutoRefresh() = false, want true (backward compatibility)")
		}
	})

	t.Run("full config with all features", func(t *testing.T) {
		tempDir := t.TempDir()
		CreateDirectoryStructure(tempDir)

		data := ProjectData{
			Name:            "full-project",
			BranchPrefix:    "feat/",
			IncludeSetup:    true,
			IncludeCleanup:  true,
			EnablePorts:     true,
			BasePort:        3000,
			SampleCommands:  []string{"doctor", "deploy"},
			Repos: []RepoData{
				{GitURL: "git@github.com:owner/repo1.git", Path: "repos"},
				{GitURL: "https://github.com/owner/repo2.git", Path: "repos"},
			},
		}

		err := GenerateConfigFile(tempDir, data)
		if err != nil {
			t.Fatalf("GenerateConfigFile() error = %v", err)
		}

		cfg, err := config.LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("failed to load generated config: %v", err)
		}

		if cfg.Setup != "scripts/setup.sh" {
			t.Errorf("config.Setup = %q, want %q", cfg.Setup, "scripts/setup.sh")
		}

		if cfg.Cleanup != "scripts/cleanup.sh" {
			t.Errorf("config.Cleanup = %q, want %q", cfg.Cleanup, "scripts/cleanup.sh")
		}

		if cfg.BasePort != 3000 {
			t.Errorf("config.BasePort = %d, want 3000", cfg.BasePort)
		}

		if cfg.MaxPorts != 100 {
			t.Errorf("config.MaxPorts = %d, want 100 (default)", cfg.MaxPorts)
		}

		if len(cfg.Commands) != 2 {
			t.Fatalf("config has %d commands, want 2", len(cfg.Commands))
		}

		// Verify command names
		commandNames := make(map[string]bool)
		for _, cmd := range cfg.Commands {
			commandNames[cmd.Name] = true
		}

		if !commandNames["doctor"] || !commandNames["deploy"] {
			t.Error("config.Commands missing expected commands")
		}
	})
}

// TestGenerateSetupScript tests setup script generation
func TestGenerateSetupScript(t *testing.T) {
	tempDir := t.TempDir()
	CreateDirectoryStructure(tempDir)

	repos := []RepoData{
		{GitURL: "git@github.com:owner/backend.git", Path: "repos"},
		{GitURL: "git@github.com:owner/frontend.git", Path: "repos"},
	}

	err := GenerateSetupScript(tempDir, repos)
	if err != nil {
		t.Fatalf("GenerateSetupScript() error = %v", err)
	}

	scriptPath := filepath.Join(tempDir, ".ramp", "scripts", "setup.sh")

	// Verify file exists
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("setup script was not created: %v", err)
	}

	// Verify executable permissions
	if info.Mode().Perm()&0111 == 0 {
		t.Error("setup script is not executable")
	}

	// Read and verify content
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read setup script: %v", err)
	}

	contentStr := string(content)

	// Verify shebang
	if !strings.HasPrefix(contentStr, "#!/bin/bash") {
		t.Error("setup script missing shebang")
	}

	// Verify standard environment variables are documented
	expectedVars := []string{
		"RAMP_PROJECT_DIR",
		"RAMP_TREES_DIR",
		"RAMP_WORKTREE_NAME",
		"RAMP_PORT",
	}

	for _, envVar := range expectedVars {
		if !strings.Contains(contentStr, envVar) {
			t.Errorf("setup script missing documentation for %s", envVar)
		}
	}

	// Verify repository-specific environment variables
	repoEnvVars := []string{
		config.GenerateEnvVarName("backend"),
		config.GenerateEnvVarName("frontend"),
	}

	for _, envVar := range repoEnvVars {
		if !strings.Contains(contentStr, envVar) {
			t.Errorf("setup script missing documentation for %s", envVar)
		}
	}
}

// TestGenerateCleanupScript tests cleanup script generation
func TestGenerateCleanupScript(t *testing.T) {
	tempDir := t.TempDir()
	CreateDirectoryStructure(tempDir)

	repos := []RepoData{
		{GitURL: "git@github.com:owner/service.git", Path: "repos"},
	}

	err := GenerateCleanupScript(tempDir, repos)
	if err != nil {
		t.Fatalf("GenerateCleanupScript() error = %v", err)
	}

	scriptPath := filepath.Join(tempDir, ".ramp", "scripts", "cleanup.sh")

	// Verify file exists and is executable
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("cleanup script was not created: %v", err)
	}

	if info.Mode().Perm()&0111 == 0 {
		t.Error("cleanup script is not executable")
	}

	// Read and verify content
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read cleanup script: %v", err)
	}

	contentStr := string(content)

	if !strings.HasPrefix(contentStr, "#!/bin/bash") {
		t.Error("cleanup script missing shebang")
	}

	if !strings.Contains(contentStr, "RAMP_WORKTREE_NAME") {
		t.Error("cleanup script missing RAMP_WORKTREE_NAME")
	}

	serviceEnvVar := config.GenerateEnvVarName("service")
	if !strings.Contains(contentStr, serviceEnvVar) {
		t.Errorf("cleanup script missing %s", serviceEnvVar)
	}
}

// TestGenerateSampleCommand tests custom command script generation
func TestGenerateSampleCommand(t *testing.T) {
	tempDir := t.TempDir()
	CreateDirectoryStructure(tempDir)

	repos := []RepoData{
		{GitURL: "git@github.com:owner/api.git", Path: "repos"},
	}

	err := GenerateSampleCommand(tempDir, "doctor", repos)
	if err != nil {
		t.Fatalf("GenerateSampleCommand() error = %v", err)
	}

	scriptPath := filepath.Join(tempDir, ".ramp", "scripts", "doctor.sh")

	// Verify file exists and is executable
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("command script was not created: %v", err)
	}

	if info.Mode().Perm()&0111 == 0 {
		t.Error("command script is not executable")
	}

	// Read and verify content
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read command script: %v", err)
	}

	contentStr := string(content)

	if !strings.HasPrefix(contentStr, "#!/bin/bash") {
		t.Error("command script missing shebang")
	}

	// Verify command name appears in content
	if !strings.Contains(contentStr, "doctor") {
		t.Error("command script missing command name in content")
	}

	// Verify usage documentation
	if !strings.Contains(contentStr, "ramp run doctor") {
		t.Error("command script missing usage documentation")
	}

	apiEnvVar := config.GenerateEnvVarName("api")
	if !strings.Contains(contentStr, apiEnvVar) {
		t.Errorf("command script missing %s", apiEnvVar)
	}
}

// TestCreateProject tests the full project creation workflow
func TestCreateProject(t *testing.T) {
	tempDir := t.TempDir()

	data := ProjectData{
		Name:            "integration-test",
		BranchPrefix:    "feature/",
		IncludeSetup:    true,
		IncludeCleanup:  true,
		EnablePorts:     true,
		BasePort:        4000,
		SampleCommands:  []string{"test", "build"},
		Repos: []RepoData{
			{GitURL: "git@github.com:owner/app.git", Path: "repos"},
			{GitURL: "git@github.com:owner/lib.git", Path: "repos"},
		},
	}

	err := CreateProject(tempDir, data)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	// Verify directory structure
	requiredDirs := []string{
		filepath.Join(tempDir, ".ramp", "scripts"),
		filepath.Join(tempDir, "repos"),
		filepath.Join(tempDir, "trees"),
	}

	for _, dir := range requiredDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("required directory %s was not created", dir)
		}
	}

	// Verify config file
	cfg, err := config.LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Name != "integration-test" {
		t.Errorf("config.Name = %q, want %q", cfg.Name, "integration-test")
	}

	if len(cfg.Repos) != 2 {
		t.Errorf("config has %d repos, want 2", len(cfg.Repos))
	}

	// Verify scripts were created
	scriptsToCheck := []string{
		"setup.sh",
		"cleanup.sh",
		"test.sh",
		"build.sh",
	}

	for _, script := range scriptsToCheck {
		scriptPath := filepath.Join(tempDir, ".ramp", "scripts", script)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			t.Errorf("script %s was not created", script)
		}
	}
}

// TestCreateProjectWithoutOptionalFeatures tests minimal project creation
func TestCreateProjectWithoutOptionalFeatures(t *testing.T) {
	tempDir := t.TempDir()

	data := ProjectData{
		Name:         "minimal-project",
		BranchPrefix: "",
		Repos: []RepoData{
			{GitURL: "git@github.com:owner/repo.git", Path: "repos"},
		},
	}

	err := CreateProject(tempDir, data)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	// Verify setup and cleanup scripts were NOT created
	setupPath := filepath.Join(tempDir, ".ramp", "scripts", "setup.sh")
	if _, err := os.Stat(setupPath); !os.IsNotExist(err) {
		t.Error("setup.sh should not exist for minimal project")
	}

	cleanupPath := filepath.Join(tempDir, ".ramp", "scripts", "cleanup.sh")
	if _, err := os.Stat(cleanupPath); !os.IsNotExist(err) {
		t.Error("cleanup.sh should not exist for minimal project")
	}

	// But config should exist
	cfg, err := config.LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Setup != "" {
		t.Errorf("config.Setup = %q, want empty", cfg.Setup)
	}

	if cfg.Cleanup != "" {
		t.Errorf("config.Cleanup = %q, want empty", cfg.Cleanup)
	}
}

// TestScriptPermissions verifies all generated scripts have executable permissions
func TestScriptPermissions(t *testing.T) {
	tempDir := t.TempDir()
	CreateDirectoryStructure(tempDir)

	repos := []RepoData{
		{GitURL: "git@github.com:owner/repo.git", Path: "repos"},
	}

	tests := []struct {
		name     string
		generate func() error
		path     string
	}{
		{
			name:     "setup script",
			generate: func() error { return GenerateSetupScript(tempDir, repos) },
			path:     filepath.Join(tempDir, ".ramp", "scripts", "setup.sh"),
		},
		{
			name:     "cleanup script",
			generate: func() error { return GenerateCleanupScript(tempDir, repos) },
			path:     filepath.Join(tempDir, ".ramp", "scripts", "cleanup.sh"),
		},
		{
			name:     "custom command",
			generate: func() error { return GenerateSampleCommand(tempDir, "custom", repos) },
			path:     filepath.Join(tempDir, ".ramp", "scripts", "custom.sh"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.generate(); err != nil {
				t.Fatalf("failed to generate script: %v", err)
			}

			info, err := os.Stat(tt.path)
			if err != nil {
				t.Fatalf("script not created: %v", err)
			}

			// Check executable bit for user
			if info.Mode().Perm()&0100 == 0 {
				t.Errorf("script is not executable (permissions: %v)", info.Mode().Perm())
			}
		})
	}
}

// TestTemplateGeneration verifies templates contain expected content
func TestTemplateGeneration(t *testing.T) {
	repos := []RepoData{
		{GitURL: "git@github.com:owner/my-app.git", Path: "repos"},
		{GitURL: "git@github.com:owner/my-lib.git", Path: "repos"},
	}

	t.Run("setup template", func(t *testing.T) {
		content := setupScriptTemplate(repos)

		// Should contain shebang
		if !strings.Contains(content, "#!/bin/bash") {
			t.Error("setup template missing shebang")
		}

		// Should document both repos
		if !strings.Contains(content, config.GenerateEnvVarName("my-app")) {
			t.Error("setup template missing my-app env var")
		}

		if !strings.Contains(content, config.GenerateEnvVarName("my-lib")) {
			t.Error("setup template missing my-lib env var")
		}
	})

	t.Run("cleanup template", func(t *testing.T) {
		content := cleanupScriptTemplate(repos)

		if !strings.Contains(content, "#!/bin/bash") {
			t.Error("cleanup template missing shebang")
		}

		if !strings.Contains(content, "Cleaning up") {
			t.Error("cleanup template missing cleanup message")
		}
	})

	t.Run("command template", func(t *testing.T) {
		content := sampleCommandTemplate("deploy", repos)

		if !strings.Contains(content, "#!/bin/bash") {
			t.Error("command template missing shebang")
		}

		if !strings.Contains(content, "deploy") {
			t.Error("command template missing command name")
		}

		if !strings.Contains(content, "ramp run deploy") {
			t.Error("command template missing usage instructions")
		}
	})
}
