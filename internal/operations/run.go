package operations

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"ramp/internal/config"
	"ramp/internal/hooks"
	"ramp/internal/ports"
)

// ErrCommandCancelled is returned when a command is cancelled
var ErrCommandCancelled = errors.New("command cancelled")

// RunOptions configures custom command execution.
type RunOptions struct {
	ProjectDir  string
	Config      *config.Config
	CommandName string
	FeatureName string           // Empty = run against source
	Args        []string         // Arguments to pass to the script
	Progress    ProgressReporter
	Output      OutputStreamer   // For streaming stdout/stderr

	// Cancel channel - when closed, the command will be killed
	Cancel <-chan struct{}

	// ProcessCallback is called after the process starts with the exec.Cmd and PGID
	// This allows the caller to track the process for cancellation
	ProcessCallback func(cmd *exec.Cmd, pgid int)
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

	// Load merged config to support commands from local and user configs
	mergedCfg, mergeErr := config.LoadMergedConfig(projectDir)

	// Find the command in configuration (try merged config first, fall back to project config)
	var command *config.Command
	if mergeErr == nil {
		command = mergedCfg.GetCommand(commandName)
	} else {
		command = cfg.GetCommand(commandName)
	}
	if command == nil {
		return nil, fmt.Errorf("command '%s' not found in configuration", commandName)
	}

	// Validate scope compatibility
	isSourceMode := featureName == ""
	if command.Scope == "source" && !isSourceMode {
		return nil, fmt.Errorf("command '%s' can only run against source repos", commandName)
	}
	if command.Scope == "feature" && isSourceMode {
		return nil, fmt.Errorf("command '%s' requires a feature name", commandName)
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
		// Don't show error message for intentional cancellation
		if !errors.Is(err, ErrCommandCancelled) {
			progress.Error(fmt.Sprintf("Command '%s' failed: %v", commandName, err))
		}
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
		}, fmt.Errorf("command '%s' failed: exited with code %d", commandName, exitCode)
	}

	// Execute run hooks (after command success)
	if mergeErr == nil && len(mergedCfg.Hooks) > 0 {
		repos := cfg.GetRepos()
		var allocatedPorts []int
		var treesDir, workDir string
		displayName := ""

		if featureName != "" {
			treesDir = filepath.Join(projectDir, "trees", featureName)
			workDir = treesDir
			displayName = LoadDisplayName(projectDir, featureName)
			if cfg.HasPortConfig() {
				portAllocations, portErr := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
				if portErr == nil {
					if p, exists := portAllocations.GetPorts(featureName); exists {
						allocatedPorts = p
					}
				}
			}
		} else {
			workDir = projectDir
		}

		hookEnv := BuildEnvVars(projectDir, treesDir, featureName, displayName, allocatedPorts, cfg, repos)
		hookEnv["RAMP_COMMAND_NAME"] = commandName
		hooks.ExecuteHooksForCommand(mergedCfg.Hooks, commandName, projectDir, workDir, hookEnv, progress)
	}

	progress.Complete(fmt.Sprintf("Command '%s' completed successfully", commandName))

	return &RunResult{
		CommandName: commandName,
		ExitCode:    0,
		Duration:    duration,
	}, nil
}

// buildBashCommand creates an exec.Cmd for running a script with login shell.
// Uses -l flag to source user's profile, ensuring tools like bun/node are available.
func buildBashCommand(scriptPath string, args []string, workDir string) *exec.Cmd {
	bashArgs := append([]string{"-l", scriptPath}, args...)
	cmd := exec.Command("/bin/bash", bashArgs...)
	cmd.Dir = workDir
	return cmd
}

// appendArgsEnv adds RAMP_ARGS to the environment if args are provided.
func appendArgsEnv(env []string, args []string) []string {
	if len(args) > 0 {
		return append(env, fmt.Sprintf("RAMP_ARGS=%s", strings.Join(args, " ")))
	}
	return env
}

// runInFeature executes a command in feature mode with feature-specific env vars.
func runInFeature(opts RunOptions, scriptPath, treesDir string) (int, error) {
	projectDir := opts.ProjectDir
	cfg := opts.Config
	featureName := opts.FeatureName
	displayName := LoadDisplayName(projectDir, featureName)

	// Get allocated ports
	var allocatedPorts []int
	portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
	if err == nil {
		if p, exists := portAllocations.GetPorts(featureName); exists {
			allocatedPorts = p
		}
	}

	cmd := buildBashCommand(scriptPath, opts.Args, treesDir)

	// Build environment variables using the standard builder, but override repo paths for worktrees
	repos := cfg.GetRepos()
	cmd.Env = BuildScriptEnv(projectDir, treesDir, featureName, displayName, allocatedPorts, cfg, repos)

	// Override repo paths to use worktree paths instead of source paths
	for name := range repos {
		envVarName := config.GenerateEnvVarName(name)
		repoPath := filepath.Join(treesDir, name)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVarName, repoPath))
	}

	cmd.Env = appendArgsEnv(cmd.Env, opts.Args)

	// Stop spinner before streaming output to avoid visual conflicts
	opts.Progress.Stop()

	return executeWithStreaming(cmd, opts.Output, opts.Cancel, opts.ProcessCallback)
}

// runInSource executes a command in source mode against the project directory.
func runInSource(opts RunOptions, scriptPath string) (int, error) {
	projectDir := opts.ProjectDir
	cfg := opts.Config

	cmd := buildBashCommand(scriptPath, opts.Args, projectDir)

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

	cmd.Env = appendArgsEnv(cmd.Env, opts.Args)

	// Stop spinner before streaming output to avoid visual conflicts
	opts.Progress.Stop()

	return executeWithStreaming(cmd, opts.Output, opts.Cancel, opts.ProcessCallback)
}

// executeWithStreaming runs a command and streams output via OutputStreamer.
// Supports cancellation via the cancel channel and process tracking via processCallback.
func executeWithStreaming(cmd *exec.Cmd, output OutputStreamer, cancel <-chan struct{}, processCallback func(*exec.Cmd, int)) (int, error) {
	// Create new process group so we can kill all child processes
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

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

	// Get process group ID (on macOS/Unix, this is the PID of the process leader)
	pgid := cmd.Process.Pid

	// Notify caller of process details for cancellation tracking
	if processCallback != nil {
		processCallback(cmd, pgid)
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

	// Wait for command in a goroutine
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- cmd.Wait()
	}()

	// Wait for completion or cancellation
	select {
	case err := <-resultCh:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitErr.ExitCode(), nil
			}
			return -1, err
		}
		return 0, nil

	case <-cancel:
		// Send SIGTERM first to allow graceful shutdown (trap handlers)
		// Negative PID sends signal to all processes in the process group
		syscall.Kill(-pgid, syscall.SIGTERM)

		// Wait up to 5 seconds for graceful termination
		select {
		case <-resultCh:
			// Process exited gracefully
		case <-time.After(5 * time.Second):
			// Force kill if still running
			syscall.Kill(-pgid, syscall.SIGKILL)
			<-resultCh
		}

		return -1, ErrCommandCancelled
	}
}
