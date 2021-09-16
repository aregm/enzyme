package runtask

import (
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"enzyme/pkg/controller"
	"enzyme/pkg/entities/cluster"
	"enzyme/pkg/entities/common"
	"enzyme/pkg/entities/storage"
	"enzyme/pkg/ssh"
)

func makeStageLogger(task *taskState, stage string) *log.Entry {
	return log.WithFields(log.Fields{
		"task":               task,
		"stage":              stage,
		"local-path":         task.localPath,
		"remote-path":        task.remotePath,
		"newline-conversion": task.convertNewline,
		"overwrite":          task.overwrite,
		"uploadFiles":        task.uploadFiles,
		"downloadFiles":      task.downloadFiles,
	})
}

func composeClusterPrereq(task *taskState, desiredStatus cluster.Status) (controller.Target, error) {
	clusterTarget, err :=
		cluster.CreateClusterTarget(task.provider, task.userVariables, task.serviceParameters, task.fetcher)
	if err != nil {
		log.WithFields(log.Fields{
			"task": task,
		}).Errorf("composeClusterPrereq: cannot make cluster target: %s", err)

		return controller.Target{}, err
	}

	return controller.Target{
		Thing:         clusterTarget,
		DesiredStatus: desiredStatus,
	}, nil
}

func composeStoragePrereq(task *taskState, desiredStatus storage.Status) (controller.Target, error) {
	storageTarget, err :=
		storage.CreateStorageTarget(task.provider, task.userVariables, task.serviceParameters, task.fetcher)
	if err != nil {
		log.WithFields(log.Fields{
			"task": task,
		}).Errorf("composeStoragePrereq: cannot make storage target: %s", err)

		return controller.Target{}, err
	}

	return controller.Target{
		Thing:         storageTarget,
		DesiredStatus: desiredStatus,
	}, nil
}

type makeConnection struct {
	task *taskState
}

func (action makeConnection) String() string {
	return fmt.Sprintf("GetConnectDetails for %s", action.task)
}

func (action makeConnection) Apply() error {
	clusterTarget, err := composeClusterPrereq(action.task, cluster.Spawned)
	if err != nil {
		return err
	}

	connect, err := cluster.GetConnectDetails(clusterTarget.Thing)
	if err != nil {
		log.WithFields(log.Fields{
			"task": action.task,
		}).Errorf("RunTask.makeConnection: cannot get connection details: %s", err)

		return err
	}

	log.WithFields(log.Fields{
		"task":        action.task,
		"hostname":    connect.PublicAddress,
		"username":    connect.UserName,
		"private-key": connect.PrivateKey,
	}).Info("RunTask.makeConnection: retrieved connection details")

	var socksProxy string = ""
	if action.task.serviceParameters.SocksProxyHost != "" {
		socksProxy = net.JoinHostPort(
			action.task.serviceParameters.SocksProxyHost,
			strconv.Itoa(action.task.serviceParameters.SocksProxyPort))
	}

	client, err := ssh.MakeZymeClient(connect.PublicAddress, connect.UserName, connect.PrivateKey, socksProxy)
	if err != nil {
		log.WithFields(log.Fields{
			"task":        action.task,
			"hostname":    connect.PublicAddress,
			"username":    connect.UserName,
			"private-key": connect.PrivateKey,
			"socks-proxy": socksProxy,
		}).Errorf("RunTask.makeConnection: cannot connect: %s", err)

		return err
	}

	action.task.client = client

	return nil
}

func (action makeConnection) IsExclusive() bool {
	return false
}

func (action makeConnection) Prerequisites() ([]controller.Target, error) {
	clusterTarget, err := composeClusterPrereq(action.task, cluster.Spawned)
	if err != nil {
		return []controller.Target{}, err
	}

	return []controller.Target{clusterTarget}, nil
}

