package cluster

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"Rhoc/pkg/config"
	"Rhoc/pkg/controller"
	"Rhoc/pkg/provider"
	"Rhoc/pkg/state"
	"Rhoc/pkg/storage"
)

// Status describes status of the cluster
type Status int

const (
	// Nothing - nothing has been done to the cluster
	Nothing Status = iota
	// Configured - cluster config file has been created
	Configured
	// Spawned - cluster has been spawned in the cloud and is ready to use
	Spawned
)

const (
	category  = "cluster-configs"
	configExt = ".tf.json"
)

// Satisfies being true means this status satisfies required "other" status
func (s Status) Satisfies(other controller.Status) bool {
	if casted, ok := other.(Status); ok {
		return s >= casted
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
		Spawned:    "spawned",
	}

	transitions = map[Status][]controller.Status{
		Nothing:    { /*Configured - this transition is not yet implemented*/ },
		Configured: {Nothing, Spawned},
		Spawned:    {Configured},
	}
)

func (s Status) String() string {
	result, ok := statusToString[s]
	if !ok {
		return "unknown"
	}

	return result
}

type clusterState struct {
	status        Status
	name          string
	imageName     string
	imageOwners   string
	provider      provider.Provider
	templatePath  string
	configPath    string
	userVariables config.Config

	connection ConnectDetails

	fetcher       state.Fetcher
	serviceParams config.ServiceParams
}

func (cluster *clusterState) String() string {
	return fmt.Sprintf("Cluster(name=%s, image=%s, status=%s)", cluster.name, cluster.imageName, cluster.status)
}

func (cluster *clusterState) GetDestroyedTarget() controller.Target {
	return controller.Target{
		Thing:         cluster,
		DesiredStatus: Configured,
		MatchExact:    true,
	}
}

func (cluster *clusterState) getClusterDir() string {
	clusterFolderPath, _ := filepath.Split(cluster.configPath)
	return clusterFolderPath
}

func (cluster *clusterState) Status() controller.Status {
	return cluster.status
}

func (cluster *clusterState) SetStatus(status controller.Status) error {
	log.Info("Cluster.SetStatus called")

	casted, ok := status.(Status)
	if !ok {
		return fmt.Errorf("cannot set status of cluster - wrong type")
	}

	cluster.status = casted
	err := cluster.fetcher.Save(cluster)

	log.WithFields(log.Fields{
		"cluster":     *cluster,
		"new-status":  casted,
		"save-result": err,
	}).Info("status saved")

	return err
}

func (cluster *clusterState) GetTransitions(to controller.Status) ([]controller.Status, error) {
	casted, ok := to.(Status)
	if !ok {
		return nil, fmt.Errorf("cannot get transitions to status %v - not a cluster status", to)
	}

	result, ok := transitions[casted]
	if !ok {
		return nil, fmt.Errorf("unexpected cluster status %v", to)
	}

	return result, nil
}

func (cluster *clusterState) Equals(other controller.Thing) bool {
	casted, ok := other.(*clusterState)
	if !ok {
		return false
	}

	return cluster.status.Equals(casted.status) &&
		cluster.name == casted.name &&
		cluster.imageName == casted.imageName &&
		cluster.provider.Equals(casted.provider) &&
		cluster.templatePath == casted.templatePath &&
		cluster.configPath == casted.configPath &&
		cluster.getUserVars() == casted.getUserVars()
}

func (cluster *clusterState) GetAction(current controller.Status, target controller.Status) (
	controller.Action, error) {
	currentStatus, ok := current.(Status)
	if !ok {
		return nil, fmt.Errorf("current status %v is not cluster status", current)
	}

	targetStatus, ok := target.(Status)
	if !ok {
		return nil, fmt.Errorf("target status %v is not cluster status", target)
	}

	switch currentStatus {
	case Nothing:
		if targetStatus == Configured {
			return &makeConfig{cluster: cluster}, nil
		}
	case Configured:
		if targetStatus == Spawned {
			return &spawnCluster{cluster: cluster}, nil
		}
	case Spawned:
		if targetStatus == Configured {
			return &destroyCluster{cluster: cluster}, nil
		}
	}

	return nil, fmt.Errorf("unsupported transition of (%v => %v)", currentStatus, targetStatus)
}

