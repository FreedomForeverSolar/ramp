package autoupdate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCache_FileDoesNotExist(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update_check.json")

	cache, err := LoadCache(cachePath)
	if err != nil {
		t.Fatalf("LoadCache() should not error on missing file, got: %v", err)
	}

	// Should return empty cache with zero time
	if !cache.LastCheck.IsZero() {
		t.Errorf("LastCheck should be zero time, got: %v", cache.LastCheck)
	}
	if cache.CurrentVersion != "" {
		t.Errorf("CurrentVersion should be empty, got: %q", cache.CurrentVersion)
	}
	if cache.LatestVersion != "" {
		t.Errorf("LatestVersion should be empty, got: %q", cache.LatestVersion)
	}
}

func TestLoadCache_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update_check.json")

	// Create a valid cache file
	testTime := time.Date(2025, 11, 15, 10, 30, 0, 0, time.UTC)
	testCache := UpdateCache{
		LastCheck:      testTime,
		CurrentVersion: "1.2.3",
		LatestVersion:  "1.2.4",
	}

	data, _ := json.Marshal(testCache)
	os.WriteFile(cachePath, data, 0644)

	// Load it
	cache, err := LoadCache(cachePath)
	if err != nil {
		t.Fatalf("LoadCache() error: %v", err)
	}

	if !cache.LastCheck.Equal(testTime) {
		t.Errorf("LastCheck = %v, want %v", cache.LastCheck, testTime)
	}
	if cache.CurrentVersion != "1.2.3" {
		t.Errorf("CurrentVersion = %q, want %q", cache.CurrentVersion, "1.2.3")
	}
	if cache.LatestVersion != "1.2.4" {
		t.Errorf("LatestVersion = %q, want %q", cache.LatestVersion, "1.2.4")
	}
}

func TestLoadCache_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update_check.json")

	// Write invalid JSON
	os.WriteFile(cachePath, []byte("invalid json"), 0644)

	// Should return empty cache on parse error
	cache, err := LoadCache(cachePath)
	if err != nil {
		t.Fatalf("LoadCache() should not error on invalid JSON, got: %v", err)
	}

	if !cache.LastCheck.IsZero() {
		t.Errorf("LastCheck should be zero time on parse error")
	}
}

func TestSaveCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update_check.json")

	testTime := time.Date(2025, 11, 15, 10, 30, 0, 0, time.UTC)
	testCache := UpdateCache{
		LastCheck:      testTime,
		CurrentVersion: "1.2.3",
		LatestVersion:  "1.2.4",
	}

	err := SaveCache(cachePath, testCache)
	if err != nil {
		t.Fatalf("SaveCache() error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache file was not created")
	}

	// Load it back and verify
	cache, err := LoadCache(cachePath)
	if err != nil {
		t.Fatalf("LoadCache() error after save: %v", err)
	}

	if !cache.LastCheck.Equal(testTime) {
		t.Errorf("LastCheck = %v, want %v", cache.LastCheck, testTime)
	}
	if cache.CurrentVersion != "1.2.3" {
		t.Errorf("CurrentVersion = %q, want %q", cache.CurrentVersion, "1.2.3")
	}
	if cache.LatestVersion != "1.2.4" {
		t.Errorf("LatestVersion = %q, want %q", cache.LatestVersion, "1.2.4")
	}
}

func TestSaveCache_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a nested path that doesn't exist
	cachePath := filepath.Join(tmpDir, "nested", "dir", "update_check.json")

	testCache := UpdateCache{
		LastCheck:      time.Now(),
		CurrentVersion: "1.0.0",
		LatestVersion:  "1.0.1",
	}

	err := SaveCache(cachePath, testCache)
	if err != nil {
		t.Fatalf("SaveCache() should create directories, got error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache file was not created in nested directory")
	}
}

func TestShouldCheck_NeverChecked(t *testing.T) {
	cache := UpdateCache{
		LastCheck: time.Time{}, // Zero time
	}

	interval := 24 * time.Hour
	if !ShouldCheck(cache, interval) {
		t.Error("ShouldCheck() should return true when never checked")
	}
}

func TestShouldCheck_RecentCheck(t *testing.T) {
	cache := UpdateCache{
		LastCheck: time.Now().Add(-1 * time.Hour), // 1 hour ago
	}

	interval := 24 * time.Hour
	if ShouldCheck(cache, interval) {
		t.Error("ShouldCheck() should return false when checked recently")
	}
}

func TestShouldCheck_StaleCheck(t *testing.T) {
	cache := UpdateCache{
		LastCheck: time.Now().Add(-25 * time.Hour), // 25 hours ago
	}

	interval := 24 * time.Hour
	if !ShouldCheck(cache, interval) {
		t.Error("ShouldCheck() should return true when check is stale")
	}
}

func TestShouldCheck_ExactInterval(t *testing.T) {
	cache := UpdateCache{
		LastCheck: time.Now().Add(-24 * time.Hour), // Exactly 24 hours ago
	}

	interval := 24 * time.Hour
	// At exactly the interval, should check (using >= comparison)
	if !ShouldCheck(cache, interval) {
		t.Error("ShouldCheck() should return true at exact interval boundary")
	}
}
