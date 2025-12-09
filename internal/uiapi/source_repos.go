package uiapi

import (
	"net/http"
	"strings"
	"sync"

	"ramp/internal/config"
	"ramp/internal/git"
	"ramp/internal/operations"

	"github.com/gorilla/mux"
)

// extractRepoName extracts the repository name from a git URL
func extractRepoName(repoPath string) string {
	// Handle git@github.com:owner/repo.git format
	if strings.Contains(repoPath, ":") {
		parts := strings.Split(repoPath, ":")
		if len(parts) > 1 {
			repoPath = parts[1]
		}
	}

	// Remove .git suffix
	repoPath = strings.TrimSuffix(repoPath, ".git")

	// Extract repo name from owner/repo format
	parts := strings.Split(repoPath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return repoPath
}

// GetSourceRepos returns git status for all source repositories in a project
func (s *Server) GetSourceRepos(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get project reference
	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", "")
		return
	}

	// Load project config
	cfg, err := config.LoadConfig(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load project config", err.Error())
		return
	}

	repos := cfg.Repos
	statuses := make([]SourceRepoStatus, len(repos))

	var wg sync.WaitGroup

	for i, repo := range repos {
		wg.Add(1)
		go func(idx int, r *config.Repo) {
			defer wg.Done()

			repoName := extractRepoName(r.Git)
			repoDir := r.GetRepoPath(ref.Path)
			status := SourceRepoStatus{
				Name:        repoName,
				IsInstalled: git.IsGitRepo(repoDir),
			}

			if !status.IsInstalled {
				statuses[idx] = status
				return
			}

			// Get current branch
			branch, err := git.GetCurrentBranch(repoDir)
			if err != nil {
				status.Error = "Failed to get branch"
				statuses[idx] = status
				return
			}
			status.Branch = branch

			// Get ahead/behind count compared to origin
			// First try origin/branch, fall back to just showing branch info
			remoteBranch := "origin/" + branch
			ahead, behind, err := git.GetAheadBehindCount(repoDir, remoteBranch)
			if err == nil {
				status.AheadCount = ahead
				status.BehindCount = behind
			}
			// If error (e.g., no remote tracking), just leave counts at 0

			statuses[idx] = status
		}(i, repo)
	}

	wg.Wait()

	writeJSON(w, http.StatusOK, SourceReposResponse{Repos: statuses})
}

// RefreshSourceRepos refreshes all source repositories
func (s *Server) RefreshSourceRepos(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get project reference
	ref, err := GetProjectRefByID(id)
	if err != nil || ref == nil {
		writeError(w, http.StatusNotFound, "Project not found", "")
		return
	}

	// Load project config
	cfg, err := config.LoadConfig(ref.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load project config", err.Error())
		return
	}

	// Create progress reporter that broadcasts via WebSocket
	progress := operations.NewWSProgressReporter("refresh", "source", s.broadcast)

	progress.Start("Refreshing source repositories...")

	// Run refresh
	results := operations.RefreshRepositories(operations.RefreshOptions{
		ProjectDir: ref.Path,
		Config:     cfg,
		Progress:   progress,
	})

	// Check for any failures
	hasWarnings := false
	for _, result := range results {
		if result.Status == "warning" {
			hasWarnings = true
			break
		}
	}

	if hasWarnings {
		progress.Warning("Refresh completed with warnings")
	} else {
		progress.Success("Source repositories refreshed")
	}

	// Signal completion so frontend stops spinner
	progress.Complete("Refresh complete")

	writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "Refresh completed"})
}
