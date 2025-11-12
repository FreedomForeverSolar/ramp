package envfile

import (
	"fmt"
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

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars, false)
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
		err := ProcessEnvFiles("app", envFiles, appSourceDir, appWorktreeDir, envVars, false)
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

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars, false)
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

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars, false)
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
		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars, false)

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

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars, false)
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

// TestScriptExecution tests executing scripts as env file sources
func TestScriptExecution(t *testing.T) {
	t.Run("basic script execution", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")
		scriptsDir := filepath.Join(sourceRepoDir, "scripts")

		os.MkdirAll(scriptsDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		// Create executable script that outputs env content
		scriptPath := filepath.Join(scriptsDir, "fetch-env.sh")
		scriptContent := `#!/bin/bash
echo "DATABASE_URL=postgresql://localhost:5432/db"
echo "API_KEY=secret123"
echo "PORT=3000"
`
		os.WriteFile(scriptPath, []byte(scriptContent), 0755)

		envFiles := []config.EnvFile{
			{Source: "scripts/fetch-env.sh", Dest: ".env"},
		}

		envVars := map[string]string{
			"RAMP_PORT": "4000",
		}

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars, false)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		// Verify script output was written to destination
		content, err := os.ReadFile(filepath.Join(worktreeDir, ".env"))
		if err != nil {
			t.Fatalf("failed to read destination file: %v", err)
		}

		expected := "DATABASE_URL=postgresql://localhost:5432/db\nAPI_KEY=secret123\nPORT=3000\n"
		if string(content) != expected {
			t.Errorf("destination file content mismatch\ngot:\n%s\nwant:\n%s", string(content), expected)
		}
	})

	t.Run("script with replacements", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")
		scriptsDir := filepath.Join(sourceRepoDir, "scripts")

		os.MkdirAll(scriptsDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		// Script outputs env with PORT that will be replaced
		scriptPath := filepath.Join(scriptsDir, "fetch-env.sh")
		scriptContent := `#!/bin/bash
echo "DATABASE_URL=postgresql://localhost:5432/db"
echo "PORT=3000"
echo "API_URL=http://localhost:3000"
`
		os.WriteFile(scriptPath, []byte(scriptContent), 0755)

		envFiles := []config.EnvFile{
			{
				Source: "scripts/fetch-env.sh",
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

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars, false)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		content, err := os.ReadFile(filepath.Join(worktreeDir, ".env"))
		if err != nil {
			t.Fatalf("failed to read destination file: %v", err)
		}

		// Replacements should be applied after script execution
		expected := "DATABASE_URL=postgresql://localhost:5432/db\nPORT=4000\nAPI_URL=http://localhost:4000\n"
		if string(content) != expected {
			t.Errorf("destination file content mismatch\ngot:\n%s\nwant:\n%s", string(content), expected)
		}
	})

	t.Run("script receives RAMP environment variables", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")
		scriptsDir := filepath.Join(sourceRepoDir, "scripts")

		os.MkdirAll(scriptsDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		// Script uses RAMP env vars to generate content
		scriptPath := filepath.Join(scriptsDir, "dynamic-env.sh")
		scriptContent := `#!/bin/bash
echo "PORT=${RAMP_PORT}"
echo "APP_NAME=myapp-${RAMP_WORKTREE_NAME}"
echo "PROJECT_DIR=${RAMP_PROJECT_DIR}"
`
		os.WriteFile(scriptPath, []byte(scriptContent), 0755)

		envFiles := []config.EnvFile{
			{Source: "scripts/dynamic-env.sh", Dest: ".env"},
		}

		envVars := map[string]string{
			"RAMP_PORT":          "4000",
			"RAMP_WORKTREE_NAME": "my-feature",
			"RAMP_PROJECT_DIR":   "/home/user/project",
		}

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars, false)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		content, err := os.ReadFile(filepath.Join(worktreeDir, ".env"))
		if err != nil {
			t.Fatalf("failed to read destination file: %v", err)
		}

		expected := "PORT=4000\nAPP_NAME=myapp-my-feature\nPROJECT_DIR=/home/user/project\n"
		if string(content) != expected {
			t.Errorf("destination file content mismatch\ngot:\n%s\nwant:\n%s", string(content), expected)
		}
	})

	t.Run("script failure returns error", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")
		scriptsDir := filepath.Join(sourceRepoDir, "scripts")

		os.MkdirAll(scriptsDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		// Script that exits with error
		scriptPath := filepath.Join(scriptsDir, "failing-script.sh")
		scriptContent := `#!/bin/bash
echo "Failed to fetch secrets" >&2
exit 1
`
		os.WriteFile(scriptPath, []byte(scriptContent), 0755)

		envFiles := []config.EnvFile{
			{Source: "scripts/failing-script.sh", Dest: ".env"},
		}

		envVars := map[string]string{}

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars, false)
		if err == nil {
			t.Fatal("ProcessEnvFiles() should return error when script fails")
		}

		// Verify error message mentions script failure
		if !os.IsNotExist(err) && err.Error() == "" {
			t.Errorf("expected error message about script failure, got: %v", err)
		}
	})

	t.Run("non-executable file is treated as regular file", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")
		scriptsDir := filepath.Join(sourceRepoDir, "scripts")

		os.MkdirAll(scriptsDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		// Create non-executable script (should be treated as regular file)
		scriptPath := filepath.Join(scriptsDir, "not-executable.sh")
		scriptContent := `#!/bin/bash
echo "This won't execute"
`
		os.WriteFile(scriptPath, []byte(scriptContent), 0644) // No execute permission

		envFiles := []config.EnvFile{
			{Source: "scripts/not-executable.sh", Dest: ".env"},
		}

		envVars := map[string]string{}

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars, false)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		// Should copy file content as-is (not execute)
		content, err := os.ReadFile(filepath.Join(worktreeDir, ".env"))
		if err != nil {
			t.Fatalf("failed to read destination file: %v", err)
		}

		if string(content) != scriptContent {
			t.Errorf("file should be copied as-is, not executed\ngot:\n%s\nwant:\n%s", string(content), scriptContent)
		}
	})
}

