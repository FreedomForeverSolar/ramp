package scaffold

import (
	"fmt"
	"strings"

	"ramp/internal/config"
)

// setupScriptTemplate provides a sample setup script showing available environment variables
func setupScriptTemplate(repos []RepoData) string {
	var repoVars strings.Builder
	for _, repo := range repos {
		repoName := extractRepoName(repo.GitURL)
		envVar := config.GenerateEnvVarName(repoName)
		repoVars.WriteString(fmt.Sprintf("#   %-20s - Path to %s repository\n", envVar, repoName))
	}

	return `#!/bin/bash

# Ramp Setup Script
# This script runs after creating a new feature branch with 'ramp up'
# Available environment variables:
#   RAMP_PROJECT_DIR     - Absolute path to project root
#   RAMP_TREES_DIR       - Absolute path to feature's trees directory
#   RAMP_WORKTREE_NAME   - Name of the feature
#   RAMP_PORT            - Allocated port number (if port management enabled)
` + repoVars.String() + `
set -e

echo "🚀 Setting up feature: $RAMP_WORKTREE_NAME"
echo "   Port: $RAMP_PORT"
echo "   Trees: $RAMP_TREES_DIR"

# Add your setup logic here
# Examples:
#   - Install dependencies
#   - Create environment files
#   - Start development servers
#   - Initialize databases

echo "✅ Setup complete!"
`
}

// cleanupScriptTemplate provides a sample cleanup script
func cleanupScriptTemplate(repos []RepoData) string {
	var repoVars strings.Builder
	for _, repo := range repos {
		repoName := extractRepoName(repo.GitURL)
		envVar := config.GenerateEnvVarName(repoName)
		repoVars.WriteString(fmt.Sprintf("#   %-20s - Path to %s repository\n", envVar, repoName))
	}

	return `#!/bin/bash

# Ramp Cleanup Script
# This script runs before tearing down a feature branch with 'ramp down'
# Available environment variables:
#   RAMP_PROJECT_DIR     - Absolute path to project root
#   RAMP_TREES_DIR       - Absolute path to feature's trees directory
#   RAMP_WORKTREE_NAME   - Name of the feature
#   RAMP_PORT            - Allocated port number (if port management enabled)
` + repoVars.String() + `
set -e

echo "🧹 Cleaning up feature: $RAMP_WORKTREE_NAME"

# Add your cleanup logic here
# Examples:
#   - Stop development servers
#   - Remove temporary files
#   - Clean up databases
#   - Remove environment files

echo "✅ Cleanup complete!"
`
}

// sampleCommandTemplate provides a template for custom commands
func sampleCommandTemplate(commandName string, repos []RepoData) string {
	var repoVars strings.Builder
	for _, repo := range repos {
		repoName := extractRepoName(repo.GitURL)
		envVar := config.GenerateEnvVarName(repoName)
		repoVars.WriteString(fmt.Sprintf("#   %-20s - Path to %s repository\n", envVar, repoName))
	}

	return `#!/bin/bash

# Ramp Custom Command: ` + commandName + `
# Run with: ramp run ` + commandName + ` [feature-name]
# Available environment variables:
#   RAMP_PROJECT_DIR     - Absolute path to project root
#   RAMP_TREES_DIR       - Absolute path to feature's trees directory
#   RAMP_WORKTREE_NAME   - Name of the feature
#   RAMP_PORT            - Allocated port number (if port management enabled)
` + repoVars.String() + `
set -e

echo "🚀 Running ` + commandName + ` for $RAMP_WORKTREE_NAME"

# Add your command logic here

echo "✅ Command complete!"
`
}

// extractRepoName extracts the repository name from a git URL
func extractRepoName(gitURL string) string {
	// Handle git@github.com:owner/repo.git format
	if strings.Contains(gitURL, ":") {
		parts := strings.Split(gitURL, ":")
		if len(parts) > 1 {
			gitURL = parts[1]
		}
	}

	// Remove .git suffix
	gitURL = strings.TrimSuffix(gitURL, ".git")

	// Extract repo name from owner/repo format
	parts := strings.Split(gitURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return gitURL
}
