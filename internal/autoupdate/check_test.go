package autoupdate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update_check.json")

	testTime := time.Date(2025, 11, 15, 10, 30, 0, 0, time.UTC)

	// Update cache
	updateCache(cachePath, "1.2.3", "1.2.4", testTime)

	// Verify it was saved
	cache, err := LoadCache(cachePath)
	if err != nil {
		t.Fatalf("LoadCache() error: %v", err)
	}

	if cache.CurrentVersion != "1.2.3" {
		t.Errorf("CurrentVersion = %q, want %q", cache.CurrentVersion, "1.2.3")
	}
	if cache.LatestVersion != "1.2.4" {
		t.Errorf("LatestVersion = %q, want %q", cache.LatestVersion, "1.2.4")
	}
	if !cache.LastCheck.Equal(testTime) {
		t.Errorf("LastCheck = %v, want %v", cache.LastCheck, testTime)
	}
}

func TestSetupLogging(t *testing.T) {
	// This will use ~/.ramp/update.log
	// Just verify it doesn't error
	logFile, err := setupLogging()
	if err != nil {
		t.Fatalf("setupLogging() error: %v", err)
	}
	defer logFile.Close()

	// Verify file was created
	logPath := getLogPath()
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	// Verify we can write to it
	logf("Test log message")
}

// Note: RunBackgroundCheck is difficult to unit test because it orchestrates
// multiple external commands (brew). It should be tested via integration tests
// or manual testing with real Homebrew.
