package cmd

import (
	"os"

	"github.com/spf13/cobra"
	
	"ramp/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "ramp",
	Short: "A CLI tool for managing multi-repo development workflows",
	Long: `Ramp is a CLI tool that helps developers manage multi-repository projects
with git worktrees and automated setup scripts.

Find a project directory with a .ramp/ramp.yaml configuration file and run
commands to initialize repositories and create feature branches.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		verbose, _ := cmd.Flags().GetBool("verbose")
		ui.Verbose = verbose
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Show detailed output during operations")
}