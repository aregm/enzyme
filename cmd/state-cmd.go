package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"Rhoc/pkg/state"
)

// MoreInfoer is a Thing companion interface which can produce additional information,
// for example connection information for a cluster
type MoreInfoer interface {
	MoreInfo() (string, error)
}

func printEntry(id string, entry state.Entry) error {
	fmt.Printf("%s - %s\n", id, entry)

	if infoer, ok := entry.(MoreInfoer); ok {
		more, err := infoer.MoreInfo()
		if err != nil {
			log.WithField("thing", entry).Errorf("Cannot get more info: %s", err)
			return err
		}

		if more != "" {
			fmt.Printf("\t%s\n", more)
		}
	}

	return nil
}

var (
	stateCmd = &cobra.Command{
		Use:   "state",
		Short: "Print the state of Rhoc",
		Long:  `This command shows all manageable entities (images, clusters, storages etc.) with their statuses.`,
		Args:  cobra.ExactValidArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			err := fetcher.Enumerate(func(id string) bool {
				return true
			}, printEntry)
			if err != nil {
				log.Fatalf("stateCmd: %s", err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(stateCmd)
}
