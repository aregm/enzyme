package storage

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"Rhoc/pkg/entities/cluster"
	"Rhoc/pkg/provider"
	"Rhoc/pkg/state"
)

func getHierarchy(prov provider.Provider, name string) ([]string, error) {
	memorizedID, err := provider.MemorizedID(prov)
	if err != nil {
		return []string{}, err
	}

	return []string{memorizedID, name}, nil
}

// implement state.Entry methods

func (storage *storageNodeState) Hierarchy() ([]string, error) {
	hier, err := getHierarchy(storage.provider, storage.name)
	if err != nil {
		return []string{}, err
	}

	return append([]string{"storage"}, hier...), nil
}

func handler(hier []string, fetcher state.Fetcher) state.Entry {
	if len(hier) != 0 && hier[0] == "storage" {
		return &storageNodeState{
			fetcher: fetcher,
		}
	}

	return nil
}

type providerPersist struct {
	Name   string
	Region string
	Zone   string
	Creds  string
}

type userVarsPersist struct {
	KeyName        string
	InstanceType   string
	UserName       string
	ProjectName    string
	SSHKeyPairPath string
}

type persistent struct {
	Status               int
	Name                 string
	ImageName            string
	DiskSize             string
	Provider             providerPersist
	TemplatePath         string
	AttachedTemplatePath string
	ConfigPath           string
	AttachedConfigPath   string
	ImportedResources    []cluster.ResourceDescriptor
	Connection           ConnectDetails

	UserVars userVarsPersist
}

func (storage *storageNodeState) getProviderVars() providerPersist {
	if storage.provider == nil {
		return providerPersist{}
	}

	return providerPersist{
		storage.provider.GetName(),
		storage.provider.GetRegion(),
		storage.provider.GetZone(),
		storage.provider.GetCredentialPath(),
	}
}

func (storage *storageNodeState) getUserVars() userVarsPersist {
	if storage.userVariables == nil {
		return userVarsPersist{}
	}

	var result userVarsPersist

	var err error

	if result.KeyName, err = storage.userVariables.GetString("storage_key_name"); err != nil {
		result.KeyName = ""
	}

	if result.InstanceType, err = storage.userVariables.GetString("storage_instance_type"); err != nil {
		result.InstanceType = ""
	}

	if result.UserName, err = storage.userVariables.GetString("user_name"); err != nil {
		result.UserName = ""
	}

	if result.ProjectName, err = storage.userVariables.GetString("project_name"); err != nil {
		result.ProjectName = ""
	}

	if result.SSHKeyPairPath, err = storage.userVariables.GetString("ssh_key_pair_path"); err != nil {
		result.SSHKeyPairPath = ""
	}

	return result
}

func (storage *storageNodeState) ToPublic() (interface{}, error) {
	return persistent{
		int(storage.status),
		storage.name,
		storage.imageName,
		storage.diskSize,
		storage.getProviderVars(),
		storage.templatePath,
		storage.attachedTemplatePath,
		storage.configPath,
		storage.attachedConfigPath,
		storage.importedResources,
		storage.connection,
		storage.getUserVars(),
	}, nil
}

func (storage *storageNodeState) FromPublic(v interface{}) (state.Entry, error) {
	persist, ok := v.(persistent)
	if !ok {
		pPersist, ok := v.(*persistent)
		if !ok {
			log.WithFields(log.Fields{
				"read": v,
			}).Error("StorageNode.FromPublic: cannot parse incoming object as storage node state")

			return nil, fmt.Errorf("incompatible intermediate type")
		}

		persist = *pPersist
	}

	if storage.userVariables != nil {
		userVars := storage.getUserVars()
		if persist.UserVars != userVars {
			log.WithFields(log.Fields{
				"loaded-vars":  persist.UserVars,
				"current-vars": userVars,
			}).Info("StorageNode.FromPublic: loaded user variables differ from current, invalidating stored state")

			return nil, nil
		}
	} else {
		log.WithFields(log.Fields{
			"loaded-vars": persist.UserVars,
		}).Info("StorageNode.FromPublic: target doesn't have variables set, use with caution")
	}

	status := Status(persist.Status)
	if status < Nothing || status > Attached {
		log.WithFields(log.Fields{
			"read": v,
		}).Errorf("StorageNode.FromPublic: incoming status is unexpected: %d", persist.Status)

		return nil, fmt.Errorf("unexpected status: %d", persist.Status)
	}

	prov, err := provider.CreateProvider(persist.Provider.Name, persist.Provider.Region,
		persist.Provider.Zone, persist.Provider.Creds)
	if err != nil {
		log.WithFields(log.Fields{
			"read": v,
		}).Errorf("StorageNode.FromPublic: cannot construct provider: %s", err)

		return nil, err
	}

	return &storageNodeState{
		status,
		persist.Name,
		persist.ImageName,
		persist.DiskSize,
		prov,
		persist.TemplatePath,
		persist.AttachedTemplatePath,
		persist.ConfigPath,
		persist.AttachedConfigPath,
		storage.userVariables,
		persist.ImportedResources,
		persist.Connection,
		storage.fetcher,
		storage.serviceParams,
	}, nil
}

func init() {
	state.RegisterHandler(handler)
}
