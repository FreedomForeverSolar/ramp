package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"ramp/internal/config"
)

var showFlag bool
var resetFlag bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure local preferences for this project",
	Long: `Configure local preferences defined in the project's ramp.yaml.

Without flags, this command will interactively prompt you to set preferences.

Flags:
  --show   Display current local preferences
  --reset  Delete local preferences (will re-prompt on next command)`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().BoolVar(&showFlag, "show", false, "Display current local preferences")
	configCmd.Flags().BoolVar(&resetFlag, "reset", false, "Delete local preferences")
}

func runConfig() error {
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

	// Handle --show flag
	if showFlag {
		return showLocalConfig(projectDir, cfg)
	}

	// Handle --reset flag
	if resetFlag {
		return resetLocalConfig(projectDir)
	}

	// Default: run interactive prompts
	return runInteractiveConfig(projectDir, cfg)
}

func showLocalConfig(projectDir string, cfg *config.Config) error {
	localCfg, err := config.LoadLocalConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load local config: %w", err)
	}

	if localCfg == nil {
		fmt.Println("⚠️  No local preferences configured yet")
		if cfg.HasPrompts() {
			fmt.Println("\n💡 Run 'ramp config' to set up your preferences")
		} else {
			fmt.Println("\n💡 No prompts defined in ramp.yaml")
		}
		return nil
	}

	fmt.Println("📋 Current local preferences:")
	fmt.Println()

	for key, value := range localCfg.Preferences {
		fmt.Printf("  %s = %s\n", key, value)
	}

	return nil
}

func resetLocalConfig(projectDir string) error {
	localPath := projectDir + "/.ramp/local.yaml"

	// Check if file exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		fmt.Println("⚠️  No local preferences to reset")
		return nil
	}

	// Delete the file
	if err := os.Remove(localPath); err != nil {
		return fmt.Errorf("failed to remove local config: %w", err)
	}

	fmt.Println("✅ Local preferences reset")
	fmt.Println("\n💡 Run 'ramp config' or use 'ramp up' to set up preferences again")

	return nil
}

func runInteractiveConfig(projectDir string, cfg *config.Config) error {
	// Check if prompts are defined
	if !cfg.HasPrompts() {
		fmt.Println("⚠️  No prompts defined in ramp.yaml")
		fmt.Println("\n💡 Add a 'prompts' section to your ramp.yaml to configure preferences")
		return nil
	}

	// Check if already configured
	localCfg, err := config.LoadLocalConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load local config: %w", err)
	}

	if localCfg != nil {
		fmt.Println("⚙️  Local preferences already configured")
		fmt.Println("\n💡 Use 'ramp config --show' to view or 'ramp config --reset' to reconfigure")
		return nil
	}

	// Run interactive prompts
	fmt.Println("⚙️  Let's set up your local preferences...")
	fmt.Println()

	preferences, err := promptForPreferences(cfg.Prompts)
	if err != nil {
		return fmt.Errorf("failed to collect preferences: %w", err)
	}

	// Save to local.yaml
	newLocalCfg := &config.LocalConfig{
		Preferences: preferences,
	}

	if err := config.SaveLocalConfig(newLocalCfg, projectDir); err != nil {
		return fmt.Errorf("failed to save local config: %w", err)
	}

	fmt.Println("\n✅ Preferences saved to .ramp/local.yaml")

	return nil
}

// promptForPreferences shows interactive prompts and returns the collected preferences
func promptForPreferences(prompts []*config.Prompt) (map[string]string, error) {
	preferences := make(map[string]string)

	for _, prompt := range prompts {
		// Build options for the select
		options := make([]huh.Option[string], len(prompt.Options))
		for i, opt := range prompt.Options {
			options[i] = huh.NewOption(opt.Label, opt.Value)
		}

		// Create the form with a single select
		var selected string
		if prompt.Default != "" {
			selected = prompt.Default
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(prompt.Question).
					Options(options...).
					Value(&selected),
			),
		)

		if err := form.Run(); err != nil {
			return nil, err
		}

		preferences[prompt.Name] = selected
	}

	return preferences, nil
}

// EnsureLocalConfig checks if local config is needed and prompts if necessary.
// This is called automatically by commands that run user scripts (up, down, run).
// Returns nil if no prompting is needed or if prompting succeeds.
// In non-interactive mode (--yes/-y), skips prompts and continues without setting preferences.
func EnsureLocalConfig(projectDir string, cfg *config.Config) error {
	// If no prompts defined, nothing to do (backwards compatible)
	if !cfg.HasPrompts() {
		return nil
	}

	// Check if local config already exists
	localCfg, err := config.LoadLocalConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load local config: %w", err)
	}

	// If local config exists, nothing to do
	if localCfg != nil {
		return nil
	}

	// In non-interactive mode, skip prompts and continue without preferences
	if NonInteractive {
		return nil
	}

	// Need to prompt for preferences
	fmt.Println("\n⚙️  Local preferences not found. Let's set them up!")
	fmt.Println()

	preferences, err := promptForPreferences(cfg.Prompts)
	if err != nil {
		return fmt.Errorf("failed to collect preferences: %w", err)
	}

	// Save to local.yaml
	newLocalCfg := &config.LocalConfig{
		Preferences: preferences,
	}

	if err := config.SaveLocalConfig(newLocalCfg, projectDir); err != nil {
		return fmt.Errorf("failed to save local config: %w", err)
	}

	fmt.Println("\n✅ Preferences saved to .ramp/local.yaml")
	fmt.Println()

	return nil
}

// GetLocalEnvVars loads local preferences and returns them as environment variables.
// Returns empty map if no local config exists (not an error).
func GetLocalEnvVars(projectDir string) (map[string]string, error) {
	localCfg, err := config.LoadLocalConfig(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load local config: %w", err)
	}

	if localCfg == nil {
		return make(map[string]string), nil
	}

	return localCfg.Preferences, nil
}
