package envfile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ramp/internal/config"
	"ramp/internal/ui"
)

// ProcessEnvFiles processes all env file configurations for a repository
// It copies files from source repo to worktree and performs variable replacements
func ProcessEnvFiles(repoName string, envFiles []config.EnvFile, sourceRepoDir string, worktreeDir string, envVars map[string]string) error {
	if len(envFiles) == 0 {
		return nil
	}

	for _, envFile := range envFiles {
		if err := processEnvFile(repoName, envFile, sourceRepoDir, worktreeDir, envVars); err != nil {
			return err
		}
	}

	return nil
}

// processEnvFile processes a single env file configuration
func processEnvFile(repoName string, envFile config.EnvFile, sourceRepoDir string, worktreeDir string, envVars map[string]string) error {
	// Resolve source path (relative to source repo directory)
	sourcePath := filepath.Join(sourceRepoDir, envFile.Source)

	// Check if source file exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		// Warn but don't fail
		ui.Warning(fmt.Sprintf("Source env file not found for %s: %s", repoName, sourcePath))
		return nil
	}

	// Read source file
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source env file %s: %w", sourcePath, err)
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
