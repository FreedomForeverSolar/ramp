package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/git"
	"ramp/internal/ports"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show project and repository status",
	Long: `Show comprehensive status information for the ramp project.

Displays current branch and status for all source repositories,
active features, and project configuration details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

type repoStatus struct {
	name               string
	path               string
	currentBranch      string
	hasUncommitted     bool
	remoteTrackingInfo string
	error              string
}

type featureInfo struct {
	name    string
	modTime time.Time
}

type featureWorktreeStatus struct {
	repoName           string
	branchName         string
	hasUncommitted     bool
	aheadCount         int
	behindCount        int
	isMerged           bool
	defaultBranch      string
	error              string
}

func runStatus() error {
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

	fmt.Printf("📦 Project: %s\n\n", cfg.Name)

	// Display source repositories status
	fmt.Println("📂 Source Repositories:")
	repos := cfg.GetRepos()
	if len(repos) == 0 {
		fmt.Println("   (no repositories configured)")
	} else {
		for name, repo := range repos {
			status := getRepoStatus(projectDir, name, repo)
			displayRepoStatus(status)
		}
	}

	fmt.Println()

	// Display project information
	err = displayProjectInfo(projectDir, cfg)
	if err != nil {
		return err
	}

	fmt.Println()

	// Display active features
	err = displayActiveFeatures(projectDir, cfg)
	if err != nil {
		return err
	}

	return nil
}

func getRepoStatus(projectDir, name string, repo *config.Repo) repoStatus {
	repoPath := repo.GetRepoPath(projectDir)

	status := repoStatus{
		name: name,
		path: repoPath,
	}

	// Check if repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		status.error = "repository not cloned"
		return status
	}

	// Get current branch
	currentBranch, err := git.GetCurrentBranch(repoPath)
	if err != nil {
		status.error = fmt.Sprintf("failed to get branch: %v", err)
		return status
	}
	status.currentBranch = currentBranch

	// Check for uncommitted changes
	hasUncommitted, err := git.HasUncommittedChanges(repoPath)
	if err != nil {
		status.error = fmt.Sprintf("failed to check uncommitted changes: %v", err)
		return status
	}
	status.hasUncommitted = hasUncommitted

	// Get remote tracking info
	remoteInfo, err := git.GetRemoteTrackingStatus(repoPath)
	if err != nil {
		// Don't treat this as an error, just no remote info
		status.remoteTrackingInfo = ""
	} else {
		status.remoteTrackingInfo = remoteInfo
	}

	return status
}

func displayRepoStatus(status repoStatus) {
	if status.error != "" {
		fmt.Printf("   ❌ %s (%s) - %s\n", status.name, status.path, status.error)
		return
	}

	statusIcon := "✅"
	if status.hasUncommitted {
		statusIcon = "⚠️"
	}

	fmt.Printf("   %s %s (%s)\n", statusIcon, status.name, status.path)
	fmt.Printf("       Branch: %s", status.currentBranch)

	if status.remoteTrackingInfo != "" {
		fmt.Printf(" %s", status.remoteTrackingInfo)
	}
	fmt.Println()

	if status.hasUncommitted {
		fmt.Println("       Status: uncommitted changes")
	} else {
		fmt.Println("       Status: clean")
	}
}

func displayProjectInfo(projectDir string, cfg *config.Config) error {
	fmt.Println("ℹ️  Project Info:")

	// Count active features
	treesDir := filepath.Join(projectDir, "trees")
	featureCount := 0
	if entries, err := os.ReadDir(treesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				featureCount++
			}
		}
	}

	fmt.Printf("   Active features: %d\n", featureCount)

	// Show port allocations only if explicitly configured in ramp.yaml
	if cfg.BasePort > 0 {
		portAlloc, err := ports.NewPortAllocations(projectDir, cfg.GetBasePort(), cfg.GetMaxPorts())
		if err != nil {
			fmt.Printf("   Port allocations: error loading (%v)\n", err)
		} else {
			allocations := portAlloc.ListAllocations()
			fmt.Printf("   Port allocations: %d in use (base: %d)\n", len(allocations), cfg.GetBasePort())
		}
	}

	return nil
}

func getFeatureWorktreeStatus(projectDir, featureName, repoName string, repo *config.Repo) featureWorktreeStatus {
	worktreePath := filepath.Join(projectDir, "trees", featureName, repoName)
	sourceRepoPath := repo.GetRepoPath(projectDir)

	status := featureWorktreeStatus{
		repoName: repoName,
	}

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		status.error = "worktree not found"
		return status
	}

	// Get branch name
	branchName, err := git.GetWorktreeBranch(worktreePath)
	if err != nil {
		status.error = fmt.Sprintf("failed to get branch: %v", err)
		return status
	}
	status.branchName = branchName

	// Get default branch from source repo
	defaultBranch, err := git.GetDefaultBranch(sourceRepoPath)
	if err != nil {
		status.error = fmt.Sprintf("failed to get default branch: %v", err)
		return status
	}
	status.defaultBranch = defaultBranch

	// Check for uncommitted changes
	hasUncommitted, err := git.HasUncommittedChanges(worktreePath)
	if err != nil {
		status.error = fmt.Sprintf("failed to check uncommitted changes: %v", err)
		return status
	}
	status.hasUncommitted = hasUncommitted

	// Get ahead/behind count compared to default branch
	ahead, behind, err := git.GetAheadBehindCount(worktreePath, defaultBranch)
	if err != nil {
		// Not a fatal error, just means we can't compare
		status.aheadCount = 0
		status.behindCount = 0
	} else {
		status.aheadCount = ahead
		status.behindCount = behind
	}

	// Check if merged into default branch
	isMerged, err := git.IsMergedInto(worktreePath, defaultBranch)
	if err != nil {
		// Not a fatal error
		status.isMerged = false
	} else {
		status.isMerged = isMerged
	}

	return status
}

func formatWorktreeStatus(status featureWorktreeStatus) string {
	if status.error != "" {
		return fmt.Sprintf("❌ %s", status.error)
	}

	var parts []string

	// Always show uncommitted changes first if present
	if status.hasUncommitted {
		parts = append(parts, "🟡 uncommitted")
	}

	// Check if merged
	if status.isMerged {
		parts = append(parts, "✔️ merged")
	} else {
		// Show ahead/behind status for unmerged branches
		if status.aheadCount > 0 && status.behindCount > 0 {
			parts = append(parts, fmt.Sprintf("🔼 %d ahead, 🔽 %d behind", status.aheadCount, status.behindCount))
		} else if status.aheadCount > 0 {
			parts = append(parts, fmt.Sprintf("🔼 %d ahead", status.aheadCount))
		} else if status.behindCount > 0 {
			parts = append(parts, fmt.Sprintf("🔽 %d behind", status.behindCount))
		}
	}

	// If nothing to report, it's clean and up to date
	if len(parts) == 0 {
		parts = append(parts, "✅ clean")
	}

	return strings.Join(parts, ", ")
}

func displayActiveFeatures(projectDir string, cfg *config.Config) error {
	treesDir := filepath.Join(projectDir, "trees")

	// Check if trees directory exists
	if _, err := os.Stat(treesDir); os.IsNotExist(err) {
		fmt.Println("🌿 Active Features:")
		fmt.Println("   (no features found)")
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
		fmt.Println("🌿 Active Features:")
		fmt.Println("   (no features found)")
		return nil
	}

	// Sort features by creation time (oldest first)
	sort.Slice(features, func(i, j int) bool {
		return features[i].modTime.Before(features[j].modTime)
	})

	fmt.Println("🌿 Active Features:")

	repos := cfg.GetRepos()
	for _, feature := range features {
		featureDir := filepath.Join(treesDir, feature.name)
		featureEntries, err := os.ReadDir(featureDir)
		if err != nil {
			fmt.Printf("   📁 %s\n", feature.name)
			fmt.Printf("      ⚠️  Error reading feature directory: %v\n", err)
			continue
		}

		// Check which repos have worktrees in this feature and get their statuses
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
			fmt.Printf("   📁 %s\n", feature.name)
			fmt.Println("      (no repository worktrees found)")
			continue
		}

		// Check if all worktrees are merged and have no uncommitted changes
		allMerged := true
		for _, status := range worktreeStatuses {
			if !status.isMerged || status.hasUncommitted {
				allMerged = false
				break
			}
		}

		// Display feature name with merged indicator if applicable
		if allMerged {
			fmt.Printf("   📁 %s [✔️ MERGED]\n", feature.name)
		} else {
			fmt.Printf("   📁 %s\n", feature.name)
		}

		// Display each worktree with its status
		for _, status := range worktreeStatuses {
			statusStr := formatWorktreeStatus(status)
			fmt.Printf("      └── %s (%s) - %s\n", status.repoName, status.branchName, statusStr)
		}
	}

	return nil
}

