package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ramp/internal/config"
)

// HookEvent represents the lifecycle event for a hook.
type HookEvent string

const (
	Up   HookEvent = "up"   // Runs after feature creation
	Down HookEvent = "down" // Runs before feature deletion
	Run  HookEvent = "run"  // Runs after command execution
)

// ProgressReporter is the interface for reporting hook execution progress.
// This matches the operations.ProgressReporter interface.
type ProgressReporter interface {
	Info(message string)
	Warning(message string)
}

// ExecuteHooks runs all hooks for a given event.
// Hooks that fail are warned about but don't stop execution.
func ExecuteHooks(
	event HookEvent,
	hooks []*config.Hook,
	projectDir string,
	workDir string,
	env map[string]string,
	progress ProgressReporter,
) {
	matchingHooks := filterHooksByEvent(hooks, event)
	if len(matchingHooks) == 0 {
		return
	}

	for _, hook := range matchingHooks {
		err := runHook(hook, projectDir, workDir, env)
		if err != nil {
			progress.Warning(fmt.Sprintf("Hook '%s' (%s) failed: %v", hook.Command, event, err))
		} else {
			progress.Info(fmt.Sprintf("Hook '%s' completed", hook.Command))
		}
	}
}

// ExecuteHooksForCommand runs all hooks for the 'run' event that match the command.
// Hooks are filtered by the 'for' field: empty matches all, exact match, or prefix pattern.
func ExecuteHooksForCommand(
	hooks []*config.Hook,
	commandName string,
	projectDir string,
	workDir string,
	env map[string]string,
	progress ProgressReporter,
) {
	runHooks := filterHooksByEvent(hooks, Run)
	if len(runHooks) == 0 {
		return
	}

	for _, hook := range runHooks {
		if !matchesCommand(hook, commandName) {
			continue
		}

		err := runHook(hook, projectDir, workDir, env)
		if err != nil {
			progress.Warning(fmt.Sprintf("Hook '%s' (run:%s) failed: %v", hook.Command, commandName, err))
		} else {
			progress.Info(fmt.Sprintf("Hook '%s' completed", hook.Command))
		}
	}
}

// filterHooksByEvent returns hooks matching the given event.
func filterHooksByEvent(hooks []*config.Hook, event HookEvent) []*config.Hook {
	result := make([]*config.Hook, 0)
	for _, hook := range hooks {
		if hook.Event == string(event) {
			result = append(result, hook)
		}
	}
	return result
}

// matchesCommand checks if a hook matches the given command name.
// Empty 'For' field matches all commands.
// Exact match or prefix pattern (e.g., "test-*") supported.
func matchesCommand(hook *config.Hook, commandName string) bool {
	if hook.For == "" {
		return true // No filter = match all
	}
	if strings.HasSuffix(hook.For, "*") {
		prefix := strings.TrimSuffix(hook.For, "*")
		return strings.HasPrefix(commandName, prefix)
	}
	return hook.For == commandName // Exact match
}

// runHook executes a single hook script.
func runHook(
	hook *config.Hook,
	projectDir string,
	workDir string,
	env map[string]string,
) error {
	// Support both absolute paths (for user-level hooks) and relative paths
	var scriptPath string
	if filepath.IsAbs(hook.Command) {
		scriptPath = hook.Command
	} else {
		scriptPath = filepath.Join(projectDir, ".ramp", hook.Command)
	}

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("hook script not found: %s", scriptPath)
	}

	// Use login shell (-l) for consistent environment
	cmd := exec.Command("/bin/bash", "-l", scriptPath)
	cmd.Dir = workDir

	// Build environment
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Capture output but don't display (hooks should be silent unless they fail)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return fmt.Errorf("%w: %s", err, string(output))
		}
		return err
	}

	return nil
}

// ValidateHookEvent checks if an event name is valid.
func ValidateHookEvent(event string) error {
	switch event {
	case "up", "down", "run":
		return nil
	default:
		return fmt.Errorf("invalid hook event: %s (valid: up, down, run)", event)
	}
}
