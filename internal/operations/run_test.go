package operations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
