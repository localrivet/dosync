package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version should be updated with each new release
var Version = "v0.0.0"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of ecrsync",
	Long:  `All software have versions. This is ecrsync's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ecrsync version " + Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