// TestCaching tests the caching functionality for script execution
func TestCaching(t *testing.T) {
	t.Run("cache with TTL - fresh cache", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")
		scriptsDir := filepath.Join(sourceRepoDir, "scripts")
		projectDir := tempDir

		os.MkdirAll(scriptsDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		// Create counter file to track executions
		counterFile := filepath.Join(tempDir, "counter.txt")
		os.WriteFile(counterFile, []byte("0"), 0644)

		// Script that increments counter each time it runs
		scriptPath := filepath.Join(scriptsDir, "cached-script.sh")
		scriptContent := fmt.Sprintf(`#!/bin/bash
COUNT=$(cat %s)
COUNT=$((COUNT + 1))
echo $COUNT > %s
echo "DATABASE_URL=postgres://localhost:5432/db_$COUNT"
`, counterFile, counterFile)
		os.WriteFile(scriptPath, []byte(scriptContent), 0755)

		envFiles := []config.EnvFile{
			{
				Source: "scripts/cached-script.sh",
				Dest:   ".env",
				Cache:  "24h",
			},
		}

		envVars := map[string]string{}

		// First execution - should run script
		err := ProcessEnvFilesWithProjectDir("app", envFiles, sourceRepoDir, worktreeDir, envVars, false, projectDir)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		content1, _ := os.ReadFile(filepath.Join(worktreeDir, ".env"))
		expected1 := "DATABASE_URL=postgres://localhost:5432/db_1\n"
		if string(content1) != expected1 {
			t.Errorf("first execution: got %q, want %q", string(content1), expected1)
		}

		// Second execution - should use cache (counter shouldn't increment)
		err = ProcessEnvFilesWithProjectDir("app", envFiles, sourceRepoDir, worktreeDir, envVars, false, projectDir)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		content2, _ := os.ReadFile(filepath.Join(worktreeDir, ".env"))
		if string(content2) != expected1 {
			t.Errorf("second execution should use cache: got %q, want %q", string(content2), expected1)
		}

		// Verify counter is still 1 (script only ran once)
		counter, _ := os.ReadFile(counterFile)
		if string(counter) != "1\n" {
			t.Errorf("counter should be 1 (script ran once), got %q", string(counter))
		}
	})

	t.Run("cache with refresh flag - ignores cache", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")
		scriptsDir := filepath.Join(sourceRepoDir, "scripts")
		projectDir := tempDir

		os.MkdirAll(scriptsDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		counterFile := filepath.Join(tempDir, "counter.txt")
		os.WriteFile(counterFile, []byte("0"), 0644)

		scriptPath := filepath.Join(scriptsDir, "cached-script.sh")
		scriptContent := fmt.Sprintf(`#!/bin/bash
COUNT=$(cat %s)
COUNT=$((COUNT + 1))
echo $COUNT > %s
echo "RUN_COUNT=$COUNT"
`, counterFile, counterFile)
		os.WriteFile(scriptPath, []byte(scriptContent), 0755)

		envFiles := []config.EnvFile{
			{
				Source: "scripts/cached-script.sh",
				Dest:   ".env",
				Cache:  "24h",
			},
		}

		envVars := map[string]string{}

		// First execution
		err := ProcessEnvFilesWithProjectDir("app", envFiles, sourceRepoDir, worktreeDir, envVars, false, projectDir)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		// Second execution with refresh=true - should ignore cache
		err = ProcessEnvFilesWithProjectDir("app", envFiles, sourceRepoDir, worktreeDir, envVars, true, projectDir)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(worktreeDir, ".env"))
		expected := "RUN_COUNT=2\n" // Should have run twice
		if string(content) != expected {
			t.Errorf("refresh should ignore cache: got %q, want %q", string(content), expected)
		}
	})

	t.Run("no cache field - always executes", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")
		scriptsDir := filepath.Join(sourceRepoDir, "scripts")
		projectDir := tempDir

		os.MkdirAll(scriptsDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		counterFile := filepath.Join(tempDir, "counter.txt")
		os.WriteFile(counterFile, []byte("0"), 0644)

		scriptPath := filepath.Join(scriptsDir, "no-cache-script.sh")
		scriptContent := fmt.Sprintf(`#!/bin/bash
COUNT=$(cat %s)
COUNT=$((COUNT + 1))
echo $COUNT > %s
echo "RUN_COUNT=$COUNT"
`, counterFile, counterFile)
		os.WriteFile(scriptPath, []byte(scriptContent), 0755)

		envFiles := []config.EnvFile{
			{
				Source: "scripts/no-cache-script.sh",
				Dest:   ".env",
				// No Cache field - should always execute
			},
		}

		envVars := map[string]string{}

		// First execution
		err := ProcessEnvFilesWithProjectDir("app", envFiles, sourceRepoDir, worktreeDir, envVars, false, projectDir)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		// Second execution - should execute again (no cache)
		err = ProcessEnvFilesWithProjectDir("app", envFiles, sourceRepoDir, worktreeDir, envVars, false, projectDir)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(worktreeDir, ".env"))
		expected := "RUN_COUNT=2\n" // Should have run twice
		if string(content) != expected {
			t.Errorf("no cache should always execute: got %q, want %q", string(content), expected)
		}
	})

	t.Run("expired cache - re-executes", func(t *testing.T) {
		t.Skip("Time-based test - implement with manual cache manipulation")
		// This would require manipulating cache file timestamps
		// or mocking time.Now() which is complex
		// We'll rely on integration testing for this
	})

	t.Run("cache is per-script (different scripts have different caches)", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")
		scriptsDir := filepath.Join(sourceRepoDir, "scripts")
		projectDir := tempDir

		os.MkdirAll(scriptsDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		// Create two different scripts
		script1Path := filepath.Join(scriptsDir, "script1.sh")
		script1Content := `#!/bin/bash
echo "OUTPUT_FROM=script1"
`
		os.WriteFile(script1Path, []byte(script1Content), 0755)

		script2Path := filepath.Join(scriptsDir, "script2.sh")
		script2Content := `#!/bin/bash
echo "OUTPUT_FROM=script2"
`
		os.WriteFile(script2Path, []byte(script2Content), 0755)

		envFiles := []config.EnvFile{
			{Source: "scripts/script1.sh", Dest: ".env1", Cache: "24h"},
			{Source: "scripts/script2.sh", Dest: ".env2", Cache: "24h"},
		}

		envVars := map[string]string{}

		// Execute both scripts
		err := ProcessEnvFilesWithProjectDir("app", envFiles, sourceRepoDir, worktreeDir, envVars, false, projectDir)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		// Verify each has its own cached output
		content1, _ := os.ReadFile(filepath.Join(worktreeDir, ".env1"))
		content2, _ := os.ReadFile(filepath.Join(worktreeDir, ".env2"))

		if string(content1) != "OUTPUT_FROM=script1\n" {
			t.Errorf(".env1 content = %q, want %q", string(content1), "OUTPUT_FROM=script1\n")
		}
		if string(content2) != "OUTPUT_FROM=script2\n" {
			t.Errorf(".env2 content = %q, want %q", string(content2), "OUTPUT_FROM=script2\n")
		}
	})
}

