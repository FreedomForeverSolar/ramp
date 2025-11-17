package autoupdate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// UpdateCache represents the cached update check information.
type UpdateCache struct {
	LastCheck      time.Time `json:"last_check"`
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version"`
}

// LoadCache loads the update cache from the specified path.
// Returns an empty cache if the file doesn't exist or can't be parsed.
func LoadCache(cachePath string) (UpdateCache, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		// File doesn't exist or can't be read - return empty cache
		return UpdateCache{}, nil
	}

	var cache UpdateCache
	if err := json.Unmarshal(data, &cache); err != nil {
		// Invalid JSON - return empty cache
		return UpdateCache{}, nil
	}

	return cache, nil
}

// SaveCache saves the update cache to the specified path.
// Creates parent directories if they don't exist.
func SaveCache(cachePath string, cache UpdateCache) error {
	// Ensure parent directory exists
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

// ShouldCheck returns true if enough time has elapsed since the last check.
func ShouldCheck(cache UpdateCache, interval time.Duration) bool {
	if cache.LastCheck.IsZero() {
		return true // Never checked before
	}

	elapsed := time.Since(cache.LastCheck)
	return elapsed >= interval
}
