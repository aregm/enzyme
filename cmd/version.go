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
	Short: "Alpha 0.0.1 Version RHOC",
	Long: `This is experimental alpha engineering version of RHOC. Very unstable. ` +
		`Use at your own risk. SPDX License indicator: MIT`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Rhoc %s\n", VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
