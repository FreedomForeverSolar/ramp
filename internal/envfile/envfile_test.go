package envfile

import (
	"os"
	"path/filepath"
	"testing"

	"ramp/internal/config"
)

// TestProcessEnvFiles tests the main env file processing function
func TestProcessEnvFiles(t *testing.T) {
	t.Run("simple copy with auto-replace", func(t *testing.T) {
		// Setup: create source repo with .env file
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")

		os.MkdirAll(sourceRepoDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		// Create source .env file with ${RAMP_*} variables
		envContent := `PORT=${RAMP_PORT}
API_PORT=${RAMP_PORT}1
APP_NAME=myapp-${RAMP_WORKTREE_NAME}
DB_NAME=db_${RAMP_WORKTREE_NAME}
`
		os.WriteFile(filepath.Join(sourceRepoDir, ".env"), []byte(envContent), 0644)

		// Process env files
		envFiles := []config.EnvFile{
			{Source: ".env", Dest: ".env"},
		}

		envVars := map[string]string{
			"RAMP_PORT":          "4000",
			"RAMP_WORKTREE_NAME": "my-feature",
		}

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		// Verify file was copied and variables replaced
		destFile := filepath.Join(worktreeDir, ".env")
		content, err := os.ReadFile(destFile)
		if err != nil {
			t.Fatalf("failed to read destination file: %v", err)
		}

		expected := `PORT=4000
API_PORT=40001
APP_NAME=myapp-my-feature
DB_NAME=db_my-feature
`
		if string(content) != expected {
			t.Errorf("destination file content mismatch\ngot:\n%s\nwant:\n%s", string(content), expected)
		}
	})

	t.Run("cross-repo copy", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create two source repos: configs and app
		configsSourceDir := filepath.Join(tempDir, "source", "configs")
		appSourceDir := filepath.Join(tempDir, "source", "app")

		// Create worktree directories
		configsWorktreeDir := filepath.Join(tempDir, "worktree", "configs")
		appWorktreeDir := filepath.Join(tempDir, "worktree", "app")

		os.MkdirAll(filepath.Join(configsSourceDir, "app"), 0755)
		os.MkdirAll(appSourceDir, 0755)
		os.MkdirAll(configsWorktreeDir, 0755)
		os.MkdirAll(appWorktreeDir, 0755)

		// Create env file in configs repo
		envContent := `PORT=3000
API_URL=http://localhost:3000
`
		os.WriteFile(filepath.Join(configsSourceDir, "app", "prod.env"), []byte(envContent), 0644)

		// Process env files for app repo with cross-repo reference
		envFiles := []config.EnvFile{
			{
				Source: "../configs/app/prod.env",
				Dest:   ".env",
				Replace: map[string]string{
					"PORT":    "${RAMP_PORT}",
					"API_URL": "http://localhost:${RAMP_PORT}",
				},
			},
		}

		envVars := map[string]string{
			"RAMP_PORT": "4000",
		}

		// Note: source path resolution happens relative to source repo
		err := ProcessEnvFiles("app", envFiles, appSourceDir, appWorktreeDir, envVars)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		// Verify file was copied with replacements
		destFile := filepath.Join(appWorktreeDir, ".env")
		content, err := os.ReadFile(destFile)
		if err != nil {
			t.Fatalf("failed to read destination file: %v", err)
		}

		expected := `PORT=4000
API_URL=http://localhost:4000
`
		if string(content) != expected {
			t.Errorf("destination file content mismatch\ngot:\n%s\nwant:\n%s", string(content), expected)
		}
	})

	t.Run("custom replacements only", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")

		os.MkdirAll(sourceRepoDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		// Create source .env with both custom keys and RAMP vars
		envContent := `PORT=3000
API_PORT=3001
UNUSED_VAR=${RAMP_PORT}
APP_NAME=default
`
		os.WriteFile(filepath.Join(sourceRepoDir, ".env"), []byte(envContent), 0644)

		// Process with explicit replacements (should only replace specified keys)
		envFiles := []config.EnvFile{
			{
				Source: ".env",
				Dest:   ".env",
				Replace: map[string]string{
					"PORT":     "${RAMP_PORT}",
					"API_PORT": "${RAMP_PORT}1",
				},
			},
		}

		envVars := map[string]string{
			"RAMP_PORT":          "4000",
			"RAMP_WORKTREE_NAME": "my-feature",
		}

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		content, err := os.ReadFile(filepath.Join(worktreeDir, ".env"))
		if err != nil {
			t.Fatalf("failed to read destination file: %v", err)
		}

		// With explicit replace, only PORT and API_PORT should be replaced
		// UNUSED_VAR should NOT be replaced (it keeps ${RAMP_PORT})
		expected := `PORT=4000
API_PORT=40001
UNUSED_VAR=${RAMP_PORT}
APP_NAME=default
`
		if string(content) != expected {
			t.Errorf("destination file content mismatch\ngot:\n%s\nwant:\n%s", string(content), expected)
		}
	})

	t.Run("multiple env files", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")

		os.MkdirAll(sourceRepoDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		// Create multiple source files
		os.WriteFile(filepath.Join(sourceRepoDir, ".env"), []byte("PORT=${RAMP_PORT}\n"), 0644)
		os.WriteFile(filepath.Join(sourceRepoDir, ".env.local"), []byte("DEBUG=true\n"), 0644)

		envFiles := []config.EnvFile{
			{Source: ".env", Dest: ".env"},
			{Source: ".env.local", Dest: ".env.local"},
		}

		envVars := map[string]string{
			"RAMP_PORT": "4000",
		}

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		// Verify both files were copied
		content1, err := os.ReadFile(filepath.Join(worktreeDir, ".env"))
		if err != nil {
			t.Fatalf("failed to read .env: %v", err)
		}
		if string(content1) != "PORT=4000\n" {
			t.Errorf(".env content = %q, want %q", string(content1), "PORT=4000\n")
		}

		content2, err := os.ReadFile(filepath.Join(worktreeDir, ".env.local"))
		if err != nil {
			t.Fatalf("failed to read .env.local: %v", err)
		}
		if string(content2) != "DEBUG=true\n" {
			t.Errorf(".env.local content = %q, want %q", string(content2), "DEBUG=true\n")
		}
	})

	t.Run("missing source file warning", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")

		os.MkdirAll(sourceRepoDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		envFiles := []config.EnvFile{
			{Source: ".env", Dest: ".env"},
		}

		envVars := map[string]string{
			"RAMP_PORT": "4000",
		}

		// Should not error, but should warn (we'll check that no file was created)
		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars)

		// Based on our design, missing files should be warnings, not errors
		if err != nil {
			t.Logf("ProcessEnvFiles() returned error (may be warning): %v", err)
		}

		// Verify destination file was not created
		_, err = os.ReadFile(filepath.Join(worktreeDir, ".env"))
		if err == nil {
			t.Errorf("destination file should not exist when source is missing")
		}
	})

	t.Run("destination subdirectory creation", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")

		os.MkdirAll(sourceRepoDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		os.WriteFile(filepath.Join(sourceRepoDir, ".env"), []byte("PORT=3000\n"), 0644)

		// Destination has a subdirectory
		envFiles := []config.EnvFile{
			{Source: ".env", Dest: "config/.env"},
		}

		envVars := map[string]string{}

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		// Verify subdirectory was created and file copied
		content, err := os.ReadFile(filepath.Join(worktreeDir, "config", ".env"))
		if err != nil {
			t.Fatalf("failed to read destination file: %v", err)
		}
		if string(content) != "PORT=3000\n" {
			t.Errorf("content mismatch: got %q, want %q", string(content), "PORT=3000\n")
		}
	})
}