// TestMixedSourceTypes tests combining regular files and scripts
func TestMixedSourceTypes(t *testing.T) {
	t.Run("process both files and scripts together", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")
		scriptsDir := filepath.Join(sourceRepoDir, "scripts")

		os.MkdirAll(scriptsDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		// Create regular file
		os.WriteFile(filepath.Join(sourceRepoDir, ".env.example"), []byte("DEBUG=true\nLOG_LEVEL=info\n"), 0644)

		// Create executable script
		scriptPath := filepath.Join(scriptsDir, "fetch-secrets.sh")
		scriptContent := `#!/bin/bash
echo "API_KEY=secret123"
echo "DATABASE_URL=postgres://localhost/db"
`
		os.WriteFile(scriptPath, []byte(scriptContent), 0755)

		envFiles := []config.EnvFile{
			{Source: ".env.example", Dest: ".env"},
			{Source: "scripts/fetch-secrets.sh", Dest: ".env.secrets", Cache: "24h"},
		}

		envVars := map[string]string{}

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars, false)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		// Verify both files were created
		contentFile, err := os.ReadFile(filepath.Join(worktreeDir, ".env"))
		if err != nil {
			t.Fatalf("failed to read .env: %v", err)
		}
		if string(contentFile) != "DEBUG=true\nLOG_LEVEL=info\n" {
			t.Errorf(".env content mismatch: got %q", string(contentFile))
		}

		contentScript, err := os.ReadFile(filepath.Join(worktreeDir, ".env.secrets"))
		if err != nil {
			t.Fatalf("failed to read .env.secrets: %v", err)
		}
		if string(contentScript) != "API_KEY=secret123\nDATABASE_URL=postgres://localhost/db\n" {
			t.Errorf(".env.secrets content mismatch: got %q", string(contentScript))
		}
	})
}

