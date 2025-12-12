package envfile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"ramp/internal/config"
	"ramp/internal/ui"
)

// ProcessEnvFiles processes all env file configurations for a repository
// It copies files from source repo to worktree and performs variable replacements
// shouldRefresh controls whether to bypass cache for script execution
func ProcessEnvFiles(repoName string, envFiles []config.EnvFile, sourceRepoDir string, worktreeDir string, envVars map[string]string, shouldRefresh bool) error {
	if len(envFiles) == 0 {
		return nil
	}

	// Determine project directory from sourceRepoDir (assume it's projectDir/repos/repoName)
	projectDir := filepath.Join(sourceRepoDir, "..", "..")

	return ProcessEnvFilesWithProjectDir(repoName, envFiles, sourceRepoDir, worktreeDir, envVars, shouldRefresh, projectDir)
}

// ProcessEnvFilesWithProjectDir is the internal version that accepts an explicit projectDir
// This is useful for testing
func ProcessEnvFilesWithProjectDir(repoName string, envFiles []config.EnvFile, sourceRepoDir string, worktreeDir string, envVars map[string]string, shouldRefresh bool, projectDir string) error {
	for _, envFile := range envFiles {
		if err := processEnvFile(repoName, envFile, sourceRepoDir, worktreeDir, envVars, shouldRefresh, projectDir); err != nil {
			return err
		}
	}

	return nil
}

// processEnvFile processes a single env file configuration
func processEnvFile(repoName string, envFile config.EnvFile, sourceRepoDir string, worktreeDir string, envVars map[string]string, shouldRefresh bool, projectDir string) error {
	// Resolve source path (relative to source repo directory)
	sourcePath := filepath.Join(sourceRepoDir, envFile.Source)

	// Get content from either file or script
	content, err := getContent(sourcePath, envFile.Cache, envVars, shouldRefresh, projectDir)
	if err != nil {
		// Check if it's a missing file
		if os.IsNotExist(err) {
			ui.Warning(fmt.Sprintf("Source env file not found for %s: %s", repoName, sourcePath))
			return nil
		}
		return err
	}

	contentStr := string(content)

	// Perform variable replacements
	if envFile.Replace != nil && len(envFile.Replace) > 0 {
		// Explicit replacements: only replace specified keys
		contentStr = replaceExplicitKeys(contentStr, envFile.Replace, envVars)
	} else {
		// Auto-replace: replace all ${RAMP_*} variables
		contentStr = replaceEnvVars(contentStr, envVars)
	}

	// Resolve destination path (relative to worktree directory)
	destPath := filepath.Join(worktreeDir, envFile.Dest)

	// Create parent directory if needed
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	// Write destination file
	if err := os.WriteFile(destPath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("failed to write destination env file %s: %w", destPath, err)
	}

	return nil
}

// getContent retrieves content from either a regular file or an executable script
func getContent(sourcePath string, cacheTTL string, envVars map[string]string, shouldRefresh bool, projectDir string) ([]byte, error) {
	// Check if source exists
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, err
	}

	// Check if it's executable
	if isExecutable(info) {
		return executeScript(sourcePath, cacheTTL, envVars, shouldRefresh, projectDir)
	}

	// Regular file - read it
	return os.ReadFile(sourcePath)
}

// isExecutable checks if a file has execute permissions
func isExecutable(info os.FileInfo) bool {
	return info.Mode()&0111 != 0
}

// executeScript runs an executable script and returns its output
// Handles caching if cacheTTL is set
func executeScript(scriptPath string, cacheTTL string, envVars map[string]string, shouldRefresh bool, projectDir string) ([]byte, error) {
	// Check cache if TTL is specified and refresh is not forced
	if cacheTTL != "" && !shouldRefresh {
		cacheContent, cacheHit := checkCache(scriptPath, cacheTTL, projectDir)
		if cacheHit {
			return cacheContent, nil
		}
	}

	// Execute the script through a login shell to ensure user's PATH is available
	// This ensures tools like bun, node, etc. are available in GUI environments
	cmd := exec.Command("/bin/bash", "-l", scriptPath)

	// Set environment variables
	cmd.Env = buildScriptEnv(envVars)

	// Capture output
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("script %s failed with exit code %d: %s", scriptPath, exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute script %s: %w", scriptPath, err)
	}

	// Cache output if TTL is specified
	if cacheTTL != "" {
		if err := cacheOutput(scriptPath, output, projectDir); err != nil {
			// Log warning but don't fail
			ui.Warning(fmt.Sprintf("Failed to cache script output: %v", err))
		}
	}

	return output, nil
}

// checkCache checks if a valid cache exists for a script
// Returns (content, true) if cache hit, (nil, false) if cache miss
func checkCache(scriptPath string, cacheTTL string, projectDir string) ([]byte, bool) {
	cachePath := getCachePath(scriptPath, projectDir)

	// Check if cache file exists
	info, err := os.Stat(cachePath)
	if err != nil {
		return nil, false
	}

	// Parse TTL duration
	duration, err := time.ParseDuration(cacheTTL)
	if err != nil {
		// Invalid TTL format, treat as no cache
		return nil, false
	}

	// Check if cache is still valid
	if time.Since(info.ModTime()) > duration {
		// Cache expired
		return nil, false
	}

	// Read cache content
	content, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}

	return content, true
}

// cacheOutput saves script output to cache
func cacheOutput(scriptPath string, output []byte, projectDir string) error {
	cachePath := getCachePath(scriptPath, projectDir)

	// Create cache directory
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	// Write cache file
	return os.WriteFile(cachePath, output, 0644)
}

// getCachePath generates a cache file path for a script
func getCachePath(scriptPath string, projectDir string) string {
	// Generate a hash of the script path for the cache key
	hash := sha256.Sum256([]byte(scriptPath))
	cacheKey := hex.EncodeToString(hash[:8])

	return filepath.Join(projectDir, ".ramp", "cache", "env_files", cacheKey+".cache")
}

// buildScriptEnv builds the environment variable array for script execution
func buildScriptEnv(envVars map[string]string) []string {
	// Start with current environment
	env := os.Environ()

	// Add/override with RAMP variables
	for key, value := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

// replaceExplicitKeys replaces only the specified keys with their replacement values
// Keys are matched as "KEY=..." patterns at the start of lines
func replaceExplicitKeys(content string, replacements map[string]string, envVars map[string]string) string {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		// Skip empty lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check if this line starts with one of our replacement keys
		for key, replacementValue := range replacements {
			if strings.HasPrefix(trimmed, key+"=") {
				// Replace the value after the =
				// First expand any ${RAMP_*} variables in the replacement value
				expandedValue := replaceEnvVars(replacementValue, envVars)
				lines[i] = key + "=" + expandedValue
				break
			}
		}
	}

	return strings.Join(lines, "\n")
}

// replaceEnvVars replaces all ${VARIABLE_NAME} patterns with their values from envVars
// Only replaces variables that are defined in envVars
func replaceEnvVars(content string, envVars map[string]string) string {
	// Pattern to match ${VARIABLE_NAME}
	re := regexp.MustCompile(`\$\{([A-Za-z0-9_]+)\}`)

	result := re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract variable name (remove ${ and })
		varName := match[2 : len(match)-1]

		// Look up in envVars
		if value, exists := envVars[varName]; exists {
			return value
		}

		// If not found, leave unchanged
		return match
	})

	return result
}
