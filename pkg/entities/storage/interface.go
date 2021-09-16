package storage

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"enzyme/pkg/controller"
	"enzyme/pkg/provider"
)

// ConnectDetails describes the details on how to connect to the storage node
type ConnectDetails struct {
	PublicAddress   string
	InternalAddress string
	UserName        string
	PrivateKey      string
}

// GetConnectDetails retrieves ConnectDetails describing how to connect to the storage node
func GetConnectDetails(from controller.Thing) (ConnectDetails, error) {
	node, ok := from.(*storageNodeState)
	if !ok {
		log.WithFields(log.Fields{
			"thing": from,
		}).Error("GetConnectDetails: not called against a storage node")

		return ConnectDetails{}, fmt.Errorf("not called against a storage node")
	}

	return node.connection, nil
}

func refreshConnectDetails(node *storageNodeState, nextStatus Status) error {
	var configName, configPath string

	switch nextStatus {
	case Detached:
		configName, configPath = "standalone", node.configPath
	case Attached:
		configName, configPath = "attached", node.attachedConfigPath
	default:
		log.WithFields(log.Fields{
			"storage": node,
			"status":  nextStatus,
		}).Error("refreshConnectDetails: storage node must be spawned")

		return fmt.Errorf("storage node must be spawned")
	}

	if disabler, err := enableConfig(configName, configPath); err == nil {
		defer disabler()
	} else {
		return err
	}

	rootDir, _ := filepath.Split(node.configPath)
	logger := log.WithField("storage", node)

	tfLogPrefix, err := node.makeToolLogPrefix("terraform")
	if err != nil {
		logger.Warnf("refreshConnectDetails: cannot make logfile name: %s", err)
	}

	json, err := provider.ParseTerraformOutputs(rootDir, tfLogPrefix, logger)
	if err != nil {
		return err
	}

	parsed, err := provider.ExtractOutputValues(json, "external_address.value", "internal_address.value",
		"user_name.value", "pkey_file.value")
	if err != nil {
		logger.Errorf("refreshConnectDetails: cannot read needed variables: %s", err)
		return err
	}

	node.connection = ConnectDetails{
		PublicAddress:   parsed[0],
		InternalAddress: parsed[1],
		UserName:        parsed[2],
		PrivateKey:      parsed[3],
	}

	return nil
}

// for additional information provided by "enzyme state"
func (node *storageNodeState) MoreInfo() (string, error) {
	if node.connection.PublicAddress != "" {
		return fmt.Sprintf("SCP files to/from %s@%s:/storage, key file=%s",
			node.connection.UserName,
			node.connection.PublicAddress,
			node.connection.PrivateKey), nil
	}

	return "", nil
}
