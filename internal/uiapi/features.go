package uiapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ramp/internal/config"
	"ramp/internal/features"
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

	// Acquire project lock to prevent concurrent feature operations
	// This serializes create/delete to avoid git worktree conflicts
	unlock := s.acquireProjectLock(id)
	defer unlock()

	var req CreateFeatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Process FromBranch if set (like CLI --from flag)
	var featureName, prefix, target string
	if req.FromBranch != "" {
		// Parse the from branch to derive prefix and feature name
		lastSlash := strings.LastIndex(req.FromBranch, "/")
		if lastSlash == -1 {
			// No slash found - entire string is feature name, no prefix
			prefix = ""
			if req.Name == "" {
				featureName = req.FromBranch
			} else {
				featureName = req.Name
			}
		} else {
			// Found slash - split into prefix and feature name
			prefix = req.FromBranch[:lastSlash+1] // Include trailing slash
			derivedName := req.FromBranch[lastSlash+1:]
			if req.Name == "" {
				featureName = derivedName
			} else {
				featureName = req.Name
			}
		}
		// Always prepend origin/ to the from value for the target
		target = "origin/" + req.FromBranch
	} else {
		// Standard flow
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "Feature name is required", "")
			return
		}
		featureName = req.Name
		prefix = req.Prefix
		target = req.Target
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
	// Include feature name as target for message filtering
	progress := operations.NewWSProgressReporter("up", featureName, func(msg interface{}) {
		s.broadcast(msg)
	})

	// Create output streamer for setup script output
	output := operations.NewWSOutputStreamerWithContext("up", featureName, "", func(msg interface{}) {
		s.broadcast(msg)
	})

	// Call operations.Up() with WebSocket progress reporter
	// All options from CreateFeatureRequest are passed through to operations.Up()
	// This ensures CLI and UI have feature parity
	// Note: Auto-refresh respects per-repo config by default (no explicit flag needed)
	result, err := operations.Up(operations.UpOptions{
		FeatureName: featureName,
		ProjectDir:  ref.Path,
		Config:      cfg,
		Progress:    progress,
		Output:      output, // Stream setup script output
		// Branch configuration
		Prefix:   prefix,
		NoPrefix: req.NoPrefix,
		Target:   target,
		// Pre-operation behavior - AutoInstall defaults to true for UI (matches CLI behavior)
		AutoInstall:  true,
		ForceRefresh: req.ForceRefresh,
		SkipRefresh:  req.SkipRefresh,
		// Display name (optional human-readable name)
		DisplayName: req.DisplayName,
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create feature", err.Error())
		return
	}

	// Return the created feature
	feature := Feature{
		Name:                  result.FeatureName,
		DisplayName:           result.DisplayName,
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

	// Acquire project lock to prevent concurrent feature operations
	// This serializes create/delete to avoid git worktree conflicts
	unlock := s.acquireProjectLock(id)
	defer unlock()

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
	// Include feature name as target for message filtering
	progress := operations.NewWSProgressReporter("down", name, func(msg interface{}) {
		s.broadcast(msg)
	})

	// Call operations.Down() with WebSocket progress reporter
	// Force=true because the UI handles uncommitted changes confirmation in the dialog
	// AutoInstall=true to match CLI behavior
	_, err = operations.Down(operations.DownOptions{
		FeatureName: name,
		ProjectDir:  ref.Path,
		Config:      cfg,
		Progress:    progress,
		Force:       true,
		AutoInstall: true,
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete feature", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "Feature deleted"})
}

// RenameFeature updates the display name of a feature
func (s *Server) RenameFeature(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	name := vars["name"]

	var req RenameFeatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", id)
		return
	}

	// Verify feature exists
	treesDir := filepath.Join(ref.Path, "trees", name)
	if _, err := os.Stat(treesDir); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "Feature not found", name)
		return
	}

	// Update metadata
	metadataStore, err := features.NewMetadataStore(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize metadata store", err.Error())
		return
	}

	if err := metadataStore.SetDisplayName(name, req.DisplayName); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to set display name", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "Display name updated"})
}

