package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/git"
	"ramp/internal/ports"
	"ramp/internal/ui"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Clean up merged feature branches automatically",
	Long: `Clean up all feature branches that have been merged into their default branch.

This command:
1. Scans all features in the trees/ directory
2. Identifies features that have been merged (based on git merge-base)
3. Shows a summary of merged features
4. Asks for confirmation once
5. Removes all confirmed merged features (worktrees, branches, and allocated resources)

Features categorized as "CLEAN" (never had any commits) are not removed by this command.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runPrune(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}

type featureToClean struct {
	name     string
	modTime  time.Time
	statuses []featureWorktreeStatus
}

func runPrune() error {
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

	progress := ui.NewProgress()
	progress.Start("Analyzing features...")

	// Find all merged features
	mergedFeatures, err := findMergedFeatures(projectDir, cfg)
	if err != nil {
		progress.Error("Failed to analyze features")
		return err
	}

	progress.Success("Analyzing features...")
	fmt.Println()

	// If no merged features, exit early
	if len(mergedFeatures) == 0 {
		fmt.Println("✓ No merged features found to clean up")
		return nil
	}

	// Display summary
	displayMergedFeaturesSummary(mergedFeatures)

	// Ask for confirmation
	if !confirmPrune(len(mergedFeatures)) {
		fmt.Println("\nPrune cancelled.")
		return nil
	}

	fmt.Println()

	// Create a single progress spinner for all cleanup operations
	cleanupProgress := ui.NewProgress()

	// Clean up each merged feature
	successCount := 0
	failedFeatures := []string{}

	for _, feature := range mergedFeatures {
		if err := cleanupFeatureWithProgress(projectDir, cfg, feature.name, cleanupProgress); err != nil {
			failedFeatures = append(failedFeatures, fmt.Sprintf("%s: %v", feature.name, err))
		} else {
			successCount++
		}
	}

	// Display final summary
	fmt.Println()
	displayCleanupSummary(len(mergedFeatures), successCount, failedFeatures)

	return nil
}

func findMergedFeatures(projectDir string, cfg *config.Config) ([]featureToClean, error) {
	treesDir := filepath.Join(projectDir, "trees")

	// Check if trees directory exists
	if _, err := os.Stat(treesDir); os.IsNotExist(err) {
		return []featureToClean{}, nil
	}

	// Read all feature directories
	entries, err := os.ReadDir(treesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read trees directory: %w", err)
	}

	// Collect feature info with creation times
	var features []featureInfo
	for _, entry := range entries {
		if entry.IsDir() {
			featurePath := filepath.Join(treesDir, entry.Name())
			stat, err := os.Stat(featurePath)
			if err != nil {
				continue
			}
			features = append(features, featureInfo{
				name:    entry.Name(),
				modTime: stat.ModTime(),
			})
		}
	}

	// Sort features by creation time (oldest first)
	sort.Slice(features, func(i, j int) bool {
		return features[i].modTime.Before(features[j].modTime)
	})

	// Categorize features and collect merged ones
	repos := cfg.GetRepos()
	var mergedFeatures []featureToClean

	for _, feature := range features {
		featureDir := filepath.Join(treesDir, feature.name)
		featureEntries, err := os.ReadDir(featureDir)
		if err != nil {
			continue
		}

		var worktreeStatuses []featureWorktreeStatus
		for _, entry := range featureEntries {
			if entry.IsDir() {
				repoName := entry.Name()
				if repo, exists := repos[repoName]; exists {
					status := getFeatureWorktreeStatus(projectDir, feature.name, repoName, repo)
					worktreeStatuses = append(worktreeStatuses, status)
				}
			}
		}

		if len(worktreeStatuses) == 0 {
			continue
		}

		// Only include features that are merged (not clean, not in-flight)
		if isMerged(worktreeStatuses) {
			mergedFeatures = append(mergedFeatures, featureToClean{
				name:     feature.name,
				modTime:  feature.modTime,
				statuses: worktreeStatuses,
			})
		}
	}

	return mergedFeatures, nil
}

func displayMergedFeaturesSummary(features []featureToClean) {
	fmt.Printf("🧹 Found %d merged feature%s to clean up:\n\n", len(features), pluralize(len(features)))

	for _, feature := range features {
		fmt.Printf("  • %s\n", feature.name)
	}
}

func confirmPrune(count int) bool {
	fmt.Printf("\nRemove all %d merged feature%s? This will delete worktrees, branches, and release ports. (y/N): ", count, pluralize(count))

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "y" || input == "yes"
}

func cleanupFeature(projectDir string, cfg *config.Config, featureName string) error {
	progress := ui.NewProgress()
	return cleanupFeatureWithProgress(projectDir, cfg, featureName, progress)
}

func cleanupFeatureWithProgress(projectDir string, cfg *config.Config, featureName string, progress *ui.ProgressUI) error {
	progress.Start(fmt.Sprintf("Cleaning up %s", featureName))

	// Get config prefix for fallback when branch detection fails
	configPrefix := cfg.GetBranchPrefix()

	treesDir := filepath.Join(projectDir, "trees", featureName)

	// Check if trees directory exists
	treesDirExists := true
	if _, err := os.Stat(treesDir); os.IsNotExist(err) {
		treesDirExists = false

		// Check if any worktrees or branches exist for this feature
		// This distinguishes between orphaned worktrees and non-existent features
		repos := cfg.GetRepos()
		featureExists := false
		for name, repo := range repos {
			repoDir := repo.GetRepoPath(projectDir)
			worktreeDir := filepath.Join(treesDir, name)

			if git.IsGitRepo(repoDir) {
				// Check if worktree is registered or branch exists
				if git.WorktreeRegistered(repoDir, worktreeDir) {
					featureExists = true
					break
				}

				// Check if branch exists
				branchName := configPrefix + featureName
				if exists, _ := git.LocalBranchExists(repoDir, branchName); exists {
					featureExists = true
					break
				}
			}
		}

		if !featureExists {
			progress.Error(fmt.Sprintf("Cleaning up %s", featureName))
			return fmt.Errorf("trees directory does not exist")
		}
	}

	// Note: We don't check for uncommitted changes here because merged features
	// shouldn't have meaningful uncommitted changes, and we already confirmed the prune

	// Run cleanup script if configured and directory exists
	if cfg.Cleanup != "" && treesDirExists {
		if err := runCleanupScriptQuiet(projectDir, treesDir, cfg.Cleanup); err != nil {
			progress.Warning(fmt.Sprintf("%s: cleanup script failed", featureName))
			// Continue anyway
		}
	}

	// Remove git worktrees and branches
	repos := cfg.GetRepos()
	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		worktreeDir := filepath.Join(treesDir, name)

		if git.IsGitRepo(repoDir) {
			var branchName string

			// Try to detect the actual branch name from the worktree
			if _, err := os.Stat(worktreeDir); err == nil {
				if detectedBranch, err := git.GetWorktreeBranch(worktreeDir); err == nil {
					branchName = detectedBranch
				} else {
					// Fallback to constructed branch name
					branchName = configPrefix + featureName
				}
			} else {
				// No worktree directory exists, use fallback branch name
				branchName = configPrefix + featureName
			}

			// Always try to remove worktree (even if directory is missing)
			// git worktree remove --force works for orphaned worktrees
			if err := git.RemoveWorktree(repoDir, worktreeDir); err != nil {
				progress.Warning(fmt.Sprintf("%s/%s: failed to remove worktree", featureName, name))
				// If worktree removal failed, prune orphaned worktrees before deleting branch
				// This handles cases where the worktree directory was manually deleted
				_ = git.PruneWorktrees(repoDir)
			}

			// Delete branch
			if err := git.DeleteBranch(repoDir, branchName); err != nil {
				progress.Warning(fmt.Sprintf("%s/%s: failed to delete branch", featureName, name))
				// Continue anyway
			}

			// Prune stale remote tracking branches
			if err := git.FetchPrune(repoDir); err != nil {
				// Ignore prune errors - not critical
			}
		}
	}

	// Release allocated port
	portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
	if err == nil {
		_ = portAllocations.ReleasePort(featureName)
	}

	// Remove trees directory if it exists
	if treesDirExists {
		if err := os.RemoveAll(treesDir); err != nil {
			progress.Error(fmt.Sprintf("Cleaning up %s", featureName))
			return fmt.Errorf("failed to remove trees directory: %w", err)
		}
	}

	progress.Success(fmt.Sprintf("Cleaned up %s", featureName))
	return nil
}

func runCleanupScriptQuiet(projectDir, treesDir, cleanupScript string) error {
	scriptPath := filepath.Join(projectDir, ".ramp", cleanupScript)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("cleanup script not found: %s", scriptPath)
	}

	// Extract feature name from treesDir path
	featureName := filepath.Base(treesDir)

	// Reuse the cleanup script logic from down.go
	// We'll run it quietly without progress UI since we're in a batch operation
	return runCleanupScriptWithoutProgress(projectDir, treesDir, featureName, cleanupScript)
}

func runCleanupScriptWithoutProgress(projectDir, treesDir, featureName, cleanupScript string) error {
	scriptPath := filepath.Join(projectDir, ".ramp", cleanupScript)

	cmd := createCleanupCommand(projectDir, treesDir, featureName, scriptPath)

	// Run quietly - capture output but don't display
	output := &strings.Builder{}
	cmd.Stdout = output
	cmd.Stderr = output

	return cmd.Run()
}

func createCleanupCommand(projectDir, treesDir, featureName, scriptPath string) *exec.Cmd {
	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = treesDir

	// Set up environment variables
	cmd.Env = append(os.Environ(), fmt.Sprintf("RAMP_PROJECT_DIR=%s", projectDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_TREES_DIR=%s", treesDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_WORKTREE_NAME=%s", featureName))

	// Add RAMP_PORT environment variable
	cfg, err := config.LoadConfig(projectDir)
	if err == nil {
		portAllocations, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
		if err == nil {
			if port, exists := portAllocations.GetPort(featureName); exists {
				cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_PORT=%d", port))
			}
		}

		repos := cfg.GetRepos()
		for name, repo := range repos {
			envVarName := config.GenerateEnvVarName(name)
			repoPath := repo.GetRepoPath(projectDir)
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVarName, repoPath))
		}
	}

	return cmd
}

func displayCleanupSummary(total, success int, failed []string) {
	if len(failed) == 0 {
		fmt.Printf("✓ Successfully cleaned up all %d merged feature%s\n", total, pluralize(total))
	} else {
		fmt.Printf("⚠️  Cleaned up %d of %d feature%s\n", success, total, pluralize(total))
		fmt.Println("\nFailed to clean up:")
		for _, failure := range failed {
			fmt.Printf("  • %s\n", failure)
		}
	}
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
