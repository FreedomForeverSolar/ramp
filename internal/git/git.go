package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	
	"ramp/internal/ui"
)

func Clone(repoURL, destDir string) error {
	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(destDir), err)
	}

	cmd := exec.Command("git", "clone", repoURL, destDir)
	message := fmt.Sprintf("cloning %s", repoURL)
	
	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to clone %s to %s: %w", repoURL, destDir, err)
	}

	return nil
}

func CreateWorktree(repoDir, worktreeDir, branchName string) error {
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
		message = fmt.Sprintf("creating worktree with existing local branch %s", branchName)
	} else if remoteExists {
		// Create local branch tracking the remote
		remoteBranch, err := getRemoteBranchName(repoDir, branchName)
		if err != nil {
			return fmt.Errorf("failed to get remote branch name: %w", err)
		}
		cmd = exec.Command("git", "worktree", "add", "-b", branchName, worktreeDir, remoteBranch)
		message = fmt.Sprintf("creating worktree with existing remote branch %s", branchName)
	} else {
		// Create new branch
		cmd = exec.Command("git", "worktree", "add", "-b", branchName, worktreeDir)
		message = fmt.Sprintf("creating worktree with new branch %s", branchName)
	}

	cmd.Dir = repoDir

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to create worktree %s with branch %s: %w", worktreeDir, branchName, err)
	}

	return nil
}

func getRemoteBranchName(repoDir, branchName string) (string, error) {
	cmd := exec.Command("git", "branch", "-r", "--list", "*/"+branchName)
	cmd.Dir = repoDir
	
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("no remote branch found for %s", branchName)
	}
	
	// Return the first match, trimmed of whitespace
	return strings.TrimSpace(lines[0]), nil
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
	cmd := exec.Command("git", "branch", "--list", branchName)
	cmd.Dir = repoDir
	
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	
	return strings.TrimSpace(string(output)) != "", nil
}

func RemoteBranchExists(repoDir, branchName string) (bool, error) {
	cmd := exec.Command("git", "branch", "-r", "--list", "*/"+branchName)
	cmd.Dir = repoDir
	
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	
	return strings.TrimSpace(string(output)) != "", nil
}

func HasUncommittedChanges(repoDir string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
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

func DeleteBranch(repoDir, branchName string) error {
	cmd := exec.Command("git", "branch", "-D", branchName)
	cmd.Dir = repoDir
	message := fmt.Sprintf("deleting branch %s", branchName)

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}

	return nil
}

func GetWorktreeBranch(worktreeDir string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "HEAD")
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
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
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

func Pull(repoDir string) error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = repoDir
	message := "pulling changes"
	
	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}
	
	return nil
}

func HasRemoteTrackingBranch(repoDir string) (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
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

func RenameBranch(repoDir, oldBranch, newBranch string) error {
	cmd := exec.Command("git", "branch", "-m", oldBranch, newBranch)
	cmd.Dir = repoDir
	message := fmt.Sprintf("renaming branch %s to %s", oldBranch, newBranch)

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to rename branch %s to %s: %w", oldBranch, newBranch, err)
	}

	return nil
}

func MoveWorktree(repoDir, oldPath, newPath string) error {
	cmd := exec.Command("git", "worktree", "move", oldPath, newPath)
	cmd.Dir = repoDir
	message := fmt.Sprintf("moving worktree from %s to %s", filepath.Base(oldPath), filepath.Base(newPath))

	if err := ui.RunCommandWithProgress(cmd, message); err != nil {
		return fmt.Errorf("failed to move worktree from %s to %s: %w", oldPath, newPath, err)
	}

	return nil
}

func IsGitRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}