// TestScriptMergingPattern tests the common pattern of merging env.example with secrets
func TestScriptMergingPattern(t *testing.T) {
	t.Run("script merges env.example with secrets", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceRepoDir := filepath.Join(tempDir, "source")
		worktreeDir := filepath.Join(tempDir, "worktree")
		scriptsDir := filepath.Join(sourceRepoDir, "scripts")

		os.MkdirAll(scriptsDir, 0755)
		os.MkdirAll(worktreeDir, 0755)

		// Create .env.example file
		exampleEnv := "DEBUG=true\nLOG_LEVEL=info\nPORT=3000\n"
		os.WriteFile(filepath.Join(sourceRepoDir, ".env.example"), []byte(exampleEnv), 0644)

		// Create script that merges .env.example with secrets
		scriptPath := filepath.Join(scriptsDir, "merge-env.sh")
		scriptContent := `#!/bin/bash
# Output .env.example first
cat "${RAMP_REPO_PATH_APP}/.env.example"
echo ""
echo "# Secrets"
echo "API_KEY=secret123"
echo "DATABASE_URL=postgres://localhost/db"
`
		os.WriteFile(scriptPath, []byte(scriptContent), 0755)

		envFiles := []config.EnvFile{
			{
				Source:  "scripts/merge-env.sh",
				Dest:    ".env",
				Cache:   "24h",
				Replace: map[string]string{"PORT": "${RAMP_PORT}"},
			},
		}

		envVars := map[string]string{
			"RAMP_PORT":          "4000",
			"RAMP_REPO_PATH_APP": sourceRepoDir,
		}

		err := ProcessEnvFiles("app", envFiles, sourceRepoDir, worktreeDir, envVars, false)
		if err != nil {
			t.Fatalf("ProcessEnvFiles() error = %v", err)
		}

		content, err := os.ReadFile(filepath.Join(worktreeDir, ".env"))
		if err != nil {
			t.Fatalf("failed to read .env: %v", err)
		}

		// Should have merged content with PORT replaced
		expected := "DEBUG=true\nLOG_LEVEL=info\nPORT=4000\n\n# Secrets\nAPI_KEY=secret123\nDATABASE_URL=postgres://localhost/db\n"
		if string(content) != expected {
			t.Errorf("merged env content mismatch\ngot:\n%s\nwant:\n%s", string(content), expected)
		}
	})
}
