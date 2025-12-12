// Package shellenv provides utilities for loading the user's shell environment.
// This is necessary when running in GUI environments (like Electron apps launched
// from Finder) that don't inherit the user's shell configuration.
package shellenv

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
)

// LoadShellEnv loads environment variables from the user's login shell.
// This ensures tools installed via Homebrew, nvm, etc. are available.
// Call this early in main() before any exec.Command calls.
func LoadShellEnv() error {
	// Only needed on macOS/Linux - Windows has different PATH handling
	if runtime.GOOS == "windows" {
		return nil
	}

	// Check if we already have a reasonable PATH (e.g., running from terminal)
	// Look for common indicators that we have a full shell environment
	currentPath := os.Getenv("PATH")
	if strings.Contains(currentPath, "/opt/homebrew") ||
		strings.Contains(currentPath, "/usr/local/bin") {
		return nil
	}

	shell := getUserShell()
	env, err := getShellEnvironment(shell)
	if err != nil {
		return fmt.Errorf("failed to load environment from %s: %w", shell, err)
	}

	// Update the current process environment
	for key, value := range env {
		os.Setenv(key, value)
	}

	return nil
}

// getUserShell returns the user's preferred shell.
func getUserShell() string {
	// First try SHELL environment variable
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	// Try to get from user info (reads /etc/passwd on Unix)
	if u, err := user.Current(); err == nil {
		// On macOS, we can check dscl for the shell
		if runtime.GOOS == "darwin" {
			cmd := exec.Command("dscl", ".", "-read", "/Users/"+u.Username, "UserShell")
			if output, err := cmd.Output(); err == nil {
				lines := strings.Split(string(output), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "UserShell:") {
						shell := strings.TrimSpace(strings.TrimPrefix(line, "UserShell:"))
						if shell != "" {
							return shell
						}
					}
				}
			}
		}
	}

	// Default to zsh on macOS (default since Catalina), bash elsewhere
	if runtime.GOOS == "darwin" {
		return "/bin/zsh"
	}
	return "/bin/bash"
}

// getShellEnvironment executes the user's shell and captures the environment.
func getShellEnvironment(shell string) (map[string]string, error) {
	// Use login (-l) and interactive (-i) flags to source profile files
	// Then run 'env' to print all environment variables
	cmd := exec.Command(shell, "-l", "-i", "-c", "env")

	// Set a minimal environment for the shell to start
	cmd.Env = []string{
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		"TERM=xterm-256color",
	}

	// Capture stdout, ignore stderr (shell startup messages)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute shell: %w", err)
	}

	// Parse the environment output
	env := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "="); idx > 0 {
			key := line[:idx]
			value := line[idx+1:]
			env[key] = value
		}
	}

	return env, nil
}

