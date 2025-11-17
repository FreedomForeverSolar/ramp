package autoupdate

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Settings represents the user's ramp settings.
type Settings struct {
	AutoUpdate AutoUpdateSettings `yaml:"auto_update"`
}

// AutoUpdateSettings contains auto-update configuration.
type AutoUpdateSettings struct {
	Enabled       bool   `yaml:"enabled"`
	CheckInterval string `yaml:"check_interval"`
}

// defaultSettings returns the default settings.
func defaultSettings() Settings {
	return Settings{
		AutoUpdate: AutoUpdateSettings{
			Enabled:       true,
			CheckInterval: "12h",
		},
	}
}

// LoadSettings loads settings from the specified path.
// Returns default settings if the file doesn't exist or can't be parsed.
func LoadSettings(settingsPath string) (Settings, error) {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		// File doesn't exist - return defaults
		return defaultSettings(), nil
	}

	var settings Settings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		// Invalid YAML - return defaults
		return defaultSettings(), nil
	}

	return settings, nil
}

// SaveSettings saves settings to the specified path.
// Creates parent directories if they don't exist.
func SaveSettings(settingsPath string, settings Settings) error {
	// Ensure parent directory exists
	dir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, data, 0644)
}

// EnsureSettings loads settings, creating the file with defaults if it doesn't exist.
func EnsureSettings(settingsPath string) (Settings, error) {
	// Check if file exists
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		// Create with defaults
		defaults := defaultSettings()
		if err := SaveSettings(settingsPath, defaults); err != nil {
			return defaults, err
		}
		return defaults, nil
	}

	// Load existing settings
	return LoadSettings(settingsPath)
}

// getSettingsPath returns the path to the settings file.
func getSettingsPath() string {
	dir, _ := getRampDir()
	return filepath.Join(dir, "settings.yaml")
}

// GetCheckInterval parses and returns the check interval duration.
// Returns error if the interval string is invalid.
func (s *Settings) GetCheckInterval() (time.Duration, error) {
	return time.ParseDuration(s.AutoUpdate.CheckInterval)
}

// GetCheckIntervalOrDefault returns the check interval or 12h if invalid.
func (s *Settings) GetCheckIntervalOrDefault() time.Duration {
	duration, err := s.GetCheckInterval()
	if err != nil {
		return 12 * time.Hour
	}
	return duration
}
