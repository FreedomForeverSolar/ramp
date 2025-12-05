package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/operations"
	"ramp/internal/ui"
)

var prefixFlag string
var noPrefixFlag bool
var targetFlag string
var fromFlag string
var refreshFlag bool
var noRefreshFlag bool

var upCmd = &cobra.Command{
	Use:   "up [feature-name]",
	Short: "Create a new feature branch with git worktrees for all repositories",
	Long: `Create a new feature branch by creating git worktrees for all repositories
from their configured locations. This creates isolated working directories for each repo
in the trees/<feature-name>/ directory.

By default, new feature branches are created from the default branch. Use the --target
flag to create the feature from a different source:
  - Existing feature name: ramp up new-feature --target my-existing-feature
  - Local branch name: ramp up new-feature --target feature/my-branch
  - Remote branch name: ramp up new-feature --target origin/feature/my-branch

Use the --from flag to create from a remote branch with automatic naming:
  - Remote branch: ramp up --from claude/feature-123
    Creates trees/feature-123/ with branch claude/feature-123 from origin/claude/feature-123
  - Override name: ramp up my-name --from claude/feature-123
    Creates trees/my-name/ with branch claude/feature-123 from origin/claude/feature-123

The operation is atomic - if any step fails, all successful operations will be
rolled back to ensure no partial feature state remains.

After creating worktrees, runs any setup script specified in the configuration.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var featureName string
		var derivedPrefix string
		var derivedTarget string

		// Handle --from flag
		if fromFlag != "" {
			// Parse the from flag to extract prefix and feature name
			lastSlash := strings.LastIndex(fromFlag, "/")
			if lastSlash == -1 {
				// No slash found - entire string is feature name, no prefix
				derivedPrefix = ""
				if len(args) == 0 {
					featureName = fromFlag
				} else {
					featureName = strings.TrimRight(args[0], "/")
				}
			} else {
				// Found slash - split into prefix and feature name
				derivedPrefix = fromFlag[:lastSlash+1] // Include trailing slash
				derivedName := fromFlag[lastSlash+1:]
				if len(args) == 0 {
					featureName = derivedName
				} else {
					featureName = strings.TrimRight(args[0], "/")
				}
			}

			// Always prepend origin/ to the from value for the target
			derivedTarget = "origin/" + fromFlag
		} else {
			// Traditional usage - feature name is required
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Error: feature-name is required when not using --from flag\n")
				os.Exit(1)
			}
			featureName = strings.TrimRight(args[0], "/")
			derivedPrefix = prefixFlag
			derivedTarget = targetFlag
		}

		if err := runUp(featureName, derivedPrefix, derivedTarget); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVar(&prefixFlag, "prefix", "", "Override the branch prefix (defaults to config default_branch_prefix)")
	upCmd.Flags().BoolVar(&noPrefixFlag, "no-prefix", false, "Disable branch prefix for this feature (mutually exclusive with --prefix)")
	upCmd.Flags().StringVar(&targetFlag, "target", "", "Create feature from existing feature name, local branch, or remote branch")
	upCmd.Flags().StringVar(&fromFlag, "from", "", "Create from remote branch with automatic prefix/name derivation (mutually exclusive with --target, --prefix, --no-prefix)")
	upCmd.Flags().BoolVar(&refreshFlag, "refresh", false, "Force refresh all repositories before creating feature (overrides auto_refresh config)")
	upCmd.Flags().BoolVar(&noRefreshFlag, "no-refresh", false, "Skip refresh for all repositories (overrides auto_refresh config)")
}

func runUp(featureName, prefix, target string) error {
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

	// Validate that feature name doesn't contain slashes
	if strings.Contains(featureName, "/") {
		return fmt.Errorf("feature name cannot contain slashes - use --prefix flag to create nested branch names (e.g., 'ramp up my-feature --prefix epic/' creates branch 'epic/my-feature')")
	}

	// Validate that --refresh and --no-refresh are not both specified
	if refreshFlag && noRefreshFlag {
		return fmt.Errorf("cannot specify both --refresh and --no-refresh flags")
	}

	// Validate that --prefix and --no-prefix are not both specified
	if prefixFlag != "" && noPrefixFlag {
		return fmt.Errorf("cannot specify both --prefix and --no-prefix flags")
	}

	// Validate that --from is mutually exclusive with --target, --prefix, and --no-prefix
	if fromFlag != "" {
		if targetFlag != "" {
			return fmt.Errorf("cannot specify both --from and --target flags")
		}
		if prefixFlag != "" {
			return fmt.Errorf("cannot specify both --from and --prefix flags")
		}
		if noPrefixFlag {
			return fmt.Errorf("cannot specify both --from and --no-prefix flags")
		}
	}

	// Auto-install if needed
	if err := AutoInstallIfNeeded(projectDir, cfg); err != nil {
		return fmt.Errorf("auto-installation failed: %w", err)
	}

	// Auto-prompt for local config if needed
	if err := EnsureLocalConfig(projectDir, cfg); err != nil {
		return fmt.Errorf("failed to configure local preferences: %w", err)
	}

	// Auto-refresh repositories based on flags and config
	repos := cfg.GetRepos()

	// Determine if we should refresh based on flags and config
	shouldRefreshRepos := false
	if noRefreshFlag {
		shouldRefreshRepos = false
	} else if refreshFlag {
		shouldRefreshRepos = true
	} else {
		for _, repo := range repos {
			if repo.ShouldAutoRefresh() {
				shouldRefreshRepos = true
				break
			}
		}
	}

	// Create a progress instance for refresh
	progress := ui.NewProgress()

	if shouldRefreshRepos {
		progress.Start("Auto-refreshing repositories before creating feature")

		reposToRefresh := make(map[string]*config.Repo)
		for name, repo := range repos {
			shouldRefreshThisRepo := false
			if refreshFlag {
				shouldRefreshThisRepo = true
			} else {
				shouldRefreshThisRepo = repo.ShouldAutoRefresh()
			}

			if shouldRefreshThisRepo {
				reposToRefresh[name] = repo
			} else {
				progress.Info(fmt.Sprintf("%s: auto-refresh disabled, skipping", name))
			}
		}

		if len(reposToRefresh) > 0 {
			results := RefreshRepositoriesParallel(projectDir, reposToRefresh, progress)

			for _, result := range results {
				switch result.status {
				case "success":
					progress.Info(fmt.Sprintf("%s: ✅ %s", result.name, result.message))
				case "warning":
					progress.Warning(fmt.Sprintf("%s: %s", result.name, result.message))
				case "skipped":
					progress.Info(fmt.Sprintf("%s: %s", result.name, result.message))
				}
			}
		}

		progress.Success("Auto-refresh completed")
	}

	// Call operations.Up() with CLI progress reporter
	result, err := operations.Up(operations.UpOptions{
		FeatureName:  featureName,
		ProjectDir:   projectDir,
		Config:       cfg,
		Progress:     operations.NewCLIProgressReporter(),
		Prefix:       prefix,
		NoPrefix:     noPrefixFlag,
		Target:       target,
		ForceRefresh: refreshFlag,
		SkipRefresh:  noRefreshFlag,
	})

	if err != nil {
		return err
	}

	fmt.Printf("Feature '%s' created at %s\n", result.FeatureName, result.TreesDir)
	return nil
}

