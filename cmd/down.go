package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/operations"
	"ramp/internal/ui"
)

var downCmd = &cobra.Command{
	Use:   "down [feature-name]",
	Short: "Clean up a feature branch by removing worktrees and branches",
	Long: `Clean up a feature branch by:
1. Running the cleanup script (if configured)
2. Removing worktree directories from trees/<feature-name>/
3. Removing the feature branches that were created
4. Prompting for confirmation if there are uncommitted changes

If no feature name is provided, ramp will attempt to auto-detect the feature
based on your current working directory.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		featureName := ""
		if len(args) > 0 {
			featureName = strings.TrimRight(args[0], "/")
		}
		if err := runDown(featureName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}

func runDown(featureName string) error {
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

	// Auto-prompt for local config if needed
	if err := EnsureLocalConfig(projectDir, cfg); err != nil {
		return fmt.Errorf("failed to configure local preferences: %w", err)
	}

	// Auto-detect feature name if not provided
	if featureName == "" {
		detected, err := config.DetectFeatureFromWorkingDir(projectDir)
		if err != nil {
			return fmt.Errorf("failed to detect feature from working directory: %w", err)
		}
		if detected != "" {
			featureName = detected
			fmt.Printf("Auto-detected feature: %s\n", featureName)
		} else {
			return fmt.Errorf("no feature name provided and could not auto-detect from current directory")
		}
	}

	treesDir := filepath.Join(projectDir, "trees", featureName)

	// Check for uncommitted changes BEFORE starting spinner (so prompt is visible)
	force := false
	if _, err := os.Stat(treesDir); err == nil {
		reposWithChanges, err := operations.CheckForUncommittedChanges(cfg, treesDir)
		if err != nil {
			return fmt.Errorf("failed to check for uncommitted changes: %w", err)
		}

		if len(reposWithChanges) > 0 {
			for _, name := range reposWithChanges {
				progress := ui.NewProgress()
				progress.Warning(fmt.Sprintf("Uncommitted changes found in %s", name))
				progress.Stop()
			}
			// In non-interactive mode, auto-confirm
			if !NonInteractive && !confirmDeletion(featureName) {
				fmt.Println("Cleanup cancelled.")
				return nil
			}
			force = true // User confirmed (or non-interactive), skip check in Down()
		}
	}

	// Call operations.Down() with CLI progress reporter
	_, err = operations.Down(operations.DownOptions{
		FeatureName: featureName,
		ProjectDir:  projectDir,
		Config:      cfg,
		Progress:    operations.NewCLIProgressReporter(),
		Force:       force,
	})

	return err
}

func confirmDeletion(featureName string) bool {
	fmt.Printf("\nThere are uncommitted changes in one or more repositories.\n")
	fmt.Printf("Are you sure you want to delete feature '%s'? This will permanently lose uncommitted changes. (y/N): ", featureName)

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "y" || input == "yes"
}
