package operations

import (
	"fmt"
	"sync"

	"ramp/internal/config"
	"ramp/internal/git"
)

// RefreshResult holds the result of refreshing a repository.
type RefreshResult struct {
	Name    string
	Status  string // "success", "warning", or "skipped"
	Message string
}

// RefreshOptions configures the refresh operation.
type RefreshOptions struct {
	ProjectDir string
	Config     *config.Config
	Progress   ProgressReporter
	RepoFilter map[string]bool // Optional: only refresh these repos (nil = all)
}

// RefreshRepositories refreshes multiple repositories concurrently.
// This is the core business logic used by both CLI and UI.
func RefreshRepositories(opts RefreshOptions) []RefreshResult {
	projectDir := opts.ProjectDir
	cfg := opts.Config
	progress := opts.Progress

	repos := cfg.GetRepos()

	// Filter repos if specified
	reposToRefresh := make(map[string]*config.Repo)
	for name, repo := range repos {
		if opts.RepoFilter == nil || opts.RepoFilter[name] {
			reposToRefresh[name] = repo
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]RefreshResult, 0, len(reposToRefresh))

	for name, repo := range reposToRefresh {
		wg.Add(1)
		go func(repoName string, r *config.Repo) {
			defer wg.Done()

			repoDir := r.GetRepoPath(projectDir)
			result := RefreshResult{Name: repoName}

			// Check if it's a git repo
			if !git.IsGitRepo(repoDir) {
				result.Status = "warning"
				result.Message = "not a git repository, skipping"
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			// Get current branch
			currentBranch, err := git.GetCurrentBranch(repoDir)
			if err != nil {
				result.Status = "warning"
				result.Message = fmt.Sprintf("failed to get current branch: %v", err)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			// Fetch all remotes
			if err := git.FetchAllQuiet(repoDir); err != nil {
				result.Status = "warning"
				result.Message = fmt.Sprintf("fetch failed: %v", err)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			// Check if current branch has a remote tracking branch
			hasRemote, err := git.HasRemoteTrackingBranch(repoDir)
			if err != nil {
				result.Status = "warning"
				result.Message = fmt.Sprintf("failed to check remote tracking branch: %v", err)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			if hasRemote {
				// Pull changes
				if err := git.PullQuiet(repoDir); err != nil {
					result.Status = "warning"
					result.Message = fmt.Sprintf("pull failed: %v", err)
					mu.Lock()
					results = append(results, result)
					mu.Unlock()
					return
				}
				result.Status = "success"
				result.Message = "updated"
			} else {
				result.Status = "skipped"
				result.Message = fmt.Sprintf("branch %s has no remote tracking branch, skipped pull", currentBranch)
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(name, repo)
	}

	wg.Wait()

	// Report results via progress
	for _, result := range results {
		switch result.Status {
		case "success":
			progress.Info(fmt.Sprintf("%s: %s", result.Name, result.Message))
		case "warning":
			progress.Warning(fmt.Sprintf("%s: %s", result.Name, result.Message))
		case "skipped":
			progress.Info(fmt.Sprintf("%s: %s", result.Name, result.Message))
		}
	}

	return results
}
