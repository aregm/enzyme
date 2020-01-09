package storage

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateDirForFile(t *testing.T) {
	dirTempFolder, errTempDir := ioutil.TempDir("", "config_unit_tests_temp") //temporary directory creation
	if errTempDir != nil {
		t.Errorf("TempDir function returned error: [%s]", errTempDir)
	}

	dirNestedFolder := filepath.Join(dirTempFolder, "nestedFolder")
	pathToFile := filepath.Join(dirNestedFolder, "testConfigFile.json")

	errCreateDirByFileName := CreateDirForFile(pathToFile)
	if errCreateDirByFileName != nil {
		t.Errorf("CreateDirForFile function returned error: [%s]", errCreateDirByFileName)
	}

	_, errStatNestedFolder := os.Stat(dirNestedFolder)
	if errStatNestedFolder != nil {
		t.Errorf("wrong behavior of CreateDirForFile function: directory wasn't created")
	}

	_, errStatFile := os.Stat(pathToFile)
	if errStatFile == nil {
		t.Errorf("wrong behavior of CreateDirForFile function: file was created")
	}

	errRemoveAll := os.RemoveAll(dirTempFolder) //removing of temporary files and path after the test execution
	if errRemoveAll != nil {
		t.Errorf("RemoveAll function returned error: [%s]", errRemoveAll)
	}
}
