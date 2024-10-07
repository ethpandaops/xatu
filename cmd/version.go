package cmd

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version of Xatu.",
	Long:  `Prints the version of Xatu.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s\n", xatu.FullVWithPlatform())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
