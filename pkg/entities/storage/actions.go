package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	action_pkg "enzyme/pkg/action"
	"enzyme/pkg/controller"
	"enzyme/pkg/entities/cluster"
	"enzyme/pkg/entities/common"
	"enzyme/pkg/entities/image"
	"enzyme/pkg/provider"
	storage_pkg "enzyme/pkg/storage"
)

func enableConfig(name, path string) (func(), error) {
	enabledPath := strings.TrimSuffix(path, configExtDisabled) + configExt
	disabler := func() {
		os.Remove(enabledPath)
	}

	err := storage_pkg.CopyFile(path, enabledPath)
	if err != nil {
		log.WithFields(log.Fields{
			"name": name,
			"path": path,
		}).Errorf("StorageNode.enableConfig: cannot copy config file: %s", err)
	}

	return disabler, err
}

type makeConfig struct {
	storage *storageNodeState
}

func (action makeConfig) String() string {
	return fmt.Sprintf("Configure for %s", action.storage)
}

func (action makeConfig) Apply() error {
	log.WithFields(log.Fields{
		"storage-node": action.storage,
	}).Info("StorageNode.makeConfig.Apply")

	configFilesDir := action.storage.getConfigFilesDir()

	tfLogPrefix, err := action.storage.makeToolLogPrefix("terraform")
	if err != nil {
		log.WithFields(log.Fields{
			"storage": action.storage,
		}).Warnf("StorageNode.makeConfig: cannot make logfile name: %s", err)
	}

	initer := func(name, template, target string) error {
		log.WithFields(log.Fields{
			"template": template,
			"target":   target,
			"name":     name,
		}).Info("StorageNode.makeConfig: generating config file")

		storageConfig, err := action.storage.provider.MakeStorageNodeConfig(template,
			action.storage.userVariables)
		if err != nil {
			log.WithFields(log.Fields{
				"template": template,
				"name":     name,
			}).Errorf("StorageNode.makeConfig: cannot create config object: %s", err)

			return err
		}

		if err = storageConfig.Serialize(target); err != nil {
			log.WithFields(log.Fields{
				"config": target,
				"name":   name,
			}).Errorf("StorageNode.makeConfig: cannot save config: %s", err)

			return err
		}

		if disabler, err := enableConfig(name, target); err == nil {
			defer disabler()
		} else {
			return err
		}

		if logname, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, configFilesDir, provider.Terraform(),
			"init"); err != nil {
			log.WithFields(log.Fields{
				"storage-dir": configFilesDir,
				"name":        name,
			}).Errorf("StorageNode.makeConfig: error initializing: %s", err)
			fmt.Fprintf(os.Stderr, "Cannot initialize tools, see log for details: %s\n", logname)

			return err
		}

		return nil
	}

	if err := initer("standalone", action.storage.templatePath, action.storage.configPath); err != nil {
		return err
	}

	if err :=
		initer("attached", action.storage.attachedTemplatePath, action.storage.attachedConfigPath); err != nil {
		return err
	}

	return nil
}

func composeImagePrereq(storage *storageNodeState, desiredStatus image.Status) (controller.Target, error) {
	imageTarget, err :=
		image.CreateImageTarget(storage.provider, storage.userVariables, storage.serviceParams, storage.fetcher)
	if err != nil {
		log.WithFields(log.Fields{
			"storage": storage,
		}).Errorf("composeImagePrereq: cannot make image target: %s", err)

		return controller.Target{}, err
	}

	return controller.Target{
		Thing:         imageTarget,
		DesiredStatus: desiredStatus,
	}, nil
}

func composeClusterPrereq(storage *storageNodeState,
	desiredStatus cluster.Status) (controller.Target, error) {
	clusterTarget, err := cluster.CreateClusterTarget(storage.provider, storage.userVariables,
		storage.serviceParams, storage.fetcher)
	if err != nil {
		log.WithFields(log.Fields{
			"storage": storage,
		}).Errorf("composeClusterPrereq: cannot make cluster target: %s", err)

		return controller.Target{}, err
	}

	return controller.Target{
		Thing:         clusterTarget,
		DesiredStatus: desiredStatus,
	}, nil
}

func (action makeConfig) IsExclusive() bool {
	return false
}

func (action makeConfig) Prerequisites() ([]controller.Target, error) {
	imageTarget, err := composeImagePrereq(action.storage, image.Configured)
	if err != nil {
		return []controller.Target{}, err
	}

	return []controller.Target{imageTarget}, nil
}

type spawnStorage struct {
	storage *storageNodeState
	stage   common.SyncedStr
}

func (action *spawnStorage) String() string {
	return fmt.Sprintf("Spawn%s for %s", action.stage.Get(), action.storage)
}

