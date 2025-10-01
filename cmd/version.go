package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the version of ramp",
	Long:  `Display the current version of the ramp CLI tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ramp version %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
