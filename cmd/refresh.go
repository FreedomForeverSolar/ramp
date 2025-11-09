package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/git"
	"ramp/internal/ui"
)

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Update all source repositories by pulling changes from their remotes",
	Long: `Update all source repositories by pulling changes from their remotes.

This command will:
1. Fetch all remotes for each configured repository
2. Pull changes if the current branch has a remote tracking branch
3. Report status for repositories without remote tracking branches

This is useful when the source repositories have been updated (either locally or remotely)
and you want to pull down the latest changes.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runRefresh(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)
}

func runRefresh() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	projectDir, err := config.FindRampProject(wd)
	if err != nil {
		return err
	}

	cfg, err := config.LoadConfig(projectDir)
	if err != nil {
		return err
	}

	// Auto-install if needed
	if err := AutoInstallIfNeeded(projectDir, cfg); err != nil {
		return fmt.Errorf("auto-installation failed: %w", err)
	}

	progress := ui.NewProgress()
	progress.Start(fmt.Sprintf("Refreshing repositories for project '%s'", cfg.Name))

	repos := cfg.GetRepos()
	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		RefreshRepository(repoDir, name, progress)
	}

	progress.Success("Refresh complete!")
	return nil
}

// RefreshRepository refreshes a single repository by fetching and pulling changes
func RefreshRepository(repoDir, name string, progress *ui.ProgressUI) error {
	if !git.IsGitRepo(repoDir) {
		progress.Warning(fmt.Sprintf("%s: not a git repository, skipping", name))
		return nil
	}

	// Get current branch
	currentBranch, err := git.GetCurrentBranch(repoDir)
	if err != nil {
		progress.Warning(fmt.Sprintf("%s: failed to get current branch: %v", name, err))
		return nil
	}

	// Fetch all remotes first (use quiet version to avoid creating another spinner)
	progress.Update(fmt.Sprintf("Refreshing %s: fetching from remotes", name))
	if err := git.FetchAllQuiet(repoDir); err != nil {
		progress.Warning(fmt.Sprintf("%s: fetch failed: %v", name, err))
		return nil
	}

	// Check if current branch has a remote tracking branch
	hasRemote, err := git.HasRemoteTrackingBranch(repoDir)
	if err != nil {
		progress.Warning(fmt.Sprintf("%s: failed to check remote tracking branch: %v", name, err))
		return nil
	}

	if hasRemote {
		progress.Update(fmt.Sprintf("Refreshing %s: pulling changes", name))
		if err := git.PullQuiet(repoDir); err != nil {
			progress.Warning(fmt.Sprintf("%s: pull failed: %v", name, err))
			return nil
		}
		progress.Info(fmt.Sprintf("%s: ✅ updated", name))
	} else {
		progress.Info(fmt.Sprintf("%s: branch %s has no remote tracking branch, skipped pull", name, currentBranch))
	}

	return nil
}