// TestReplaceEnvVars tests variable replacement logic
func TestReplaceEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		content string
		envVars map[string]string
		want    string
	}{
		{
			name:    "simple replacement",
			content: "PORT=${RAMP_PORT}",
			envVars: map[string]string{"RAMP_PORT": "4000"},
			want:    "PORT=4000",
		},
		{
			name:    "concatenation",
			content: "API_PORT=${RAMP_PORT}1",
			envVars: map[string]string{"RAMP_PORT": "4000"},
			want:    "API_PORT=40001",
		},
		{
			name:    "multiple variables",
			content: "URL=http://${RAMP_HOST}:${RAMP_PORT}",
			envVars: map[string]string{
				"RAMP_HOST": "localhost",
				"RAMP_PORT": "4000",
			},
			want: "URL=http://localhost:4000",
		},
		{
			name:    "variable in string",
			content: "APP_NAME=myapp-${RAMP_WORKTREE_NAME}-prod",
			envVars: map[string]string{"RAMP_WORKTREE_NAME": "feature-1"},
			want:    "APP_NAME=myapp-feature-1-prod",
		},
		{
			name:    "no variables",
			content: "DEBUG=true",
			envVars: map[string]string{"RAMP_PORT": "4000"},
			want:    "DEBUG=true",
		},
		{
			name:    "undefined variable unchanged",
			content: "VAL=${UNDEFINED_VAR}",
			envVars: map[string]string{"RAMP_PORT": "4000"},
			want:    "VAL=${UNDEFINED_VAR}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceEnvVars(tt.content, tt.envVars)
			if got != tt.want {
				t.Errorf("replaceEnvVars() = %q, want %q", got, tt.want)
			}
		})
	}
}
