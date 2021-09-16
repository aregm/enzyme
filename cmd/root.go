package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"enzyme/pkg/logging"
	"enzyme/pkg/provider"
	"enzyme/pkg/state"
)

var (
	verbose  bool
	simulate bool
	fetcher  state.Fetcher

	rootCmd = &cobra.Command{
		Use:   "enzyme",
		Short: "enzyme is easy to use hybrid cloud HPCaaS creator and runner",
		Long: `enzyme is a software tool which allows users to set up Virtual Private
Cluster in the cloud in a cloud provider-agnostic way. enzyme creates
Intel HPC Platform compatible cluster anywhere where the term "cluster"
is applicable and runs a workload there.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				log.Fatalf("cmd.Help function failed: %s", err)
			}
		},
	}
)

func init() {
	cobra.OnInitialize(initenzyme)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&simulate, "simulate", "s", false,
		"simulate running the execution - do not perform any actual actions")

	log.SetOutput(os.Stdout)
}

func initenzyme() {
	if verbose {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}

	logging.InitLogging(verbose)
	provider.InitTools()

	if simulate {
		fetcher = state.Fetcher{
			Chest: &state.MemChest{},
		}
	} else {
		fetcher = state.Fetcher{
			Chest: &state.JSONChest{},
		}
	}
}

func checkFileExists(fileName string) {
	fileNameStat, err := os.Stat(fileName)
	if err != nil {
		log.WithField("path", fileName).Fatalf("checkFileExists: file not found: %s", err)
	}

	if !fileNameStat.Mode().IsRegular() {
		log.WithField("path", fileName).Fatalf("checkFileExists: path is not pointing to a regular file")
	}
}

// Execute is used as entry point for parsing user input via cobra package
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
