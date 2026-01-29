package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ramp/internal/ui"
)

func Clone(repoURL, destDir string, shallow bool) error {
	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(destDir), err)
	}

	args := []string{"clone"}
	if shallow {
		args = append(args, "--depth", "1")
	}
	args = append(args, repoURL, destDir)

	prefix := "cloning"
	if shallow {
		prefix = "shallow cloning"
	}

	cmd := exec.Command("git", args...)
	if err := ui.RunCommandWithProgress(cmd, fmt.Sprintf("%s %s", prefix, repoURL)); err != nil {
		return fmt.Errorf("failed to clone %s to %s: %w", repoURL, destDir, err)
	}

	return nil
}

func CreateWorktreeFromSource(repoDir, worktreeDir, branchName, sourceBranch, repoName string) error {
	if err := os.MkdirAll(filepath.Dir(worktreeDir), 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(worktreeDir), err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreeDir); err == nil {
		return fmt.Errorf("worktree directory already exists: %s", worktreeDir)
	}

	// Check if target branch already exists locally
	localExists, err := LocalBranchExists(repoDir, branchName)
	if err != nil {
		return fmt.Errorf("failed to check if local branch exists: %w", err)
	}

	if localExists {
		return fmt.Errorf("branch %s already exists locally", branchName)
	}

	// Verify source branch exists
	if err := validateSourceBranch(repoDir, sourceBranch); err != nil {
		return fmt.Errorf("source branch validation failed: %w", err)
	}

	// Create new branch from source
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreeDir, sourceBranch)
	cmd.Dir = repoDir
	message := fmt.Sprintf("%s: creating worktree with new branch %s from %s", repoName, branchName, sourceBranch)

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to create worktree %s with branch %s from %s: %w", worktreeDir, branchName, sourceBranch, err)
	}

	return nil
}

func CreateWorktreeFromSourceQuiet(repoDir, worktreeDir, branchName, sourceBranch, repoName string) error {
	if err := os.MkdirAll(filepath.Dir(worktreeDir), 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(worktreeDir), err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreeDir); err == nil {
		return fmt.Errorf("worktree directory already exists: %s", worktreeDir)
	}

	// Check if target branch already exists locally
	localExists, err := LocalBranchExists(repoDir, branchName)
	if err != nil {
		return fmt.Errorf("failed to check if local branch exists: %w", err)
	}

	if localExists {
		return fmt.Errorf("branch %s already exists locally", branchName)
	}

	// Verify source branch exists
	if err := validateSourceBranch(repoDir, sourceBranch); err != nil {
		return fmt.Errorf("source branch validation failed: %w", err)
	}

	// Create new branch from source
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreeDir, sourceBranch)
	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create worktree %s with branch %s from %s: %w", worktreeDir, branchName, sourceBranch, err)
	}

	return nil
}

func CreateWorktree(repoDir, worktreeDir, branchName, repoName string) error {
	if err := os.MkdirAll(filepath.Dir(worktreeDir), 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(worktreeDir), err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreeDir); err == nil {
		return fmt.Errorf("worktree directory already exists: %s", worktreeDir)
	}

	// Check branch status
	localExists, err := LocalBranchExists(repoDir, branchName)
	if err != nil {
		return fmt.Errorf("failed to check if local branch exists: %w", err)
	}

	remoteExists, err := RemoteBranchExists(repoDir, branchName)
	if err != nil {
		return fmt.Errorf("failed to check if remote branch exists: %w", err)
	}

	var cmd *exec.Cmd
	var message string

	if localExists {
		// Use existing local branch
		cmd = exec.Command("git", "worktree", "add", worktreeDir, branchName)
		message = fmt.Sprintf("%s: creating worktree with existing local branch %s", repoName, branchName)
	} else if remoteExists {
		// Create local branch tracking the remote
		remoteBranch, err := getRemoteBranchName(repoDir, branchName)
		if err != nil {
			return fmt.Errorf("failed to get remote branch name: %w", err)
		}
		cmd = exec.Command("git", "worktree", "add", "-b", branchName, worktreeDir, remoteBranch)
		message = fmt.Sprintf("%s: creating worktree with existing remote branch %s", repoName, branchName)
	} else {
		// Create new branch
		cmd = exec.Command("git", "worktree", "add", "-b", branchName, worktreeDir)
		message = fmt.Sprintf("%s: creating worktree with new branch %s", repoName, branchName)
	}

	cmd.Dir = repoDir

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to create worktree %s with branch %s: %w", worktreeDir, branchName, err)
	}

	return nil
}

