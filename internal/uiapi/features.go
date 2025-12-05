package uiapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"ramp/internal/config"
	"ramp/internal/git"
	"ramp/internal/operations"

	"github.com/gorilla/mux"
)

// ListFeatures returns all features for a project
func (s *Server) ListFeatures(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", id)
		return
	}

	features, err := getProjectFeatures(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list features", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, FeaturesResponse{Features: features})
}

// CreateFeature creates a new feature (ramp up)
func (s *Server) CreateFeature(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req CreateFeatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Feature name is required", "")
		return
	}

	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", id)
		return
	}

	// Load project config
	cfg, err := config.LoadConfig(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load project config", err.Error())
		return
	}

	// Create progress reporter that broadcasts to WebSocket
	progress := operations.NewWSProgressReporter("up", func(msg interface{}) {
		s.broadcast(msg)
	})

	// Call operations.Up() with WebSocket progress reporter
	result, err := operations.Up(operations.UpOptions{
		FeatureName: req.Name,
		ProjectDir:  ref.Path,
		Config:      cfg,
		Progress:    progress,
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create feature", err.Error())
		return
	}

	// Return the created feature
	feature := Feature{
		Name:                  result.FeatureName,
		Repos:                 result.Repos,
		HasUncommittedChanges: false,
	}

	writeJSON(w, http.StatusCreated, feature)
}

// DeleteFeature deletes a feature (ramp down)
func (s *Server) DeleteFeature(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	name := vars["name"]

	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", id)
		return
	}

	// Load project config
	cfg, err := config.LoadConfig(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load project config", err.Error())
		return
	}

	// Create progress reporter that broadcasts to WebSocket
	progress := operations.NewWSProgressReporter("down", func(msg interface{}) {
		s.broadcast(msg)
	})

	// Call operations.Down() with WebSocket progress reporter
	// Force=true because the UI shows uncommitted changes warnings in the feature list
	_, err = operations.Down(operations.DownOptions{
		FeatureName: name,
		ProjectDir:  ref.Path,
		Config:      cfg,
		Progress:    progress,
		Force:       true,
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete feature", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "Feature deleted"})
}

// getProjectFeatures returns detailed feature information for a project
func getProjectFeatures(projectPath string) ([]Feature, error) {
	treesDir := filepath.Join(projectPath, "trees")
	features := []Feature{}

	entries, err := os.ReadDir(treesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return features, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() || isHiddenDir(entry.Name()) {
			continue
		}

		featurePath := filepath.Join(treesDir, entry.Name())

		// Get repos in this feature
		repoEntries, err := os.ReadDir(featurePath)
		if err != nil {
			continue
		}

		repos := []string{}
		hasUncommitted := false

		for _, repoEntry := range repoEntries {
			if !repoEntry.IsDir() || isHiddenDir(repoEntry.Name()) {
				continue
			}
			repos = append(repos, repoEntry.Name())

			// Check for uncommitted changes
			repoPath := filepath.Join(featurePath, repoEntry.Name())
			hasChanges, _ := git.HasUncommittedChanges(repoPath)
			if hasChanges {
				hasUncommitted = true
			}
		}

		// Get creation time from directory info
		info, _ := entry.Info()
		var created = info.ModTime()

		features = append(features, Feature{
			Name:                  entry.Name(),
			Repos:                 repos,
			Created:               created,
			HasUncommittedChanges: hasUncommitted,
		})
	}

	return features, nil
}