func (cluster *clusterState) makeToolLogPrefix(tool string) (string, error) {
	hier, err := getHierarchy(cluster.provider, cluster.name)
	if err != nil {
		log.WithFields(log.Fields{
			"cluster": cluster,
		}).Errorf("Cluster.makeToolLogPrefix: cannot compute hierarchy: %s", err)

		return "", err
	}

	return storage.MakeStorageFilename(storage.LogCategory, append(hier, tool), ""), nil
}

func getVariable(userVariables config.Config, clusterTemplatePath string, varname string) (string, error) {
	value, err := userVariables.GetString(varname)
	if err != nil {
		template, err := config.CreateJSONConfigFromFile(clusterTemplatePath)
		if err != nil {
			log.WithFields(log.Fields{
				"config-path": clusterTemplatePath,
			}).Errorf("getVariable: cannot read default config: %s", err)

			return "", err
		}

		value, err = template.GetString(fmt.Sprintf("variable.%s.default", varname))
		if err != nil {
			log.WithFields(log.Fields{
				"config-path": clusterTemplatePath,
			}).Errorf("getVariable: cannot read default %s: %s", varname, err)

			return "", err
		}
	}

	return value, nil
}

// CreateClusterTarget creates a Thing for controller package
// that represents the cluster described by userVariables
func CreateClusterTarget(prov provider.Provider, userVariables config.Config,
	serviceParams config.ServiceParams, fetcher state.Fetcher) (controller.Thing, error) {
	if err := prov.CheckUserVars(userVariables); err != nil {
		log.Errorf("Cluster.CreateClusterTarget: user variables aren't supported or correct")

		return nil, err
	}

	clusterTemplatePath, err := provider.GetDefaultTemplate(prov.GetName(), provider.ClusterDescriptor)
	if err != nil {
		log.WithFields(log.Fields{
			"providerName": prov.GetName(),
		}).Errorf("CreateCluster: cannot get template: %s", err)

		return nil, err
	}

	name, err := getVariable(userVariables, clusterTemplatePath, "cluster_name")
	if err != nil {
		return nil, err
	}

	imageName, err := getVariable(userVariables, clusterTemplatePath, "image_name")
	if err != nil {
		return nil, err
	}

	imageOwners, err := getVariable(userVariables, clusterTemplatePath, "owners")
	if err != nil {
		return nil, err
	}

	hier, err := getHierarchy(prov, name)
	if err != nil {
		return nil, err
	}

	clusterConfigPath := storage.MakeStorageFilename(category, append(hier, "config"), configExt)
	cluster := clusterState{
		status:        Nothing,
		name:          name,
		imageName:     imageName,
		imageOwners:   imageOwners,
		provider:      prov,
		templatePath:  clusterTemplatePath,
		configPath:    clusterConfigPath,
		userVariables: userVariables,
		fetcher:       fetcher,
		serviceParams: serviceParams,
	}

	if clusterFromDisk, err := cluster.fetcher.Load(&cluster); err == nil {
		if clusterFromDisk != nil {
			cluster = *clusterFromDisk.(*clusterState)

			log.WithFields(log.Fields{
				"name":       name,
				"image-name": imageName,
			}).Info("CreateCluster: cluster state loaded from disk")
		} else {
			log.WithFields(log.Fields{
				"name":       name,
				"image-name": imageName,
			}).Info("CreateCluster: cluster state not found on disk")
		}
	} else {
		log.WithFields(log.Fields{
			"name":       name,
			"image-name": imageName,
		}).Errorf("CreateCluster: cannot load cluster state: %s", err)

		return nil, err
	}

	return &cluster, nil
}
