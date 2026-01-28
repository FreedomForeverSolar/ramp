package operations

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ramp/internal/config"
)

// MockOutputStreamer captures command output for testing
type MockOutputStreamer struct {
	Lines      []string
	ErrorLines []string
}

func (m *MockOutputStreamer) WriteLine(line string) {
	m.Lines = append(m.Lines, line)
}

func (m *MockOutputStreamer) WriteErrorLine(line string) {
	m.ErrorLines = append(m.ErrorLines, line)
}

// AddCommand adds a custom command to the test project config
func (tp *TestProject) AddCommand(name, scriptContent string) {
	tp.t.Helper()

	scriptsDir := filepath.Join(tp.RampDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		tp.t.Fatalf("failed to create scripts dir: %v", err)
	}

	scriptPath := filepath.Join(scriptsDir, name+".sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		tp.t.Fatalf("failed to write script: %v", err)
	}

	tp.Config.Commands = append(tp.Config.Commands, &config.Command{
		Name:    name,
		Command: "scripts/" + name + ".sh",
	})

	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		tp.t.Fatalf("failed to save config: %v", err)
	}
}

// === RUN OPERATION TESTS ===

func TestRunCommand_RepoPathEnvVars_FeatureMode(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("frontend")
	tp.InitRepo("api")

	// Create a command that prints RAMP_REPO_PATH_* env vars
	tp.AddCommand("print-paths", `#!/bin/bash
env | grep RAMP_REPO_PATH | sort
`)

	progress := &MockProgressReporter{}

	// Create a feature first
	_, err := Up(UpOptions{
		FeatureName: "test-feature",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		SkipRefresh: true,
	})
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	// Run command in feature mode
	output := &MockOutputStreamer{}
	_, err = RunCommand(RunOptions{
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		CommandName: "print-paths",
		FeatureName: "test-feature",
		Progress:    progress,
		Output:      output,
	})
	if err != nil {
		t.Fatalf("RunCommand() error = %v", err)
	}

	// In feature mode, RAMP_REPO_PATH_* should point to trees/<feature>/<repo>
	expectedTreesDir := filepath.Join(tp.Dir, "trees", "test-feature")

	foundFrontend := false
	foundAPI := false

	for _, line := range output.Lines {
		if path, ok := strings.CutPrefix(line, "RAMP_REPO_PATH_FRONTEND="); ok {
			expectedPath := filepath.Join(expectedTreesDir, "frontend")
			if path != expectedPath {
				t.Errorf("RAMP_REPO_PATH_FRONTEND = %q, want %q (worktree path)", path, expectedPath)
			}
			foundFrontend = true
		}
		if path, ok := strings.CutPrefix(line, "RAMP_REPO_PATH_API="); ok {
			expectedPath := filepath.Join(expectedTreesDir, "api")
			if path != expectedPath {
				t.Errorf("RAMP_REPO_PATH_API = %q, want %q (worktree path)", path, expectedPath)
			}
			foundAPI = true
		}
	}

	if !foundFrontend {
		t.Error("RAMP_REPO_PATH_FRONTEND not found in output")
	}
	if !foundAPI {
		t.Error("RAMP_REPO_PATH_API not found in output")
	}
}

