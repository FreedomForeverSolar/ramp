package cmd

import (
	"github.com/spf13/cobra"
	"ramp/internal/autoupdate"
)

var internalCmd = &cobra.Command{
	Use:    "__internal_update_check",
	Hidden: true, // Don't show in help output
	Short:  "Internal command for background update checking",
	Run: func(cmd *cobra.Command, args []string) {
		// Run the background check with current version
		autoupdate.RunBackgroundCheck(version)
	},
}

func init() {
	rootCmd.AddCommand(internalCmd)
}
