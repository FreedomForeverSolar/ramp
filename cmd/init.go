package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/git"
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

	fmt.Printf("Initializing ramp project '%s'\n", cfg.Name)
	repos := cfg.GetRepos()
	fmt.Printf("Found %d repositories to clone\n", len(repos))

	for name, repo := range repos {
		// Get the configured path for this repository
		repoDir := repo.GetRepoPath(projectDir)
		
		// Create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(repoDir), err)
		}
		
		if git.IsGitRepo(repoDir) {
			fmt.Printf("  %s: already exists at %s, skipping\n", name, repoDir)
			continue
		}

		gitURL := repo.GetGitURL()
		fmt.Printf("  %s: cloning from %s to %s\n", name, gitURL, repoDir)
		if err := git.Clone(gitURL, repoDir); err != nil {
			return fmt.Errorf("failed to clone %s: %w", name, err)
		}
	}

	fmt.Println("✅ Initialization complete!")
	return nil
}