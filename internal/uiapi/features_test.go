package uiapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"ramp/internal/config"
	"ramp/internal/scaffold"

	"github.com/gorilla/mux"
)

// TestProjectForUI is a helper for creating test projects for UI tests
type TestProjectForUI struct {
	Dir      string
	ReposDir string
	TreesDir string
	Config   *config.Config
	t        *testing.T
}

// NewTestProjectForUI creates a test project and adds it to the app config
func NewTestProjectForUI(t *testing.T) *TestProjectForUI {
	t.Helper()

	// Disable user config to prevent personal hooks/commands from interfering with tests
	t.Setenv("RAMP_USER_CONFIG_DIR", "")

	projectDir := t.TempDir()

	// Resolve symlinks
	canonicalDir, err := filepath.EvalSymlinks(projectDir)
	if err != nil {
		canonicalDir = projectDir
	}

	// Create project structure
	data := scaffold.ProjectData{
		Name:           "test-project",
		BranchPrefix:   "feature/",
		IncludeSetup:   false,
		IncludeCleanup: false,
		EnablePorts:    true,
		BasePort:       3000,
		Repos:          []scaffold.RepoData{},
	}

	if err := scaffold.CreateProject(canonicalDir, data); err != nil {
		t.Fatalf("failed to create test project: %v", err)
	}

	cfg, err := config.LoadConfig(canonicalDir)
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}

	tp := &TestProjectForUI{
		Dir:      canonicalDir,
		ReposDir: filepath.Join(canonicalDir, "repos"),
		TreesDir: filepath.Join(canonicalDir, "trees"),
		Config:   cfg,
		t:        t,
	}

	return tp
}

// InitRepo initializes a git repository
func (tp *TestProjectForUI) InitRepo(name string) {
	tp.t.Helper()

	// Create bare remote repository
	remoteDir := filepath.Join(tp.t.TempDir(), name+"-remote")
	runGitCmdUI(tp.t, remoteDir, "init", "--bare")

	// Create source repository
	sourceDir := filepath.Join(tp.ReposDir, name)
	runGitCmdUI(tp.t, sourceDir, "init")
	runGitCmdUI(tp.t, sourceDir, "config", "user.email", "test@example.com")
	runGitCmdUI(tp.t, sourceDir, "config", "user.name", "Test User")
	runGitCmdUI(tp.t, sourceDir, "config", "commit.gpgsign", "false")

	// Create initial commit
	readmeFile := filepath.Join(sourceDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# "+name), 0644); err != nil {
		tp.t.Fatalf("failed to create README: %v", err)
	}
	runGitCmdUI(tp.t, sourceDir, "add", "README.md")
	runGitCmdUI(tp.t, sourceDir, "commit", "-m", "initial commit")
	runGitCmdUI(tp.t, sourceDir, "branch", "-M", "main")
	runGitCmdUI(tp.t, sourceDir, "remote", "add", "origin", remoteDir)
	runGitCmdUI(tp.t, sourceDir, "push", "-u", "origin", "main")

	// Update config
	autoRefresh := false // Disable auto-refresh for faster tests
	tp.Config.Repos = append(tp.Config.Repos, &config.Repo{
		Path:        "repos",
		Git:         sourceDir,
		AutoRefresh: &autoRefresh,
	})

	if err := config.SaveConfig(tp.Config, tp.Dir); err != nil {
		tp.t.Fatalf("failed to save updated config: %v", err)
	}
}

// AddToAppConfig adds this project to the app config
func (tp *TestProjectForUI) AddToAppConfig() string {
	tp.t.Helper()
	id, err := AddProjectToConfig(tp.Dir)
	if err != nil {
		tp.t.Fatalf("failed to add project to config: %v", err)
	}
	return id
}

func runGitCmdUI(t *testing.T, dir string, args ...string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %v\nOutput: %s", args, dir, err, string(output))
	}
}

// === LIST FEATURES TESTS ===

func TestListFeatures_Empty(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	tp := NewTestProjectForUI(t)
	tp.InitRepo("repo1")
	id := tp.AddToAppConfig()

	server := NewServer()

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+id+"/features", nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()

	server.ListFeatures(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListFeatures() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response FeaturesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Features) != 0 {
		t.Errorf("ListFeatures() returned %d features, want 0", len(response.Features))
	}
}

