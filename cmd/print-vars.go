package cmd

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"enzyme/pkg/provider"
)

const (
	imageTargetObject   = "image"
	clusterTargetObject = "cluster"
	taskTargetObject    = "task"
	storageTargetObject = "storage"
)

var (
	targetProvider    string
	validPrintTargets []string = []string{imageTargetObject, clusterTargetObject, taskTargetObject,
		storageTargetObject}

	printVarsCommand = &cobra.Command{
		Use:       fmt.Sprintf("print-vars [{%s}, ...]", strings.Join(validPrintTargets, ", ")),
		Short:     "prints variables that user can set",
		Long:      `Use this command with one of the additional args: ` + strings.Join(validPrintTargets, ", "),
		ValidArgs: validPrintTargets,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
				return err
			}

			if err := cobra.OnlyValidArgs(cmd, args); err != nil {
				return err
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			printVars(args)
		},
	}
)

func init() {
	rootCmd.AddCommand(printVarsCommand)

	printVarsCommand.Flags().StringVar(&targetProvider, "provider", "gcp",
		"prints variables that can be setted for a provider")
}

func printVars(args []string) {
	var variables map[string]interface{}

	var err error

	for _, targetObject := range args {
		switch targetObject {
		case imageTargetObject:
			_, variables, err = provider.GetImageVariables(targetProvider, "")
			if err != nil {
				log.WithFields(log.Fields{
					"targetProvider": targetProvider,
				}).Fatalf("printVarsCommand: %s", err)
			}
		case clusterTargetObject:
			_, variables, err = provider.GetClusterVariables(targetProvider, "", true)
			if err != nil {
				log.WithFields(log.Fields{
					"targetProvider": targetProvider,
				}).Fatalf("printVarsCommand: %s", err)
			}
		case taskTargetObject:
			variables, err = provider.GetTaskVariables(targetProvider)
			if err != nil {
				log.WithFields(log.Fields{
					"targetProvider": targetProvider,
				}).Fatalf("printVarsCommand: %s", err)
			}
		case storageTargetObject:
			removeDefaultLayer := true

			_, variables, err = provider.GetStorageVariables(targetProvider, "", removeDefaultLayer)
			if err != nil {
				log.WithFields(log.Fields{
					"targetProvider": targetProvider,
				}).Fatalf("printVarsCommand: %s", err)
			}
		default:
			log.WithFields(log.Fields{
				"target": targetObject,
			}).Fatal("variables for the object cannot be printed")
		}

		printVariables(targetObject, variables)
	}
}

func printVariables(targetObject string, vars map[string]interface{}) {
	fmt.Println(targetObject + ":")

	for key, value := range vars {
		fmt.Printf("\t%-25s: %s\n", key, value)
	}
}
