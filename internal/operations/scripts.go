package operations

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ramp/internal/config"
	"ramp/internal/ports"
	"ramp/internal/ui"
)

// RunSetupScript runs the setup script for a feature with progress reporting.
// If output is provided, stdout/stderr will be streamed via the OutputStreamer.
func RunSetupScript(projectDir, treesDir, featureName string, cfg *config.Config, allocatedPorts []int, repos map[string]*config.Repo, progress ProgressReporter, output OutputStreamer) error {
	if cfg.Setup == "" {
		return nil
	}

	scriptPath := filepath.Join(projectDir, ".ramp", cfg.Setup)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("setup script not found: %s", scriptPath)
	}

	progress.Update(fmt.Sprintf("Running setup script: %s", cfg.Setup))

	// Use login shell (-l) to source user's profile and get full PATH
	// This ensures tools like bun, node, etc. are available in GUI environments
	cmd := exec.Command("/bin/bash", "-l", scriptPath)
	cmd.Dir = treesDir

	// Set up environment variables
	cmd.Env = buildScriptEnv(projectDir, treesDir, featureName, allocatedPorts, cfg, repos)

	// If output streamer provided, stream output in real-time
	if output != nil {
		exitCode, err := executeWithStreaming(cmd, output, nil, nil)
		if err != nil {
			return err
		}
		if exitCode != 0 {
			return fmt.Errorf("setup script exited with code %d", exitCode)
		}
		return nil
	}

	// Fall back to capture mode (for CLI without streaming)
	return runScriptWithCapture(cmd)
}

// RunCleanupScript runs the cleanup script for a feature with progress reporting.
func RunCleanupScript(projectDir, treesDir, featureName string, cfg *config.Config, progress ProgressReporter) error {
	if cfg.Cleanup == "" {
		return nil
	}

	scriptPath := filepath.Join(projectDir, ".ramp", cfg.Cleanup)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("cleanup script not found: %s", scriptPath)
	}

	progress.Update(fmt.Sprintf("Running cleanup script: %s", cfg.Cleanup))

	// Use login shell (-l) to source user's profile and get full PATH
	// This ensures tools like bun, node, etc. are available in GUI environments
	cmd := exec.Command("/bin/bash", "-l", scriptPath)
	cmd.Dir = treesDir

	// Get allocated ports for this feature
	var allocatedPorts []int
	if cfg.HasPortConfig() {
		portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
		if err == nil {
			if p, exists := portAllocations.GetPorts(featureName); exists {
				allocatedPorts = p
			}
		}
	}

	repos := cfg.GetRepos()

	// Set up environment variables
	cmd.Env = buildScriptEnv(projectDir, treesDir, featureName, allocatedPorts, cfg, repos)

	return runScriptWithCapture(cmd)
}

// buildScriptEnv builds the environment variables for script execution.
func buildScriptEnv(projectDir, treesDir, featureName string, allocatedPorts []int, cfg *config.Config, repos map[string]*config.Repo) []string {
	env := os.Environ()

	// Standard RAMP variables
	env = append(env, fmt.Sprintf("RAMP_PROJECT_DIR=%s", projectDir))
	env = append(env, fmt.Sprintf("RAMP_TREES_DIR=%s", treesDir))
	env = append(env, fmt.Sprintf("RAMP_WORKTREE_NAME=%s", featureName))

	// Add port environment variables if configured
	if cfg.HasPortConfig() && len(allocatedPorts) > 0 {
		env = append(env, fmt.Sprintf("RAMP_PORT=%d", allocatedPorts[0]))
		for i, port := range allocatedPorts {
			env = append(env, fmt.Sprintf("RAMP_PORT_%d=%d", i+1, port))
		}
	}

	// Add repo path variables
	for name, repo := range repos {
		envVarName := config.GenerateEnvVarName(name)
		repoPath := repo.GetRepoPath(projectDir)
		env = append(env, fmt.Sprintf("%s=%s", envVarName, repoPath))
	}

	// Add local config environment variables
	localCfg, err := config.LoadLocalConfig(projectDir)
	if err == nil && localCfg != nil {
		for key, value := range localCfg.Preferences {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return env
}

// runScriptWithCapture runs a script, showing output only on error (unless verbose mode).
func runScriptWithCapture(cmd *exec.Cmd) error {
	// In verbose mode, show output directly
	if ui.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// In non-verbose mode, capture output and only show on error
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Show captured output on error
		if stdout.Len() > 0 {
			fmt.Print(stdout.String())
		}
		if stderr.Len() > 0 {
			fmt.Fprint(os.Stderr, stderr.String())
		}
		return err
	}

	return nil
}
