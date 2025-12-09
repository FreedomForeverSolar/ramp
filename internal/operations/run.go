package operations

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"ramp/internal/config"
	"ramp/internal/ports"
)

// RunOptions configures custom command execution.
type RunOptions struct {
	ProjectDir  string
	Config      *config.Config
	CommandName string
	FeatureName string           // Empty = run against source
	Progress    ProgressReporter
	Output      OutputStreamer   // For streaming stdout/stderr
}

// RunResult contains the results of command execution.
type RunResult struct {
	CommandName string
	ExitCode    int
	Duration    time.Duration
}

// RunCommand executes a custom command defined in ramp.yaml.
// If FeatureName is provided, runs in feature mode with feature-specific env vars.
// If FeatureName is empty, runs in source mode against the project directory.
func RunCommand(opts RunOptions) (*RunResult, error) {
	projectDir := opts.ProjectDir
	cfg := opts.Config
	progress := opts.Progress
	commandName := opts.CommandName
	featureName := opts.FeatureName

	// Find the command in configuration
	command := cfg.GetCommand(commandName)
	if command == nil {
		return nil, fmt.Errorf("command '%s' not found in configuration", commandName)
	}

	scriptPath := filepath.Join(projectDir, ".ramp", command.Command)

	// Validate script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("command script not found: %s", scriptPath)
	}

	start := time.Now()

	var err error
	var exitCode int

	if featureName == "" {
		// Source mode
		progress.Start(fmt.Sprintf("Running '%s' against source repositories", commandName))
		exitCode, err = runInSource(opts, scriptPath)
	} else {
		// Feature mode
		treesDir := filepath.Join(projectDir, "trees", featureName)

		// Validate feature exists
		if _, statErr := os.Stat(treesDir); os.IsNotExist(statErr) {
			return nil, fmt.Errorf("feature '%s' not found (trees directory does not exist)", featureName)
		}

		progress.Start(fmt.Sprintf("Running '%s' for feature '%s'", commandName, featureName))
		exitCode, err = runInFeature(opts, scriptPath, treesDir)
	}

	duration := time.Since(start)

	if err != nil {
		progress.Error(fmt.Sprintf("Command '%s' failed: %v", commandName, err))
		return &RunResult{
			CommandName: commandName,
			ExitCode:    exitCode,
			Duration:    duration,
		}, err
	}

	if exitCode != 0 {
		progress.Error(fmt.Sprintf("Command '%s' exited with code %d", commandName, exitCode))
		return &RunResult{
			CommandName: commandName,
			ExitCode:    exitCode,
			Duration:    duration,
		}, fmt.Errorf("command exited with code %d", exitCode)
	}

	progress.Complete(fmt.Sprintf("Command '%s' completed successfully", commandName))

	return &RunResult{
		CommandName: commandName,
		ExitCode:    0,
		Duration:    duration,
	}, nil
}

// runInFeature executes a command in feature mode with feature-specific env vars.
func runInFeature(opts RunOptions, scriptPath, treesDir string) (int, error) {
	projectDir := opts.ProjectDir
	cfg := opts.Config
	featureName := opts.FeatureName

	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = treesDir

	// Build environment variables
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("RAMP_PROJECT_DIR=%s", projectDir),
		fmt.Sprintf("RAMP_TREES_DIR=%s", treesDir),
		fmt.Sprintf("RAMP_WORKTREE_NAME=%s", featureName),
	)

	// Add port environment variables
	portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
	if err == nil {
		if allocatedPorts, exists := portAllocations.GetPorts(featureName); exists {
			addPortEnvVars(cmd, allocatedPorts)
		}
	}

	// Add repo path variables
	repos := cfg.GetRepos()
	for name, repo := range repos {
		envVarName := config.GenerateEnvVarName(name)
		repoPath := repo.GetRepoPath(projectDir)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVarName, repoPath))
	}

	// Add local config environment variables
	localEnvVars, err := GetLocalEnvVars(projectDir)
	if err == nil {
		for key, value := range localEnvVars {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return executeWithStreaming(cmd, opts.Output)
}

// runInSource executes a command in source mode against the project directory.
func runInSource(opts RunOptions, scriptPath string) (int, error) {
	projectDir := opts.ProjectDir
	cfg := opts.Config

	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = projectDir

	// Build environment variables (excluding feature-specific vars)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("RAMP_PROJECT_DIR=%s", projectDir),
	)

	// Add repo path variables
	repos := cfg.GetRepos()
	for name, repo := range repos {
		envVarName := config.GenerateEnvVarName(name)
		repoPath := repo.GetRepoPath(projectDir)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVarName, repoPath))
	}

	// Add local config environment variables
	localEnvVars, err := GetLocalEnvVars(projectDir)
	if err == nil {
		for key, value := range localEnvVars {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return executeWithStreaming(cmd, opts.Output)
}

// executeWithStreaming runs a command and streams output via OutputStreamer.
func executeWithStreaming(cmd *exec.Cmd, output OutputStreamer) (int, error) {
	// Set up pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return -1, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return -1, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("failed to start command: %w", err)
	}

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if output != nil {
				output.WriteLine(scanner.Text())
			}
		}
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			if output != nil {
				output.WriteErrorLine(scanner.Text())
			}
		}
	}()

	// Wait for command to complete
	err = cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return -1, err
	}

	return 0, nil
}

// addPortEnvVars adds port environment variables to a command.
func addPortEnvVars(cmd *exec.Cmd, ports []int) {
	if len(ports) == 0 {
		return
	}

	// Set RAMP_PORT to first port (backward compatibility)
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_PORT=%d", ports[0]))

	// Set indexed ports (RAMP_PORT_1, RAMP_PORT_2, etc.)
	for i, port := range ports {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_PORT_%d=%d", i+1, port))
	}
}
