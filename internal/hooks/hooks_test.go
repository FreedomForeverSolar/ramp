package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"ramp/internal/config"
)

// MockProgressReporter captures progress messages for testing
type MockProgressReporter struct {
	InfoMessages    []string
	WarningMessages []string
}

func (m *MockProgressReporter) Info(message string) {
	m.InfoMessages = append(m.InfoMessages, message)
}

func (m *MockProgressReporter) Warning(message string) {
	m.WarningMessages = append(m.WarningMessages, message)
}

func TestRunHook_UsesBaseDir(t *testing.T) {
	// Create a temp directory structure simulating user config
	userConfigDir := t.TempDir()
	projectDir := t.TempDir()

	// Create hook script in user config dir
	hooksDir := filepath.Join(userConfigDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("failed to create hooks dir: %v", err)
	}

	// Create a simple hook script that writes to a marker file
	markerFile := filepath.Join(t.TempDir(), "marker.txt")
	scriptContent := `#!/bin/bash
echo "HOOK_EXECUTED" > "` + markerFile + `"
`
	scriptPath := filepath.Join(hooksDir, "user-hook.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Create hook with BaseDir pointing to user config dir
	hook := &config.Hook{
		Event:   "up",
		Command: "hooks/user-hook.sh",
		BaseDir: userConfigDir, // This is the key: BaseDir points to user config
	}

	// Run the hook - projectDir is different from BaseDir
	err := runHook(hook, projectDir, projectDir, nil)
	if err != nil {
		t.Fatalf("runHook() error = %v", err)
	}

	// Verify the hook was executed by checking marker file
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("hook did not create marker file: %v", err)
	}

	if string(content) != "HOOK_EXECUTED\n" {
		t.Errorf("marker file content = %q, want %q", string(content), "HOOK_EXECUTED\n")
	}
}

func TestRunHook_FallbackWithoutBaseDir(t *testing.T) {
	// Test backward compatibility: when BaseDir is empty, use projectDir/.ramp/
	projectDir := t.TempDir()
	rampDir := filepath.Join(projectDir, ".ramp")
	hooksDir := filepath.Join(rampDir, "hooks")

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("failed to create hooks dir: %v", err)
	}

	// Create hook script in project .ramp dir
	markerFile := filepath.Join(t.TempDir(), "marker.txt")
	scriptContent := `#!/bin/bash
echo "PROJECT_HOOK" > "` + markerFile + `"
`
	scriptPath := filepath.Join(hooksDir, "project-hook.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Create hook WITHOUT BaseDir (backward compatibility)
	hook := &config.Hook{
		Event:   "up",
		Command: "hooks/project-hook.sh",
		BaseDir: "", // Empty - should fall back to projectDir/.ramp/
	}

	err := runHook(hook, projectDir, projectDir, nil)
	if err != nil {
		t.Fatalf("runHook() error = %v", err)
	}

	// Verify execution
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("hook did not create marker file: %v", err)
	}

	if string(content) != "PROJECT_HOOK\n" {
		t.Errorf("marker file content = %q, want %q", string(content), "PROJECT_HOOK\n")
	}
}

func TestRunHook_AbsolutePathIgnoresBaseDir(t *testing.T) {
	// Test that absolute paths are used directly, ignoring BaseDir
	scriptDir := t.TempDir()
	projectDir := t.TempDir()

	markerFile := filepath.Join(t.TempDir(), "marker.txt")
	scriptContent := `#!/bin/bash
echo "ABSOLUTE_PATH" > "` + markerFile + `"
`
	absoluteScriptPath := filepath.Join(scriptDir, "absolute-hook.sh")
	if err := os.WriteFile(absoluteScriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Create hook with absolute path and arbitrary BaseDir
	hook := &config.Hook{
		Event:   "up",
		Command: absoluteScriptPath, // Absolute path
		BaseDir: "/some/other/dir",  // Should be ignored for absolute paths
	}

	err := runHook(hook, projectDir, projectDir, nil)
	if err != nil {
		t.Fatalf("runHook() error = %v", err)
	}

	// Verify execution
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("hook did not create marker file: %v", err)
	}

	if string(content) != "ABSOLUTE_PATH\n" {
		t.Errorf("marker file content = %q, want %q", string(content), "ABSOLUTE_PATH\n")
	}
}

