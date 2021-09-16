package cluster

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"enzyme/pkg/config"
	"enzyme/pkg/controller"
	"enzyme/pkg/provider"
)

// ResourceDescriptor is a descriptor for uniquely identifying a resource, e.g. used by Terraform
type ResourceDescriptor struct {
	Address string
	ID      string
}

// GetNetworkResources retrieves network resources managed by the cluster, usually
// "network" and "subnetwork" parts
func GetNetworkResources(from controller.Thing) ([]ResourceDescriptor, error) {
	result := []ResourceDescriptor{}

	cluster, ok := from.(*clusterState)
	if !ok {
		log.WithField("thing", from).Error("GetNetworkResources: not called against a cluster")
		return result, fmt.Errorf("not called against a cluster")
	}

	if !cluster.status.Satisfies(Spawned) {
		log.WithFields(log.Fields{
			"cluster": cluster,
			"status":  cluster.status,
		}).Error("GetNetworkResources: cluster must be spawned")

		return result, fmt.Errorf("cluster must be spawned")
	}

	json, err := parseTerraformJSON(cluster)
	if err != nil {
		return result, err
	}

	for idx := 1; ; idx++ {
		parsed, err := provider.ExtractOutputValues(json,
			fmt.Sprintf("network_resource_address_%d.value", idx),
			fmt.Sprintf("network_resource_id_%d.value", idx))
		if err != nil {
			if missing, ok := err.(provider.MissingKey); ok {
				log.WithField("cluster", cluster).Infof(
					"GetNetworkResources: stop looping at %d index - no more network resources: %s",
					idx, missing)

				break
			}

			log.WithField("cluster", cluster).Errorf("GetNetworkResources: cannot parse network resource: %s",
				err)

			return result, err
		}

		result = append(result, ResourceDescriptor{
			Address: parsed[0],
			ID:      parsed[1],
		})
	}

	log.WithFields(log.Fields{
		"cluster": cluster,
		"found":   result,
	}).Info("GetNetworkResources: found network resources")

	return result, nil
}

// ConnectDetails contains hostname, username and private key to connect
// to a spawned cluster
type ConnectDetails struct {
	PublicAddress string
	UserName      string
	PrivateKey    string
}

// GetConnectDetails retrieves hostname, username and private key to connect
// to the spawned cluster
func GetConnectDetails(from controller.Thing) (ConnectDetails, error) {
	cluster, ok := from.(*clusterState)
	if !ok {
		log.WithFields(log.Fields{
			"thing": from,
		}).Error("GetConnectDetails: not called against a cluster")

		return ConnectDetails{}, fmt.Errorf("not called against a cluster")
	}

	return cluster.connection, nil
}

func refreshConnectDetails(cluster *clusterState, nextStatus Status) error {
	if !nextStatus.Satisfies(Spawned) {
		log.WithFields(log.Fields{
			"cluster": cluster,
			"status":  nextStatus,
		}).Error("refreshConnectDetails: cluster must be spawned")

		return fmt.Errorf("cluster must be spawned")
	}

	json, err := parseTerraformJSON(cluster)
	if err != nil {
		return err
	}

	parsed, err := provider.ExtractOutputValues(json, "login_address.value", "username.value", "pkey_file.value")
	if err != nil {
		log.WithField("cluster", cluster).Errorf("refreshConnectDetails: cannot read needed variables: %s", err)
		return err
	}

	cluster.connection = ConnectDetails{
		PublicAddress: parsed[0],
		UserName:      parsed[1],
		PrivateKey:    parsed[2],
	}

	return nil
}

func parseTerraformJSON(cluster *clusterState) (config.Config, error) {
	clusterRootDir, _ := filepath.Split(cluster.configPath)
	logger := log.WithField("cluster", cluster)

	tfLogPrefix, err := cluster.makeToolLogPrefix("terraform")
	if err != nil {
		logger.Warnf("parseTerraformJSON: cannot make logfile name: %s", err)
	}

	return provider.ParseTerraformOutputs(clusterRootDir, tfLogPrefix, logger)
}

// for additional information provided by "enzyme state"
func (cluster *clusterState) MoreInfo() (string, error) {
	if cluster.connection.PublicAddress != "" {
		return fmt.Sprintf("SSH to %s@%s, key file=%s",
			cluster.connection.UserName,
			cluster.connection.PublicAddress,
			cluster.connection.PrivateKey), nil
	}

	return "", nil
}
