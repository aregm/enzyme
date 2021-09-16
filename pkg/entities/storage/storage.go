package storage

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"enzyme/pkg/config"
	"enzyme/pkg/controller"
	"enzyme/pkg/entities/cluster"
	"enzyme/pkg/provider"
	"enzyme/pkg/state"
	storage_pkg "enzyme/pkg/storage"
)

// Status describes status of the storage node
type Status int

const (
	// Nothing - nothing has been done to the storage node
	Nothing Status = iota
	// Configured - storage node config file has been created
	Configured
	// Detached - storage node is running as a separate entity
	Detached
	// Attached - storage node is attached to a cluster
	Attached
)

const (
	category          = "storage-configs"
	configExt         = ".tf.json"
	configExtDisabled = ".disabled.json"
)

// Satisfies being true means this status satisfies required "other" status
func (s Status) Satisfies(other controller.Status) bool {
	if casted, ok := other.(Status); ok {
		// Attached is handled separately because a node in Attached status
		// does not satisfy a "Detached" requirement and vice versa
		return (s == Attached && s == casted) || (s < Attached && s >= casted)
	}

	return false
}

// Equals is only true if "other" status is exactly equal to this status
func (s Status) Equals(other controller.Status) bool {
	if casted, ok := other.(Status); ok {
		return s == casted
	}

	return false
}

var (
	statusToString = map[Status]string{
		Nothing:    "nothing",
		Configured: "configured",
		Detached:   "standalone access",
		Attached:   "attached to cluster",
	}
	// transitions maps "to" status to all "from" statuses where an action from "from" to "to" exists
	transitions = map[Status][]controller.Status{
		Nothing:    {},
		Configured: {Nothing, Detached, Attached},
		Detached:   {Configured, Attached},
		Attached:   {Detached, Configured},
	}
)

func (s Status) String() string {
	result, ok := statusToString[s]
	if !ok {
		return "unknown"
	}

	return result
}

type storageNodeState struct {
	status               Status
	name                 string
	imageName            string
	diskSize             string
	provider             provider.Provider
	templatePath         string
	attachedTemplatePath string
	configPath           string
	attachedConfigPath   string
	userVariables        config.Config

	// resources imported during detached->attached transition
	importedResources []cluster.ResourceDescriptor

	connection ConnectDetails

	fetcher       state.Fetcher
	serviceParams config.ServiceParams
}

func (storage *storageNodeState) String() string {
	return fmt.Sprintf("Storage(name=%s, size=%s, status=%s)", storage.name, storage.diskSize, storage.status)
}

func (storage *storageNodeState) GetDestroyedTarget() controller.Target {
	return controller.Target{
		Thing:         storage,
		DesiredStatus: Configured,
		MatchExact:    true,
	}
}

func (storage *storageNodeState) getConfigFilesDir() string {
	storageConfigDir, _ := filepath.Split(storage.configPath)
	return storageConfigDir
}

func (storage *storageNodeState) Status() controller.Status {
	return storage.status
}

func (storage *storageNodeState) SetStatus(status controller.Status) error {
	log.Info("StorageNode.SetStatus called")

	casted, ok := status.(Status)
	if !ok {
		return fmt.Errorf("cannot set status of storage node - wrong type")
	}

	storage.status = casted
	err := storage.fetcher.Save(storage)

	log.WithFields(log.Fields{
		"storage":     *storage,
		"new-status":  casted,
		"save-result": err,
	}).Info("status saved")

	return err
}

func (storage *storageNodeState) GetTransitions(to controller.Status) ([]controller.Status, error) {
	casted, ok := to.(Status)
	if !ok {
		return nil, fmt.Errorf("cannot get transitions to status %v - not a storage node status", to)
	}

	result, ok := transitions[casted]
	if !ok {
		return nil, fmt.Errorf("unexpected storage node status %v", to)
	}

	return result, nil
}

func (storage *storageNodeState) Equals(other controller.Thing) bool {
	casted, ok := other.(*storageNodeState)
	if !ok {
		return false
	}

	return storage.status.Equals(casted.status) &&
		storage.name == casted.name &&
		storage.imageName == casted.imageName &&
		storage.diskSize == casted.diskSize &&
		storage.provider.Equals(casted.provider) &&
		storage.templatePath == casted.templatePath &&
		storage.configPath == casted.configPath &&
		storage.getUserVars() == casted.getUserVars()
}