func TestExecuteHooks_WithMixedBaseDirs(t *testing.T) {
	projectDir := t.TempDir()
	userConfigDir := t.TempDir()
	outputFile := filepath.Join(t.TempDir(), "output.txt")

	// Create project hook
	projectHooksDir := filepath.Join(projectDir, ".ramp", "hooks")
	if err := os.MkdirAll(projectHooksDir, 0755); err != nil {
		t.Fatalf("failed to create project hooks dir: %v", err)
	}
	projectScript := `#!/bin/bash
echo "PROJECT" >> "` + outputFile + `"
`
	if err := os.WriteFile(filepath.Join(projectHooksDir, "project.sh"), []byte(projectScript), 0755); err != nil {
		t.Fatalf("failed to write project script: %v", err)
	}

	// Create user hook
	userHooksDir := filepath.Join(userConfigDir, "hooks")
	if err := os.MkdirAll(userHooksDir, 0755); err != nil {
		t.Fatalf("failed to create user hooks dir: %v", err)
	}
	userScript := `#!/bin/bash
echo "USER" >> "` + outputFile + `"
`
	if err := os.WriteFile(filepath.Join(userHooksDir, "user.sh"), []byte(userScript), 0755); err != nil {
		t.Fatalf("failed to write user script: %v", err)
	}

	// Create hooks with different BaseDirs
	hooks := []*config.Hook{
		{
			Event:   "up",
			Command: "hooks/project.sh",
			BaseDir: filepath.Join(projectDir, ".ramp"),
		},
		{
			Event:   "up",
			Command: "hooks/user.sh",
			BaseDir: userConfigDir,
		},
	}

	progress := &MockProgressReporter{}

	ExecuteHooks(Up, hooks, projectDir, projectDir, nil, progress)

	// Verify both hooks executed
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("hooks did not create output file: %v", err)
	}

	expected := "PROJECT\nUSER\n"
	if string(content) != expected {
		t.Errorf("output file content = %q, want %q", string(content), expected)
	}

	// Verify progress messages
	if len(progress.InfoMessages) != 2 {
		t.Errorf("expected 2 info messages, got %d", len(progress.InfoMessages))
	}
}

func TestFilterHooksByEvent(t *testing.T) {
	hooks := []*config.Hook{
		{Event: "up", Command: "up1.sh"},
		{Event: "down", Command: "down1.sh"},
		{Event: "up", Command: "up2.sh"},
		{Event: "run", Command: "run1.sh"},
	}

	upHooks := filterHooksByEvent(hooks, Up)
	if len(upHooks) != 2 {
		t.Errorf("expected 2 up hooks, got %d", len(upHooks))
	}

	downHooks := filterHooksByEvent(hooks, Down)
	if len(downHooks) != 1 {
		t.Errorf("expected 1 down hook, got %d", len(downHooks))
	}

	runHooks := filterHooksByEvent(hooks, Run)
	if len(runHooks) != 1 {
		t.Errorf("expected 1 run hook, got %d", len(runHooks))
	}
}

func TestMatchesCommand(t *testing.T) {
	tests := []struct {
		name        string
		hookFor     string
		commandName string
		want        bool
	}{
		{
			name:        "empty for matches all",
			hookFor:     "",
			commandName: "anything",
			want:        true,
		},
		{
			name:        "exact match",
			hookFor:     "deploy",
			commandName: "deploy",
			want:        true,
		},
		{
			name:        "exact match fails",
			hookFor:     "deploy",
			commandName: "build",
			want:        false,
		},
		{
			name:        "prefix pattern matches",
			hookFor:     "test-*",
			commandName: "test-unit",
			want:        true,
		},
		{
			name:        "prefix pattern matches integration",
			hookFor:     "test-*",
			commandName: "test-integration",
			want:        true,
		},
		{
			name:        "prefix pattern fails",
			hookFor:     "test-*",
			commandName: "build",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := &config.Hook{For: tt.hookFor}
			got := matchesCommand(hook, tt.commandName)
			if got != tt.want {
				t.Errorf("matchesCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateHookEvent(t *testing.T) {
	tests := []struct {
		event   string
		wantErr bool
	}{
		{"up", false},
		{"down", false},
		{"run", false},
		{"invalid", true},
		{"", true},
		{"UP", true}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			err := ValidateHookEvent(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHookEvent(%q) error = %v, wantErr %v", tt.event, err, tt.wantErr)
			}
		})
	}
}