// getProjectFeatures returns detailed feature information for a project
func getProjectFeatures(projectPath string) ([]Feature, error) {
	treesDir := filepath.Join(projectPath, "trees")
	featuresList := []Feature{}

	entries, err := os.ReadDir(treesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return featuresList, nil
		}
		return nil, err
	}

	// Load feature metadata for display names
	metadataStore, _ := features.NewMetadataStore(projectPath)

	// Try to load project config for detailed status
	// If config loading fails, we'll fall back to basic feature info
	cfg, cfgErr := config.LoadConfig(projectPath)
	var repos map[string]*config.Repo
	if cfgErr == nil {
		repos = cfg.GetRepos()

		// Fetch all repos in parallel for accurate ahead/behind info
		var wg sync.WaitGroup
		for _, repo := range repos {
			wg.Add(1)
			go func(r *config.Repo) {
				defer wg.Done()
				repoPath := r.GetRepoPath(projectPath)
				if _, err := os.Stat(repoPath); err == nil && git.IsGitRepo(repoPath) {
					_ = git.FetchAllQuiet(repoPath)
				}
			}(repo)
		}
		wg.Wait()
	}

	// Collect features with detailed status
	for _, entry := range entries {
		if !entry.IsDir() || isHiddenDir(entry.Name()) {
			continue
		}

		featurePath := filepath.Join(treesDir, entry.Name())
		featureName := entry.Name()

		// Get repos in this feature
		repoEntries, err := os.ReadDir(featurePath)
		if err != nil {
			continue
		}

		repoNames := []string{}
		hasUncommitted := false
		var worktreeStatuses []FeatureWorktreeStatus

		for _, repoEntry := range repoEntries {
			if !repoEntry.IsDir() || isHiddenDir(repoEntry.Name()) {
				continue
			}
			repoName := repoEntry.Name()
			repoNames = append(repoNames, repoName)

			// Get detailed worktree status if we have repo config
			if repos != nil {
				if repo, exists := repos[repoName]; exists {
					status := getFeatureWorktreeStatus(projectPath, featureName, repoName, repo)
					worktreeStatuses = append(worktreeStatuses, status)
					if status.HasUncommitted {
						hasUncommitted = true
					}
					continue
				}
			}
			// Fallback for repos not in config or when config not available
			repoPath := filepath.Join(featurePath, repoName)
			hasChanges, _ := git.HasUncommittedChanges(repoPath)
			if hasChanges {
				hasUncommitted = true
			}
		}

		// Get creation time from directory info
		info, _ := entry.Info()
		created := info.ModTime()

		// Categorize the feature
		category := categorizeFeature(worktreeStatuses)

		// Get display name from metadata
		var displayName string
		if metadataStore != nil {
			displayName = metadataStore.GetDisplayName(featureName)
		}

		featuresList = append(featuresList, Feature{
			Name:                  featureName,
			DisplayName:           displayName,
			Repos:                 repoNames,
			Created:               created,
			HasUncommittedChanges: hasUncommitted,
			Category:              category,
			WorktreeStatuses:      worktreeStatuses,
		})
	}

	return featuresList, nil
}

// getFeatureWorktreeStatus collects detailed status for a single repo worktree
func getFeatureWorktreeStatus(projectDir, featureName, repoName string, repo *config.Repo) FeatureWorktreeStatus {
	worktreePath := filepath.Join(projectDir, "trees", featureName, repoName)
	sourceRepoPath := repo.GetRepoPath(projectDir)

	status := FeatureWorktreeStatus{
		RepoName: repoName,
	}

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		status.Error = "worktree not found"
		return status
	}

	// Get branch name
	branchName, err := git.GetWorktreeBranch(worktreePath)
	if err != nil {
		status.Error = fmt.Sprintf("failed to get branch: %v", err)
		return status
	}
	status.BranchName = branchName

	// Get default branch from source repo
	defaultBranch, err := git.GetDefaultBranch(sourceRepoPath)
	if err != nil {
		status.Error = fmt.Sprintf("failed to get default branch: %v", err)
		return status
	}

	// Check for uncommitted changes
	hasUncommitted, err := git.HasUncommittedChanges(worktreePath)
	if err != nil {
		status.Error = fmt.Sprintf("failed to check uncommitted changes: %v", err)
		return status
	}
	status.HasUncommitted = hasUncommitted

	// Get diff stats and status stats if there are uncommitted changes
	if hasUncommitted {
		if diffStats, err := git.GetDiffStats(worktreePath); err == nil && diffStats != nil {
			status.DiffStats = &DiffStats{
				FilesChanged: diffStats.FilesChanged,
				Insertions:   diffStats.Insertions,
				Deletions:    diffStats.Deletions,
			}
		}

		if statusStats, err := git.GetStatusStats(worktreePath); err == nil && statusStats != nil {
			status.StatusStats = &StatusStats{
				UntrackedFiles: statusStats.UntrackedFiles,
				StagedFiles:    statusStats.StagedFiles,
				ModifiedFiles:  statusStats.ModifiedFiles,
			}
		}
	}

	// Get ahead/behind count compared to default branch
	ahead, behind, err := git.GetAheadBehindCount(worktreePath, defaultBranch)
	if err != nil {
		status.AheadCount = 0
		status.BehindCount = 0
	} else {
		status.AheadCount = ahead
		status.BehindCount = behind
	}

	// Check if merged into default branch
	isMerged, err := git.IsMergedInto(worktreePath, defaultBranch)
	if err != nil {
		status.IsMerged = false
	} else {
		status.IsMerged = isMerged
	}

	return status
}

