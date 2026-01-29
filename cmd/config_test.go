package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ramp/internal/config"
)

// TestConfigCommandBasic tests the basic config command flow
func TestConfigCommandBasic(t *testing.T) {
	tp := NewTestProject(t)
	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Add prompts to the config
	cfg, err := config.LoadConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	cfg.Prompts = []*config.Prompt{
		{
			Name:     "RAMP_IDE",
			Question: "Which IDE do you use?",
			Options: []*config.PromptOption{
				{Value: "vscode", Label: "VSCode"},
				{Value: "vim", Label: "Vim"},
			},
			Default: "vscode",
		},
	}

	if err := config.SaveConfig(cfg, tp.Dir); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify local.yaml doesn't exist yet
	localPath := filepath.Join(tp.Dir, ".ramp", "local.yaml")
	if _, err := os.Stat(localPath); !os.IsNotExist(err) {
		t.Fatalf("local.yaml should not exist yet")
	}

	// Note: We can't test the interactive prompting in unit tests
	// because it requires stdin interaction. We'll test the
	// helper functions directly instead.
}

// TestConfigShowBeforeSet tests showing config when no local.yaml exists
func TestConfigShowBeforeSet(t *testing.T) {
	tp := NewTestProject(t)
	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Add prompts to the config
	cfg, err := config.LoadConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	cfg.Prompts = []*config.Prompt{
		{
			Name:     "RAMP_IDE",
			Question: "Which IDE?",
			Options: []*config.PromptOption{
				{Value: "vscode", Label: "VSCode"},
			},
			Default: "vscode",
		},
	}

	if err := config.SaveConfig(cfg, tp.Dir); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Load local config (should be nil)
	localCfg, err := config.LoadLocalConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadLocalConfig failed: %v", err)
	}

	if localCfg != nil {
		t.Errorf("Expected nil local config, got %v", localCfg)
	}
}

// TestConfigShowAfterSet tests showing config after preferences are set
func TestConfigShowAfterSet(t *testing.T) {
	tp := NewTestProject(t)
	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a local config
	localCfg := &config.LocalConfig{
		Preferences: map[string]string{
			"RAMP_IDE":      "vscode",
			"RAMP_DATABASE": "postgres",
		},
	}

	if err := config.SaveLocalConfig(localCfg, tp.Dir); err != nil {
		t.Fatalf("SaveLocalConfig failed: %v", err)
	}

	// Load it back
	loaded, err := config.LoadLocalConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadLocalConfig failed: %v", err)
	}

	if loaded == nil {
		t.Fatal("Expected local config, got nil")
	}

	if loaded.Preferences["RAMP_IDE"] != "vscode" {
		t.Errorf("RAMP_IDE = %q, want vscode", loaded.Preferences["RAMP_IDE"])
	}

	if loaded.Preferences["RAMP_DATABASE"] != "postgres" {
		t.Errorf("RAMP_DATABASE = %q, want postgres", loaded.Preferences["RAMP_DATABASE"])
	}
}

// TestConfigReset tests resetting local configuration
func TestConfigReset(t *testing.T) {
	tp := NewTestProject(t)
	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Create a local config
	localCfg := &config.LocalConfig{
		Preferences: map[string]string{
			"RAMP_IDE": "vscode",
		},
	}

	if err := config.SaveLocalConfig(localCfg, tp.Dir); err != nil {
		t.Fatalf("SaveLocalConfig failed: %v", err)
	}

	// Verify it exists
	localPath := filepath.Join(tp.Dir, ".ramp", "local.yaml")
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		t.Fatal("local.yaml should exist")
	}

	// Reset (delete the file)
	if err := os.Remove(localPath); err != nil {
		t.Fatalf("Failed to remove local.yaml: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(localPath); !os.IsNotExist(err) {
		t.Fatal("local.yaml should not exist after reset")
	}

	// Verify LoadLocalConfig returns nil
	loaded, err := config.LoadLocalConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadLocalConfig failed: %v", err)
	}

	if loaded != nil {
		t.Errorf("Expected nil after reset, got %v", loaded)
	}
}

// TestConfigWithoutPrompts tests config command when no prompts are defined
func TestConfigWithoutPrompts(t *testing.T) {
	tp := NewTestProject(t)
	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Load config (no prompts by default)
	cfg, err := config.LoadConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.HasPrompts() {
		t.Errorf("Expected no prompts, got %d", len(cfg.Prompts))
	}

	// Should not create local.yaml if no prompts defined
	// This is tested by verifying HasPrompts returns false
}

