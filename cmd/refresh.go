package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/git"
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

	// Auto-initialize if needed
	if err := autoInitializeIfNeeded(projectDir, cfg); err != nil {
		return fmt.Errorf("auto-initialization failed: %w", err)
	}

	fmt.Printf("Refreshing repositories for project '%s'\n", cfg.Name)

	repos := cfg.GetRepos()
	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)

		if !git.IsGitRepo(repoDir) {
			fmt.Printf("  %s: not a git repository, skipping\n", name)
			continue
		}

		fmt.Printf("  %s: ", name)

		// Get current branch
		currentBranch, err := git.GetCurrentBranch(repoDir)
		if err != nil {
			fmt.Printf("failed to get current branch: %v\n", err)
			continue
		}

		// Fetch all remotes first
		fmt.Printf("fetching... ")
		if err := git.FetchAll(repoDir); err != nil {
			fmt.Printf("fetch failed: %v\n", err)
			continue
		}

		// Check if current branch has a remote tracking branch
		hasRemote, err := git.HasRemoteTrackingBranch(repoDir)
		if err != nil {
			fmt.Printf("failed to check remote tracking branch: %v\n", err)
			continue
		}

		if hasRemote {
			fmt.Printf("pulling %s... ", currentBranch)
			if err := git.Pull(repoDir); err != nil {
				fmt.Printf("pull failed: %v\n", err)
				continue
			}
			fmt.Printf("✅ updated\n")
		} else {
			fmt.Printf("branch %s has no remote tracking branch, skipped pull\n", currentBranch)
		}
	}

	fmt.Printf("✅ Refresh complete!\n")
	return nil
}