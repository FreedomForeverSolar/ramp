package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"ramp/internal/config"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all current feature worktrees",
	Long: `List all current feature worktrees in the project.
	
Shows all features that have been created with 'ramp new' and displays
which repositories have active worktrees for each feature.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runList(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

type featureInfo struct {
	name    string
	modTime time.Time
}

func runList() error {
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

	treesDir := filepath.Join(projectDir, "trees")
	
	// Check if trees directory exists
	if _, err := os.Stat(treesDir); os.IsNotExist(err) {
		fmt.Printf("No features found for project '%s'\n", cfg.Name)
		fmt.Println("Use 'ramp new <feature-name>' to create a new feature")
		return nil
	}

	// Read all feature directories
	entries, err := os.ReadDir(treesDir)
	if err != nil {
		return fmt.Errorf("failed to read trees directory: %w", err)
	}

	// Collect feature info with creation times
	var features []featureInfo
	for _, entry := range entries {
		if entry.IsDir() {
			featurePath := filepath.Join(treesDir, entry.Name())
			stat, err := os.Stat(featurePath)
			if err != nil {
				continue // Skip entries we can't stat
			}
			features = append(features, featureInfo{
				name:    entry.Name(),
				modTime: stat.ModTime(),
			})
		}
	}

	if len(features) == 0 {
		fmt.Printf("No features found for project '%s'\n", cfg.Name)
		fmt.Println("Use 'ramp new <feature-name>' to create a new feature")
		return nil
	}

	// Sort features by creation time (oldest first)
	sort.Slice(features, func(i, j int) bool {
		return features[i].modTime.Before(features[j].modTime)
	})

	fmt.Printf("Active features for project '%s':\n\n", cfg.Name)

	repos := cfg.GetRepos()
	for _, feature := range features {
		fmt.Printf("📁 %s\n", feature.name)
		
		featureDir := filepath.Join(treesDir, feature.name)
		featureEntries, err := os.ReadDir(featureDir)
		if err != nil {
			fmt.Printf("   ⚠️  Error reading feature directory: %v\n", err)
			continue
		}

		// Check which repos have worktrees in this feature
		var repoWorktrees []string
		for _, entry := range featureEntries {
			if entry.IsDir() {
				repoName := entry.Name()
				if _, exists := repos[repoName]; exists {
					repoWorktrees = append(repoWorktrees, repoName)
				}
			}
		}

		if len(repoWorktrees) == 0 {
			fmt.Println("   (no repository worktrees found)")
		} else {
			for _, repoName := range repoWorktrees {
				fmt.Printf("   └── %s\n", repoName)
			}
		}
		fmt.Println()
	}

	return nil
}