func CreateWorktreeQuiet(repoDir, worktreeDir, branchName, repoName string) error {
	if err := os.MkdirAll(filepath.Dir(worktreeDir), 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(worktreeDir), err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreeDir); err == nil {
		return fmt.Errorf("worktree directory already exists: %s", worktreeDir)
	}

	// Check branch status
	localExists, err := LocalBranchExists(repoDir, branchName)
	if err != nil {
		return fmt.Errorf("failed to check if local branch exists: %w", err)
	}

	remoteExists, err := RemoteBranchExists(repoDir, branchName)
	if err != nil {
		return fmt.Errorf("failed to check if remote branch exists: %w", err)
	}

	var cmd *exec.Cmd

	if localExists {
		// Use existing local branch
		cmd = exec.Command("git", "worktree", "add", worktreeDir, branchName)
	} else if remoteExists {
		// Create local branch tracking the remote
		remoteBranch, err := getRemoteBranchName(repoDir, branchName)
		if err != nil {
			return fmt.Errorf("failed to get remote branch name: %w", err)
		}
		cmd = exec.Command("git", "worktree", "add", "-b", branchName, worktreeDir, remoteBranch)
	} else {
		// Create new branch
		cmd = exec.Command("git", "worktree", "add", "-b", branchName, worktreeDir)
	}

	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create worktree %s with branch %s: %w", worktreeDir, branchName, err)
	}

	return nil
}

func getRemoteBranchName(repoDir, branchName string) (string, error) {
	// Get all remote branches and check for exact matches
	cmd := exec.Command("git", "--no-optional-locks", "branch", "-r")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip HEAD references
		if strings.Contains(line, "HEAD ->") {
			continue
		}
		// Check if this line matches "origin/branchName" exactly
		if line == "origin/"+branchName {
			return line, nil
		}
	}

	return "", fmt.Errorf("no remote branch found for %s", branchName)
}

func BranchExists(repoDir, branchName string) (bool, error) {
	local, err := LocalBranchExists(repoDir, branchName)
	if err != nil {
		return false, err
	}
	if local {
		return true, nil
	}

	return RemoteBranchExists(repoDir, branchName)
}

func LocalBranchExists(repoDir, branchName string) (bool, error) {
	cmd := exec.Command("git", "--no-optional-locks", "branch", "--list", branchName)
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(output)) != "", nil
}

func RemoteBranchExists(repoDir, branchName string) (bool, error) {
	// Get all remote branches and check for exact matches
	cmd := exec.Command("git", "--no-optional-locks", "branch", "-r")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip HEAD references
		if strings.Contains(line, "HEAD ->") {
			continue
		}
		// Check if this line matches "origin/branchName" exactly
		if line == "origin/"+branchName {
			return true, nil
		}
	}

	return false, nil
}

func HasUncommittedChanges(repoDir string) (bool, error) {
	cmd := exec.Command("git", "--no-optional-locks", "status", "--porcelain")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(output)) != "", nil
}

func RemoveWorktree(repoDir, worktreeDir string) error {
	cmd := exec.Command("git", "worktree", "remove", worktreeDir, "--force")
	cmd.Dir = repoDir
	message := fmt.Sprintf("removing worktree %s", worktreeDir)

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to remove worktree %s: %w", worktreeDir, err)
	}

	return nil
}

func RemoveWorktreeQuiet(repoDir, worktreeDir string) error {
	cmd := exec.Command("git", "worktree", "remove", worktreeDir, "--force")
	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove worktree %s: %w", worktreeDir, err)
	}

	return nil
}

func PruneWorktrees(repoDir string) error {
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to prune worktrees: %w", err)
	}

	return nil
}

func DeleteBranch(repoDir, branchName string) error {
	cmd := exec.Command("git", "branch", "-D", branchName)
	cmd.Dir = repoDir
	message := fmt.Sprintf("deleting branch %s", branchName)

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}

	return nil
}

func DeleteBranchQuiet(repoDir, branchName string) error {
	cmd := exec.Command("git", "branch", "-D", branchName)
	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}

	return nil
}

func GetWorktreeBranch(worktreeDir string) (string, error) {
	cmd := exec.Command("git", "--no-optional-locks", "symbolic-ref", "HEAD")
	cmd.Dir = worktreeDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get branch name from worktree: %w", err)
	}

	branchRef := strings.TrimSpace(string(output))
	// Remove "refs/heads/" prefix to get just the branch name
	if strings.HasPrefix(branchRef, "refs/heads/") {
		return strings.TrimPrefix(branchRef, "refs/heads/"), nil
	}

	return branchRef, nil
}

