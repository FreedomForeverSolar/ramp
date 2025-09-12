package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Clone(repoURL, destDir string) error {
	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(destDir), err)
	}

	cmd := exec.Command("git", "clone", repoURL, destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove worktree %s: %w", worktreeDir, err)
	}

	return nil
}

func DeleteBranch(repoDir, branchName string) error {
	cmd := exec.Command("git", "branch", "-D", branchName)
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
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

func IsGitRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}