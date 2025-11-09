package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ramp/internal/config"
)

// TestRunCommandWithFeature tests running a command for a feature
func TestRunCommandWithFeature(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	// Create a simple test command script
	scriptPath := filepath.Join(tp.RampDir, "scripts", "test-cmd.sh")
	scriptContent := `#!/bin/bash
echo "Command executed for feature: $RAMP_WORKTREE_NAME"
echo "Project dir: $RAMP_PROJECT_DIR"
echo "Trees dir: $RAMP_TREES_DIR"
exit 0
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	// Add command to config
	tp.Config.Commands = []*config.Command{
		{Name: "test", Command: "scripts/test-cmd.sh"},
	}
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a feature first
	err := runUp("test-feature", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Run the command for the feature
	err = runCustomCommand("test", "test-feature")
	if err != nil {
		t.Fatalf("runCustomCommand() error = %v", err)
	}
}

// TestRunCommandWithoutFeature tests running a command without a feature (source mode)
func TestRunCommandWithoutFeature(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	// Create a simple test command script
	scriptPath := filepath.Join(tp.RampDir, "scripts", "source-cmd.sh")
	scriptContent := `#!/bin/bash
echo "Command executed in source mode"
echo "Project dir: $RAMP_PROJECT_DIR"
exit 0
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	// Add command to config
	tp.Config.Commands = []*config.Command{
		{Name: "source-test", Command: "scripts/source-cmd.sh"},
	}
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Run the command without a feature
	err := runCustomCommand("source-test", "")
	if err != nil {
		t.Fatalf("runCustomCommand() error = %v", err)
	}
}

// TestRunCommandNotFound tests error when command doesn't exist in config
func TestRunCommandNotFound(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Try to run a non-existent command
	err := runCustomCommand("nonexistent", "")
	if err == nil {
		t.Fatal("runCustomCommand() should fail for non-existent command")
	}

	expectedMsg := "command 'nonexistent' not found in configuration"
	if err.Error() != expectedMsg {
		t.Errorf("error = %q, want %q", err.Error(), expectedMsg)
	}
}

