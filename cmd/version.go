package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "v0.1.24"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of DOSync",
	Long:  `All software has versions. This is DOSync's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("DOSync", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
