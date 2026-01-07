package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/features"
)

var renameCmd = &cobra.Command{
	Use:   "rename <feature> <display-name>",
	Short: "Set or change the display name of a feature",
	Long: `Set or change the human-readable display name of a feature.

The display name is shown in status output and the UI as an alternative
to the technical feature identifier (directory/branch name).

Pass an empty string to clear the display name.

Examples:
  ramp rename my-feature "User Authentication Feature"
  ramp rename my-feature ""  # Clear display name`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		featureName := args[0]
		displayName := args[1]

		if err := runRename(featureName, displayName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}

func runRename(featureName, displayName string) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	projectDir, err := config.FindRampProject(wd)
	if err != nil {
		return err
	}

	// Verify feature exists
	treesDir := filepath.Join(projectDir, "trees", featureName)
	if _, err := os.Stat(treesDir); os.IsNotExist(err) {
		return fmt.Errorf("feature '%s' not found", featureName)
	}

	// Update metadata
	metadataStore, err := features.NewMetadataStore(projectDir)
	if err != nil {
		return fmt.Errorf("failed to initialize metadata store: %w", err)
	}

	if err := metadataStore.SetDisplayName(featureName, displayName); err != nil {
		return fmt.Errorf("failed to set display name: %w", err)
	}

	if displayName == "" {
		fmt.Printf("Cleared display name for '%s'\n", featureName)
	} else {
		fmt.Printf("Set display name for '%s' to '%s'\n", featureName, displayName)
	}

	return nil
}
