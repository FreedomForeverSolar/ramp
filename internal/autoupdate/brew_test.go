package autoupdate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsHomebrewInstall(t *testing.T) {
	tests := []struct {
		name       string
		binaryPath string
		want       bool
	}{
		{
			name:       "installed via Homebrew on Intel Mac",
			binaryPath: "/usr/local/Cellar/ramp/1.2.3/bin/ramp",
			want:       true,
		},
		{
			name:       "installed via Homebrew on Apple Silicon",
			binaryPath: "/opt/homebrew/Cellar/ramp/1.2.3/bin/ramp",
			want:       true,
		},
		{
			name:       "installed via Homebrew in custom tap",
			binaryPath: "/opt/homebrew/Cellar/ramp/1.0.0/bin/ramp",
			want:       true,
		},
		{
			name:       "installed manually in /usr/local/bin",
			binaryPath: "/usr/local/bin/ramp",
			want:       false,
		},
		{
			name:       "installed in home directory",
			binaryPath: "/home/user/bin/ramp",
			want:       false,
		},
		{
			name:       "installed in /usr/bin",
			binaryPath: "/usr/bin/ramp",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHomebrewPath(tt.binaryPath)
			if got != tt.want {
				t.Errorf("isHomebrewPath(%q) = %v, want %v", tt.binaryPath, got, tt.want)
			}
		})
	}
}

func TestParseBrewInfo(t *testing.T) {
	tests := []struct {
		name        string
		jsonOutput  string
		wantVersion string
		wantTap     string
		wantErr     bool
	}{
		{
			name: "valid homebrew/core formula",
			jsonOutput: `{
				"formulae": [
					{
						"name": "ramp",
						"tap": "homebrew/core",
						"versions": {
							"stable": "1.2.3"
						}
					}
				]
			}`,
			wantVersion: "1.2.3",
			wantTap:     "homebrew/core",
			wantErr:     false,
		},
		{
			name: "custom tap",
			jsonOutput: `{
				"formulae": [
					{
						"name": "ramp",
						"tap": "robrichardson13/ramp",
						"versions": {
							"stable": "2.0.0"
						}
					}
				]
			}`,
			wantVersion: "2.0.0",
			wantTap:     "robrichardson13/ramp",
			wantErr:     false,
		},
		{
			name:        "invalid JSON",
			jsonOutput:  "not json",
			wantVersion: "",
			wantTap:     "",
			wantErr:     true,
		},
		{
			name:        "empty formulae array",
			jsonOutput:  `{"formulae": []}`,
			wantVersion: "",
			wantTap:     "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, tap, err := parseBrewInfo([]byte(tt.jsonOutput))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBrewInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if version != tt.wantVersion {
				t.Errorf("parseBrewInfo() version = %q, want %q", version, tt.wantVersion)
			}
			if tap != tt.wantTap {
				t.Errorf("parseBrewInfo() tap = %q, want %q", tap, tt.wantTap)
			}
		})
	}
}

func TestGetRampDir(t *testing.T) {
	// Test that getRampDir returns a path
	dir, err := getRampDir()
	if err != nil {
		t.Fatalf("getRampDir() error: %v", err)
	}

	// Should end with .ramp
	if !strings.HasSuffix(dir, ".ramp") {
		t.Errorf("getRampDir() = %q, should end with .ramp", dir)
	}

	// Should be in home directory
	home, _ := os.UserHomeDir()
	if !strings.HasPrefix(dir, home) {
		t.Errorf("getRampDir() = %q, should be in home directory %q", dir, home)
	}
}

func TestGetCachePath(t *testing.T) {
	path := getCachePath()

	// Should end with update_check.json
	if !strings.HasSuffix(path, "update_check.json") {
		t.Errorf("getCachePath() = %q, should end with update_check.json", path)
	}

	// Should contain .ramp
	if !strings.Contains(path, ".ramp") {
		t.Errorf("getCachePath() = %q, should contain .ramp", path)
	}
}

func TestGetLockPath(t *testing.T) {
	path := getLockPath()

	// Should end with update.lock
	if !strings.HasSuffix(path, "update.lock") {
		t.Errorf("getLockPath() = %q, should end with update.lock", path)
	}

	// Should contain .ramp
	if !strings.Contains(path, ".ramp") {
		t.Errorf("getLockPath() = %q, should contain .ramp", path)
	}
}

func TestGetLogPath(t *testing.T) {
	path := getLogPath()

	// Should end with update.log
	if !strings.HasSuffix(path, "update.log") {
		t.Errorf("getLogPath() = %q, should end with update.log", path)
	}

	// Should contain .ramp
	if !strings.Contains(path, ".ramp") {
		t.Errorf("getLogPath() = %q, should contain .ramp", path)
	}
}

func TestGetCheckInterval(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     string // duration as string for comparison
	}{
		{
			name:     "default when not set",
			envValue: "",
			want:     "24h0m0s",
		},
		{
			name:     "custom 12h",
			envValue: "12h",
			want:     "12h0m0s",
		},
		{
			name:     "custom 48h",
			envValue: "48h",
			want:     "48h0m0s",
		},
		{
			name:     "custom 30m",
			envValue: "30m",
			want:     "30m0s",
		},
		{
			name:     "invalid falls back to default",
			envValue: "invalid",
			want:     "24h0m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env var
			if tt.envValue != "" {
				os.Setenv("RAMP_UPDATE_CHECK_INTERVAL", tt.envValue)
			} else {
				os.Unsetenv("RAMP_UPDATE_CHECK_INTERVAL")
			}
			defer os.Unsetenv("RAMP_UPDATE_CHECK_INTERVAL")

			got := getCheckInterval()
			if got.String() != tt.want {
				t.Errorf("getCheckInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAutoUpdateEnabled(t *testing.T) {
	// Create temp directory for fake Homebrew binary
	tmpDir := t.TempDir()
	fakeBinary := filepath.Join(tmpDir, "Cellar", "ramp", "1.0.0", "bin", "ramp")
	os.MkdirAll(filepath.Dir(fakeBinary), 0755)
	os.WriteFile(fakeBinary, []byte("fake"), 0755)

	tests := []struct {
		name         string
		envValue     string
		binaryPath   string
		want         bool
	}{
		{
			name:       "enabled by default with Homebrew install",
			envValue:   "",
			binaryPath: fakeBinary,
			want:       true,
		},
		{
			name:       "explicitly disabled",
			envValue:   "false",
			binaryPath: fakeBinary,
			want:       false,
		},
		{
			name:       "disabled with 0",
			envValue:   "0",
			binaryPath: fakeBinary,
			want:       false,
		},
		{
			name:       "disabled if not Homebrew install",
			envValue:   "",
			binaryPath: "/usr/local/bin/ramp",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env var
			if tt.envValue != "" {
				os.Setenv("RAMP_AUTO_UPDATE", tt.envValue)
			} else {
				os.Unsetenv("RAMP_AUTO_UPDATE")
			}
			defer os.Unsetenv("RAMP_AUTO_UPDATE")

			got := isAutoUpdateEnabled(tt.binaryPath)
			if got != tt.want {
				t.Errorf("isAutoUpdateEnabled(%q) = %v, want %v", tt.binaryPath, got, tt.want)
			}
		})
	}
}
