package autoupdate

import (
	"strconv"
	"strings"
)

// IsNewer returns true if latest version is newer than current version.
// Handles semantic versioning (e.g., "1.2.3") and strips "v" prefix if present.
// Returns false if either version is invalid.
func IsNewer(latest, current string) bool {
	// Strip "v" prefix if present
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	// Parse versions
	latestParts, err := parseVersion(latest)
	if err != nil {
		return false
	}

	currentParts, err := parseVersion(current)
	if err != nil {
		return false
	}

	// Compare major, minor, patch in order
	for i := 0; i < 3; i++ {
		if latestParts[i] > currentParts[i] {
			return true
		}
		if latestParts[i] < currentParts[i] {
			return false
		}
	}

	// Versions are equal
	return false
}

// parseVersion parses a semantic version string into [major, minor, patch].
// Returns error if the version is not in valid format.
func parseVersion(version string) ([3]int, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return [3]int{}, strconv.ErrSyntax
	}

	var result [3]int
	for i, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return [3]int{}, err
		}
		result[i] = num
	}

	return result, nil
}
