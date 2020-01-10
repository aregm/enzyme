package action

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMakeLogWriter(t *testing.T) {
	testFileNamePrefix := filepath.Join(os.TempDir(), "TestMakeLogWriter")
	testFileContent := []byte("test makeLogWriter content")

	logName, logFile, err := makeLogWriter(testFileNamePrefix)
	if err != nil {
		t.Errorf("makeLogWriter function returned error: [%s]", err)
	}
	defer logFile.Close()

	n, err := logFile.Write(testFileContent)
	if err != nil {
		t.Errorf("error occurred while trying to write to the log file: [%s]", err)
	}

	if n != len(testFileContent) {
		t.Errorf("unexpected written bytes count: %d, expected: %d", n, len(testFileContent))
	}

	readContent, err := ioutil.ReadFile(logName)

	if err != nil {
		t.Errorf("error occurred while trying to read from the log file: [%s]", err)
	}

	if equal := reflect.DeepEqual(readContent, testFileContent); equal != true {
		t.Errorf("recorded content does not match read content")
	}

	// still need to close explicitly instead of relying on defer so os.Remove succeeds
	logFile.Close()

	if err = os.Remove(logName); err != nil {
		t.Errorf("error deleting file; file: [%s], error: [%s]", logName, err)
	}
}
