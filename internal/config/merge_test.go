package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeConfigs_SetsBaseDir(t *testing.T) {
	projectDir := "/home/user/myproject"
	rampDir := filepath.Join(projectDir, ".ramp")

	projectCfg := &Config{
		Name: "test-project",
		Commands: []*Command{
			{Name: "build", Command: "scripts/build.sh"},
		},
		Hooks: []*Hook{
			{Event: "up", Command: "hooks/up.sh"},
		},
	}

	localCfg := &LocalConfig{
		Commands: []*Command{
			{Name: "local-cmd", Command: "scripts/local.sh"},
		},
		Hooks: []*Hook{
			{Event: "down", Command: "hooks/down.sh"},
		},
	}

	// Note: User config is nil in this test, tested separately below

	merged := MergeConfigs(projectCfg, localCfg, nil, projectDir)

	// Verify project command has BaseDir set to .ramp dir
	if len(merged.Commands) < 1 {
		t.Fatal("expected at least 1 command")
	}
	if merged.Commands[0].BaseDir != rampDir {
		t.Errorf("project command BaseDir = %q, want %q", merged.Commands[0].BaseDir, rampDir)
	}

	// Verify local command has BaseDir set to .ramp dir
	if len(merged.Commands) < 2 {
		t.Fatal("expected at least 2 commands")
	}
	if merged.Commands[1].BaseDir != rampDir {
		t.Errorf("local command BaseDir = %q, want %q", merged.Commands[1].BaseDir, rampDir)
	}

	// Verify project hook has BaseDir set to .ramp dir
	if len(merged.Hooks) < 1 {
		t.Fatal("expected at least 1 hook")
	}
	if merged.Hooks[0].BaseDir != rampDir {
		t.Errorf("project hook BaseDir = %q, want %q", merged.Hooks[0].BaseDir, rampDir)
	}

	// Verify local hook has BaseDir set to .ramp dir
	if len(merged.Hooks) < 2 {
		t.Fatal("expected at least 2 hooks")
	}
	if merged.Hooks[1].BaseDir != rampDir {
		t.Errorf("local hook BaseDir = %q, want %q", merged.Hooks[1].BaseDir, rampDir)
	}
}

func TestMergeConfigs_UserConfigBaseDir(t *testing.T) {
	projectDir := "/home/user/myproject"
	rampDir := filepath.Join(projectDir, ".ramp")

	// Get expected user config dir
	userConfigDir, err := GetUserConfigDir()
	if err != nil {
		t.Fatalf("GetUserConfigDir() error = %v", err)
	}

	projectCfg := &Config{
		Name: "test-project",
		Commands: []*Command{
			{Name: "build", Command: "scripts/build.sh"},
		},
		Hooks: []*Hook{
			{Event: "up", Command: "hooks/up.sh"},
		},
	}

	userCfg := &UserConfig{
		Commands: []*Command{
			{Name: "user-cmd", Command: "scripts/user.sh"},
		},
		Hooks: []*Hook{
			{Event: "up", Command: "hooks/user-up.sh"},
		},
	}

	merged := MergeConfigs(projectCfg, nil, userCfg, projectDir)

	// Verify project command has BaseDir set to .ramp dir
	if merged.Commands[0].BaseDir != rampDir {
		t.Errorf("project command BaseDir = %q, want %q", merged.Commands[0].BaseDir, rampDir)
	}

	// Verify user command has BaseDir set to user config dir
	if len(merged.Commands) < 2 {
		t.Fatal("expected at least 2 commands")
	}
	if merged.Commands[1].BaseDir != userConfigDir {
		t.Errorf("user command BaseDir = %q, want %q", merged.Commands[1].BaseDir, userConfigDir)
	}

	// Verify project hook has BaseDir set to .ramp dir
	if merged.Hooks[0].BaseDir != rampDir {
		t.Errorf("project hook BaseDir = %q, want %q", merged.Hooks[0].BaseDir, rampDir)
	}

	// Verify user hook has BaseDir set to user config dir
	if len(merged.Hooks) < 2 {
		t.Fatal("expected at least 2 hooks")
	}
	if merged.Hooks[1].BaseDir != userConfigDir {
		t.Errorf("user hook BaseDir = %q, want %q", merged.Hooks[1].BaseDir, userConfigDir)
	}
}

func TestMergeConfigs_CommandPrecedence(t *testing.T) {
	projectDir := "/home/user/myproject"

	projectCfg := &Config{
		Name: "test-project",
		Commands: []*Command{
			{Name: "build", Command: "project-build.sh"},
		},
	}

	localCfg := &LocalConfig{
		Commands: []*Command{
			{Name: "build", Command: "local-build.sh"},  // Same name - should be ignored
			{Name: "local", Command: "local-only.sh"},   // Unique name
		},
	}

	userCfg := &UserConfig{
		Commands: []*Command{
			{Name: "build", Command: "user-build.sh"},  // Same name - should be ignored
			{Name: "local", Command: "user-local.sh"},  // Same as local - should be ignored
			{Name: "user", Command: "user-only.sh"},    // Unique name
		},
	}

	merged := MergeConfigs(projectCfg, localCfg, userCfg, projectDir)

	// Should have 3 commands: build (project), local (local), user (user)
	if len(merged.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(merged.Commands))
	}

	// Verify project command wins
	buildCmd := merged.GetCommand("build")
	if buildCmd.Command != "project-build.sh" {
		t.Errorf("build command = %q, want %q (project wins)", buildCmd.Command, "project-build.sh")
	}

	// Verify local command wins over user
	localCmd := merged.GetCommand("local")
	if localCmd.Command != "local-only.sh" {
		t.Errorf("local command = %q, want %q (local wins over user)", localCmd.Command, "local-only.sh")
	}

	// Verify user command is included
	userCmd := merged.GetCommand("user")
	if userCmd == nil || userCmd.Command != "user-only.sh" {
		t.Errorf("user command not found or incorrect")
	}
}