// TestRunCommandFeatureNotFound tests error when feature doesn't exist
func TestRunCommandFeatureNotFound(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	// Create a test command
	scriptPath := filepath.Join(tp.RampDir, "scripts", "test-cmd.sh")
	scriptContent := `#!/bin/bash
exit 0
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	tp.Config.Commands = []*config.Command{
		{Name: "test", Command: "scripts/test-cmd.sh"},
	}
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Try to run for non-existent feature
	err := runCustomCommand("test", "nonexistent-feature")
	if err == nil {
		t.Fatal("runCustomCommand() should fail for non-existent feature")
	}

	expectedMsg := "feature 'nonexistent-feature' not found (trees directory does not exist)"
	if err.Error() != expectedMsg {
		t.Errorf("error = %q, want %q", err.Error(), expectedMsg)
	}
}

// TestRunCommandScriptNotFound tests error when command script file doesn't exist
func TestRunCommandScriptNotFound(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	// Add command to config but don't create the script file
	tp.Config.Commands = []*config.Command{
		{Name: "missing", Command: "scripts/missing.sh"},
	}
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Try to run command with missing script
	err := runCustomCommand("missing", "")
	if err == nil {
		t.Fatal("runCustomCommand() should fail for missing script file")
	}

	// Error should contain "command script not found"
	if !strings.Contains(err.Error(), "command script not found") {
		t.Errorf("error should contain 'command script not found', got %q", err.Error())
	}
}

// TestRunCommandWithEnvironmentVariables tests that environment variables are set correctly
func TestRunCommandWithEnvironmentVariables(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")
	tp.InitRepo("repo2")

	// Create a command that checks environment variables
	scriptPath := filepath.Join(tp.RampDir, "scripts", "env-check.sh")
	scriptContent := `#!/bin/bash
set -e
# Check that required env vars are set
if [ -z "$RAMP_PROJECT_DIR" ]; then
  echo "RAMP_PROJECT_DIR not set"
  exit 1
fi
if [ -z "$RAMP_TREES_DIR" ]; then
  echo "RAMP_TREES_DIR not set"
  exit 1
fi
if [ -z "$RAMP_WORKTREE_NAME" ]; then
  echo "RAMP_WORKTREE_NAME not set"
  exit 1
fi
if [ -z "$RAMP_REPO_PATH_REPO1" ]; then
  echo "RAMP_REPO_PATH_REPO1 not set"
  exit 1
fi
if [ -z "$RAMP_REPO_PATH_REPO2" ]; then
  echo "RAMP_REPO_PATH_REPO2 not set"
  exit 1
fi
echo "All environment variables are set correctly"
exit 0
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	tp.Config.Commands = []*config.Command{
		{Name: "env-check", Command: "scripts/env-check.sh"},
	}
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a feature
	err := runUp("env-test", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Run command and verify env vars are set
	err = runCustomCommand("env-check", "env-test")
	if err != nil {
		t.Fatalf("runCustomCommand() error = %v (env vars not set correctly)", err)
	}
}

// TestRunCommandWithPort tests that RAMP_PORT is set when port is allocated
func TestRunCommandWithPort(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	// Enable port management
	basePort := 3000
	maxPorts := 10
	tp.Config.BasePort = basePort
	tp.Config.MaxPorts = maxPorts

	// Create a command that checks port env var
	scriptPath := filepath.Join(tp.RampDir, "scripts", "port-check.sh")
	scriptContent := `#!/bin/bash
if [ -z "$RAMP_PORT" ]; then
  echo "RAMP_PORT not set"
  exit 1
fi
echo "Port is set to: $RAMP_PORT"
exit 0
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	tp.Config.Commands = []*config.Command{
		{Name: "port-check", Command: "scripts/port-check.sh"},
	}
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a feature (which allocates a port)
	err := runUp("port-test", "", "")
	if err != nil {
		t.Fatalf("runUp() error = %v", err)
	}

	// Run command and verify port is set
	err = runCustomCommand("port-check", "port-test")
	if err != nil {
		t.Fatalf("runCustomCommand() error = %v (port not set correctly)", err)
	}
}

// TestRunCommandFailure tests that command failures are properly reported
func TestRunCommandFailure(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	// Create a command that exits with error
	scriptPath := filepath.Join(tp.RampDir, "scripts", "fail-cmd.sh")
	scriptContent := `#!/bin/bash
echo "This command will fail"
exit 1
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	tp.Config.Commands = []*config.Command{
		{Name: "fail", Command: "scripts/fail-cmd.sh"},
	}
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Run command and expect failure
	err := runCustomCommand("fail", "")
	if err == nil {
		t.Fatal("runCustomCommand() should fail when script exits with non-zero")
	}

	if !strings.Contains(err.Error(), "command 'fail' failed") {
		t.Errorf("error should mention command failure, got %q", err.Error())
	}
}

// TestRunCommandSourceMode tests running commands in source mode
func TestRunCommandSourceMode(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")
	tp.InitRepo("repo2")

	// Create a command that verifies source mode env vars
	scriptPath := filepath.Join(tp.RampDir, "scripts", "source-mode.sh")
	scriptContent := `#!/bin/bash
set -e
# In source mode, RAMP_TREES_DIR and RAMP_WORKTREE_NAME should not be set
if [ -n "$RAMP_TREES_DIR" ]; then
  echo "RAMP_TREES_DIR should not be set in source mode"
  exit 1
fi
if [ -n "$RAMP_WORKTREE_NAME" ]; then
  echo "RAMP_WORKTREE_NAME should not be set in source mode"
  exit 1
fi
# But RAMP_PROJECT_DIR and repo paths should be set
if [ -z "$RAMP_PROJECT_DIR" ]; then
  echo "RAMP_PROJECT_DIR not set"
  exit 1
fi
if [ -z "$RAMP_REPO_PATH_REPO1" ]; then
  echo "RAMP_REPO_PATH_REPO1 not set"
  exit 1
fi
echo "Source mode env vars are correct"
exit 0
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	tp.Config.Commands = []*config.Command{
		{Name: "source-mode", Command: "scripts/source-mode.sh"},
	}
	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Run in source mode (no feature name)
	err := runCustomCommand("source-mode", "")
	if err != nil {
		t.Fatalf("runCustomCommand() error = %v", err)
	}
}
