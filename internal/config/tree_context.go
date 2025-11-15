package config

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectFeatureFromWorkingDir detects if the current working directory is inside
// a feature's trees directory and returns the feature name if found.
//
// Returns:
//   - feature name if current directory is under <projectDir>/trees/<feature-name>/
//   - empty string if not in a trees directory
//   - error only for actual failures (not for "not found" cases)
//
// Examples:
//   - /path/to/project/trees/my-feature/repo/src/ -> "my-feature"
//   - /path/to/project/trees/my-feature/ -> "my-feature"
//   - /path/to/project/repos/repo/ -> ""
//   - /path/to/project/ -> ""
func DetectFeatureFromWorkingDir(projectDir string) (string, error) {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Convert both to absolute paths and resolve symlinks for comparison
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		// If we can't resolve the project dir, we're probably not in it
		return "", nil
	}

	// Resolve symlinks to handle macOS /var/folders -> /private/var/folders
	absProjectDir, err = filepath.EvalSymlinks(absProjectDir)
	if err != nil {
		// If we can't evaluate symlinks, we're probably not in it
		return "", nil
	}

	absWd, err := filepath.Abs(wd)
	if err != nil {
		return "", err
	}

	// Resolve symlinks for working directory too
	absWd, err = filepath.EvalSymlinks(absWd)
	if err != nil {
		return "", err
	}

	// Check if working directory is under project directory
	if !strings.HasPrefix(absWd, absProjectDir) {
		// Not in the project directory at all
		return "", nil
	}

	// Get the relative path from project dir to working dir
	relPath, err := filepath.Rel(absProjectDir, absWd)
	if err != nil {
		return "", nil
	}

	// Split the path into components
	parts := strings.Split(relPath, string(filepath.Separator))

	// Check if path starts with "trees"
	if len(parts) < 2 || parts[0] != "trees" {
		// Not in trees directory
		return "", nil
	}

	// The feature name is the first directory under trees/
	featureName := parts[1]

	// Return empty string if we're in the trees directory itself
	if featureName == "" || featureName == "." {
		return "", nil
	}

	return featureName, nil
}