func TestListFeatures_ProjectNotFound(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	server := NewServer()

	req := httptest.NewRequest(http.MethodGet, "/api/projects/nonexistent/features", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	server.ListFeatures(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("ListFeatures() status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// === CREATE FEATURE TESTS ===

func TestCreateFeature_Basic(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	tp := NewTestProjectForUI(t)
	tp.InitRepo("repo1")
	id := tp.AddToAppConfig()

	server := NewServer()

	body := CreateFeatureRequest{
		Name:        "test-feature",
		SkipRefresh: true,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+id+"/features", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()

	server.CreateFeature(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("CreateFeature() status = %d, want %d. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var feature Feature
	if err := json.Unmarshal(w.Body.Bytes(), &feature); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if feature.Name != "test-feature" {
		t.Errorf("Feature.Name = %q, want %q", feature.Name, "test-feature")
	}

	// Verify feature directory exists
	featureDir := filepath.Join(tp.TreesDir, "test-feature")
	if _, err := os.Stat(featureDir); os.IsNotExist(err) {
		t.Error("Feature directory should exist")
	}
}

func TestCreateFeature_InvalidJSON(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	tp := NewTestProjectForUI(t)
	tp.InitRepo("repo1")
	id := tp.AddToAppConfig()

	server := NewServer()

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+id+"/features", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()

	server.CreateFeature(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateFeature() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateFeature_MissingName(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	tp := NewTestProjectForUI(t)
	tp.InitRepo("repo1")
	id := tp.AddToAppConfig()

	server := NewServer()

	body := CreateFeatureRequest{
		Name: "", // Missing name
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+id+"/features", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()

	server.CreateFeature(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateFeature() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateFeature_ProjectNotFound(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	server := NewServer()

	body := CreateFeatureRequest{Name: "test"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/nonexistent/features", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	server.CreateFeature(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("CreateFeature() status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestCreateFeature_WithFromBranch(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	tp := NewTestProjectForUI(t)
	tp.InitRepo("repo1")
	id := tp.AddToAppConfig()

	server := NewServer()

	// FromBranch parses the branch name to derive feature name
	body := CreateFeatureRequest{
		FromBranch:  "feature/existing",
		SkipRefresh: true,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+id+"/features", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()

	server.CreateFeature(w, req)

	// This will fail because origin/feature/existing doesn't exist, but we can verify the parsing
	// by checking that the error is about the feature creation, not about missing name
	if w.Code == http.StatusBadRequest {
		var errResp ErrorResponse
		json.Unmarshal(w.Body.Bytes(), &errResp)
		if errResp.Error == "Feature name is required" {
			t.Error("FromBranch should provide feature name")
		}
	}
}

// === DELETE FEATURE TESTS ===

func TestDeleteFeature_Basic(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	tp := NewTestProjectForUI(t)
	tp.InitRepo("repo1")
	id := tp.AddToAppConfig()

	server := NewServer()

	// First create a feature
	createBody := CreateFeatureRequest{
		Name:        "to-delete",
		SkipRefresh: true,
	}
	createBodyBytes, _ := json.Marshal(createBody)

	createReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+id+"/features", bytes.NewReader(createBodyBytes))
	createReq.Header.Set("Content-Type", "application/json")
	createReq = mux.SetURLVars(createReq, map[string]string{"id": id})
	createW := httptest.NewRecorder()

	server.CreateFeature(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("CreateFeature() failed: %s", createW.Body.String())
	}

	// Verify feature exists
	featureDir := filepath.Join(tp.TreesDir, "to-delete")
	if _, err := os.Stat(featureDir); os.IsNotExist(err) {
		t.Fatal("Feature should exist before deletion")
	}

	// Now delete the feature
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/projects/"+id+"/features/to-delete", nil)
	deleteReq = mux.SetURLVars(deleteReq, map[string]string{"id": id, "name": "to-delete"})
	deleteW := httptest.NewRecorder()

	server.DeleteFeature(deleteW, deleteReq)

	if deleteW.Code != http.StatusOK {
		t.Errorf("DeleteFeature() status = %d, want %d. Body: %s", deleteW.Code, http.StatusOK, deleteW.Body.String())
	}

	// Verify feature is deleted
	if _, err := os.Stat(featureDir); !os.IsNotExist(err) {
		t.Error("Feature directory should be deleted")
	}
}

func TestDeleteFeature_ProjectNotFound(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	server := NewServer()

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/nonexistent/features/test", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent", "name": "test"})
	w := httptest.NewRecorder()

	server.DeleteFeature(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("DeleteFeature() status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteFeature_FeatureNotFound(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	tp := NewTestProjectForUI(t)
	tp.InitRepo("repo1")
	id := tp.AddToAppConfig()

	server := NewServer()

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+id+"/features/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": id, "name": "nonexistent"})
	w := httptest.NewRecorder()

	server.DeleteFeature(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("DeleteFeature() status = %d, want %d (feature not found)", w.Code, http.StatusInternalServerError)
	}
}

// === CATEGORIZATION TESTS ===

func TestCategorizeFeature(t *testing.T) {
	tests := []struct {
		name     string
		statuses []FeatureWorktreeStatus
		want     string
	}{
		{
			name:     "empty statuses",
			statuses: []FeatureWorktreeStatus{},
			want:     "clean",
		},
		{
			name: "clean feature",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: false, AheadCount: 0, BehindCount: 0, IsMerged: false},
			},
			want: "clean",
		},
		{
			name: "in_flight - uncommitted changes",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: true, AheadCount: 0, BehindCount: 0, IsMerged: false},
			},
			want: "in_flight",
		},
		{
			name: "in_flight - unpushed commits",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: false, AheadCount: 2, BehindCount: 0, IsMerged: false},
			},
			want: "in_flight",
		},
		{
			name: "merged - all merged and behind",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: false, AheadCount: 0, BehindCount: 5, IsMerged: true},
			},
			want: "merged",
		},
		{
			name: "clean - merged but not behind",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: false, AheadCount: 0, BehindCount: 0, IsMerged: true},
			},
			want: "clean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := categorizeFeature(tt.statuses)
			if got != tt.want {
				t.Errorf("categorizeFeature() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNeedsAttention(t *testing.T) {
	tests := []struct {
		name     string
		statuses []FeatureWorktreeStatus
		want     bool
	}{
		{
			name:     "empty",
			statuses: []FeatureWorktreeStatus{},
			want:     false,
		},
		{
			name: "clean",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: false, AheadCount: 0},
			},
			want: false,
		},
		{
			name: "uncommitted",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: true, AheadCount: 0},
			},
			want: true,
		},
		{
			name: "ahead and not merged",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: false, AheadCount: 3, IsMerged: false},
			},
			want: true,
		},
		{
			name: "ahead but merged",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: false, AheadCount: 3, IsMerged: true},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsAttention(tt.statuses)
			if got != tt.want {
				t.Errorf("needsAttention() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsMerged(t *testing.T) {
	tests := []struct {
		name     string
		statuses []FeatureWorktreeStatus
		want     bool
	}{
		{
			name:     "empty",
			statuses: []FeatureWorktreeStatus{},
			want:     false, // No statuses means false
		},
		{
			name: "merged and behind",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: false, AheadCount: 0, BehindCount: 5, IsMerged: true},
			},
			want: true,
		},
		{
			name: "merged but not behind",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: false, AheadCount: 0, BehindCount: 0, IsMerged: true},
			},
			want: false,
		},
		{
			name: "not merged",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: false, AheadCount: 0, BehindCount: 5, IsMerged: false},
			},
			want: false,
		},
		{
			name: "has uncommitted",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: true, AheadCount: 0, BehindCount: 5, IsMerged: true},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMerged(tt.statuses)
			if got != tt.want {
				t.Errorf("isMerged() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsClean(t *testing.T) {
	tests := []struct {
		name     string
		statuses []FeatureWorktreeStatus
		want     bool
	}{
		{
			name:     "empty",
			statuses: []FeatureWorktreeStatus{},
			want:     true,
		},
		{
			name: "clean",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: false, AheadCount: 0},
			},
			want: true,
		},
		{
			name: "uncommitted",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: true, AheadCount: 0},
			},
			want: false,
		},
		{
			name: "ahead",
			statuses: []FeatureWorktreeStatus{
				{HasUncommitted: false, AheadCount: 1},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isClean(tt.statuses)
			if got != tt.want {
				t.Errorf("isClean() = %v, want %v", got, tt.want)
			}
		})
	}
}

// === PRUNE TESTS ===

func TestPruneFeatures_NoMergedFeatures(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	tp := NewTestProjectForUI(t)
	tp.InitRepo("repo1")
	id := tp.AddToAppConfig()

	server := NewServer()

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+id+"/features/prune", nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()

	server.PruneFeatures(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("PruneFeatures() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response PruneResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Pruned) != 0 {
		t.Errorf("Pruned = %d, want 0", len(response.Pruned))
	}

	if response.Message != "No merged features to prune" {
		t.Errorf("Message = %q, want %q", response.Message, "No merged features to prune")
	}
}

func TestPruneFeatures_ProjectNotFound(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	server := NewServer()

	req := httptest.NewRequest(http.MethodPost, "/api/projects/nonexistent/features/prune", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	server.PruneFeatures(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("PruneFeatures() status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
