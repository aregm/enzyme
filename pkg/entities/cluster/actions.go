package cluster

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	action_pkg "Rhoc/pkg/action"
	"Rhoc/pkg/controller"
	"Rhoc/pkg/entities/common"
	"Rhoc/pkg/entities/image"
	"Rhoc/pkg/provider"
)

type makeConfig struct {
	cluster *clusterState
}

func (action makeConfig) String() string {
	return fmt.Sprintf("Configure for %s", action.cluster)
}

func (action makeConfig) Apply() error {
	log.WithFields(log.Fields{
		"cluster": action.cluster,
	}).Info("Cluster.makeConfig.Apply")

	clusterConfig, err := action.cluster.provider.MakeCreateClusterConfig(action.cluster.templatePath,
		action.cluster.userVariables)
	if err != nil {
		log.WithFields(log.Fields{
			"clusterTemplatePath": action.cluster.templatePath,
		}).Errorf("Cluster.makeConfig: cannot create config object: %s", err)

		return err
	}

	if err = clusterConfig.Serialize(action.cluster.configPath); err != nil {
		log.WithFields(log.Fields{
			"clusterConfigPath": action.cluster.configPath,
		}).Errorf("Cluster.makeConfig: cannot save config: %s", err)

		return err
	}

	clusterDir := action.cluster.getClusterDir()

	tfLogPrefix, err := action.cluster.makeToolLogPrefix("terraform")
	if err != nil {
		log.WithFields(log.Fields{
			"cluster": action.cluster,
		}).Warnf("Cluster.makeConfig: cannot make logfile name: %s", err)
	}

	if logname, err :=
		action_pkg.RunLoggedCmdDir(tfLogPrefix, clusterDir, provider.Terraform(), "init"); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": clusterDir,
		}).Errorf("Cluster.makeConfig: error initializing: %s", err)
		fmt.Fprintf(os.Stderr, "Failed to initialize tools, see log for details: %s\n", logname)

		return err
	}

	return nil
}

func composeImagePrereq(cluster *clusterState, desiredStatus image.Status) (controller.Target, error) {
	imageTarget, err := image.CreateImageTarget(cluster.provider, cluster.userVariables, cluster.serviceParams,
		cluster.fetcher)
	if err != nil {
		log.WithFields(log.Fields{
			"cluster": cluster,
		}).Errorf("composeImagePrereq: cannot make image target: %s", err)

		return controller.Target{}, err
	}

	return controller.Target{
		Thing:         imageTarget,
		DesiredStatus: desiredStatus,
	}, nil
}

func (action makeConfig) IsExclusive() bool {
	return false
}

func (action makeConfig) Prerequisites() ([]controller.Target, error) {
	imageTarget, err := composeImagePrereq(action.cluster, image.Configured)
	if err != nil {
		return []controller.Target{}, err
	}

	return []controller.Target{imageTarget}, nil
}

type spawnCluster struct {
	cluster *clusterState
	stage   common.SyncedStr
}

func (action *spawnCluster) String() string {
	return fmt.Sprintf("Spawn%s for %s", action.stage.Get(), action.cluster)
}

func (action *spawnCluster) Apply() error {
	log.WithFields(log.Fields{
		"cluster": action.cluster,
	}).Info("Cluster.spawnCluster.Apply")

	clusterDir := action.cluster.getClusterDir()

	tfLogPrefix, err := action.cluster.makeToolLogPrefix("terraform")
	if err != nil {
		log.WithFields(log.Fields{
			"cluster": action.cluster,
		}).Warnf("Cluster.spawnCluster: cannot make logfile name: %s", err)
	}

	if logname, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, clusterDir, provider.Terraform(),
		"apply", "-auto-approve"); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": clusterDir,
		}).Errorf("Cluster.spawnCluster: error spawning cluster: %s", err)
		fmt.Fprintf(os.Stderr, "Cannot spawn cluster, see log for details: %s\n", logname)

		log.Info("Cluster.spawnCluster: destroying half-spawned cluster ...")
		action.stage.Set(":destroying half-spawned")

		if logname, destroyErr := action_pkg.RunLoggedCmdDir(tfLogPrefix, clusterDir, provider.Terraform(),
			"destroy", "-force"); destroyErr != nil {
			log.WithFields(log.Fields{
				"storage-dir": clusterDir,
			}).Errorf("Cluster.spawnCluster: error destroying half-spawned cluster: %s, you can try to "+
				"destroy manually by going to %s and working with 'terraform destroy'", destroyErr, clusterDir)
			fmt.Fprintf(os.Stderr, "Cannot destroy half-spawned cluster, see log for details: %s\n", logname)
		}

		return err
	}

	action.stage.Set(":getting connect info")

	if err := refreshConnectDetails(action.cluster, Spawned); err != nil {
		log.WithField("cluster", action.cluster).Errorf(
			"Cluster.spawnCluster: cannot get connection details: %s", err)
		return err
	}

	action.stage.Reset()

	return nil
}

func (action *spawnCluster) IsExclusive() bool {
	return false
}

func (action *spawnCluster) Prerequisites() ([]controller.Target, error) {
	imageTarget, err := composeImagePrereq(action.cluster, image.Created)
	if err != nil {
		return []controller.Target{}, err
	}

	return []controller.Target{imageTarget}, nil
}

type destroyCluster struct {
	cluster *clusterState
}

func (action destroyCluster) String() string {
	return fmt.Sprintf("Destroy for %s", action.cluster)
}

func (action destroyCluster) Apply() error {
	log.WithFields(log.Fields{
		"cluster": action.cluster,
	}).Info("Cluster.destroyCluster.Apply")

	clusterDir := action.cluster.getClusterDir()

	tfLogPrefix, err := action.cluster.makeToolLogPrefix("terraform")
	if err != nil {
		log.WithFields(log.Fields{
			"cluster": action.cluster,
		}).Warnf("Cluster.destroyCluster: cannot make logfile name: %s", err)
	}

	// reset connect details as cluster is being destroyed now, so assume it's no longer accessible
	action.cluster.connection = ConnectDetails{}

	if logname, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, clusterDir, provider.Terraform(),
		"destroy", "-force"); err != nil {
		log.WithFields(log.Fields{
			"storage-dir": clusterDir,
		}).Errorf("Cluster.destroyCluster: error destroying cluster: %s", err)
		fmt.Fprintf(os.Stderr, "Cannot destroy cluser, see log for details: %s\n", logname)

		return err
	}

	return nil
}

func (action destroyCluster) IsExclusive() bool {
	return false
}

func (action destroyCluster) Prerequisites() ([]controller.Target, error) {
	return []controller.Target{}, nil
}