// TestNeedsPrompting tests the logic for determining when to show prompts
func TestNeedsPrompting(t *testing.T) {
	tests := []struct {
		name         string
		hasPrompts   bool
		hasLocalCfg  bool
		wantPrompt   bool
	}{
		{
			name:        "prompts defined, no local config",
			hasPrompts:  true,
			hasLocalCfg: false,
			wantPrompt:  true,
		},
		{
			name:        "prompts defined, has local config",
			hasPrompts:  true,
			hasLocalCfg: true,
			wantPrompt:  false,
		},
		{
			name:        "no prompts, no local config",
			hasPrompts:  false,
			hasLocalCfg: false,
			wantPrompt:  false,
		},
		{
			name:        "no prompts, has local config",
			hasPrompts:  false,
			hasLocalCfg: true,
			wantPrompt:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := NewTestProject(t)
			cleanup := tp.ChangeToProjectDir()
			defer cleanup()

			// Set up prompts if needed
			if tt.hasPrompts {
				cfg, err := config.LoadConfig(tp.Dir)
				if err != nil {
					t.Fatalf("LoadConfig failed: %v", err)
				}

				cfg.Prompts = []*config.Prompt{
					{
						Name:     "RAMP_IDE",
						Question: "IDE?",
						Options: []*config.PromptOption{
							{Value: "vscode", Label: "VSCode"},
						},
						Default: "vscode",
					},
				}

				if err := config.SaveConfig(cfg, tp.Dir); err != nil {
					t.Fatalf("SaveConfig failed: %v", err)
				}
			}

			// Set up local config if needed
			if tt.hasLocalCfg {
				localCfg := &config.LocalConfig{
					Preferences: map[string]string{"RAMP_IDE": "vscode"},
				}
				if err := config.SaveLocalConfig(localCfg, tp.Dir); err != nil {
					t.Fatalf("SaveLocalConfig failed: %v", err)
				}
			}

			// Check if prompting is needed
			cfg, _ := config.LoadConfig(tp.Dir)
			localCfg, _ := config.LoadLocalConfig(tp.Dir)

			needsPrompt := cfg.HasPrompts() && localCfg == nil
			if needsPrompt != tt.wantPrompt {
				t.Errorf("needsPrompt = %v, want %v", needsPrompt, tt.wantPrompt)
			}
		})
	}
}

// TestLocalConfigFileLocation tests that local.yaml is created in the right place
func TestLocalConfigFileLocation(t *testing.T) {
	tp := NewTestProject(t)
	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	localCfg := &config.LocalConfig{
		Preferences: map[string]string{"TEST": "value"},
	}

	if err := config.SaveLocalConfig(localCfg, tp.Dir); err != nil {
		t.Fatalf("SaveLocalConfig failed: %v", err)
	}

	// Verify it's in .ramp/local.yaml
	expectedPath := filepath.Join(tp.Dir, ".ramp", "local.yaml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("local.yaml not found at expected path: %s", expectedPath)
	}

	// Verify it's not in project root
	wrongPath := filepath.Join(tp.Dir, "local.yaml")
	if _, err := os.Stat(wrongPath); !os.IsNotExist(err) {
		t.Error("local.yaml should not exist in project root")
	}
}

// TestLocalConfigYAMLFormat tests that local.yaml is valid YAML
func TestLocalConfigYAMLFormat(t *testing.T) {
	tp := NewTestProject(t)
	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	localCfg := &config.LocalConfig{
		Preferences: map[string]string{
			"RAMP_IDE":      "vscode",
			"RAMP_DATABASE": "postgres",
		},
	}

	if err := config.SaveLocalConfig(localCfg, tp.Dir); err != nil {
		t.Fatalf("SaveLocalConfig failed: %v", err)
	}

	// Read the file
	localPath := filepath.Join(tp.Dir, ".ramp", "local.yaml")
	content, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read local.yaml: %v", err)
	}

	contentStr := string(content)

	// Check for expected YAML structure
	if !strings.Contains(contentStr, "preferences:") {
		t.Error("local.yaml should contain 'preferences:' key")
	}

	if !strings.Contains(contentStr, "RAMP_IDE:") {
		t.Error("local.yaml should contain 'RAMP_IDE:' key")
	}

	if !strings.Contains(contentStr, "vscode") {
		t.Error("local.yaml should contain 'vscode' value")
	}
}