func GetCurrentBranch(repoDir string) (string, error) {
	cmd := exec.Command("git", "--no-optional-locks", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func FetchAll(repoDir string) error {
	cmd := exec.Command("git", "fetch", "--all")
	cmd.Dir = repoDir
	message := "fetching from all remotes"

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	return nil
}

func FetchAllQuiet(repoDir string) error {
	cmd := exec.Command("git", "fetch", "--all", "--quiet")
	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	return nil
}

func Pull(repoDir string) error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = repoDir
	message := "pulling changes"

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	return nil
}

func PullQuiet(repoDir string) error {
	cmd := exec.Command("git", "pull", "--quiet")
	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	return nil
}

func HasRemoteTrackingBranch(repoDir string) (bool, error) {
	cmd := exec.Command("git", "--no-optional-locks", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	cmd.Dir = repoDir

	_, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
			return false, nil
		}
		return false, fmt.Errorf("failed to check remote tracking branch: %w", err)
	}

	return true, nil
}

func IsGitRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}

func Checkout(repoDir, branchName string) error {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = repoDir
	message := fmt.Sprintf("checking out branch %s", branchName)

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	return nil
}

func CheckoutQuiet(repoDir, branchName string) error {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	return nil
}

func FetchBranch(repoDir, branchName string) error {
	cmd := exec.Command("git", "fetch", "origin", branchName)
	cmd.Dir = repoDir
	message := fmt.Sprintf("fetching branch %s from origin", branchName)

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to fetch branch %s: %w", branchName, err)
	}

	return nil
}

func FetchBranchQuiet(repoDir, branchName string) error {
	cmd := exec.Command("git", "fetch", "origin", branchName, "--quiet")
	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch branch %s: %w", branchName, err)
	}

	return nil
}

func FetchPrune(repoDir string) error {
	cmd := exec.Command("git", "fetch", "--prune")
	cmd.Dir = repoDir
	message := "pruning stale remote tracking branches"

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to prune remote tracking branches: %w", err)
	}

	return nil
}

func FetchPruneQuiet(repoDir string) error {
	cmd := exec.Command("git", "fetch", "--prune", "--quiet")
	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to prune remote tracking branches: %w", err)
	}

	return nil
}

func StashChanges(repoDir string) (bool, error) {
	// First check if there are changes to stash
	hasChanges, err := HasUncommittedChanges(repoDir)
	if err != nil {
		return false, err
	}

	if !hasChanges {
		return false, nil
	}

	cmd := exec.Command("git", "stash", "push", "-m", "ramp rebase stash")
	cmd.Dir = repoDir
	message := "stashing uncommitted changes"

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return false, fmt.Errorf("failed to stash changes: %w", err)
	}

	return true, nil
}

func StashChangesQuiet(repoDir string) (bool, error) {
	// First check if there are changes to stash
	hasChanges, err := HasUncommittedChanges(repoDir)
	if err != nil {
		return false, err
	}

	if !hasChanges {
		return false, nil
	}

	cmd := exec.Command("git", "stash", "push", "-m", "ramp rebase stash", "--quiet")
	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to stash changes: %w", err)
	}

	return true, nil
}

func PopStash(repoDir string) error {
	cmd := exec.Command("git", "stash", "pop")
	cmd.Dir = repoDir
	message := "restoring stashed changes"

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to pop stash: %w", err)
	}

	return nil
}

func PopStashQuiet(repoDir string) error {
	cmd := exec.Command("git", "stash", "pop", "--quiet")
	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pop stash: %w", err)
	}

	return nil
}

func CheckoutRemoteBranch(repoDir, branchName string) error {
	// First try to fetch the branch
	if err := FetchBranch(repoDir, branchName); err != nil {
		return err
	}

	// Create local branch tracking the remote
	remoteBranch := "origin/" + branchName
	cmd := exec.Command("git", "checkout", "-b", branchName, remoteBranch)
	cmd.Dir = repoDir
	message := fmt.Sprintf("creating local branch %s tracking %s", branchName, remoteBranch)

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to checkout remote branch %s: %w", branchName, err)
	}

	return nil
}

func CheckoutRemoteBranchQuiet(repoDir, branchName string) error {
	// First try to fetch the branch (use quiet version to avoid nested spinner)
	if err := FetchBranchQuiet(repoDir, branchName); err != nil {
		return err
	}

	// Create local branch tracking the remote
	remoteBranch := "origin/" + branchName
	cmd := exec.Command("git", "checkout", "-b", branchName, remoteBranch)
	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout remote branch %s: %w", branchName, err)
	}

	return nil
}

