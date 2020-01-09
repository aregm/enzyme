package runtask

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"Rhoc/pkg/config"
	"Rhoc/pkg/controller"
	"Rhoc/pkg/provider"
	"Rhoc/pkg/ssh"
	"Rhoc/pkg/state"
)

// Status describes status of the "run" task
type Status int

const (
	// NotRunning is the very first status of the task when nothing was done yet
	NotRunning Status = iota
	// Connected is when SSH connection to frontend node is established
	Connected
	// DataUploaded is when data needed for the task was uploaded
	DataUploaded
	// CommandFinished is when the command finished running on the cluster
	CommandFinished
	// ResultsDownloaded is when results of the command were downloaded locally
	ResultsDownloaded
	// ClusterCleaned is when cluster used to run the task was shut down
	ClusterCleaned
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
		NotRunning:        "not running",
		Connected:         "connection established",
		DataUploaded:      "data uploaded",
		CommandFinished:   "command finished",
		ResultsDownloaded: "results downloaded",
		ClusterCleaned:    "cluster cleaned",
	}

	transitions = map[Status][]controller.Status{
		NotRunning:        {},
		Connected:         {NotRunning},
		DataUploaded:      {Connected},
		CommandFinished:   {DataUploaded},
		ResultsDownloaded: {CommandFinished},
		ClusterCleaned:    {ResultsDownloaded},
	}
)

func (s Status) String() string {
	result, ok := statusToString[s]
	if !ok {
		return "unknown"
	}

	return result
}

type taskState struct {
	status         Status
	provider       provider.Provider
	userVariables  config.Config
	localPath      string
	remotePath     string
	args           []string
	overwrite      bool
	convertNewline bool
	useStorage     bool

	client ssh.ZymeClient

	fetcher           state.Fetcher
	serviceParameters config.ServiceParams
	uploadFiles       []string
	downloadFiles     []string
}

func (task *taskState) String() string {
	return fmt.Sprintf("RunTask(local=%s, remote=%s, args=%s, status=%s)", task.localPath, task.remotePath,
		task.args, task.status)
}

func (task *taskState) Status() controller.Status {
	return task.status
}

func (task *taskState) SetStatus(status controller.Status) error {
	casted, ok := status.(Status)
	if !ok {
		return fmt.Errorf("cannot set status of task - wrong type")
	}

	task.status = casted

	log.WithFields(log.Fields{
		"task":       *task,
		"new-status": casted,
	}).Info("status set")

	return nil
}

func (task *taskState) GetTransitions(to controller.Status) ([]controller.Status, error) {
	casted, ok := to.(Status)
	if !ok {
		return nil, fmt.Errorf("cannot get transitions to status %v - not a RunTask status", to)
	}

	result, ok := transitions[casted]
	if !ok {
		return nil, fmt.Errorf("unexpected RunTask status %v", to)
	}

	return result, nil
}

func (task *taskState) Equals(other controller.Thing) bool {
	casted, ok := other.(*taskState)
	if !ok {
		return false
	}

	if len(task.args) != len(casted.args) {
		return false
	}

	for i, arg := range task.args {
		if arg != casted.args[i] {
			return false
		}
	}

	if task.client == nil {
		if casted.client != nil {
			return false
		}
	} else if !task.client.Equals(casted.client) {
		return false
	}

	return task.status.Equals(task.status) &&
		task.provider.Equals(casted.provider) &&
		task.localPath == casted.localPath &&
		task.remotePath == casted.remotePath &&
		task.overwrite == casted.overwrite
}

func (task *taskState) GetAction(current controller.Status, target controller.Status) (
	controller.Action, error) {
	currentStatus, ok := current.(Status)
	if !ok {
		return nil, fmt.Errorf("current status %v is not RunTask status", current)
	}

	targetStatus, ok := target.(Status)
	if !ok {
		return nil, fmt.Errorf("target status %v is not RunTask status", target)
	}

	switch currentStatus {
	case NotRunning:
		if targetStatus == Connected {
			return &makeConnection{task: task}, nil
		}
	case Connected:
		if targetStatus == DataUploaded {
			return &uploadData{task: task}, nil
		}
	case DataUploaded:
		if targetStatus == CommandFinished {
			return &runRemote{task: task}, nil
		}
	case CommandFinished:
		if targetStatus == ResultsDownloaded {
			return &downloadResults{task: task}, nil
		}
	case ResultsDownloaded:
		if targetStatus == ClusterCleaned {
			return &cleanCluster{task: task}, nil
		}
	}

	return nil, fmt.Errorf("unsupported transition of (%v => %v)", currentStatus, targetStatus)
}

// CreateTaskTarget is function, that returns a structure describing the task that
// the controller package (planner and executor) can use
func CreateTaskTarget(prov provider.Provider, userVariables config.Config, serviceParams config.ServiceParams,
	fetcher state.Fetcher, localPath, remotePath string, convertNewline, overwrite, useStorage bool,
	args []string, uploadFiles []string, downloadFiles []string) (controller.Thing, error) {
	return &taskState{
		status:            NotRunning,
		provider:          prov,
		userVariables:     userVariables,
		localPath:         localPath,
		remotePath:        remotePath,
		args:              args,
		overwrite:         overwrite,
		convertNewline:    convertNewline,
		useStorage:        useStorage,
		client:            nil,
		fetcher:           fetcher,
		serviceParameters: serviceParams,
		uploadFiles:       uploadFiles,
		downloadFiles:     downloadFiles,
	}, nil
}