func (action *spawnStorage) Apply() error {
	log.WithFields(log.Fields{
		"storage": action.storage,
	}).Info("StorageNode.spawnStorage.Apply")

	if disabler, err := enableConfig("standalone", action.storage.configPath); err == nil {
		defer disabler()
	} else {
		return err
	}

	configFilesDir := action.storage.getConfigFilesDir()

	tfLogPrefix, err := action.storage.makeToolLogPrefix("terraform")
	if err != nil {
		log.WithFields(log.Fields{
			"storage": action.storage,
		}).Warnf("StorageNode.spawnStorage: cannot make logfile name: %s", err)
	}

	diskResourceName := action.storage.provider.GetTFStorageResourceName()
	if _, err = action_pkg.RunLoggedCmdDir(tfLogPrefix, configFilesDir, provider.Terraform(),
		"import", "-no-color", diskResourceName+".storage", action.storage.name+"-disk"); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
		}).Infof("StorageNode.spawnStorage: cannot import disk, err=%s", err)
	} else {
		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
		}).Info("StorageNode.spawnStorage: successfully imported disk")
	}

	if logname, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, configFilesDir, provider.Terraform(),
		"apply", "-auto-approve", "-no-color"); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
		}).Errorf("StorageNode.spawnStorage: error spawning storage node: %s", err)
		fmt.Fprintf(os.Stderr, "Cannot spawn storage node, see log for details: %s\n", logname)
		log.Info("StorageNode.spawnStorage: destroying half-spawned storage node ...")
		action.stage.Set(":destroying half-spawned")
		// exclude disk from removal
		_, rmErr := action_pkg.RunLoggedCmdDir(tfLogPrefix, configFilesDir, provider.Terraform(),
			"state", "rm", diskResourceName+".storage")

		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
		}).Infof("StorageNode.spawnStorage: tried to exclude disk from destruction, err=%s", rmErr)

		if logname, destroyErr := action_pkg.RunLoggedCmdDir(tfLogPrefix, configFilesDir, provider.Terraform(),
			"destroy", "-auto-approve", "-no-color"); destroyErr != nil {
			log.WithFields(log.Fields{
				"storage-dir": configFilesDir,
			}).Errorf("StorageNode.spawnStorage: error destroying half-spawned storage node: %s, "+
				"you can try to destroy manually by going to %s and working with 'terraform destroy'",
				destroyErr, configFilesDir)
			fmt.Fprintf(os.Stderr, "Cannot destroy half-spawned storage node, see log for details: %s\n",
				logname)
		}

		return err
	}

	action.stage.Set(":getting connect info")

	if err := refreshConnectDetails(action.storage, Detached); err != nil {
		log.WithField("storage", action.storage).Errorf(
			"StorageNode.spawnStorage: cannot refresh connection details: %s", err)
		return err
	}

	action.stage.Reset()

	return nil
}

func (action *spawnStorage) IsExclusive() bool {
	return false
}

func (action *spawnStorage) Prerequisites() ([]controller.Target, error) {
	imageTarget, err := composeImagePrereq(action.storage, image.Created)
	if err != nil {
		return []controller.Target{}, err
	}

	return []controller.Target{imageTarget}, nil
}

type destroyStorage struct {
	storage *storageNodeState
}

func (action destroyStorage) String() string {
	return fmt.Sprintf("Destroy for %s", action.storage)
}

func (action destroyStorage) Apply() error {
	log.WithFields(log.Fields{
		"storage": action.storage,
	}).Info("StorageNode.destroyStorage.Apply")

	var configName, configPath string

	var unmanagedResources []string

	switch action.storage.status {
	case Attached:
		configName, configPath = "attached", action.storage.attachedConfigPath

		for _, resource := range action.storage.importedResources {
			unmanagedResources = append(unmanagedResources, resource.Address)
		}
	case Detached:
		configName, configPath = "standalone", action.storage.configPath
	default:
		err := fmt.Errorf("unexpected storage node status: %s", action.storage.status)
		log.WithField("storage", action.storage).Errorf("StorageNode.destroyStorage: %s", err)

		return err
	}

	if disabler, err := enableConfig(configName, configPath); err == nil {
		defer disabler()
	} else {
		return err
	}

	configFilesDir := action.storage.getConfigFilesDir()

	tfLogPrefix, err := action.storage.makeToolLogPrefix("terraform")
	if err != nil {
		log.WithFields(log.Fields{
			"storage": action.storage,
		}).Warnf("StorageNode.destroyStorage: cannot make logfile name: %s", err)
	}

	newTfState, currentTfState := filepath.Join(configFilesDir, "destroying.tfstate"),
		filepath.Join(configFilesDir, "terraform.tfstate")
	if err := storage_pkg.CopyFile(currentTfState, newTfState); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
		}).Errorf("StorageNode.destroyStorage: cannot copy state: %s", err)

		return err
	}

	// reset connection details as storage node is most likely no longer accessible
	// after running "terraform destroy"
	action.storage.connection = ConnectDetails{}

	// exclude disk from removal
	diskResourceName := action.storage.provider.GetTFStorageResourceName()
	unmanagedResources = append(unmanagedResources, diskResourceName+".storage")

	rmArgs := []string{"state", "rm", "-state=" + newTfState}
	rmArgs = append(rmArgs, unmanagedResources...)

	if logname, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, configFilesDir, provider.Terraform(),
		rmArgs...); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
			"resources":   unmanagedResources,
		}).Errorf("StorageNode.spawnStorage: cannot exclude resources from destruction: %s", err)
		fmt.Fprintf(os.Stderr, "Cannot exclude unmanaged resources, see log for details: %s\n", logname)

		return err
	}

	if logname, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, configFilesDir, provider.Terraform(),
		"destroy", "-auto-approve", "-no-color", "-state="+newTfState); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
		}).Errorf("StorageNode.destroyStorage: error destroying storage node: %s", err)
		fmt.Fprintf(os.Stderr, "Cannot destroy storage node, see log for details: %s\n", logname)

		return err
	}

	if err := os.Rename(newTfState, currentTfState); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
		}).Errorf("StorageNode.destroyStorage: cannot rename state: %s", err)

		return err
	}

	return nil
}

