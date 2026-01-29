package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/operations"
)

var runCmd = &cobra.Command{
	Use:   "run <command-name> [feature-name] [-- args...]",
	Short: "Run a custom command defined in the configuration",
	Long: `Run a custom command defined in the ramp.yaml configuration.

If a feature name is provided, the command is executed from within that
feature's trees directory with access to feature-specific environment variables.

If no feature name is provided, ramp will attempt to auto-detect the feature
based on your current working directory. If not in a feature tree, the command
is executed from the source directory with access to source repository paths.

Arguments after -- are passed directly to the script as positional arguments
($1, $2, etc.) and also via the RAMP_ARGS environment variable.

Note: RAMP_ARGS is space-joined, so arguments containing spaces will lose
their boundaries. Use positional arguments ($1, $2, $@) for such cases.

Example:
  ramp run open my-feature    # Run 'open' command for 'my-feature'
  ramp run open               # Auto-detect feature from current directory
  ramp run deploy             # Run 'deploy' command against source repos
  ramp run check -- --cwd backend    # Pass args to the script
  ramp run test my-feature -- --all  # Feature name + args`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dashIndex := cmd.ArgsLenAtDash()

		var commandName, featureName string
		var scriptArgs []string

		if dashIndex == -1 {
			// No -- separator, use original logic
			commandName = args[0]
			if len(args) > 1 {
				featureName = strings.TrimRight(args[1], "/")
			}
		} else {
			// -- was used to separate script args
			// dashIndex indicates where -- appears in args:
			//   dashIndex == 1: "run cmd -- args" (no feature name)
			//   dashIndex > 1:  "run cmd feature -- args" (feature at args[1])
			commandName = args[0]
			if dashIndex > 1 {
				featureName = strings.TrimRight(args[1], "/")
			}
			scriptArgs = args[dashIndex:]
		}

		if err := runCustomCommand(commandName, featureName, scriptArgs); err != nil {
			// Don't print error for intentional cancellation (Ctrl+C)
			if errors.Is(err, operations.ErrCommandCancelled) {
				os.Exit(130) // Standard exit code for SIGINT (128 + 2)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runCustomCommand(commandName, featureName string, args []string) error {
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

	// Auto-prompt for local config if needed
	if err := EnsureLocalConfig(projectDir, cfg); err != nil {
		return fmt.Errorf("failed to configure local preferences: %w", err)
	}

	// Auto-install if needed
	if err := AutoInstallIfNeeded(projectDir, cfg); err != nil {
		return fmt.Errorf("auto-installation failed: %w", err)
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
		}
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Create cancel channel to signal command termination
	cancel := make(chan struct{})

	// Done channel to clean up signal goroutine on normal exit
	done := make(chan struct{})
	defer close(done)

	// Handle signals in goroutine
	go func() {
		select {
		case <-sigChan:
			close(cancel)
		case <-done:
			// Command completed normally, exit goroutine
		}
	}()

	// Use shared operations.RunCommand for consistent behavior with UI
	// This ensures hooks execute for both CLI and UI
	_, err = operations.RunCommand(operations.RunOptions{
		ProjectDir:  projectDir,
		Config:      cfg,
		CommandName: commandName,
		FeatureName: featureName,
		Args:        args,
		Progress:    operations.NewCLIProgressReporter(),
		Output:      &operations.CLIOutputStreamer{},
		Cancel:      cancel,
	})

	return err
}