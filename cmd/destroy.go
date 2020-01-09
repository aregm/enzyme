package cmd

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"Rhoc/pkg/controller"
	"Rhoc/pkg/state"
)

var (
	destroyCommand = &cobra.Command{
		Use:   "destroy [destroyObjectID]",
		Short: "destroy object created by using create command",
		Long:  `You can destroy image, cluster or storage by destroyObjectID which can be found by checking state.`,
		Args:  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			destroy(args)
		},
	}
)

func init() {
	rootCmd.AddCommand(destroyCommand)
}

// Destructible interface should be implemented for an object that should be
// destroyed via Rhoc's destroy command
type Destructible interface {
	GetDestroyedTarget() controller.Target
}

func destroy(args []string) {
	destroyObjectID := args[0]
	candidates := []state.Entry{}

	if err := fetcher.Enumerate(
		func(id string) bool {
			return id == destroyObjectID
		}, func(id string, entry state.Entry) error {
			candidates = append(candidates, entry)
			return nil
		}); err != nil {
		log.WithField("destroyObjectID", destroyObjectID).Fatalf("destroy: Enumerate failed: %s", err)
	}

	msg := ""
	fields := log.Fields{
		"target-id": destroyObjectID,
	}

	switch {
	case len(candidates) == 0:
		msg = "destroy: cannot find an object by id"
	case len(candidates) > 1:
		foundObjs := []string{}
		for _, candidate := range candidates {
			foundObjs = append(foundObjs, fmt.Sprintf("%s", candidate))
		}

		fields["found-objects"] = strings.Join(foundObjs, ", ")
		msg = "destroy: too many matching objects found"
	default:
		fields["found-object"] = candidates[0]

		if destruct, ok := candidates[0].(Destructible); !ok {
			msg = "destroy: found object is not destructible"
		} else {
			if thing, ok := destruct.(controller.Thing); !ok {
				msg = "destroy: found object is not controllable"
			} else {
				destroyedTarget := destruct.GetDestroyedTarget()
				fields["desired-status"] = destroyedTarget.DesiredStatus
				if thing.Status().Equals(destroyedTarget.DesiredStatus) {
					log.WithFields(fields).Info("destroy: object already destroyed")
					fmt.Printf("destroy: already destroyed: %s\n", candidates[0])

					return
				}
				if err := controller.ReachTargetEx(destroyedTarget, simulate); err != nil {
					msg = fmt.Sprintf("destroy: cannot destroy found object: %s", err)
				} else {
					log.WithFields(fields).Info("destroy: successfully destroyed object")
					fmt.Printf("destroy: successfully destroyed object: %s\n", candidates[0])

					return
				}
			}
		}
	}

	log.WithFields(fields).Fatal(msg)
}