func TestRunCommand_RepoPathEnvVars_SourceMode(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("frontend")
	tp.InitRepo("api")

	// Create a command that prints RAMP_REPO_PATH_* env vars
	tp.AddCommand("print-paths", `#!/bin/bash
env | grep RAMP_REPO_PATH | sort
`)

	progress := &MockProgressReporter{}
	output := &MockOutputStreamer{}

	// Run command in source mode (no feature name)
	_, err := RunCommand(RunOptions{
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		CommandName: "print-paths",
		FeatureName: "", // source mode
		Progress:    progress,
		Output:      output,
	})
	if err != nil {
		t.Fatalf("RunCommand() error = %v", err)
	}

	// In source mode, RAMP_REPO_PATH_* should point to repos/<repo>
	foundFrontend := false
	foundAPI := false

	for _, line := range output.Lines {
		if path, ok := strings.CutPrefix(line, "RAMP_REPO_PATH_FRONTEND="); ok {
			expectedPath := filepath.Join(tp.Dir, "repos", "frontend")
			if path != expectedPath {
				t.Errorf("RAMP_REPO_PATH_FRONTEND = %q, want %q (source path)", path, expectedPath)
			}
			foundFrontend = true
		}
		if path, ok := strings.CutPrefix(line, "RAMP_REPO_PATH_API="); ok {
			expectedPath := filepath.Join(tp.Dir, "repos", "api")
			if path != expectedPath {
				t.Errorf("RAMP_REPO_PATH_API = %q, want %q (source path)", path, expectedPath)
			}
			foundAPI = true
		}
	}

	if !foundFrontend {
		t.Error("RAMP_REPO_PATH_FRONTEND not found in output")
	}
	if !foundAPI {
		t.Error("RAMP_REPO_PATH_API not found in output")
	}
}

func TestRunCommand_RepoPathEnvVars_FeatureMode_NotSourcePath(t *testing.T) {
	// This test explicitly verifies the bug: in feature mode,
	// RAMP_REPO_PATH_* should NOT point to source repos
	tp := NewTestProject(t)
	tp.InitRepo("myrepo")

	tp.AddCommand("print-paths", `#!/bin/bash
env | grep RAMP_REPO_PATH | sort
`)

	progress := &MockProgressReporter{}

	// Create a feature
	_, err := Up(UpOptions{
		FeatureName: "my-feature",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		SkipRefresh: true,
	})
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	output := &MockOutputStreamer{}
	_, err = RunCommand(RunOptions{
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		CommandName: "print-paths",
		FeatureName: "my-feature",
		Progress:    progress,
		Output:      output,
	})
	if err != nil {
		t.Fatalf("RunCommand() error = %v", err)
	}

	// The bug: RAMP_REPO_PATH_* incorrectly points to repos/ instead of trees/
	sourceReposPath := filepath.Join(tp.Dir, "repos")

	for _, line := range output.Lines {
		if strings.HasPrefix(line, "RAMP_REPO_PATH_") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			path := parts[1]

			// In feature mode, paths should NOT contain /repos/
			if strings.HasPrefix(path, sourceReposPath) {
				t.Errorf("Bug detected: %s points to source repos path %q, should point to trees path", parts[0], path)
			}

			// Paths SHOULD contain /trees/<feature>/
			expectedPrefix := filepath.Join(tp.Dir, "trees", "my-feature")
			if !strings.HasPrefix(path, expectedPrefix) {
				t.Errorf("%s = %q, should start with %q", parts[0], path, expectedPrefix)
			}
		}
	}
}

// === CANCEL/SIGNAL HANDLING TESTS ===