func (action destroyStorage) IsExclusive() bool {
	return false
}

func (action destroyStorage) Prerequisites() ([]controller.Target, error) {
	return []controller.Target{}, nil
}

type attachStorage struct {
	storage *storageNodeState
	stage   common.SyncedStr
}

func (action *attachStorage) String() string {
	return fmt.Sprintf("Attach%s for %s", action.stage.Get(), action.storage)
}

func (action *attachStorage) Apply() error {
	log.WithFields(log.Fields{
		"storage": action.storage,
	}).Info("StorageNode.attachStorage.Apply")

	clusterTarget, err := composeClusterPrereq(action.storage, cluster.Spawned)
	if err != nil {
		return err
	}

	action.stage.Set(":getting imported resources")

	networkResources, err := cluster.GetNetworkResources(clusterTarget.Thing)
	if err != nil {
		log.WithFields(log.Fields{
			"storage": action.storage,
		}).Errorf("StorageNode.attachStorage: cannot get cluster network resources: %s", err)

		return err
	}

	configFilesDir := action.storage.getConfigFilesDir()

	tfLogPrefix, err := action.storage.makeToolLogPrefix("terraform")
	if err != nil {
		log.WithFields(log.Fields{
			"storage": action.storage,
		}).Warnf("StorageNode.attachStorage: cannot make logfile name: %s", err)
	}

	newTfState, currentTfState := filepath.Join(configFilesDir, "imported.tfstate"),
		filepath.Join(configFilesDir, "terraform.tfstate")

	_, statErr := os.Stat(currentTfState)
	if action.storage.status != Configured || os.IsExist(statErr) {
		if err := storage_pkg.CopyFile(currentTfState, newTfState); err != nil {
			log.WithFields(log.Fields{
				"storage-dir": configFilesDir,
			}).Errorf("StorageNode.attachStorage: cannot copy state: %s", err)

			return err
		}
	}

	if disabler, err := enableConfig("attached", action.storage.attachedConfigPath); err == nil {
		defer disabler()
	} else {
		return err
	}

	diskResourceName := action.storage.provider.GetTFStorageResourceName()
	if action.storage.status == Configured {
		if _, err = action_pkg.RunLoggedCmdDir(tfLogPrefix, configFilesDir, provider.Terraform(),
			"import", "-no-color", "-state="+newTfState, diskResourceName+".storage", action.storage.name+"-disk"); err != nil {
			log.WithFields(log.Fields{
				"storage-dir": configFilesDir,
			}).Infof("StorageNode.attachStorage: cannot import disk, err=%s", err)
		} else {
			log.WithFields(log.Fields{
				"storage-dir": configFilesDir,
			}).Info("StorageNode.attachStorage: successfully imported disk")
		}
	}

	for _, resource := range networkResources {
		if logname, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, configFilesDir, provider.Terraform(),
			"import", "-no-color", "-state="+newTfState, resource.Address, resource.ID); err != nil {
			log.WithFields(log.Fields{
				"storage-dir": configFilesDir,
			}).Errorf("StorageNode.attachStorage: cannot import resource %s: %s", resource.Address, err)
			fmt.Fprintf(os.Stderr, "Cannot import resource, see log for details: %s\n", logname)

			return err
		}
	}

	action.stage.Reset()

	if logname, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, configFilesDir, provider.Terraform(),
		"apply", "-no-color", "-auto-approve", "-state="+newTfState); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
		}).Errorf("StorageNode.attachStorage: cannot attach storage: %s", err)
		fmt.Fprintf(os.Stderr, "Cannot attach storage to cluser, see log for details: %s\n", logname)

		return err
	}

	if err := os.Rename(newTfState, currentTfState); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
		}).Errorf("StorageNode.attachStorage: cannot rename state: %s", err)

		return err
	}

	action.stage.Set(":getting connect info")

	if err := refreshConnectDetails(action.storage, Attached); err != nil {
		log.WithField("storage", action.storage).Errorf(
			"StorageNode.attachStorage: cannot refresh connection details: %s", err)

		return err
	}

	action.stage.Reset()

	action.storage.importedResources = networkResources

	return nil
}

