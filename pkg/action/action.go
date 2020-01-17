package action

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"

	"Rhoc/pkg/logging"
)

// RunLoggedCmd executes the program <name> in current working directory
// with arguments <args> while redirecting its output to the logger
func RunLoggedCmd(logfilePrefix string, name string, args ...string) (string, error) {
	return RunLoggedCmdDir(logfilePrefix, "", name, args...)
}

// RunLoggedCmdDir executes the program <name> in working directory <workDir>
// with arguments <args> while redirecting its output to the logger
func RunLoggedCmdDir(logfilePrefix string, workDir string, name string, args ...string) (string, error) {
	return RunLoggedCmdDirOutput(logfilePrefix, workDir, nil, name, args...)
}

// RunLoggedCmdDirOutput executes the command and reads its stdout into <output>
func RunLoggedCmdDirOutput(logfilePrefix string, workDir string, output io.Writer, name string,
	args ...string) (string, error) {
	var cmdLog io.Writer

	logname, cmdLogfile, err := logging.MakeLogWriter(logfilePrefix)
	if err != nil {
		log.Warnf("RunLoggedCmdDirOutput: cannot redirect log to file: %s, using default log instead", err)

		cmdLog = logging.GetLogWriter()
	} else {
		logStr := fmt.Sprintf("Rhoc: running command: %s %s\n-----\n\n", name, strings.Join(args, " "))
		if n, err := io.WriteString(cmdLogfile, logStr); err != nil {
			log.WithField("logFile", cmdLogfile).Fatalf(
				"RunLoggedCmdDirOutput: only %d bytes from %d were written into logFile: %s", n, len(logStr),
				err)
		}
		defer cmdLogfile.Close()
		cmdLog = cmdLogfile
	}

	var cmdOut, cmdErr io.Writer

	switch {
	case output != nil:
		cmdOut = io.MultiWriter(output, cmdLog)
	case logging.GetRepeatLogs():
		cmdOut = io.MultiWriter(os.Stdout, cmdLog)
	default:
		cmdOut = cmdLog
	}

	if logging.GetRepeatLogs() {
		cmdErr = io.MultiWriter(os.Stderr, cmdLog)
	} else {
		cmdErr = cmdLog
	}

	return logname, runLoggedCmdDirRedirect(logname, workDir, cmdOut, cmdErr, name, args...)
}

func runLoggedCmdDirRedirect(logname string, workDir string, cmdOut io.Writer, cmdErr io.Writer, name string,
	args ...string) error {
	command := exec.Command(name, args...)
	command.Stdout = cmdOut
	command.Stderr = cmdErr

	if workDir != "" {
		command.Dir = workDir
	}

	log.WithFields(log.Fields{
		"name": name,
		"dir":  workDir,
		"args": args,
		"log":  logname,
	}).Info("running command")

	return command.Run()
}
