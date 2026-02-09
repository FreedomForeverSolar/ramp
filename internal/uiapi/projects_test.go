package uiapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

func TestIsHiddenDir(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"hidden dir", ".git", true},
		{"hidden file", ".env", true},
		{"normal dir", "src", false},
		{"normal file", "main.go", false},
		{"empty string", "", false},
		{"double dot", "..", true},
		{"single dot", ".", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHiddenDir(tt.input)
			if result != tt.expected {
				t.Errorf("isHiddenDir(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestListExistingFeatures(t *testing.T) {
	// Create a temp project directory
	tempDir := t.TempDir()
	treesDir := filepath.Join(tempDir, "trees")
	os.MkdirAll(treesDir, 0755)

	// Create some feature directories
	os.MkdirAll(filepath.Join(treesDir, "feature-1"), 0755)
	os.MkdirAll(filepath.Join(treesDir, "feature-2"), 0755)
	os.MkdirAll(filepath.Join(treesDir, ".hidden"), 0755) // Should be ignored

	features := listExistingFeatures(tempDir)

	if len(features) != 2 {
		t.Errorf("listExistingFeatures() returned %d features, want 2", len(features))
	}

	// Check that hidden directories are excluded
	for _, f := range features {
		if f == ".hidden" {
			t.Error("listExistingFeatures() should not include hidden directories")
		}
	}
}

func TestListExistingFeatures_NoTreesDir(t *testing.T) {
	tempDir := t.TempDir()

	features := listExistingFeatures(tempDir)

	if len(features) != 0 {
		t.Errorf("listExistingFeatures() returned %d features, want 0 when trees dir doesn't exist", len(features))
	}
}

func TestListExistingFeatures_EmptyTreesDir(t *testing.T) {
	tempDir := t.TempDir()
	treesDir := filepath.Join(tempDir, "trees")
	os.MkdirAll(treesDir, 0755)

	features := listExistingFeatures(tempDir)

	if len(features) != 0 {
		t.Errorf("listExistingFeatures() returned %d features, want 0 when trees dir is empty", len(features))
	}
}

func TestListProjects_Empty(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	server := NewServer()

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	w := httptest.NewRecorder()

	server.ListProjects(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListProjects() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response ProjectsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Projects) != 0 {
		t.Errorf("ListProjects() returned %d projects, want 0", len(response.Projects))
	}
}

func TestAddProject_PathNotExist(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	server := NewServer()

	body := AddProjectRequest{Path: "/non/existent/path"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.AddProject(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("AddProject() status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "Path does not exist" {
		t.Errorf("AddProject() error = %q, want %q", response.Error, "Path does not exist")
	}
}

func TestAddProject_NotRampProject(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Create a directory without .ramp/ramp.yaml
	tempDir := t.TempDir()

	server := NewServer()

	body := AddProjectRequest{Path: tempDir}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.AddProject(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("AddProject() status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "Not a valid Ramp project" {
		t.Errorf("AddProject() error = %q, want %q", response.Error, "Not a valid Ramp project")
	}
}

func TestAddProject_Success(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Create a valid ramp project (with .ramp/ramp.yaml, no .git)
	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".ramp"), 0755)
	os.WriteFile(filepath.Join(tempDir, ".ramp", "ramp.yaml"), []byte("name: test-project\nrepos: []\n"), 0644)

	server := NewServer()

	body := AddProjectRequest{Path: tempDir}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.AddProject(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("AddProject() status = %d, want %d. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestAddProject_SuccessWithGit(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Create a valid ramp project that also has .git
	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".ramp"), 0755)
	os.WriteFile(filepath.Join(tempDir, ".ramp", "ramp.yaml"), []byte("name: test-project\nrepos: []\n"), 0644)
	os.MkdirAll(filepath.Join(tempDir, ".git"), 0755)

	server := NewServer()

	body := AddProjectRequest{Path: tempDir}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.AddProject(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("AddProject() status = %d, want %d. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestAddProject_InvalidJSON(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	server := NewServer()

	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.AddProject(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("AddProject() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRemoveProject(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Add a project first
	id, _ := AddProjectToConfig("/test/path")

	server := NewServer()

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+id, nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()

	server.RemoveProject(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("RemoveProject() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("RemoveProject() success = false, want true")
	}

	// Verify it was removed
	ref, _ := GetProjectRefByID(id)
	if ref != nil {
		t.Error("Project should have been removed")
	}
}