func TestExecuteWithStreaming_CancelSendsSIGTERM(t *testing.T) {
	// Create a temporary script that:
	// 1. Writes "STARTED" to a marker file
	// 2. Sets up a trap handler that writes "TRAPPED" on SIGTERM
	// 3. Sleeps indefinitely
	// 4. The trap handler writes "CLEANUP" before exiting

	markerFile := filepath.Join(t.TempDir(), "marker.txt")
	scriptFile := filepath.Join(t.TempDir(), "test-trap.sh")

	script := `#!/bin/bash
MARKER_FILE="` + markerFile + `"

cleanup() {
    echo "TRAPPED" >> "$MARKER_FILE"
    echo "CLEANUP" >> "$MARKER_FILE"
    exit 0
}
trap cleanup TERM INT

echo "STARTED" >> "$MARKER_FILE"

# Keep running
while true; do
    sleep 0.1
done
`

	if err := os.WriteFile(scriptFile, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	// Create cancel channel
	cancel := make(chan struct{})

	output := &MockOutputStreamer{}

	// Run in a goroutine
	done := make(chan struct{})
	var exitCode int
	var execErr error

	go func() {
		defer close(done)
		cmd := createTestCommand(scriptFile)
		exitCode, execErr = executeWithStreaming(cmd, output, cancel, nil)
	}()

	// Wait for script to start (poll for marker file)
	started := waitForMarker(t, markerFile, "STARTED", 5*time.Second)
	if !started {
		t.Fatal("Script did not start within timeout")
	}

	// Close cancel channel to trigger termination
	close(cancel)

	// Wait for execution to complete
	select {
	case <-done:
		// OK
	case <-time.After(10 * time.Second):
		t.Fatal("Command did not terminate within timeout")
	}

	// Verify we got the cancelled error
	if execErr != ErrCommandCancelled {
		t.Errorf("Expected ErrCommandCancelled, got: %v", execErr)
	}

	if exitCode != -1 {
		t.Errorf("Expected exit code -1, got: %d", exitCode)
	}

	// Read the marker file to verify SIGTERM was received (not SIGKILL)
	// If SIGKILL was sent, the trap handler wouldn't run and "TRAPPED" wouldn't appear
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Failed to read marker file: %v", err)
	}

	markerContent := string(content)

	if !strings.Contains(markerContent, "TRAPPED") {
		t.Errorf("Trap handler did not run (SIGKILL was likely sent instead of SIGTERM).\nMarker file contents: %s", markerContent)
	}

	if !strings.Contains(markerContent, "CLEANUP") {
		t.Errorf("Cleanup did not run.\nMarker file contents: %s", markerContent)
	}
}

