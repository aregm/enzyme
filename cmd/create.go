package cmd

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"enzyme/pkg/controller"
	"enzyme/pkg/entities/cluster"
	"enzyme/pkg/entities/image"
	"enzyme/pkg/entities/storage"
)

var (
	validCreateTargets []string = []string{imageTargetObject, clusterTargetObject, storageTargetObject}

	createCommand = &cobra.Command{
		Use:   fmt.Sprintf("create {%s}", strings.Join(validCreateTargets, ", ")),
		Short: fmt.Sprintf("creates the {%s} in the public cloud", strings.Join(validCreateTargets, ", ")),
		Long: `This command tells enzyme to create a VM image, to spawn VM instances forming
a cluster or to create VM instance based on a disk that holds your data.`,
		ValidArgs: validCreateTargets,
		Args:      cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			config, prov, serviceParams, err := createArgs()
			if err != nil {
				log.Fatal()
			}

			creatingObject := args[0]

			var thing controller.Thing
			var desired controller.Status
			logger := log.WithFields(log.Fields{
				"providerName":    providerName,
				"credentialsFile": credentialsFile,
				"region":          region,
				"parametersFile":  parametersFile,
				"creatingObject":  creatingObject,
			})

			switch creatingObject {
			case imageTargetObject:
				if thing, err = image.CreateImageTarget(prov, config, serviceParams, fetcher); err != nil {
					logger.Fatalf("createCommand: cannot create image thing: %s", err)
				}
				desired = image.Created
			case clusterTargetObject:
				if thing, err = cluster.CreateClusterTarget(prov, config, serviceParams, fetcher); err != nil {
					logger.Fatalf("createCommand: cannot create cluster thing: %s", err)
				}
				desired = cluster.Spawned
			case storageTargetObject:
				if thing, err = storage.CreateStorageTarget(prov, config, serviceParams, fetcher); err != nil {
					logger.Fatalf("createCommand: cannot create storage node thing: %s", err)
				}
				desired = storage.Detached
			default:
				logger.Fatal("this object cannot be created")
			}

			if err := controller.ReachTarget(thing, desired, simulate); err != nil {
				logger.WithFields(log.Fields{
					"thing":          thing,
					"desired-status": desired,
					"simulate":       simulate,
				}).Fatalf("createCommand: cannot make thing reach desired status: %s", err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(createCommand)
	addServiceParams(createCommand)
}
