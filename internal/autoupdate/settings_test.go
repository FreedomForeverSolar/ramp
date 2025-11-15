package autoupdate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadSettings_FileDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.yaml")

	settings, err := LoadSettings(settingsPath)
	if err != nil {
		t.Fatalf("LoadSettings() should not error on missing file, got: %v", err)
	}

	// Should return default settings
	if !settings.AutoUpdate.Enabled {
		t.Error("Default AutoUpdate.Enabled should be true")
	}
	if settings.AutoUpdate.CheckInterval != "12h" {
		t.Errorf("Default CheckInterval = %q, want %q", settings.AutoUpdate.CheckInterval, "12h")
	}
}

func TestLoadSettings_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.yaml")

	// Create a valid settings file
	settingsYAML := `auto_update:
  enabled: false
  check_interval: 12h
`
	os.WriteFile(settingsPath, []byte(settingsYAML), 0644)

	settings, err := LoadSettings(settingsPath)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}

	if settings.AutoUpdate.Enabled {
		t.Error("AutoUpdate.Enabled should be false")
	}
	if settings.AutoUpdate.CheckInterval != "12h" {
		t.Errorf("CheckInterval = %q, want %q", settings.AutoUpdate.CheckInterval, "12h")
	}
}

func TestLoadSettings_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.yaml")

	// Write invalid YAML
	os.WriteFile(settingsPath, []byte("invalid: yaml: content:"), 0644)

	// Should return defaults on parse error
	settings, err := LoadSettings(settingsPath)
	if err != nil {
		t.Fatalf("LoadSettings() should not error on invalid YAML, got: %v", err)
	}

	// Should have defaults
	if !settings.AutoUpdate.Enabled {
		t.Error("Should fallback to default enabled=true on parse error")
	}
}

func TestSaveSettings(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.yaml")

	settings := Settings{
		AutoUpdate: AutoUpdateSettings{
			Enabled:       false,
			CheckInterval: "6h",
		},
	}

	err := SaveSettings(settingsPath, settings)
	if err != nil {
		t.Fatalf("SaveSettings() error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Fatal("Settings file was not created")
	}

	// Load it back and verify
	loaded, err := LoadSettings(settingsPath)
	if err != nil {
		t.Fatalf("LoadSettings() error after save: %v", err)
	}

	if loaded.AutoUpdate.Enabled {
		t.Error("AutoUpdate.Enabled should be false")
	}
	if loaded.AutoUpdate.CheckInterval != "6h" {
		t.Errorf("CheckInterval = %q, want %q", loaded.AutoUpdate.CheckInterval, "6h")
	}
}

func TestSaveSettings_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "nested", "dir", "settings.yaml")

	settings := Settings{
		AutoUpdate: AutoUpdateSettings{
			Enabled:       true,
			CheckInterval: "24h",
		},
	}

	err := SaveSettings(settingsPath, settings)
	if err != nil {
		t.Fatalf("SaveSettings() should create directories, got error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Fatal("Settings file was not created in nested directory")
	}
}

func TestGetSettingsPath(t *testing.T) {
	path := getSettingsPath()

	// Should end with settings.yaml
	if !strings.HasSuffix(path, "settings.yaml") {
		t.Errorf("getSettingsPath() = %q, should end with settings.yaml", path)
	}

	// Should contain .ramp
	if !strings.Contains(path, ".ramp") {
		t.Errorf("getSettingsPath() = %q, should contain .ramp", path)
	}
}