func escapeArg(arg string) string {
	arg = strings.ReplaceAll(arg, `\`, `\\`)
	arg = strings.ReplaceAll(arg, `"`, `\"`)

	return fmt.Sprintf(`"%s"`, arg)
}

func expandExe(exe string) string {
	if !strings.HasPrefix(exe, "/") && !strings.HasPrefix(exe, "./") && !strings.HasPrefix(exe, "~/") {
		return "./" + exe
	}

	if strings.HasPrefix(exe, "~/") {
		// "~/" is not actually a path but a template for shell to substitute; we don't run it via shell,
		// so replace it with "./" because those two are equivalent
		// when running like "ssh user@host ~/some-exe"
		return "./" + strings.TrimPrefix(exe, "~/")
	}

	return exe
}

func runRemoteCommand(client ssh.ZymeClient, logger *log.Entry, exe string, args ...string) error {
	cmd := []string{escapeArg(exe)}

	for _, arg := range args {
		cmd = append(cmd, escapeArg(arg))
	}

	remoteCmd := strings.Join(cmd, " ")
	if err := client.ExecuteCommand(remoteCmd, true); err != nil {
		logger.WithField("command", remoteCmd).Errorf("runRemoteCommand: cannot execute command: %s", err)
		return err
	}

	return nil
}

type uploadData struct {
	task  *taskState
	stage common.SyncedStr
}

func (action *uploadData) String() string {
	return fmt.Sprintf("UploadData%s for %s", action.stage.Get(), action.task)
}

func (action *uploadData) Apply() error {
	logger := makeStageLogger(action.task, "upload-data")

	if action.task.useStorage {
		action.stage.Set(":mounting storage")

		storageTarget, err := composeStoragePrereq(action.task, storage.Attached)
		if err != nil {
			return err
		}

		connect, err := storage.GetConnectDetails(storageTarget.Thing)
		if err != nil {
			logger.Errorf("RunTask.uploadData: cannot get connection details for storage node: %s", err)
			return err
		}

		if err := runRemoteCommand(action.task.client, logger,
			expandExe("~/zyme-postprocess/storage/attach-on-head.sh"), connect.InternalAddress); err != nil {
			logger.Errorf("RunTask.uploadData: cannot attach storage node: %s", err)
			return err
		}
	}

	uploadFolder := "enzyme-upload"

	// upload desired files
	for _, localFilePath := range action.task.uploadFiles {
		_, localName := filepath.Split(localFilePath)
		remoteName := strings.Join([]string{uploadFolder, localName}, "/")

		action.stage.Set(fmt.Sprintf(":uploading %s", localName))

		if err :=
			action.task.client.PutFile(localFilePath, remoteName, false, action.task.overwrite, false); err != nil {
			logger.Errorf("RunTask.uploadData: cannot upload local file %s: %s", localFilePath, err)
			return err
		}
	}

	action.stage.Reset()

	if err := action.task.client.PutFile(action.task.localPath, action.task.remotePath,
		action.task.convertNewline, action.task.overwrite, true); err != nil {
		logger.Errorf("RunTask.uploadData: cannot upload local script: %s", err)
		return err
	}

	return nil
}

func (action *uploadData) IsExclusive() bool {
	return false
}

func (action *uploadData) Prerequisites() ([]controller.Target, error) {
	clusterTarget, err := composeClusterPrereq(action.task, cluster.Spawned)
	if err != nil {
		return []controller.Target{}, err
	}

	if !action.task.useStorage {
		return []controller.Target{clusterTarget}, nil
	}

	storageTarget, err := composeStoragePrereq(action.task, storage.Attached)
	if err != nil {
		return []controller.Target{}, err
	}

	return []controller.Target{clusterTarget, storageTarget}, nil
}

type runRemote struct {
	task *taskState
}

func (action runRemote) String() string {
	return fmt.Sprintf("RunRemote for %s", action.task)
}

func (action runRemote) Apply() error {
	logger := makeStageLogger(action.task, "run-remote")

	return runRemoteCommand(action.task.client, logger, expandExe(action.task.remotePath), action.task.args...)
}

func (action runRemote) IsExclusive() bool {
	return true
}

func (action runRemote) Prerequisites() ([]controller.Target, error) {
	clusterTarget, err := composeClusterPrereq(action.task, cluster.Spawned)
	if err != nil {
		return []controller.Target{}, err
	}

	return []controller.Target{clusterTarget}, nil
}

type downloadResults struct {
	task  *taskState
	stage common.SyncedStr
}

func (action *downloadResults) String() string {
	return fmt.Sprintf("DownloadResults%s for %s", action.stage.Get(), action.task)
}

func (action *downloadResults) Apply() error {
	logger := makeStageLogger(action.task, "download-data")

	if action.task.useStorage {
		action.stage.Set(":detaching storage")

		if err := runRemoteCommand(action.task.client, logger,
			expandExe("~/zyme-postprocess/storage/detach-on-head.sh")); err != nil {
			logger.Errorf("RunTask.downloadResults: cannot detach storage node: %s", err)
			return err
		}
	}

	downloadFolder := "enzyme-download"

	// download desired files
	for _, remoteFilePath := range action.task.downloadFiles {
		_, remoteName := action.task.client.Split(remoteFilePath)

		action.stage.Set(fmt.Sprintf(":downloading %s", remoteName))

		localName := filepath.Join(downloadFolder, remoteName)

		if err := action.task.client.GetFile(remoteFilePath, localName, action.task.overwrite); err != nil {
			logger.Errorf("RunTask.downloadResults: cannot download remote file %s: %s", remoteFilePath, err)
			return err
		}
	}

	action.stage.Reset()
	action.task.client.Close()
	logger.Info("RunTask.downloadResults: closed connection to the cluster")

	return nil
}

func (action *downloadResults) IsExclusive() bool {
	return false
}

func (action *downloadResults) Prerequisites() ([]controller.Target, error) {
	clusterTarget, err := composeClusterPrereq(action.task, cluster.Spawned)
	if err != nil {
		return []controller.Target{}, err
	}

	if !action.task.useStorage {
		return []controller.Target{clusterTarget}, nil
	}

	storageTarget, err := composeStoragePrereq(action.task, storage.Detached)
	if err != nil {
		return []controller.Target{}, err
	}

	return []controller.Target{clusterTarget, storageTarget}, nil
}

type cleanCluster struct {
	task *taskState
}

func (action cleanCluster) String() string {
	return fmt.Sprintf("CleanCluster for %s", action.task)
}

func (action cleanCluster) Apply() error {
	log.Info("RunTask.cleanCluster: empty action, needed for prerequisites only")
	return nil
}

func (action cleanCluster) IsExclusive() bool {
	return false
}

func (action cleanCluster) Prerequisites() ([]controller.Target, error) {
	clusterTarget, err := composeClusterPrereq(action.task, cluster.Configured)
	if err != nil {
		return []controller.Target{}, err
	}

	clusterTarget.MatchExact = true

	return []controller.Target{clusterTarget}, nil
}
