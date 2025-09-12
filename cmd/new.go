package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/git"
)

var prefixFlag string

var newCmd = &cobra.Command{
	Use:   "new <feature-name>",
	Short: "Create a new feature branch with git worktrees for all repositories",
	Long: `Create a new feature branch by creating git worktrees for all repositories
from their configured locations. This creates isolated working directories for each repo
in the trees/<feature-name>/ directory.

After creating worktrees, runs any setup script specified in the configuration.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		featureName := args[0]
		if err := runNew(featureName, prefixFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
	newCmd.Flags().StringVar(&prefixFlag, "prefix", "", "Override the branch prefix (defaults to config default_branch_prefix)")
}

func runNew(featureName, prefix string) error {
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

	// Auto-initialize if needed
	if err := autoInitializeIfNeeded(projectDir, cfg); err != nil {
		return fmt.Errorf("auto-initialization failed: %w", err)
	}

	// Determine effective prefix - flag takes precedence, then config, then empty
	effectivePrefix := prefix
	if effectivePrefix == "" {
		effectivePrefix = cfg.GetBranchPrefix()
	}

	treesDir := filepath.Join(projectDir, "trees", featureName)

	if err := os.MkdirAll(treesDir, 0755); err != nil {
		return fmt.Errorf("failed to create trees directory: %w", err)
	}

	fmt.Printf("Creating feature '%s' for project '%s'\n", featureName, cfg.Name)
	fmt.Printf("Creating worktrees in %s\n", treesDir)

	repos := cfg.GetRepos()
	for name, repo := range repos {
		repoDir := repo.GetRepoPath(projectDir)
		worktreeDir := filepath.Join(treesDir, name)

		if !git.IsGitRepo(repoDir) {
			return fmt.Errorf("source repo not found at %s even after auto-initialization", repoDir)
		}

		branchName := effectivePrefix + featureName
		
		// Check branch status to provide informative message
		localExists, err := git.LocalBranchExists(repoDir, branchName)
		if err != nil {
			return fmt.Errorf("failed to check local branch for %s: %w", name, err)
		}
		
		remoteExists, err := git.RemoteBranchExists(repoDir, branchName)
		if err != nil {
			return fmt.Errorf("failed to check remote branch for %s: %w", name, err)
		}

		if localExists {
			fmt.Printf("  %s: creating worktree with existing local branch %s\n", name, branchName)
		} else if remoteExists {
			fmt.Printf("  %s: creating worktree with existing remote branch %s\n", name, branchName)
		} else {
			fmt.Printf("  %s: creating worktree with new branch %s\n", name, branchName)
		}

		if err := git.CreateWorktree(repoDir, worktreeDir, branchName); err != nil {
			return fmt.Errorf("failed to create worktree for %s: %w", name, err)
		}
	}

	if cfg.Setup != "" {
		fmt.Printf("Running setup script: %s\n", cfg.Setup)
		if err := runSetupScript(projectDir, treesDir, cfg.Setup); err != nil {
			return fmt.Errorf("setup script failed: %w", err)
		}
	}

	fmt.Printf("✅ Feature '%s' created successfully!\n", featureName)
	fmt.Printf("📁 Worktrees are located in: %s\n", treesDir)
	return nil
}

func runSetupScript(projectDir, treesDir, setupScript string) error {
	scriptPath := filepath.Join(projectDir, ".ramp", setupScript)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("setup script not found: %s", scriptPath)
	}

	// Extract feature name from treesDir path
	featureName := filepath.Base(treesDir)

	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = treesDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up environment variables that the setup script expects
	cmd.Env = append(os.Environ(), fmt.Sprintf("RAMP_PROJECT_DIR=%s", projectDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_TREES_DIR=%s", treesDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_WORKTREE_NAME=%s", featureName))

	// Add dynamic repository path environment variables
	cfg, err := config.LoadConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load config for env vars: %w", err)
	}
	
	repos := cfg.GetRepos()
	for name, repo := range repos {
		envVarName := config.GenerateEnvVarName(name)
		repoPath := repo.GetRepoPath(projectDir)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVarName, repoPath))
	}

	return cmd.Run()
}
