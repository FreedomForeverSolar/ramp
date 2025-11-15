package autoupdate

import (
	"os"
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

// Note: IsAutoUpdateEnabled() tests are now in settings_test.go
// since the function behavior depends on the settings file.
