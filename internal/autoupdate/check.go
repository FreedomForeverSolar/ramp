package autoupdate

import (
	"log"
	"os"
	"time"
)

// RunBackgroundCheck performs the update check and upgrade if needed.
// This is the main entry point for the background update process.
func RunBackgroundCheck(currentVersion string) {
	// Set up logging
	logFile, err := setupLogging()
	if err != nil {
		// Can't log, just exit silently
		return
	}
	defer logFile.Close()

	logf("Starting background update check (current version: %s)", currentVersion)

	// STEP 1: Acquire lock (prevents concurrent checks)
	lockPath := getLockPath()
	lock, err := AcquireLock(lockPath)
	if err != nil {
		logf("Lock held by another process, exiting")
		return
	}
	defer lock.Release()

	// STEP 2: Load settings
	settingsPath := getSettingsPath()
	settings, err := EnsureSettings(settingsPath)
	if err != nil {
		logf("Failed to load settings: %v", err)
		// Use default settings
		settings = defaultSettings()
	}

	// STEP 3: Check cache (rate limiting)
	cachePath := getCachePath()
	cache, err := LoadCache(cachePath)
	if err != nil {
		logf("Failed to load cache: %v", err)
		// Continue anyway
	}

	interval := settings.GetCheckIntervalOrDefault()
	if !ShouldCheck(cache, interval) {
		logf("Checked recently (%s ago), skipping", time.Since(cache.LastCheck))
		return
	}

	logf("Cache is stale or empty, proceeding with check")

	// STEP 4: Get brew info (tap and version)
	latestVersion, tap, err := GetBrewInfo()
	if err != nil {
		logf("Failed to get brew info: %v", err)
		// Update cache with current time to avoid hammering on repeated failures
		updateCache(cachePath, currentVersion, "", time.Now())
		return
	}

	logf("Latest version from brew: %s (tap: %s)", latestVersion, tap)

	// STEP 5: Update Homebrew tap to get latest formula
	logf("Updating Homebrew tap: %s", tap)
	if err := RunBrewUpdate(tap); err != nil {
		logf("Failed to update brew tap: %v", err)
		updateCache(cachePath, currentVersion, latestVersion, time.Now())
		return
	}

	// STEP 6: Get brew info again after tap update
	latestVersion, tap, err = GetBrewInfo()
	if err != nil {
		logf("Failed to get brew info after update: %v", err)
		updateCache(cachePath, currentVersion, "", time.Now())
		return
	}

	logf("Latest version after tap update: %s", latestVersion)

	// STEP 7: Compare versions
	if !IsNewer(latestVersion, currentVersion) {
		logf("Already on latest version: %s", currentVersion)
		updateCache(cachePath, currentVersion, latestVersion, time.Now())
		return
	}

	logf("Update available: %s → %s", currentVersion, latestVersion)

	// STEP 8: Run brew upgrade
	logf("Running brew upgrade ramp...")
	if err := RunBrewUpgrade(); err != nil {
		logf("Brew upgrade failed: %v", err)
		updateCache(cachePath, currentVersion, latestVersion, time.Now())
		return
	}

	logf("Successfully updated to %s", latestVersion)
	updateCache(cachePath, latestVersion, latestVersion, time.Now())
}

// setupLogging sets up logging to the update log file.
func setupLogging() (*os.File, error) {
	logPath := getLogPath()

	// Ensure directory exists
	dir, _ := getRampDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Open log file in append mode
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// Set log output to file
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime)

	return logFile, nil
}

// logf logs a message with timestamp.
func logf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// updateCache updates the cache file with new data.
func updateCache(cachePath, currentVersion, latestVersion string, lastCheck time.Time) {
	cache := UpdateCache{
		LastCheck:      lastCheck,
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
	}

	if err := SaveCache(cachePath, cache); err != nil {
		logf("Failed to save cache: %v", err)
	}
}
