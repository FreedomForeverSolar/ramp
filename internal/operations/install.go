package operations

import (
	"fmt"
	"os"
	"path/filepath"

	"ramp/internal/config"
	"ramp/internal/git"
)

// IsProjectInstalled checks if all configured repositories are present.
// This is used by both CLI and UI to validate before operations.
func IsProjectInstalled(cfg *config.Config, projectDir string) bool {
	repos := cfg.GetRepos()
	for _, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		if !git.IsGitRepo(repoDir) {
			return false
		}
	}
	return true
}

// InstallOptions configures the install operation.
type InstallOptions struct {
	ProjectDir string
	Config     *config.Config
	Progress   ProgressReporter
	Shallow    bool
}

// InstallResult contains the results of installation.
type InstallResult struct {
	ClonedRepos  []string
	SkippedRepos []string
}

// Install clones all configured repositories.
// This is the core business logic used by both CLI and UI.
func Install(opts InstallOptions) (*InstallResult, error) {
	projectDir := opts.ProjectDir
	cfg := opts.Config
	progress := opts.Progress

	progress.Start(fmt.Sprintf("Installing repositories for ramp project '%s'", cfg.Name))

	repos := cfg.GetRepos()
	progress.Info(fmt.Sprintf("Found %d repositories to clone", len(repos)))

	result := &InstallResult{
		ClonedRepos:  []string{},
		SkippedRepos: []string{},
	}

	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)

		// Create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
			progress.Error(fmt.Sprintf("Failed to create directory %s", filepath.Dir(repoDir)))
			return nil, fmt.Errorf("failed to create directory %s: %w", filepath.Dir(repoDir), err)
		}

		if git.IsGitRepo(repoDir) {
			progress.Info(fmt.Sprintf("%s: already exists at %s, skipping", name, repoDir))
			result.SkippedRepos = append(result.SkippedRepos, name)
			continue
		}

		gitURL := repo.GetGitURL()
		progress.Info(fmt.Sprintf("%s: cloning from %s to %s", name, gitURL, repoDir))
		if err := git.Clone(gitURL, repoDir, opts.Shallow); err != nil {
			progress.Error(fmt.Sprintf("Failed to clone %s", name))
			return nil, fmt.Errorf("failed to clone %s: %w", name, err)
		}
		result.ClonedRepos = append(result.ClonedRepos, name)
	}

	progress.Complete("Installation complete!")
	return result, nil
}