// TestEnsureLocalConfigNonInteractive tests the non-interactive mode behavior
// In non-interactive mode, EnsureLocalConfig should always succeed (skip prompts)
func TestEnsureLocalConfigNonInteractive(t *testing.T) {
	tests := []struct {
		name        string
		hasPrompts  bool
		hasLocalCfg bool
	}{
		{
			name:        "prompts defined, no local config - should skip and succeed",
			hasPrompts:  true,
			hasLocalCfg: false,
		},
		{
			name:        "prompts defined, has local config - should succeed",
			hasPrompts:  true,
			hasLocalCfg: true,
		},
		{
			name:        "no prompts, no local config - should succeed",
			hasPrompts:  false,
			hasLocalCfg: false,
		},
		{
			name:        "no prompts, has local config - should succeed",
			hasPrompts:  false,
			hasLocalCfg: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := NewTestProject(t)
			cleanup := tp.ChangeToProjectDir()
			defer cleanup()

			// Enable non-interactive mode
			origNonInteractive := NonInteractive
			NonInteractive = true
			defer func() { NonInteractive = origNonInteractive }()

			// Set up prompts if needed
			cfg, err := config.LoadConfig(tp.Dir)
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			if tt.hasPrompts {
				cfg.Prompts = []*config.Prompt{
					{
						Name:     "RAMP_IDE",
						Question: "IDE?",
						Options: []*config.PromptOption{
							{Value: "vscode", Label: "VSCode"},
						},
						Default: "vscode",
					},
				}

				if err := config.SaveConfig(cfg, tp.Dir); err != nil {
					t.Fatalf("SaveConfig failed: %v", err)
				}
				// Reload config after save
				cfg, _ = config.LoadConfig(tp.Dir)
			}

			// Set up local config if needed
			if tt.hasLocalCfg {
				localCfg := &config.LocalConfig{
					Preferences: map[string]string{"RAMP_IDE": "vscode"},
				}
				if err := config.SaveLocalConfig(localCfg, tp.Dir); err != nil {
					t.Fatalf("SaveLocalConfig failed: %v", err)
				}
			}

			// Test EnsureLocalConfig - should always succeed in non-interactive mode
			err = EnsureLocalConfig(tp.Dir, cfg)
			if err != nil {
				t.Errorf("EnsureLocalConfig returned unexpected error: %v", err)
			}

			// Verify no local.yaml was created if it didn't exist
			if !tt.hasLocalCfg {
				localCfg, _ := config.LoadLocalConfig(tp.Dir)
				if localCfg != nil {
					t.Error("local.yaml should not have been created in non-interactive mode")
				}
			}
		})
	}
}

// TestEnsureLocalConfigInteractive tests that interactive mode doesn't error (it would prompt)
func TestEnsureLocalConfigInteractive(t *testing.T) {
	tp := NewTestProject(t)
	cleanup := tp.ChangeToProjectDir()
	defer cleanup()

	// Ensure interactive mode (default)
	origNonInteractive := NonInteractive
	NonInteractive = false
	defer func() { NonInteractive = origNonInteractive }()

	cfg, err := config.LoadConfig(tp.Dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// No prompts defined - should succeed without prompting
	err = EnsureLocalConfig(tp.Dir, cfg)
	if err != nil {
		t.Errorf("EnsureLocalConfig with no prompts should succeed: %v", err)
	}

	// With prompts but already configured - should succeed
	cfg.Prompts = []*config.Prompt{
		{
			Name:     "RAMP_IDE",
			Question: "IDE?",
			Options: []*config.PromptOption{
				{Value: "vscode", Label: "VSCode"},
			},
			Default: "vscode",
		},
	}
	if err := config.SaveConfig(cfg, tp.Dir); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}
	cfg, _ = config.LoadConfig(tp.Dir)

	localCfg := &config.LocalConfig{
		Preferences: map[string]string{"RAMP_IDE": "vscode"},
	}
	if err := config.SaveLocalConfig(localCfg, tp.Dir); err != nil {
		t.Fatalf("SaveLocalConfig failed: %v", err)
	}

	err = EnsureLocalConfig(tp.Dir, cfg)
	if err != nil {
		t.Errorf("EnsureLocalConfig with existing local config should succeed: %v", err)
	}
}
