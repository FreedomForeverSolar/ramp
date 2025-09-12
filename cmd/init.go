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
and cloning all specified repositories into the source/ directory.

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

	sourceDir := filepath.Join(projectDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		return fmt.Errorf("failed to create source directory: %w", err)
	}

	fmt.Printf("Initializing ramp project '%s'\n", cfg.Name)
	repos := cfg.GetRepos()
	fmt.Printf("Found %d repositories to clone\n", len(repos))

	for name, repo := range repos {
		repoDir := filepath.Join(sourceDir, name)
		
		if git.IsGitRepo(repoDir) {
			fmt.Printf("  %s: already exists, skipping\n", name)
			continue
		}

		fmt.Printf("  %s: cloning from %s\n", name, repo.Path)
		if err := git.Clone(repo.Path, repoDir); err != nil {
			return fmt.Errorf("failed to clone %s: %w", name, err)
		}
	}

	fmt.Println("✅ Initialization complete!")
	return nil
}