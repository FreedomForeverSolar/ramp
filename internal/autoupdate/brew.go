package autoupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BrewInfo represents the JSON output from `brew info --json=v2`
type BrewInfo struct {
	Formulae []struct {
		Name     string `json:"name"`
		Tap      string `json:"tap"`
		Versions struct {
			Stable string `json:"stable"`
		} `json:"versions"`
	} `json:"formulae"`
}

// isHomebrewPath checks if the given path is a Homebrew installation.
func isHomebrewPath(binaryPath string) bool {
	return strings.Contains(binaryPath, "/Cellar/ramp/")
}

// IsHomebrewInstall checks if the current ramp binary is installed via Homebrew.
func IsHomebrewInstall() bool {
	exePath, err := os.Executable()
	if err != nil {
		return false
	}

	// Resolve symlink to get actual path (e.g., /opt/homebrew/bin/ramp -> ../Cellar/ramp/1.3.3/bin/ramp)
	resolvedPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		// If we can't resolve, fall back to original path
		resolvedPath = exePath
	}

	return isHomebrewPath(resolvedPath)
}

// IsAutoUpdateEnabled checks if auto-update should be enabled.
// Checks settings file and Homebrew install status.
func IsAutoUpdateEnabled() bool {
	// Only enable for Homebrew installs
	if !IsHomebrewInstall() {
		return false
	}

	// Load settings to check if enabled
	settingsPath := getSettingsPath()
	settings, err := EnsureSettings(settingsPath)
	if err != nil {
		// If we can't load settings, default to enabled
		return true
	}

	return settings.AutoUpdate.Enabled
}

// parseBrewInfo parses the JSON output from `brew info --json=v2`.
func parseBrewInfo(jsonData []byte) (version string, tap string, err error) {
	var info BrewInfo
	if err := json.Unmarshal(jsonData, &info); err != nil {
		return "", "", fmt.Errorf("failed to parse brew info: %w", err)
	}

	if len(info.Formulae) == 0 {
		return "", "", fmt.Errorf("no formulae found in brew info output")
	}

	formula := info.Formulae[0]
	return formula.Versions.Stable, formula.Tap, nil
}

// GetBrewInfo gets the latest version and tap name from Homebrew.
func GetBrewInfo() (version string, tap string, err error) {
	cmd := exec.Command("brew", "info", "ramp", "--json=v2")
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("brew info failed: %w", err)
	}

	return parseBrewInfo(output)
}

// RunBrewUpdate updates all Homebrew taps to get the latest formulas.
func RunBrewUpdate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "brew", "update")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("brew update failed: %w\n%s", err, output)
	}

	return nil
}

// RunBrewUpgrade upgrades the ramp package.
func RunBrewUpgrade() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "brew", "upgrade", "ramp")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("brew upgrade failed: %w\n%s", err, output)
	}

	return nil
}

// getRampDir returns the ~/.ramp directory path.
func getRampDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ramp"), nil
}

// getCachePath returns the path to the update check cache file.
func getCachePath() string {
	dir, _ := getRampDir()
	return filepath.Join(dir, "update_check.json")
}

// getLockPath returns the path to the update lock file.
func getLockPath() string {
	dir, _ := getRampDir()
	return filepath.Join(dir, "update.lock")
}

// getLogPath returns the path to the update log file.
func getLogPath() string {
	dir, _ := getRampDir()
	return filepath.Join(dir, "update.log")
}
