package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// VERSION value is replaced by make during build
	VERSION = "development"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Alpha 0.0.1 Version enzyme",
	Long: `This is experimental alpha engineering version of enzyme. Very unstable. ` +
		`Use at your own risk. SPDX License indicator: MIT`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("enzyme %s\n", VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
