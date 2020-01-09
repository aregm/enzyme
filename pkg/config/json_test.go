package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
)

const testImageName = "testName"

func init() {
	log.SetOutput(ioutil.Discard) // logs hide
}

var testJSONString = []byte(fmt.Sprintf(`{"variables": {"image_name": "%s"}}`, testImageName))

//function for initialized configs creation
func createInitializedJSONConfig(jsonString []byte) (Config, error) {
	jsonConfig := CreateJSONConfig()
	errDeserializeBytes := jsonConfig.DeserializeBytes(jsonString)

	return jsonConfig, errDeserializeBytes
}

//unit-tests of CreateJSONConfigs and CreateJSONConfigsFromFile functions
func TestCreateJSONConfigs(t *testing.T) {
	jsonConfig := CreateJSONConfig()
	if jsonConfig == nil {
		t.Errorf("JSON config wasn't created; return value: [nil]")
	}

	dirTempFolder, errTempDir := ioutil.TempDir("", "config_unit_tests_temp") //temporary directory creation
	if errTempDir != nil {
		t.Errorf("TempDir function returned error: [%s]", errTempDir)
	}

	pathToFile := filepath.Join(dirTempFolder, "testConfigFile.json")

	errWriteFile := ioutil.WriteFile(pathToFile, testJSONString, 0644)
	if errWriteFile != nil {
		t.Errorf("WriteFile function returned error: [%s]", errWriteFile)
	}

	jsonConfigFile, errCreateConf := CreateJSONConfigFromFile(pathToFile)
	if errCreateConf != nil {
		t.Errorf("CreateJSONConfigFromFile function returned error: [%s]", errCreateConf)
	} else if jsonConfigFile == nil {
		t.Errorf("JSON config wasn't created; return value: [nil]")
	}

	errRemoveAll := os.RemoveAll(dirTempFolder) //removing of temporary files and path after the test execution
	if errRemoveAll != nil {
		t.Errorf("RemoveAll function returned error: [%s]", errRemoveAll)
	}
}

//unit-tests of GetString, GetStringMap and GetValue methods
func TestGetters(t *testing.T) {
	jsonConfig := CreateJSONConfig()

	//empty config access attempt
	outputGetString1, errGetString1 := jsonConfig.GetString("")
	if errGetString1 == nil {
		t.Errorf("for empty key GetString method returned error: [nil]")
	} else if outputGetString1 != "" {
		t.Errorf("for empty key GetString method returned non-empty value: [%s]", outputGetString1)
	}

	outputGetStringMap1, errGetStringMap1 := jsonConfig.GetStringMap("")
	if errGetStringMap1 == nil {
		t.Errorf("for empty key GetStringMap method returned error: [nil]")
	} else if len(outputGetStringMap1) != 0 {
		t.Errorf("for empty key GetStringMap method returned non-empty map container")
	}

	outputGetValue1, errGetValue1 := jsonConfig.GetValue("")
	if errGetValue1 == nil {
		t.Errorf("for empty key GetValue method returned error: [nil]")
	} else if outputGetValue1 != nil {
		t.Errorf("for empty key GetValue method returned non-empty value: [%s]", outputGetValue1)
	}

	//initialized config access attempt
	testJSONConfig, errJSONConfigCreate := createInitializedJSONConfig(testJSONString)
	if errJSONConfigCreate != nil {
		t.Errorf("createInitializedJSONConfig returned error: [%s]", errJSONConfigCreate)
	}

	outputGetString2, errGetString2 := testJSONConfig.GetString("variables.image_name")
	if errGetString2 != nil {
		t.Errorf("for test key GetString method returned error: [%s]", errGetString2)
	} else if outputGetString2 != testImageName {
		t.Errorf("for test key GetString method returned wrong value: [%s]!=[%s]", testImageName, outputGetString2)
	}

	outputGetStringMap2, errGetStringMap2 := testJSONConfig.GetStringMap("variables")
	if errGetStringMap2 != nil {
		t.Errorf("for test key GetStringMap method returned error: [%s]", errGetStringMap2)
	} else if outputGetStringMap2["image_name"] != testImageName {
		t.Errorf("for test key GetStringMap method returned wrong value: [%s]!=[%s]", testImageName,
			outputGetStringMap2["image_name"])
	}

	outputGetValue2, errGetValue2 := testJSONConfig.GetValue("variables.image_name")
	if errGetValue2 != nil {
		t.Errorf("for test key GetValue method returned error: [%s]", errGetValue2)
	} else if outputGetValue2 != testImageName {
		t.Errorf("for test key GetValue method returned wrong value: [%s]!=[%s]", testImageName, outputGetValue2)
	}
}

