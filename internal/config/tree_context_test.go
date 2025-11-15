package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFeatureFromWorkingDir(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, projectDir string) string // Returns the working directory to test from
		expectedFeature string
		expectError    bool
	}{
		{
			name: "DetectFromFeatureRoot",
			setupFunc: func(t *testing.T, projectDir string) string {
				// Create trees/my-feature/
				featureDir := filepath.Join(projectDir, "trees", "my-feature")
				if err := os.MkdirAll(featureDir, 0755); err != nil {
					t.Fatalf("Failed to create feature dir: %v", err)
				}
				return featureDir
			},
			expectedFeature: "my-feature",
			expectError:     false,
		},
		{
			name: "DetectFromRepoInFeature",
			setupFunc: func(t *testing.T, projectDir string) string {
				// Create trees/my-feature/my-repo/
				repoDir := filepath.Join(projectDir, "trees", "my-feature", "my-repo")
				if err := os.MkdirAll(repoDir, 0755); err != nil {
					t.Fatalf("Failed to create repo dir: %v", err)
				}
				return repoDir
			},
			expectedFeature: "my-feature",
			expectError:     false,
		},
		{
			name: "DetectFromNestedPath",
			setupFunc: func(t *testing.T, projectDir string) string {
				// Create trees/my-feature/my-repo/src/components/Button.tsx
				deepDir := filepath.Join(projectDir, "trees", "my-feature", "my-repo", "src", "components")
				if err := os.MkdirAll(deepDir, 0755); err != nil {
					t.Fatalf("Failed to create deep dir: %v", err)
				}
				return deepDir
			},
			expectedFeature: "my-feature",
			expectError:     false,
		},
		{
			name: "DetectFeatureWithDashesAndUnderscores",
			setupFunc: func(t *testing.T, projectDir string) string {
				// Create trees/feature-name_with-special123/repo/
				featureDir := filepath.Join(projectDir, "trees", "feature-name_with-special123", "repo")
				if err := os.MkdirAll(featureDir, 0755); err != nil {
					t.Fatalf("Failed to create feature dir: %v", err)
				}
				return featureDir
			},
			expectedFeature: "feature-name_with-special123",
			expectError:     false,
		},
		{
			name: "NotInTreesDirectory",
			setupFunc: func(t *testing.T, projectDir string) string {
				// Return project root (not in trees/)
				return projectDir
			},
			expectedFeature: "",
			expectError:     false,
		},
		{
			name: "InReposDirectory",
			setupFunc: func(t *testing.T, projectDir string) string {
				// Create repos/my-repo/src/
				repoDir := filepath.Join(projectDir, "repos", "my-repo", "src")
				if err := os.MkdirAll(repoDir, 0755); err != nil {
					t.Fatalf("Failed to create repos dir: %v", err)
				}
				return repoDir
			},
			expectedFeature: "",
			expectError:     false,
		},
		{
			name: "InTreesButNotInFeature",
			setupFunc: func(t *testing.T, projectDir string) string {
				// Create trees/ directory itself
				treesDir := filepath.Join(projectDir, "trees")
				if err := os.MkdirAll(treesDir, 0755); err != nil {
					t.Fatalf("Failed to create trees dir: %v", err)
				}
				return treesDir
			},
			expectedFeature: "",
			expectError:     false,
		},
		{
			name: "WithTrailingSlashInFeatureName",
			setupFunc: func(t *testing.T, projectDir string) string {
				// Create trees/my-feature/repo/
				repoDir := filepath.Join(projectDir, "trees", "my-feature", "repo")
				if err := os.MkdirAll(repoDir, 0755); err != nil {
					t.Fatalf("Failed to create repo dir: %v", err)
				}
				return repoDir
			},
			expectedFeature: "my-feature",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary project directory
			projectDir := t.TempDir()

			// Setup the directory structure and get the working directory to test from
			workingDir := tt.setupFunc(t, projectDir)

			// Save current directory
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer os.Chdir(originalDir)

			// Change to the test working directory
			if err := os.Chdir(workingDir); err != nil {
				t.Fatalf("Failed to change to working directory: %v", err)
			}

			// Call the function under test
			feature, err := DetectFeatureFromWorkingDir(projectDir)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check feature name
			if feature != tt.expectedFeature {
				t.Errorf("Expected feature '%s', got '%s'", tt.expectedFeature, feature)
			}
		})
	}
}

func TestDetectFeatureFromWorkingDir_EdgeCases(t *testing.T) {
	t.Run("ProjectDirDoesNotExist", func(t *testing.T) {
		nonExistentDir := "/path/that/does/not/exist"
		_, err := DetectFeatureFromWorkingDir(nonExistentDir)
		// Should not error, just return empty string since we're not in trees/
		if err != nil {
			t.Errorf("Should not error for non-existent project dir, got: %v", err)
		}
	})

	t.Run("WorkingDirOutsideProjectDir", func(t *testing.T) {
		// Create a project dir
		projectDir := t.TempDir()

		// Working directory is somewhere else entirely
		otherDir := t.TempDir()
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)

		if err := os.Chdir(otherDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		// Should return empty string (not in this project's trees)
		feature, err := DetectFeatureFromWorkingDir(projectDir)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if feature != "" {
			t.Errorf("Expected empty feature, got '%s'", feature)
		}
	})

	t.Run("NestedTreesDirectories", func(t *testing.T) {
		// Edge case: trees/my-feature/trees/something
		// Should detect "my-feature" as the feature name
		projectDir := t.TempDir()
		nestedDir := filepath.Join(projectDir, "trees", "my-feature", "trees", "something")
		if err := os.MkdirAll(nestedDir, 0755); err != nil {
			t.Fatalf("Failed to create nested dir: %v", err)
		}

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)

		if err := os.Chdir(nestedDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		feature, err := DetectFeatureFromWorkingDir(projectDir)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if feature != "my-feature" {
			t.Errorf("Expected 'my-feature', got '%s'", feature)
		}
	})
}