func TestGetSettingsCheckInterval(t *testing.T) {
	tests := []struct {
		name             string
		intervalString   string
		wantDuration     string
		wantErr          bool
	}{
		{
			name:           "valid 24h",
			intervalString: "24h",
			wantDuration:   "24h0m0s",
			wantErr:        false,
		},
		{
			name:           "valid 12h",
			intervalString: "12h",
			wantDuration:   "12h0m0s",
			wantErr:        false,
		},
		{
			name:           "valid 30m",
			intervalString: "30m",
			wantDuration:   "30m0s",
			wantErr:        false,
		},
		{
			name:           "invalid format",
			intervalString: "invalid",
			wantDuration:   "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := Settings{
				AutoUpdate: AutoUpdateSettings{
					CheckInterval: tt.intervalString,
				},
			}

			duration, err := settings.GetCheckInterval()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCheckInterval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && duration.String() != tt.wantDuration {
				t.Errorf("GetCheckInterval() = %v, want %v", duration, tt.wantDuration)
			}
		})
	}
}

func TestDefaultSettings(t *testing.T) {
	settings := defaultSettings()

	if !settings.AutoUpdate.Enabled {
		t.Error("Default settings should have AutoUpdate.Enabled = true")
	}
	if settings.AutoUpdate.CheckInterval != "12h" {
		t.Errorf("Default CheckInterval = %q, want %q", settings.AutoUpdate.CheckInterval, "12h")
	}
}

func TestEnsureSettings_CreatesFileIfMissing(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.yaml")

	// Ensure settings (should create file with defaults)
	settings, err := EnsureSettings(settingsPath)
	if err != nil {
		t.Fatalf("EnsureSettings() error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Fatal("Settings file was not created by EnsureSettings()")
	}

	// Verify defaults
	if !settings.AutoUpdate.Enabled {
		t.Error("Default settings should have AutoUpdate.Enabled = true")
	}
	if settings.AutoUpdate.CheckInterval != "12h" {
		t.Errorf("Default CheckInterval = %q, want %q", settings.AutoUpdate.CheckInterval, "12h")
	}
}

func TestEnsureSettings_UsesExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.yaml")

	// Create existing settings file
	settingsYAML := `auto_update:
  enabled: false
  check_interval: 6h
`
	os.WriteFile(settingsPath, []byte(settingsYAML), 0644)

	// Ensure settings (should use existing file)
	settings, err := EnsureSettings(settingsPath)
	if err != nil {
		t.Fatalf("EnsureSettings() error: %v", err)
	}

	// Should use existing values
	if settings.AutoUpdate.Enabled {
		t.Error("Should use existing enabled=false")
	}
	if settings.AutoUpdate.CheckInterval != "6h" {
		t.Errorf("Should use existing interval=6h, got %q", settings.AutoUpdate.CheckInterval)
	}
}

func TestSettingsYAMLFormat(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.yaml")

	settings := defaultSettings()
	err := SaveSettings(settingsPath, settings)
	if err != nil {
		t.Fatalf("SaveSettings() error: %v", err)
	}

	// Read the file and check format
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	contentStr := string(content)

	// Check for expected YAML structure
	if !strings.Contains(contentStr, "auto_update:") {
		t.Error("YAML should contain 'auto_update:' key")
	}
	if !strings.Contains(contentStr, "enabled:") {
		t.Error("YAML should contain 'enabled:' key")
	}
	if !strings.Contains(contentStr, "check_interval:") {
		t.Error("YAML should contain 'check_interval:' key")
	}
	if !strings.Contains(contentStr, "12h") {
		t.Error("YAML should contain default '12h' interval")
	}
}

func TestGetCheckIntervalOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		settings Settings
		want     time.Duration
	}{
		{
			name: "valid interval",
			settings: Settings{
				AutoUpdate: AutoUpdateSettings{
					CheckInterval: "12h",
				},
			},
			want: 12 * time.Hour,
		},
		{
			name: "invalid interval returns default",
			settings: Settings{
				AutoUpdate: AutoUpdateSettings{
					CheckInterval: "invalid",
				},
			},
			want: 12 * time.Hour,
		},
		{
			name: "empty interval returns default",
			settings: Settings{
				AutoUpdate: AutoUpdateSettings{
					CheckInterval: "",
				},
			},
			want: 12 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.settings.GetCheckIntervalOrDefault()
			if got != tt.want {
				t.Errorf("GetCheckIntervalOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}
