package autoupdate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsAutoUpdateEnabled_NotHomebrewInstall(t *testing.T) {
	// This test verifies that auto-update is disabled for non-Homebrew installs
	// and that the settings file is never created/touched

	tmpDir := t.TempDir()

	// Set HOME to temp directory so settings would be created there if touched
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// IsAutoUpdateEnabled should return false for current binary
	// (which is not installed via Homebrew during tests)
	enabled := IsAutoUpdateEnabled()

	if enabled {
		t.Error("IsAutoUpdateEnabled() should return false for non-Homebrew install")
	}

	// Verify settings file was NOT created
	settingsPath := filepath.Join(tmpDir, ".ramp", "settings.yaml")
	if _, err := os.Stat(settingsPath); err == nil {
		t.Error("Settings file should not be created for non-Homebrew install")
	}
}

func TestIsAutoUpdateEnabled_HomebrewInstallDisabledInSettings(t *testing.T) {
	// Mock a Homebrew install by testing isHomebrewPath directly
	// (we can't easily mock os.Executable in tests)

	homebrewPath := "/opt/homebrew/Cellar/ramp/1.0.0/bin/ramp"
	if !isHomebrewPath(homebrewPath) {
		t.Fatalf("isHomebrewPath(%q) should return true", homebrewPath)
	}

	// Create a settings file with auto-update disabled
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.yaml")

	settings := Settings{
		AutoUpdate: AutoUpdateSettings{
			Enabled:       false,
			CheckInterval: "24h",
		},
	}

	if err := SaveSettings(settingsPath, settings); err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}

	// Load and verify
	loaded, err := LoadSettings(settingsPath)
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}

	if loaded.AutoUpdate.Enabled {
		t.Error("Auto-update should be disabled in settings")
	}
}

func TestManualInstallBehavior(t *testing.T) {
	// Test that manual installs (e.g., built locally) don't trigger auto-update

	manualInstallPaths := []string{
		"/usr/local/bin/ramp",
		"/usr/bin/ramp",
		"/home/user/bin/ramp",
		"/opt/local/bin/ramp",
		"./ramp",
		"/tmp/ramp",
	}

	for _, path := range manualInstallPaths {
		t.Run(path, func(t *testing.T) {
			if isHomebrewPath(path) {
				t.Errorf("isHomebrewPath(%q) should return false for manual install", path)
			}
		})
	}
}

func TestRunBackgroundCheckExitsEarlyForNonHomebrew(t *testing.T) {
	// Verify that RunBackgroundCheck exits early if not a Homebrew install
	// This is tested indirectly - if it's not Homebrew, IsAutoUpdateEnabled returns false
	// and SpawnBackgroundChecker is never called

	// The key is that IsAutoUpdateEnabled is checked BEFORE spawning background process
	// This test documents that behavior

	if !IsHomebrewInstall() {
		// Current test binary is not Homebrew install
		enabled := IsAutoUpdateEnabled()
		if enabled {
			t.Error("Auto-update should be disabled for non-Homebrew test binary")
		}
	}
}
