package uiapi

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestConfig sets up a temporary config path for testing
func setupTestConfig(t *testing.T) func() {
	t.Helper()
	tempDir := t.TempDir()
	original := getConfigPath
	getConfigPath = func() (string, error) {
		return filepath.Join(tempDir, "config.json"), nil
	}
	return func() { getConfigPath = original }
}

func TestGetAppConfigPath(t *testing.T) {
	path, err := GetAppConfigPath()
	if err != nil {
		t.Fatalf("GetAppConfigPath() error = %v", err)
	}

	if path == "" {
		t.Error("GetAppConfigPath() returned empty path")
	}

	// Should end with config.json
	if filepath.Base(path) != "config.json" {
		t.Errorf("GetAppConfigPath() = %v, want path ending in config.json", path)
	}

	// Parent directory should be ramp-ui
	if filepath.Base(filepath.Dir(path)) != "ramp-ui" {
		t.Errorf("GetAppConfigPath() parent dir = %v, want ramp-ui", filepath.Base(filepath.Dir(path)))
	}
}

func TestLoadAppConfig_DefaultConfig(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	config, err := LoadAppConfig()
	if err != nil {
		t.Fatalf("LoadAppConfig() error = %v", err)
	}

	// Should return default config when file doesn't exist
	if config == nil {
		t.Fatal("LoadAppConfig() returned nil config")
	}

	if len(config.Projects) != 0 {
		t.Errorf("LoadAppConfig() default projects = %v, want empty", config.Projects)
	}

	if config.Preferences.Theme != "system" {
		t.Errorf("LoadAppConfig() default theme = %v, want system", config.Preferences.Theme)
	}

	if !config.Preferences.ShowGitStatus {
		t.Error("LoadAppConfig() default ShowGitStatus = false, want true")
	}
}

func TestSaveAndLoadAppConfig(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Create a config
	config := &AppConfig{
		Projects: []ProjectRef{
			{ID: "test-id", Path: "/test/path"},
		},
		Preferences: Preferences{
			Theme:         "dark",
			ShowGitStatus: false,
		},
	}

	// Save it
	err := SaveAppConfig(config)
	if err != nil {
		t.Fatalf("SaveAppConfig() error = %v", err)
	}

	// Load it back
	loaded, err := LoadAppConfig()
	if err != nil {
		t.Fatalf("LoadAppConfig() error = %v", err)
	}

	if len(loaded.Projects) != 1 {
		t.Fatalf("LoadAppConfig() projects count = %d, want 1", len(loaded.Projects))
	}

	if loaded.Projects[0].ID != "test-id" {
		t.Errorf("LoadAppConfig() project ID = %v, want test-id", loaded.Projects[0].ID)
	}

	if loaded.Projects[0].Path != "/test/path" {
		t.Errorf("LoadAppConfig() project Path = %v, want /test/path", loaded.Projects[0].Path)
	}

	if loaded.Preferences.Theme != "dark" {
		t.Errorf("LoadAppConfig() theme = %v, want dark", loaded.Preferences.Theme)
	}
}

func TestLoadAppConfig_InvalidJSON(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Write invalid JSON
	configPath, _ := GetAppConfigPath()
	os.WriteFile(configPath, []byte("invalid json"), 0644)

	_, err := LoadAppConfig()
	if err == nil {
		t.Error("LoadAppConfig() with invalid JSON should return error")
	}
}

func TestAddProjectToConfig(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Add a project
	id, err := AddProjectToConfig("/test/project")
	if err != nil {
		t.Fatalf("AddProjectToConfig() error = %v", err)
	}

	if id == "" {
		t.Error("AddProjectToConfig() returned empty ID")
	}

	if len(id) != 8 {
		t.Errorf("AddProjectToConfig() ID length = %d, want 8", len(id))
	}

	// Verify it was saved
	config, err := LoadAppConfig()
	if err != nil {
		t.Fatalf("LoadAppConfig() error = %v", err)
	}

	if len(config.Projects) != 1 {
		t.Fatalf("Config projects count = %d, want 1", len(config.Projects))
	}

	if config.Projects[0].Path != "/test/project" {
		t.Errorf("Project path = %v, want /test/project", config.Projects[0].Path)
	}

	if config.Projects[0].AddedAt.IsZero() {
		t.Error("Project AddedAt should not be zero")
	}
}