func TestMergeConfigs_HookAggregation(t *testing.T) {
	projectDir := "/home/user/myproject"

	projectCfg := &Config{
		Name: "test-project",
		Hooks: []*Hook{
			{Event: "up", Command: "project-up.sh"},
		},
	}

	localCfg := &LocalConfig{
		Hooks: []*Hook{
			{Event: "up", Command: "local-up.sh"},
		},
	}

	userCfg := &UserConfig{
		Hooks: []*Hook{
			{Event: "up", Command: "user-up.sh"},
		},
	}

	merged := MergeConfigs(projectCfg, localCfg, userCfg, projectDir)

	// All hooks should be present (hooks aggregate, don't override)
	if len(merged.Hooks) != 3 {
		t.Fatalf("expected 3 hooks, got %d", len(merged.Hooks))
	}

	// Verify order: project -> local -> user
	if merged.Hooks[0].Command != "project-up.sh" {
		t.Errorf("first hook should be project hook")
	}
	if merged.Hooks[1].Command != "local-up.sh" {
		t.Errorf("second hook should be local hook")
	}
	if merged.Hooks[2].Command != "user-up.sh" {
		t.Errorf("third hook should be user hook")
	}
}

func TestMergeConfigs_DoesNotMutateOriginal(t *testing.T) {
	projectDir := "/home/user/myproject"

	originalCmd := &Command{Name: "build", Command: "build.sh"}
	originalHook := &Hook{Event: "up", Command: "up.sh"}

	projectCfg := &Config{
		Name:     "test-project",
		Commands: []*Command{originalCmd},
		Hooks:    []*Hook{originalHook},
	}

	_ = MergeConfigs(projectCfg, nil, nil, projectDir)

	// Original structs should not have BaseDir set
	if originalCmd.BaseDir != "" {
		t.Errorf("original command was mutated: BaseDir = %q, want empty", originalCmd.BaseDir)
	}
	if originalHook.BaseDir != "" {
		t.Errorf("original hook was mutated: BaseDir = %q, want empty", originalHook.BaseDir)
	}
}

func TestGetUserConfigDir(t *testing.T) {
	dir, err := GetUserConfigDir()
	if err != nil {
		t.Fatalf("GetUserConfigDir() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "ramp")

	if dir != expected {
		t.Errorf("GetUserConfigDir() = %q, want %q", dir, expected)
	}
}

func TestLoadMergedConfig(t *testing.T) {
	tempDir := t.TempDir()

	// Create project config
	projectContent := `name: test-project
repos:
  - path: repos
    git: git@github.com:owner/repo.git
commands:
  - name: build
    command: scripts/build.sh
hooks:
  - event: up
    command: hooks/up.sh
`
	rampDir := filepath.Join(tempDir, ".ramp")
	if err := os.MkdirAll(rampDir, 0755); err != nil {
		t.Fatalf("failed to create .ramp dir: %v", err)
	}

	configPath := filepath.Join(rampDir, "ramp.yaml")
	if err := os.WriteFile(configPath, []byte(projectContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	merged, err := LoadMergedConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadMergedConfig() error = %v", err)
	}

	// Verify BaseDir is set correctly on project command
	// Note: may also have user commands from ~/.config/ramp/ramp.yaml
	expectedBaseDir := rampDir
	if len(merged.Commands) < 1 {
		t.Fatalf("expected at least 1 command, got %d", len(merged.Commands))
	}
	// Find our project command
	var buildCmd *Command
	for _, cmd := range merged.Commands {
		if cmd.Name == "build" {
			buildCmd = cmd
			break
		}
	}
	if buildCmd == nil {
		t.Fatal("build command not found")
	}
	if buildCmd.BaseDir != expectedBaseDir {
		t.Errorf("command BaseDir = %q, want %q", buildCmd.BaseDir, expectedBaseDir)
	}

	// Verify BaseDir is set correctly on project hook
	// Note: may also have user hooks from ~/.config/ramp/ramp.yaml
	if len(merged.Hooks) < 1 {
		t.Fatalf("expected at least 1 hook, got %d", len(merged.Hooks))
	}
	// The first hook should be from project config (project hooks come first)
	if merged.Hooks[0].BaseDir != expectedBaseDir {
		t.Errorf("hook BaseDir = %q, want %q", merged.Hooks[0].BaseDir, expectedBaseDir)
	}
}
