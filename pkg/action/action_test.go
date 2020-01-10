package action

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMakeLogWriter(t *testing.T) {
	testFileName := filepath.Join(os.TempDir(), "TestMakeLogWriter")
	testFileContent := []byte("test makeLogWriter content")

	logName, logFile, err := makeLogWriter(testFileName)
	if err != nil {
		t.Errorf("makeLogWriter function returned error: [%s]", err)
	}

	n, err := logFile.Write(testFileContent)
	if err != nil {
		t.Errorf("while trying to write to the log file occurrated error: [%s]", err)
	}

	if n != len(testFileContent) {
		t.Errorf("wrong entry in file")
	}

	readContent, err := ioutil.ReadFile(logName)

	if err != nil {
		t.Errorf("error occurred while trying to read from the log file: [%s]", err)
	}

	if equal := reflect.DeepEqual(readContent, testFileContent); equal != true {
		t.Errorf("recorded content does not match read content")
	}

	logFile.Close()
	os.Remove(logName)
}
