package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/git"
	"ramp/internal/ui"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Clone all configured repositories from ramp.yaml",
	Long: `Clone all repositories specified in the .ramp/ramp.yaml configuration file
into their configured locations.

This command must be run from within a directory containing a .ramp/ramp.yaml file.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInstall(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}

// isProjectInstalled checks if all configured repositories are present
func isProjectInstalled(cfg *config.Config, projectDir string) bool {
	repos := cfg.GetRepos()
	for _, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		if !git.IsGitRepo(repoDir) {
			return false
		}
	}
	return true
}

// AutoInstallIfNeeded checks if the project repos are cloned, and if not, clones them
func AutoInstallIfNeeded(projectDir string, cfg *config.Config) error {
	if isProjectInstalled(cfg, projectDir) {
		return nil
	}

	progress := ui.NewProgress()
	progress.Info("🚀 Repositories not installed, running auto-installation...")
	progress.Stop()
	return runInstallForProject(projectDir, cfg)
}

func runInstall() error {
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

	return runInstallForProject(projectDir, cfg)
}

func runInstallForProject(projectDir string, cfg *config.Config) error {
	progress := ui.NewProgress()
	progress.Start(fmt.Sprintf("Installing repositories for ramp project '%s'", cfg.Name))
	progress.Success(fmt.Sprintf("Installing repositories for ramp project '%s'", cfg.Name))

	repos := cfg.GetRepos()
	progress.Info(fmt.Sprintf("Found %d repositories to clone", len(repos)))

	for name, repo := range repos {
		// Get the configured path for this repository
		repoDir := repo.GetRepoPath(projectDir)

		// Create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
			progress.Error(fmt.Sprintf("Failed to create directory %s", filepath.Dir(repoDir)))
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(repoDir), err)
		}

		if git.IsGitRepo(repoDir) {
			progress.Info(fmt.Sprintf("%s: already exists at %s, skipping", name, repoDir))
			continue
		}

		gitURL := repo.GetGitURL()
		progress.Info(fmt.Sprintf("%s: cloning from %s to %s", name, gitURL, repoDir))
		if err := git.Clone(gitURL, repoDir); err != nil {
			progress.Error(fmt.Sprintf("Failed to clone %s", name))
			return fmt.Errorf("failed to clone %s: %w", name, err)
		}
	}

	progress.Success("Installation complete!")
	return nil
}
