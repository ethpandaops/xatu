package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	release  = "dev"
	commit   = "unknown"
	platform = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version of Xatu.",
	Long:  `Prints the version of Xatu.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("xatu version %s %s %s\n", release, commit, platform)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
