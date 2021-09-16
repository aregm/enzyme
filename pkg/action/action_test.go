package action

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunLoggedCmdDirOutput(t *testing.T) {
	testLogFilePrefix := filepath.Join(os.TempDir(), "TestRunLoggedCmdDirOutput")
	workdir := ""
	name := "go"
	arg := "version"
	var buffer bytes.Buffer

	expectedBytesContent, err := exec.Command(name, arg).Output()
	if err != nil {
		t.Errorf("exec.Command: [%s %s], error: [%s]", name, arg, err)
	}

	expectedContent := string(expectedBytesContent)

	logName, err := RunLoggedCmdDirOutput(testLogFilePrefix, workdir, &buffer, name, arg)
	if err != nil {
		t.Errorf("error occurred while trying to run command: [%s %s], error: [%s]", name, arg, err)
	}

	content := buffer.String()
	expectedLogContent := "enzyme: running command: go version"

	if strings.HasPrefix(content, expectedContent) != true {
		t.Errorf("recorded content: [%s] does not match read content: [%s]", content, expectedContent)
	}

	logContent, err := ioutil.ReadFile(logName)
	if err != nil {
		t.Errorf("error occurred while trying to read from the log file: [%s]", err)
	}

	if strings.HasPrefix(string(logContent), expectedLogContent) != true {
		t.Errorf("recorded content: [%s] does not match read content: [%s]", logContent, expectedLogContent)
	}
}

func TestLoggedCmd(t *testing.T) {
	testLogFilePrefix := filepath.Join(os.TempDir(), "TestRunLoggedCmdDirOutput")
	name := "go"
	arg := "version"
	expectedLogContent := "enzyme: running command: go version"

	logName, err := RunLoggedCmd(testLogFilePrefix, name, arg)
	if err != nil {
		t.Errorf("error occurred while trying to run command: [%s %s], error: [%s]", name, arg, err)
	}

	logContent, err := ioutil.ReadFile(logName)
	if err != nil {
		t.Errorf("error occurred while trying to read from the log file: [%s]", err)
	}

	if strings.HasPrefix(string(logContent), expectedLogContent) != true {
		t.Errorf("recorded content: [%s] does not match read content: [%s]", logContent, expectedLogContent)
	}
}