func TestSetValue(t *testing.T) {
	jsonConfig := CreateJSONConfig()
	jsonConfig.SetValue("variables.image_name", testImageName)

	configKey := jsonConfig.Keys()

	if len(configKey) != 1 {
		t.Errorf("wrong number of config key-value pairs")
	} else if configKey[0] != "variables.image_name" {
		t.Errorf("wrong test key name: [variables.image_name]!=[%s]", configKey[0])
	}

	configValue, _ := jsonConfig.GetValue(configKey[0])

	if configValue != testImageName {
		t.Errorf("wrong test value name: [%s]!=[%s]", testImageName, configValue)
	}
}

//unit-tests of Serialize and Deserialize methods
func TestSerializeDeserialize(t *testing.T) {
	jsonConfigSerialize, errJSONConfigCreate := createInitializedJSONConfig(testJSONString)
	if errJSONConfigCreate != nil {
		t.Errorf("createInitializedJSONConfig function returned error: [%s]", errJSONConfigCreate)
	}

	jsonConfigDeserialize := CreateJSONConfig()

	dirTempFolder, errTempDir := ioutil.TempDir("", "config_unit_tests_temp") //temporary directory creation
	if errTempDir != nil {
		t.Errorf("TempDir function returned error: [%s]", errTempDir)
	}

	pathToFile := filepath.Join(dirTempFolder, "testConfigFile.json")

	errSerialize := jsonConfigSerialize.Serialize(pathToFile)
	if errSerialize != nil {
		t.Errorf("Serialize method returned error: [%s]", errSerialize)
	}

	errDeserialize := jsonConfigDeserialize.Deserialize(pathToFile)
	if errDeserialize != nil {
		t.Errorf("Deserialize method returned error: [%s]", errDeserialize)
	}

	deserializeValue, _ := jsonConfigDeserialize.GetValue("variables.image_name")
	if deserializeValue != testImageName {
		t.Errorf("Deserialize method returned wrong value: [%s]!=[%s]", testImageName, deserializeValue)
	}

	errRemoveAll := os.RemoveAll(dirTempFolder) //removing of temporary files and path after the test execution
	if errRemoveAll != nil {
		t.Errorf("RemoveAll function returned error: [%s]", errRemoveAll)
	}
}

func TestDeserializeBytes(t *testing.T) {
	jsonConfig := CreateJSONConfig()

	if err := jsonConfig.DeserializeBytes(testJSONString); err != nil {
		t.Errorf("DeserializeBytes method returned error: [%s]", err)
	}

	imageName, err := jsonConfig.GetString("variables.image_name")
	if err != nil {
		t.Errorf("GetString method returned error: [%s]", err)
	} else if imageName != testImageName {
		t.Errorf("GetString method returned wrong value: '%s' != [%s]", testImageName, imageName)
	}
}

func TestDeleteKey(t *testing.T) {
	jsonConfig, errJSONConfigCreate := createInitializedJSONConfig(testJSONString)
	if errJSONConfigCreate != nil {
		t.Errorf("createInitializedJSONConfig function returned error: [%s]", errJSONConfigCreate)
	}

	errDeleteKey := jsonConfig.DeleteKey("variables")
	if errDeleteKey != nil {
		t.Errorf("DeleteKey method returned error: [%s]", errDeleteKey)
	}

	configValue, errGetValue := jsonConfig.GetValue("variables")
	if errGetValue == nil || configValue != nil {
		t.Errorf("DeleteKey method error: test key wasn't deleted")
	}
}

func TestCopy(t *testing.T) {
	jsonConfig, errJSONConfigCreate := createInitializedJSONConfig(testJSONString)
	if errJSONConfigCreate != nil {
		t.Errorf("createInitializedJSONConfig function returned error: [%s]", errJSONConfigCreate)
	}

	copyJSONConfig, errCopy := jsonConfig.Copy()
	if errCopy != nil {
		t.Errorf("Copy method returned error: [%s]", errCopy)
	}

	configCopyValue, errGetValue := copyJSONConfig.GetValue("variables.image_name")
	if errGetValue != nil {
		t.Errorf("copied data access error: [%s]", errGetValue)
	} else if configCopyValue != testImageName {
		t.Errorf("test value copied with error: [%s]!=[%s]", testImageName, configCopyValue)
	}
}
