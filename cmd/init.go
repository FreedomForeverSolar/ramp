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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a ramp project by cloning all configured repositories",
	Long: `Initialize a ramp project by reading the .ramp/ramp.yaml configuration file
and cloning all specified repositories into their configured locations.

This command must be run from within a directory containing a .ramp/ramp.yaml file.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInit(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// isProjectInitialized checks if all configured repositories are present
func isProjectInitialized(cfg *config.Config, projectDir string) bool {
	repos := cfg.GetRepos()
	for _, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		if !git.IsGitRepo(repoDir) {
			return false
		}
	}
	return true
}

// autoInitializeIfNeeded checks if the project is initialized, and if not, runs initialization
func autoInitializeIfNeeded(projectDir string, cfg *config.Config) error {
	if isProjectInitialized(cfg, projectDir) {
		return nil
	}

	progress := ui.NewProgress()
	progress.Info("🚀 Project not initialized, running auto-initialization...")
	progress.Stop()
	return runInitForProject(projectDir, cfg)
}

func runInit() error {
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

	return runInitForProject(projectDir, cfg)
}

func runInitForProject(projectDir string, cfg *config.Config) error {
	progress := ui.NewProgress()
	progress.Start(fmt.Sprintf("Initializing ramp project '%s'", cfg.Name))
	progress.Success(fmt.Sprintf("Initializing ramp project '%s'", cfg.Name))
	
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

	progress.Success("Initialization complete!")
	return nil
}