func validateSourceBranch(repoDir, sourceBranch string) error {
	// Check if it's a local branch
	localExists, err := LocalBranchExists(repoDir, sourceBranch)
	if err != nil {
		return fmt.Errorf("failed to check local branch: %w", err)
	}

	if localExists {
		return nil
	}

	// Check if it's a remote branch (like origin/branch-name)
	if strings.Contains(sourceBranch, "/") {
		if err := validateRemoteBranch(repoDir, sourceBranch); err != nil {
			return err
		}
		return nil
	}

	// Check if it exists as a remote branch with origin/ prefix
	remoteExists, err := RemoteBranchExists(repoDir, sourceBranch)
	if err != nil {
		return fmt.Errorf("failed to check remote branch: %w", err)
	}

	if remoteExists {
		return nil
	}

	return fmt.Errorf("source branch '%s' not found locally or on remote", sourceBranch)
}

func validateRemoteBranch(repoDir, remoteBranch string) error {
	cmd := exec.Command("git", "--no-optional-locks", "show-ref", "--verify", "refs/remotes/"+remoteBranch)
	cmd.Dir = repoDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("remote branch '%s' not found", remoteBranch)
	}

	return nil
}

func ResolveSourceBranch(repoDir, target, effectivePrefix string) (string, error) {
	// If target starts with a remote prefix, validate as remote branch
	if strings.HasPrefix(target, "origin/") {
		// Validate that the remote branch actually exists
		if err := validateRemoteBranch(repoDir, target); err != nil {
			return "", err
		}
		return target, nil
	}

	// If target contains a slash (but isn't remote), check if it's a local branch
	if strings.Contains(target, "/") {
		localExists, err := LocalBranchExists(repoDir, target)
		if err != nil {
			return "", fmt.Errorf("failed to check local branch: %w", err)
		}
		if localExists {
			return target, nil
		}
		return "", fmt.Errorf("local branch '%s' not found", target)
	}

	// Check if target is a direct branch name (without prefix)
	localExists, err := LocalBranchExists(repoDir, target)
	if err != nil {
		return "", fmt.Errorf("failed to check local branch: %w", err)
	}
	if localExists {
		return target, nil
	}

	remoteExists, err := RemoteBranchExists(repoDir, target)
	if err != nil {
		return "", fmt.Errorf("failed to check remote branch: %w", err)
	}
	if remoteExists {
		return "origin/" + target, nil
	}

	// Try as a feature name (with prefix)
	featureBranchName := effectivePrefix + target
	localExists, err = LocalBranchExists(repoDir, featureBranchName)
	if err != nil {
		return "", fmt.Errorf("failed to check local feature branch: %w", err)
	}
	if localExists {
		return featureBranchName, nil
	}

	remoteExists, err = RemoteBranchExists(repoDir, featureBranchName)
	if err != nil {
		return "", fmt.Errorf("failed to check remote feature branch: %w", err)
	}
	if remoteExists {
		return "origin/" + featureBranchName, nil
	}

	return "", fmt.Errorf("target '%s' not found as feature name, branch name, or remote branch", target)
}

func GetRemoteTrackingStatus(repoDir string) (string, error) {
	// Check if current branch has a remote tracking branch
	hasRemote, err := HasRemoteTrackingBranch(repoDir)
	if err != nil {
		return "", err
	}

	if !hasRemote {
		return "(no remote tracking)", nil
	}

	// Get ahead/behind status using git rev-list
	cmd := exec.Command("git", "--no-optional-locks", "rev-list", "--count", "--left-right", "HEAD...@{upstream}")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		// If this fails, the remote tracking might be broken
		return "(remote tracking broken)", nil
	}

	status := strings.TrimSpace(string(output))
	parts := strings.Fields(status)

	if len(parts) != 2 {
		return "", nil
	}

	ahead := parts[0]
	behind := parts[1]

	if ahead == "0" && behind == "0" {
		return "(up to date)", nil
	}

	var statusParts []string
	if ahead != "0" {
		statusParts = append(statusParts, fmt.Sprintf("ahead %s", ahead))
	}
	if behind != "0" {
		statusParts = append(statusParts, fmt.Sprintf("behind %s", behind))
	}

	return fmt.Sprintf("(%s)", strings.Join(statusParts, ", ")), nil
}