func (action *attachStorage) IsExclusive() bool {
	return false
}

func (action *attachStorage) Prerequisites() ([]controller.Target, error) {
	clusterTarget, err := composeClusterPrereq(action.storage, cluster.Spawned)
	if err != nil {
		return []controller.Target{}, err
	}

	imageTarget, err := composeImagePrereq(action.storage, image.Created)
	if err != nil {
		return []controller.Target{}, err
	}

	return []controller.Target{imageTarget, clusterTarget}, nil
}

type detachStorage struct {
	storage *storageNodeState
	stage   common.SyncedStr
}

func (action *detachStorage) String() string {
	return fmt.Sprintf("Detach%s for %s", action.stage.Get(), action.storage)
}

func (action *detachStorage) Apply() error {
	log.WithFields(log.Fields{
		"storage": action.storage,
	}).Info("StorageNode.detachStorage.Apply")

	configFilesDir := action.storage.getConfigFilesDir()

	tfLogPrefix, err := action.storage.makeToolLogPrefix("terraform")
	if err != nil {
		log.WithFields(log.Fields{
			"storage": action.storage,
		}).Warnf("StorageNode.detachStorage: cannot make logfile name: %s", err)
	}

	attachedDisabler, err := enableConfig("attached", action.storage.attachedConfigPath)
	if err != nil {
		return err
	}

	defer attachedDisabler()

	clusterTarget, err := composeClusterPrereq(action.storage, cluster.Spawned)
	if err != nil {
		return err
	}

	action.stage.Set(":getting imported resources")

	networkResources, err := cluster.GetNetworkResources(clusterTarget.Thing)
	if err != nil {
		log.WithFields(log.Fields{
			"storage": action.storage,
		}).Errorf("StorageNode.detachStorage: cannot get cluster network resources: %s", err)

		return err
	}

	newTfState, currentTfState := filepath.Join(configFilesDir, "destroying.tfstate"),
		filepath.Join(configFilesDir, "terraform.tfstate")
	if err := storage_pkg.CopyFile(currentTfState, newTfState); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
		}).Errorf("StorageNode.detachStorage: cannot copy state: %s", err)

		return err
	}

	stateRmArgs := []string{"state", "rm", "-state=" + newTfState}

	for _, resource := range networkResources {
		stateRmArgs = append(stateRmArgs, resource.Address)
	}

	if logname, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, configFilesDir, provider.Terraform(),
		stateRmArgs...); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
		}).Errorf("StorageNode.detachStorage: cannot remove resources not managed by storage node: %s", err)
		fmt.Fprintf(os.Stderr, "Cannot remove imported resources, see log for details: %s\n", logname)

		return err
	}

	attachedDisabler() // remove "attached" config before enabling other

	if disabler, err := enableConfig("standalone", action.storage.configPath); err == nil {
		defer disabler()
	} else {
		return err
	}

	action.stage.Reset()

	if logname, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, configFilesDir, provider.Terraform(),
		"apply", "-no-color", "-auto-approve", "-state="+newTfState); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
		}).Errorf("StorageNode.detachStorage: cannot detach storage: %s", err)
		fmt.Fprintf(os.Stderr, "Cannot detach storage node, see log for details: %s\n", logname)

		return err
	}

	if err := os.Rename(newTfState, currentTfState); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": configFilesDir,
		}).Errorf("StorageNode.detachStorage: cannot rename state: %s", err)

		return err
	}

	action.stage.Set(":getting connect info")

	if err := refreshConnectDetails(action.storage, Detached); err != nil {
		log.WithField("storage", action.storage).Errorf(
			"StorageNode.detachStorage: cannot refresh connection details: %s", err)
		return err
	}

	action.stage.Reset()

	return nil
}

func (action *detachStorage) IsExclusive() bool {
	return false
}

func (action *detachStorage) Prerequisites() ([]controller.Target, error) {
	clusterTarget, err := composeClusterPrereq(action.storage, cluster.Spawned)
	if err != nil {
		return []controller.Target{}, err
	}

	return []controller.Target{clusterTarget}, nil
}