func TestAddProjectToConfig_Duplicate(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Add the same project twice
	id1, err := AddProjectToConfig("/test/project")
	if err != nil {
		t.Fatalf("AddProjectToConfig() first call error = %v", err)
	}

	id2, err := AddProjectToConfig("/test/project")
	if err != nil {
		t.Fatalf("AddProjectToConfig() second call error = %v", err)
	}

	// Should return the same ID
	if id1 != id2 {
		t.Errorf("AddProjectToConfig() duplicate returned different IDs: %v vs %v", id1, id2)
	}

	// Should only have one project
	config, _ := LoadAppConfig()
	if len(config.Projects) != 1 {
		t.Errorf("Config projects count = %d, want 1 (duplicate should not be added)", len(config.Projects))
	}
}

func TestAddProjectToConfig_MultipleProjects(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Add multiple projects
	id1, _ := AddProjectToConfig("/test/project1")
	id2, _ := AddProjectToConfig("/test/project2")
	id3, _ := AddProjectToConfig("/test/project3")

	// All IDs should be unique
	if id1 == id2 || id2 == id3 || id1 == id3 {
		t.Errorf("AddProjectToConfig() generated duplicate IDs: %v, %v, %v", id1, id2, id3)
	}

	// Should have three projects
	config, _ := LoadAppConfig()
	if len(config.Projects) != 3 {
		t.Errorf("Config projects count = %d, want 3", len(config.Projects))
	}
}

func TestRemoveProjectFromConfig(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Add a project
	id, _ := AddProjectToConfig("/test/project")

	// Remove it
	err := RemoveProjectFromConfig(id)
	if err != nil {
		t.Fatalf("RemoveProjectFromConfig() error = %v", err)
	}

	// Verify it was removed
	config, _ := LoadAppConfig()
	if len(config.Projects) != 0 {
		t.Errorf("Config projects count = %d, want 0", len(config.Projects))
	}
}

func TestRemoveProjectFromConfig_NonExistent(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Remove a non-existent project (should not error)
	err := RemoveProjectFromConfig("non-existent-id")
	if err != nil {
		t.Fatalf("RemoveProjectFromConfig() error = %v, want nil", err)
	}
}

func TestRemoveProjectFromConfig_KeepsOthers(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Add multiple projects
	id1, _ := AddProjectToConfig("/test/project1")
	_, _ = AddProjectToConfig("/test/project2")

	// Remove only the first one
	RemoveProjectFromConfig(id1)

	// Should still have one project
	config, _ := LoadAppConfig()
	if len(config.Projects) != 1 {
		t.Errorf("Config projects count = %d, want 1", len(config.Projects))
	}

	if config.Projects[0].Path != "/test/project2" {
		t.Errorf("Remaining project = %v, want /test/project2", config.Projects[0].Path)
	}
}

func TestGetProjectRefByID(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Add a project
	id, _ := AddProjectToConfig("/test/project")

	// Get it by ID
	ref, err := GetProjectRefByID(id)
	if err != nil {
		t.Fatalf("GetProjectRefByID() error = %v", err)
	}

	if ref == nil {
		t.Fatal("GetProjectRefByID() returned nil")
	}

	if ref.ID != id {
		t.Errorf("GetProjectRefByID() ID = %v, want %v", ref.ID, id)
	}

	if ref.Path != "/test/project" {
		t.Errorf("GetProjectRefByID() Path = %v, want /test/project", ref.Path)
	}
}

func TestGetProjectRefByID_NotFound(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	ref, err := GetProjectRefByID("non-existent")
	if err != nil {
		t.Fatalf("GetProjectRefByID() error = %v", err)
	}

	if ref != nil {
		t.Errorf("GetProjectRefByID() = %v, want nil for non-existent ID", ref)
	}
}