func GetDefaultBranch(repoDir string) (string, error) {
	// Try to get the default branch from remote's HEAD
	cmd := exec.Command("git", "--no-optional-locks", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err == nil {
		// Parse "refs/remotes/origin/main" to "main"
		ref := strings.TrimSpace(string(output))
		if strings.HasPrefix(ref, "refs/remotes/origin/") {
			return strings.TrimPrefix(ref, "refs/remotes/origin/"), nil
		}
	}

	// Fallback: check if 'main' exists
	mainExists, err := LocalBranchExists(repoDir, "main")
	if err == nil && mainExists {
		return "main", nil
	}

	// Fallback: check if 'master' exists
	masterExists, err := LocalBranchExists(repoDir, "master")
	if err == nil && masterExists {
		return "master", nil
	}

	// Ultimate fallback
	return "main", nil
}

func GetAheadBehindCount(worktreeDir, baseBranch string) (ahead int, behind int, err error) {
	// Get ahead/behind status compared to base branch
	cmd := exec.Command("git", "--no-optional-locks", "rev-list", "--count", "--left-right", fmt.Sprintf("HEAD...%s", baseBranch))
	cmd.Dir = worktreeDir

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get ahead/behind count: %w", err)
	}

	status := strings.TrimSpace(string(output))
	parts := strings.Fields(status)

	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected output format: %s", status)
	}

	ahead = 0
	behind = 0

	if parts[0] != "0" {
		fmt.Sscanf(parts[0], "%d", &ahead)
	}
	if parts[1] != "0" {
		fmt.Sscanf(parts[1], "%d", &behind)
	}

	return ahead, behind, nil
}

func IsMergedInto(worktreeDir, targetBranch string) (bool, error) {
	// Use git merge-base to check if HEAD is an ancestor of targetBranch
	// This means all commits from current branch are in targetBranch
	cmd := exec.Command("git", "--no-optional-locks", "merge-base", "--is-ancestor", "HEAD", targetBranch)
	cmd.Dir = worktreeDir

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Exit code 1 means not an ancestor (not merged)
			return false, nil
		}
		// Other errors (e.g., invalid branch name)
		return false, fmt.Errorf("failed to check merge status: %w", err)
	}

	// Exit code 0 means HEAD is an ancestor of targetBranch (merged)
	return true, nil
}

type DiffStats struct {
	FilesChanged int
	Insertions   int
	Deletions    int
}

type StatusStats struct {
	UntrackedFiles int
	ModifiedFiles  int
	StagedFiles    int
}

func GetDiffStats(repoDir string) (*DiffStats, error) {
	cmd := exec.Command("git", "--no-optional-locks", "diff", "--shortstat")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff stats: %w", err)
	}

	stats := &DiffStats{}
	outputStr := strings.TrimSpace(string(output))

	if outputStr == "" {
		return stats, nil
	}

	// Parse output like: " 2 files changed, 15 insertions(+), 3 deletions(-)"
	fmt.Sscanf(outputStr, "%d file", &stats.FilesChanged)

	if strings.Contains(outputStr, "insertion") {
		insertionIdx := strings.Index(outputStr, "insertion")
		// Find the number before "insertion"
		parts := strings.Fields(outputStr[:insertionIdx])
		if len(parts) > 0 {
			fmt.Sscanf(parts[len(parts)-1], "%d", &stats.Insertions)
		}
	}

	if strings.Contains(outputStr, "deletion") {
		deletionIdx := strings.Index(outputStr, "deletion")
		// Find the number before "deletion"
		parts := strings.Fields(outputStr[:deletionIdx])
		if len(parts) > 0 {
			fmt.Sscanf(parts[len(parts)-1], "%d", &stats.Deletions)
		}
	}

	return stats, nil
}

func GetStatusStats(repoDir string) (*StatusStats, error) {
	cmd := exec.Command("git", "--no-optional-locks", "status", "--porcelain")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	stats := &StatusStats{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if len(line) < 2 {
			continue
		}

		// Format: XY filename
		// X = index status, Y = working tree status
		// ?? = untracked
		x := line[0]
		y := line[1]

		if x == '?' && y == '?' {
			stats.UntrackedFiles++
		} else if x != ' ' && x != '?' {
			stats.StagedFiles++
		} else if y != ' ' && y != '?' {
			stats.ModifiedFiles++
		}
	}

	return stats, nil
}

// WorktreeRegistered checks if a worktree path is registered with git, even if the directory doesn't exist
func WorktreeRegistered(repoDir, worktreeDir string) bool {
	cmd := exec.Command("git", "--no-optional-locks", "worktree", "list", "--porcelain")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse worktree list output to find matching worktree path
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			if path == worktreeDir {
				return true
			}
		}
	}

	return false
}
