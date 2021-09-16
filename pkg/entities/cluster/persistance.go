package cluster

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"enzyme/pkg/provider"
	"enzyme/pkg/state"
)

func getHierarchy(prov provider.Provider, name string) ([]string, error) {
	memorizedID, err := provider.MemorizedID(prov)
	if err != nil {
		return []string{}, err
	}

	return []string{memorizedID, name}, nil
}

// implement state.Entry methods

func (cluster *clusterState) Hierarchy() ([]string, error) {
	hier, err := getHierarchy(cluster.provider, cluster.name)
	if err != nil {
		return []string{}, err
	}

	return append([]string{"cluster"}, hier...), nil
}

type providerPersist struct {
	Name   string
	Region string
	Zone   string
	Creds  string
}

type userVarsPersist struct {
	KeyName            string
	WorkerCount        string
	LoginInstanceType  string
	WorkedInstanceType string
	LoginRootDiskSize  string
	UserName           string
	ProjectName        string
	SSHKeyPairPath     string
}

type persistent struct {
	Status       int
	Name         string
	ImageName    string
	ImageOwners  string
	Provider     providerPersist
	TemplatePath string
	ConfigPath   string
	UserVars     userVarsPersist

	Connection ConnectDetails
}

func (cluster *clusterState) getProviderVars() providerPersist {
	if cluster.provider == nil {
		return providerPersist{}
	}

	return providerPersist{
		cluster.provider.GetName(),
		cluster.provider.GetRegion(),
		cluster.provider.GetZone(),
		cluster.provider.GetCredentialPath(),
	}
}

func (cluster *clusterState) getUserVars() userVarsPersist {
	if cluster.userVariables == nil {
		return userVarsPersist{}
	}

	var result userVarsPersist

	var err error

	if result.KeyName, err = cluster.userVariables.GetString("key_name"); err != nil {
		result.KeyName = ""
	}

	if result.WorkerCount, err = cluster.userVariables.GetString("worker_count"); err != nil {
		result.WorkerCount = ""
	}

	if result.LoginInstanceType, err = cluster.userVariables.GetString("instance_type_login_node"); err != nil {
		result.LoginInstanceType = ""
	}

	if result.WorkedInstanceType, err =
		cluster.userVariables.GetString("instance_type_worker_node"); err != nil {
		result.WorkedInstanceType = ""
	}

	if result.LoginRootDiskSize, err = cluster.userVariables.GetString("login_node_root_size"); err != nil {
		result.LoginRootDiskSize = ""
	}

	if result.UserName, err = cluster.userVariables.GetString("user_name"); err != nil {
		result.UserName = ""
	}

	if result.ProjectName, err = cluster.userVariables.GetString("project_name"); err != nil {
		result.ProjectName = ""
	}

	if result.SSHKeyPairPath, err = cluster.userVariables.GetString("ssh_key_pair_path"); err != nil {
		result.SSHKeyPairPath = ""
	}

	return result
}

func (cluster *clusterState) ToPublic() (interface{}, error) {
	return persistent{
		int(cluster.status),
		cluster.name,
		cluster.imageName,
		cluster.imageOwners,
		cluster.getProviderVars(),
		cluster.templatePath,
		cluster.configPath,
		cluster.getUserVars(),
		cluster.connection,
	}, nil
}

func (cluster *clusterState) FromPublic(v interface{}) (state.Entry, error) {
	persist, ok := v.(persistent)
	if !ok {
		pPersist, ok := v.(*persistent)
		if !ok {
			log.WithFields(log.Fields{
				"read": v,
			}).Error("Cluster.FromPublic: cannot parse incoming object as clusterState")

			return nil, fmt.Errorf("incompatible intermediate type")
		}

		persist = *pPersist
	}

	if cluster.userVariables != nil {
		userVars := cluster.getUserVars()
		if persist.UserVars != userVars {
			log.WithFields(log.Fields{
				"loaded-vars":  persist.UserVars,
				"current-vars": userVars,
			}).Info("Cluster.FromPublic: loaded user variables differ from current, invalidating stored state")

			return nil, nil
		}
	} else {
		log.WithFields(log.Fields{
			"loaded-vars": persist.UserVars,
		}).Info("Cluster.FromPublic: target doesn't have variables set, use with caution")
	}

	status := Status(persist.Status)
	if status < Nothing || status > Spawned {
		log.WithFields(log.Fields{
			"read": v,
		}).Errorf("Cluster.FromPublic: incoming status is unexpected: %d", persist.Status)

		return nil, fmt.Errorf("unexpected status: %d", persist.Status)
	}

	prov, err := provider.CreateProvider(persist.Provider.Name, persist.Provider.Region,
		persist.Provider.Zone, persist.Provider.Creds)
	if err != nil {
		log.WithFields(log.Fields{
			"read": v,
		}).Errorf("Cluster.FromPublic: cannot construct provider: %s", err)

		return nil, err
	}

	return &clusterState{
		status,
		persist.Name,
		persist.ImageName,
		persist.ImageOwners,
		prov,
		persist.TemplatePath,
		persist.ConfigPath,
		cluster.userVariables,
		persist.Connection,
		cluster.fetcher,
		cluster.serviceParams,
	}, nil
}

func handler(hier []string, fetcher state.Fetcher) state.Entry {
	if len(hier) != 0 && hier[0] == "cluster" {
		return &clusterState{
			fetcher: fetcher,
		}
	}

	return nil
}

func init() {
	state.RegisterHandler(handler)
}
