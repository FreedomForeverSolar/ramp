package uiapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"ramp/internal/config"
	"ramp/internal/git"

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

	// Send progress via WebSocket
	s.broadcast(WSMessage{
		Type:      "progress",
		Operation: "up",
		Message:   "Starting feature creation...",
	})

	// Create feature directory
	treesDir := filepath.Join(ref.Path, "trees", req.Name)
	if err := os.MkdirAll(treesDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create feature directory", err.Error())
		return
	}

	// Get repos map for name lookup
	repos := cfg.GetRepos()
	repoNames := []string{}
	i := 0
	total := len(repos)

	// Create worktrees for each repo
	for repoName, repo := range repos {
		repoNames = append(repoNames, repoName)

		s.broadcast(WSMessage{
			Type:       "progress",
			Operation:  "up",
			Message:    "Creating worktree for " + repoName + "...",
			Percentage: (i + 1) * 100 / total,
		})

		repoPath := filepath.Join(ref.Path, repo.Path, repoName)
		worktreePath := filepath.Join(treesDir, repoName)

		// Determine branch name
		branchName := req.Name
		if cfg.DefaultBranchPrefix != "" {
			branchName = cfg.DefaultBranchPrefix + req.Name
		}

		// Create the worktree
		if err := git.CreateWorktreeQuiet(repoPath, worktreePath, branchName, repoName); err != nil {
			s.broadcast(WSMessage{
				Type:      "error",
				Operation: "up",
				Message:   "Failed to create worktree for " + repoName + ": " + err.Error(),
			})
			writeError(w, http.StatusInternalServerError, "Failed to create worktree", err.Error())
			return
		}
		i++
	}

	s.broadcast(WSMessage{
		Type:       "complete",
		Operation:  "up",
		Message:    "Feature created successfully",
		Percentage: 100,
	})

	// Return the created feature
	feature := Feature{
		Name:                  req.Name,
		Repos:                 repoNames,
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

	s.broadcast(WSMessage{
		Type:      "progress",
		Operation: "down",
		Message:   "Starting feature deletion...",
	})

	treesDir := filepath.Join(ref.Path, "trees", name)
	repos := cfg.GetRepos()
	i := 0
	total := len(repos)

	// Remove worktrees for each repo
	for repoName, repo := range repos {
		s.broadcast(WSMessage{
			Type:       "progress",
			Operation:  "down",
			Message:    "Removing worktree for " + repoName + "...",
			Percentage: (i + 1) * 100 / total,
		})

		repoPath := filepath.Join(ref.Path, repo.Path, repoName)
		worktreePath := filepath.Join(treesDir, repoName)

		// Check for uncommitted changes
		hasChanges, _ := git.HasUncommittedChanges(worktreePath)
		if hasChanges {
			s.broadcast(WSMessage{
				Type:      "error",
				Operation: "down",
				Message:   "Uncommitted changes in " + repoName,
			})
			// Continue anyway for now - in a real implementation, we might want to prompt the user
		}

		// Remove the worktree
		if err := git.RemoveWorktreeQuiet(repoPath, worktreePath); err != nil {
			// Log but continue - the directory might not exist
			i++
			continue
		}

		// Determine branch name and try to delete it
		branchName := name
		if cfg.DefaultBranchPrefix != "" {
			branchName = cfg.DefaultBranchPrefix + name
		}
		git.DeleteBranchQuiet(repoPath, branchName)
		i++
	}

	// Remove the feature directory
	os.RemoveAll(treesDir)

	s.broadcast(WSMessage{
		Type:       "complete",
		Operation:  "down",
		Message:    "Feature deleted successfully",
		Percentage: 100,
	})

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
