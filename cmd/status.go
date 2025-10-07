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
	diffStats          *git.DiffStats
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

	// Get diff stats if there are uncommitted changes
	if hasUncommitted {
		diffStats, err := git.GetDiffStats(worktreePath)
		if err != nil {
			// Not fatal, just skip the stats
			status.diffStats = nil
		} else {
			status.diffStats = diffStats
		}
	}

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

func formatCompactStatus(status featureWorktreeStatus, showAll bool) string {
	if status.error != "" {
		return fmt.Sprintf("◉ error: %s", status.error)
	}

	var symbol string
	var parts []string

	// Determine symbol based on state
	hasLocalWork := status.hasUncommitted || status.aheadCount > 0

	if hasLocalWork {
		symbol = "◉"
	} else {
		symbol = "○"
	}

	// Show uncommitted changes
	if status.hasUncommitted {
		if status.diffStats != nil && (status.diffStats.FilesChanged > 0 || status.diffStats.Insertions > 0 || status.diffStats.Deletions > 0) {
			diffParts := []string{}
			if status.diffStats.FilesChanged > 0 {
				diffParts = append(diffParts, fmt.Sprintf("+%d", status.diffStats.FilesChanged))
			}
			if status.diffStats.Insertions > 0 {
				diffParts = append(diffParts, fmt.Sprintf("+%d", status.diffStats.Insertions))
			}
			if status.diffStats.Deletions > 0 {
				diffParts = append(diffParts, fmt.Sprintf("-%d", status.diffStats.Deletions))
			}
			parts = append(parts, strings.Join(diffParts, " "))
		} else {
			parts = append(parts, "uncommitted")
		}
	}

	// Show ahead status - this indicates work that needs attention
	if status.aheadCount > 0 {
		parts = append(parts, fmt.Sprintf("%d ahead", status.aheadCount))
	}

	// Don't show "merged" or "behind" status in needs attention section
	// It's confusing and not actionable - you only care about uncommitted/ahead

	// If no interesting status and not showing all, return empty
	if len(parts) == 0 && !showAll {
		return ""
	}

	// If showing all and no status, just show symbol
	if len(parts) == 0 {
		return symbol
	}

	return fmt.Sprintf("%s %s", symbol, strings.Join(parts, ", "))
}

func needsAttention(statuses []featureWorktreeStatus) bool {
	for _, status := range statuses {
		// Has uncommitted changes
		if status.hasUncommitted {
			return true
		}
		// Has commits ahead (not merged yet)
		if status.aheadCount > 0 && !status.isMerged {
			return true
		}
	}
	return false
}

func isMerged(statuses []featureWorktreeStatus) bool {
	for _, status := range statuses {
		// Must have had commits (was ahead) and now merged
		if status.aheadCount == 0 && status.isMerged && status.behindCount > 0 && !status.hasUncommitted {
			continue
		}
		return false
	}
	return true
}

func isClean(statuses []featureWorktreeStatus) bool {
	for _, status := range statuses {
		// Never had any commits (0 ahead, 0 behind or just behind)
		// No uncommitted changes
		if status.hasUncommitted || status.aheadCount > 0 {
			return false
		}
	}
	return true
}

func displayActiveFeatures(projectDir string, cfg *config.Config) error {
	treesDir := filepath.Join(projectDir, "trees")

	// Check if trees directory exists
	if _, err := os.Stat(treesDir); os.IsNotExist(err) {
		fmt.Println("🌿 No active features")
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
				continue
			}
			features = append(features, featureInfo{
				name:    entry.Name(),
				modTime: stat.ModTime(),
			})
		}
	}

	if len(features) == 0 {
		fmt.Println("🌿 No active features")
		return nil
	}

	// Sort features by creation time (oldest first)
	sort.Slice(features, func(i, j int) bool {
		return features[i].modTime.Before(features[j].modTime)
	})

	// Categorize features
	repos := cfg.GetRepos()
	var inFlightFeatures []struct {
		name     string
		statuses []featureWorktreeStatus
	}
	var mergedFeatures []string
	var cleanFeatures []string

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

		if needsAttention(worktreeStatuses) {
			inFlightFeatures = append(inFlightFeatures, struct {
				name     string
				statuses []featureWorktreeStatus
			}{feature.name, worktreeStatuses})
		} else if isMerged(worktreeStatuses) {
			mergedFeatures = append(mergedFeatures, feature.name)
		} else if isClean(worktreeStatuses) {
			cleanFeatures = append(cleanFeatures, feature.name)
		}
	}

	// Print summary
	totalFeatures := len(features)
	inFlightCount := len(inFlightFeatures)
	mergedCount := len(mergedFeatures)
	cleanCount := len(cleanFeatures)

	summaryParts := []string{fmt.Sprintf("%d active", totalFeatures)}
	if inFlightCount > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d in flight", inFlightCount))
	}
	if mergedCount > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d merged", mergedCount))
	}
	if cleanCount > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d clean", cleanCount))
	}
	fmt.Printf("🌿 Features: %s\n\n", strings.Join(summaryParts, "  •  "))

	// Display in-flight features
	if len(inFlightFeatures) > 0 {
		fmt.Println("━━━ IN FLIGHT ━━━\n")
		for _, feature := range inFlightFeatures {
			fmt.Printf("%s\n", feature.name)
			for _, status := range feature.statuses {
				// Only show repos with local work (uncommitted or ahead)
				hasLocalWork := status.hasUncommitted || status.aheadCount > 0
				if !hasLocalWork {
					continue
				}
				statusStr := formatCompactStatus(status, false)
				if statusStr != "" {
					fmt.Printf("  %s: %s\n", status.repoName, statusStr)
				}
			}
			fmt.Println()
		}
	}

	// Display merged features
	if len(mergedFeatures) > 0 {
		fmt.Printf("━━━ MERGED (%d) ━━━\n", len(mergedFeatures))
		const maxWidth = 70
		line := ""
		for i, name := range mergedFeatures {
			if i > 0 {
				line += ", "
			}
			if len(line)+len(name) > maxWidth && line != "" {
				fmt.Println(line)
				line = name
			} else {
				line += name
			}
		}
		if line != "" {
			fmt.Println(line)
		}
		fmt.Println()
	}

	// Display clean features
	if len(cleanFeatures) > 0 {
		fmt.Printf("━━━ CLEAN (%d) ━━━\n", len(cleanFeatures))
		const maxWidth = 70
		line := ""
		for i, name := range cleanFeatures {
			if i > 0 {
				line += ", "
			}
			if len(line)+len(name) > maxWidth && line != "" {
				fmt.Println(line)
				line = name
			} else {
				line += name
			}
		}
		if line != "" {
			fmt.Println(line)
		}
		fmt.Println()
	}


	return nil
}