// categorizeFeature determines the category for a feature based on worktree statuses
func categorizeFeature(statuses []FeatureWorktreeStatus) string {
	if len(statuses) == 0 {
		return "clean"
	}

	if needsAttention(statuses) {
		return "in_flight"
	}
	if isMerged(statuses) {
		return "merged"
	}
	if isClean(statuses) {
		return "clean"
	}
	return "clean"
}

// needsAttention returns true if any worktree has uncommitted changes or unpushed commits
func needsAttention(statuses []FeatureWorktreeStatus) bool {
	for _, status := range statuses {
		if status.HasUncommitted {
			return true
		}
		if status.AheadCount > 0 && !status.IsMerged {
			return true
		}
	}
	return false
}

// isMerged returns true if all worktrees are merged and clean
func isMerged(statuses []FeatureWorktreeStatus) bool {
	anyBehind := false

	for _, status := range statuses {
		if status.HasUncommitted || status.AheadCount > 0 {
			return false
		}
		if !status.IsMerged {
			return false
		}
		if status.BehindCount > 0 {
			anyBehind = true
		}
	}

	// All repos are merged and clean, AND at least one is behind default
	return anyBehind
}

// isClean returns true if all worktrees are clean (no uncommitted, no ahead)
func isClean(statuses []FeatureWorktreeStatus) bool {
	for _, status := range statuses {
		if status.HasUncommitted || status.AheadCount > 0 {
			return false
		}
	}
	return true
}

// PruneFeatures deletes all merged features for a project
func (s *Server) PruneFeatures(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Acquire project lock to prevent concurrent feature operations
	unlock := s.acquireProjectLock(id)
	defer unlock()

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

	// Get all features and identify merged ones
	features, err := getProjectFeatures(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list features", err.Error())
		return
	}

	var mergedFeatures []string
	for _, feature := range features {
		if feature.Category == "merged" {
			mergedFeatures = append(mergedFeatures, feature.Name)
		}
	}

	if len(mergedFeatures) == 0 {
		writeJSON(w, http.StatusOK, PruneResponse{
			Pruned:  []string{},
			Failed:  []PruneFailure{},
			Message: "No merged features to prune",
		})
		return
	}

	// Create progress reporter that broadcasts to WebSocket
	progress := operations.NewWSProgressReporter("prune", "", func(msg interface{}) {
		s.broadcast(msg)
	})

	// Delete each merged feature
	var pruned []string
	var failed []PruneFailure

	progress.Start("Pruning merged features...")

	for _, featureName := range mergedFeatures {
		progress.Update(fmt.Sprintf("Removing %s...", featureName))

		_, err := operations.Down(operations.DownOptions{
			FeatureName: featureName,
			ProjectDir:  ref.Path,
			Config:      cfg,
			Progress:    progress,
			Force:       true, // Skip confirmation - merged features are safe to delete
			AutoInstall: false,
		})

		if err != nil {
			failed = append(failed, PruneFailure{
				Name:  featureName,
				Error: err.Error(),
			})
		} else {
			pruned = append(pruned, featureName)
		}
	}

	progress.Complete("Prune complete")

	// Build response message
	var message string
	if len(failed) == 0 {
		message = fmt.Sprintf("Successfully pruned %d merged feature(s)", len(pruned))
	} else {
		message = fmt.Sprintf("Pruned %d feature(s), %d failed", len(pruned), len(failed))
	}

	writeJSON(w, http.StatusOK, PruneResponse{
		Pruned:  pruned,
		Failed:  failed,
		Message: message,
	})
}