func (storage *storageNodeState) GetAction(current controller.Status,
	target controller.Status) (controller.Action, error) {
	currentStatus, ok := current.(Status)
	if !ok {
		return nil, fmt.Errorf("current status %v is not a storage node status", current)
	}

	targetStatus, ok := target.(Status)
	if !ok {
		return nil, fmt.Errorf("target status %v is not a storage node status", target)
	}

	switch currentStatus {
	case Nothing:
		if targetStatus == Configured {
			return &makeConfig{storage: storage}, nil
		}
	case Configured:
		switch targetStatus {
		case Detached:
			return &spawnStorage{storage: storage}, nil
		case Attached:
			return &attachStorage{storage: storage}, nil
		}
	case Detached:
		switch targetStatus {
		case Configured:
			return &destroyStorage{storage: storage}, nil
		case Attached:
			return &attachStorage{storage: storage}, nil
		}
	case Attached:
		switch targetStatus {
		case Detached:
			return &detachStorage{storage: storage}, nil
		case Configured:
			return &destroyStorage{storage: storage}, nil
		}
	}

	return nil, fmt.Errorf("unsupported transition of (%v => %v)", currentStatus, targetStatus)
}

func (storage *storageNodeState) makeToolLogPrefix(tool string) (string, error) {
	hier, err := getHierarchy(storage.provider, storage.name)
	if err != nil {
		log.WithFields(log.Fields{
			"storage": storage,
		}).Errorf("StorageNode.makeToolLogPrefix: cannot compute hierarchy: %s", err)

		return "", err
	}

	return storage_pkg.MakeStorageFilename(storage_pkg.LogCategory, append(hier, tool), ""), nil
}

func getVariable(userVariables config.Config, templatePath string, varname string) (string, error) {
	value, err := userVariables.GetString(varname)
	if err != nil {
		template, err := config.CreateJSONConfigFromFile(templatePath)
		if err != nil {
			log.WithFields(log.Fields{
				"config-path": templatePath,
			}).Errorf("getVariable: cannot read default config: %s", err)

			return "", err
		}

		value, err = template.GetString(fmt.Sprintf("variable.%s.default", varname))
		if err != nil {
			log.WithFields(log.Fields{
				"config-path": templatePath,
			}).Errorf("getVariable: cannot read default %s: %s", varname, err)

			return "", err
		}
	}

	return value, nil
}

// CreateStorageTarget creates a Thing for controller package
// that represents the storage node described by userVariables
func CreateStorageTarget(prov provider.Provider, userVariables config.Config,
	serviceParams config.ServiceParams, fetcher state.Fetcher) (controller.Thing, error) {
	if err := prov.CheckUserVars(userVariables); err != nil {
		log.Errorf("Storage.CreateStorageTarget: user variables aren't supported or correct")

		return nil, err
	}

	storageTemplatePath, err := provider.GetDefaultTemplate(prov.GetName(), provider.StorageNodeDescriptor)
	if err != nil {
		log.WithFields(log.Fields{
			"providerName": prov.GetName(),
		}).Errorf("CreateStorageTarget: cannot get template: %s", err)

		return nil, err
	}

	attachedTemplatePath, err :=
		provider.GetDefaultTemplate(prov.GetName(), provider.StorageAttachedDescriptor)
	if err != nil {
		log.WithFields(log.Fields{
			"providerName": prov.GetName(),
		}).Errorf("CreateStorageTarget: cannot get attached template: %s", err)

		return nil, err
	}

	name, err := getVariable(userVariables, storageTemplatePath, "storage_name")
	if err != nil {
		return nil, err
	}

	imageName, err := getVariable(userVariables, storageTemplatePath, "image_name")
	if err != nil {
		return nil, err
	}

	diskSize, err := getVariable(userVariables, storageTemplatePath, "storage_disk_size")
	if err != nil {
		return nil, err
	}

	hier, err := getHierarchy(prov, name)
	if err != nil {
		return nil, err
	}

	configPath := storage_pkg.MakeStorageFilename(category, append(hier, "config"), configExtDisabled)
	attachedConfigPath := storage_pkg.MakeStorageFilename(category, append(hier, "config-attached"),
		configExtDisabled)
	storage := storageNodeState{
		status:               Nothing,
		name:                 name,
		imageName:            imageName,
		diskSize:             diskSize,
		provider:             prov,
		templatePath:         storageTemplatePath,
		attachedTemplatePath: attachedTemplatePath,
		configPath:           configPath,
		attachedConfigPath:   attachedConfigPath,
		userVariables:        userVariables,
		fetcher:              fetcher,
		serviceParams:        serviceParams,
	}

	if storageFromDisk, err := storage.fetcher.Load(&storage); err == nil {
		if storageFromDisk != nil {
			storage = *storageFromDisk.(*storageNodeState)

			log.WithFields(log.Fields{
				"name": name,
			}).Info("CreateStorageTarget: storage node state loaded from disk")
		} else {
			log.WithFields(log.Fields{
				"name": name,
			}).Info("CreateStorageTarget: storage node state not found on disk")
		}
	} else {
		log.WithFields(log.Fields{
			"name": name,
		}).Errorf("CreateStorageTarget: cannot load storage node state: %s", err)

		return nil, err
	}

	return &storage, nil
}