func TestExecuteWithStreaming_CancelKillsChildProcesses(t *testing.T) {
	// Create a script that spawns a child process writing to a file,
	// then verify the child is also terminated when parent is cancelled

	markerFile := filepath.Join(t.TempDir(), "child-marker.txt")
	scriptFile := filepath.Join(t.TempDir(), "parent-child.sh")

	script := `#!/bin/bash
MARKER_FILE="` + markerFile + `"

# Spawn child that writes to file every 100ms
(
    while true; do
        echo "CHILD_RUNNING" >> "$MARKER_FILE"
        sleep 0.1
    done
) &

echo "PARENT_STARTED" >> "$MARKER_FILE"

# Wait forever
while true; do
    sleep 0.1
done
`

	if err := os.WriteFile(scriptFile, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	cancel := make(chan struct{})
	output := &MockOutputStreamer{}

	done := make(chan struct{})
	go func() {
		defer close(done)
		cmd := createTestCommand(scriptFile)
		executeWithStreaming(cmd, output, cancel, nil)
	}()

	// Wait for parent and child to start
	started := waitForMarker(t, markerFile, "PARENT_STARTED", 5*time.Second)
	if !started {
		t.Fatal("Parent script did not start within timeout")
	}

	// Wait a bit for child to write a few times
	time.Sleep(300 * time.Millisecond)

	// Count how many CHILD_RUNNING entries we have before cancellation
	contentBefore, _ := os.ReadFile(markerFile)
	countBefore := strings.Count(string(contentBefore), "CHILD_RUNNING")

	// Cancel
	close(cancel)

	// Wait for execution to complete
	select {
	case <-done:
		// OK
	case <-time.After(10 * time.Second):
		t.Fatal("Command did not terminate within timeout")
	}

	// Wait a bit more to see if child is still writing
	time.Sleep(500 * time.Millisecond)

	// Check that child stopped writing
	contentAfter, _ := os.ReadFile(markerFile)
	countAfter := strings.Count(string(contentAfter), "CHILD_RUNNING")

	// Allow for maybe 1-2 more writes during shutdown, but not many more
	additionalWrites := countAfter - countBefore
	if additionalWrites > 5 {
		t.Errorf("Child process appears to still be running after cancel (wrote %d additional times)", additionalWrites)
	}
}

func TestRunCommand_WithCancelChannel(t *testing.T) {
	tp := NewTestProject(t)
	tp.InitRepo("repo1")

	// Create a long-running command
	tp.AddCommand("long-running", `#!/bin/bash
echo "STARTED"
while true; do
    sleep 0.1
done
`)

	// Create feature
	progress := &MockProgressReporter{}
	_, err := Up(UpOptions{
		FeatureName: "cancel-test",
		ProjectDir:  tp.Dir,
		Config:      tp.Config,
		Progress:    progress,
		SkipRefresh: true,
	})
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	// Create cancel channel
	cancel := make(chan struct{})
	output := &MockOutputStreamer{}

	// Run in goroutine
	done := make(chan error)
	go func() {
		_, err := RunCommand(RunOptions{
			ProjectDir:  tp.Dir,
			Config:      tp.Config,
			CommandName: "long-running",
			FeatureName: "cancel-test",
			Progress:    progress,
			Output:      output,
			Cancel:      cancel,
		})
		done <- err
	}()

	// Wait for command to start (check output)
	startTime := time.Now()
	for {
		if time.Since(startTime) > 5*time.Second {
			t.Fatal("Command did not start within timeout")
		}
		if len(output.Lines) > 0 && output.Lines[0] == "STARTED" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Cancel the command
	close(cancel)

	// Wait for completion
	select {
	case err := <-done:
		if err != ErrCommandCancelled {
			t.Errorf("Expected ErrCommandCancelled, got: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Command did not terminate within timeout after cancel")
	}
}

func TestExecuteWithStreaming_FallbackToSIGKILL(t *testing.T) {
	// Create a script that ignores SIGTERM (simulating a stuck process)
	// The implementation should fallback to SIGKILL after 5 seconds
	// We'll use a shorter timeout for testing by modifying the test expectations

	if testing.Short() {
		t.Skip("Skipping SIGKILL fallback test in short mode (takes >5 seconds)")
	}

	markerFile := filepath.Join(t.TempDir(), "ignore-term.txt")
	scriptFile := filepath.Join(t.TempDir(), "ignore-term.sh")

	// This script ignores SIGTERM (trap '' TERM)
	script := `#!/bin/bash
MARKER_FILE="` + markerFile + `"

# Ignore SIGTERM
trap '' TERM

echo "STARTED" >> "$MARKER_FILE"

# Keep running
while true; do
    sleep 0.1
done
`

	if err := os.WriteFile(scriptFile, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	cancel := make(chan struct{})
	output := &MockOutputStreamer{}

	done := make(chan struct{})
	var startTime, endTime time.Time

	go func() {
		defer close(done)
		startTime = time.Now()
		cmd := createTestCommand(scriptFile)
		executeWithStreaming(cmd, output, cancel, nil)
		endTime = time.Now()
	}()

	// Wait for script to start
	started := waitForMarker(t, markerFile, "STARTED", 5*time.Second)
	if !started {
		t.Fatal("Script did not start within timeout")
	}

	// Cancel
	close(cancel)

	// Wait for execution to complete (should take ~5 seconds for SIGKILL fallback)
	select {
	case <-done:
		// OK
	case <-time.After(15 * time.Second):
		t.Fatal("Command did not terminate within timeout (SIGKILL fallback may have failed)")
	}

	// Verify it took at least 4 seconds (allowing some tolerance)
	// This proves the SIGKILL fallback occurred after waiting for SIGTERM
	elapsed := endTime.Sub(startTime)
	if elapsed < 4*time.Second {
		t.Errorf("Process terminated too quickly (%v), SIGKILL fallback may not be working", elapsed)
	}
}

// Helper to create a test command
func createTestCommand(scriptPath string) *exec.Cmd {
	return exec.Command("/bin/bash", "-l", scriptPath)
}

// Helper to wait for a specific string to appear in a marker file
func waitForMarker(t *testing.T, markerFile, expected string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		content, err := os.ReadFile(markerFile)
		if err == nil && strings.Contains(string(content), expected) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
