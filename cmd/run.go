package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"Rhoc/pkg/controller"
	"Rhoc/pkg/entities/runtask"
)

var (
	remotePath        string
	newlineConversion bool
	overwrite         bool
	keepCluster       bool
	useStorage        bool
	uploadFiles       []string
	downloadFiles     []string

	runCommand = &cobra.Command{
		Use:   "run [script path] [script args]",
		Short: "Runs command on public cloud",
		Long: `This command will make sure image and cluster required for the task are created and ` +
			`then will run this task.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			config, prov, serviceParams, err := createArgs()
			if err != nil {
				log.Fatal()
			}
			localPath := args[0]
			scriptArgs := args[1:]
			task, err := runtask.CreateTaskTarget(prov, config, serviceParams, fetcher, localPath, remotePath,
				newlineConversion, overwrite, useStorage, scriptArgs, uploadFiles, downloadFiles)
			if err != nil {
				log.WithFields(log.Fields{
					"provider":  prov,
					"script":    localPath,
					"remote":    remotePath,
					"overwrite": overwrite,
				}).Fatalf("run: cannot create run task: %s", err)
			}

			desired := runtask.ClusterCleaned
			if keepCluster {
				desired = runtask.ResultsDownloaded
			}

			if err = controller.ReachTarget(task, desired, simulate); err != nil {
				log.WithFields(log.Fields{
					"provider": prov,
					"script":   localPath,
					"remote":   remotePath,
					"args":     scriptArgs,
				}).Fatalf("run: cannot execute the run task: %s", err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(runCommand)
	addServiceParams(runCommand)

	runCommand.Flags().StringVar(&remotePath, "remote-path", "Rhoc-script",
		"name for the transmitted program on the remote machine")
	runCommand.Flags().BoolVar(&newlineConversion, "newline-conversion", false,
		"enable conversion of DOS/Windows newlines to UNIX newlines")
	runCommand.Flags().BoolVar(&overwrite, "overwrite", false,
		"overwrite the content of the remote file with the content of the local file")
	runCommand.Flags().BoolVar(&keepCluster, "keep-cluster", false,
		"keep the cluster running after script is done")
	runCommand.Flags().BoolVar(&useStorage, "use-storage", false,
		"use external storage node during execution")
	runCommand.Flags().StringSliceVar(&uploadFiles, "upload-files", nil,
		"files for copying into the cluster (into '~/Rhoc-upload' folder with the same names)")
	runCommand.Flags().StringSliceVar(&downloadFiles, "download-files", nil,
		"files for copying from the cluster (into './Rhoc-download' folder with the same names)")
}
