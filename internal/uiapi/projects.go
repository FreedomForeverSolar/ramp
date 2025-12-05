package uiapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"ramp/internal/config"

	"github.com/gorilla/mux"
)

// ListProjects returns all projects in the app config
func (s *Server) ListProjects(w http.ResponseWriter, r *http.Request) {
	appConfig, err := LoadAppConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load app config", err.Error())
		return
	}

	projects := make([]Project, 0, len(appConfig.Projects))

	for _, ref := range appConfig.Projects {
		project, err := loadProjectFromPath(ref.ID, ref.Path, ref.AddedAt)
		if err != nil {
			// Skip projects that can't be loaded (might have been moved/deleted)
			continue
		}
		projects = append(projects, *project)
	}

	writeJSON(w, http.StatusOK, ProjectsResponse{Projects: projects})
}

// AddProject adds a new project to the app config
func (s *Server) AddProject(w http.ResponseWriter, r *http.Request) {
	var req AddProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate path exists
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		writeError(w, http.StatusBadRequest, "Path does not exist", req.Path)
		return
	}

	// Check for .ramp/ramp.yaml
	configPath := filepath.Join(req.Path, ".ramp", "ramp.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		writeError(w, http.StatusBadRequest, "Not a valid Ramp project", "Missing .ramp/ramp.yaml")
		return
	}

	// Add to app config
	id, err := AddProjectToConfig(req.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to add project", err.Error())
		return
	}

	// Load and return the project
	appConfig, _ := LoadAppConfig()
	var addedAt = appConfig.Projects[len(appConfig.Projects)-1].AddedAt
	project, err := loadProjectFromPath(id, req.Path, addedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load project", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

// RemoveProject removes a project from the app config
func (s *Server) RemoveProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := RemoveProjectFromConfig(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove project", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "Project removed"})
}

// loadProjectFromPath loads a project from its filesystem path
func loadProjectFromPath(id, projectPath string, addedAt interface{}) (*Project, error) {
	cfg, err := config.LoadConfig(projectPath)
	if err != nil {
		return nil, err
	}

	// Convert repos using GetRepos() which returns a map with repo names as keys
	reposMap := cfg.GetRepos()
	repos := make([]Repo, 0, len(reposMap))
	for repoName, repo := range reposMap {
		autoRefresh := true
		if repo.AutoRefresh != nil {
			autoRefresh = *repo.AutoRefresh
		}
		repos = append(repos, Repo{
			Name:        repoName,
			Path:        repo.Path,
			Git:         repo.Git,
			AutoRefresh: autoRefresh,
		})
	}

	// Get existing features (worktrees)
	features := listExistingFeatures(projectPath)

	project := &Project{
		ID:                  id,
		Name:                cfg.Name,
		Path:                projectPath,
		Repos:               repos,
		Features:            features,
		BasePort:            cfg.BasePort,
		DefaultBranchPrefix: cfg.DefaultBranchPrefix,
	}

	// Set addedAt if it's a time.Time
	if t, ok := addedAt.(interface{ IsZero() bool }); ok && !t.IsZero() {
		if tt, ok := addedAt.(interface{ UTC() interface{} }); ok {
			_ = tt // Just to suppress unused warning
		}
	}

	return project, nil
}

// listExistingFeatures returns the list of feature directories in the trees folder
func listExistingFeatures(projectPath string) []string {
	treesDir := filepath.Join(projectPath, "trees")
	features := []string{}

	entries, err := os.ReadDir(treesDir)
	if err != nil {
		return features
	}

	for _, entry := range entries {
		if entry.IsDir() && !isHiddenDir(entry.Name()) {
			features = append(features, entry.Name())
		}
	}

	return features
}

// isHiddenDir checks if a directory name is hidden (starts with .)
func isHiddenDir(name string) bool {
	return len(name) > 0 && name[0] == '.'
}
