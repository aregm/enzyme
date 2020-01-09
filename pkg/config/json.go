package config

import (
	"encoding/json"
	"errors"

	"github.com/intel-go/viper"
	log "github.com/sirupsen/logrus"

	"Rhoc/pkg/storage"
)

type jsonConfig struct {
	data *viper.Viper
}

func newViperConfig() *viper.Viper {
	vpr := viper.New()
	vpr.SetKeysCaseSensitive(true)

	return vpr
}

// CreateJSONConfig is function for creation of empty JSON config object
func CreateJSONConfig() Config {
	vpr := newViperConfig()
	vpr.SetConfigType("json")

	return &jsonConfig{data: vpr}
}

// CreateJSONConfigFromFile is function for creation of JSON config object from JSON file
func CreateJSONConfigFromFile(fileName string) (Config, error) {
	jsonConf := CreateJSONConfig()

	if err := jsonConf.Deserialize(fileName); err != nil {
		log.WithFields(log.Fields{
			"filename": fileName,
		}).Errorf("CreateJSONConfigFromFile: %s", err)

		return nil, err
	}

	return jsonConf, nil
}

func (img_conf *jsonConfig) Keys() []string {
	return img_conf.data.AllKeys()
}

func (img_conf *jsonConfig) IsSet(key string) bool {
	return img_conf.data.IsSet(key)
}

func (img_conf *jsonConfig) GetString(key string) (string, error) {
	var err error

	errorMessage := "cast to string failed"

	value, err := img_conf.GetValue(key)
	if err != nil {
		return "", err
	}

	stringValue, ok := value.(string)
	if !ok {
		log.WithFields(log.Fields{
			"key": key,
		}).Errorf("jsonConfig.GetString: %s", errorMessage)

		err = errors.New(errorMessage)
	}

	return stringValue, err
}

func (img_conf *jsonConfig) GetStringMap(key string) (map[string]interface{}, error) {
	var err error

	errorMessage := "cast to map[string]interface{} failed"

	value, err := img_conf.GetValue(key)
	if err != nil {
		return nil, err
	}

	mapStringValue, ok := value.(map[string]interface{})
	if !ok {
		log.WithFields(log.Fields{
			"key": key,
		}).Errorf("jsonConfig.GetStringMap: %s", errorMessage)

		err = errors.New(errorMessage)
	}

	return mapStringValue, err
}

func (img_conf *jsonConfig) GetValue(key string) (interface{}, error) {
	var err error

	errorMessage := "key not found"

	value := img_conf.data.Get(key)
	if value == nil {
		log.WithFields(log.Fields{
			"key": key,
		}).Tracef("jsonConfig.GetValue failed: %s", errorMessage)

		err = errors.New(errorMessage)
	}

	return value, err
}

func (img_conf *jsonConfig) SetValue(key string, value interface{}) {
	img_conf.data.Set(key, value)
}

func (img_conf *jsonConfig) Serialize(configName string) error {
	if err := storage.CreateDirForFile(configName); err != nil {
		log.WithFields(log.Fields{
			"configName": configName,
		}).Errorf("jsonConfg.Serialize: %s", err)

		return err
	}

	log.WithFields(log.Fields{
		"configName": configName,
	}).Info("jsonConfig.Serialization ...")

	// TODO force option
	return img_conf.data.WriteConfigAs(configName)
}

func (img_conf *jsonConfig) Deserialize(configName string) error {
	log.WithFields(log.Fields{
		"configName": configName,
	}).Info("jsonConfig.Deserialization ...")

	img_conf.data.SetConfigFile(configName)

	return img_conf.data.ReadInConfig()
}

func (img_conf *jsonConfig) DeserializeBytes(jsonContent []byte) error {
	var jsonMap map[string]interface{}

	err := json.Unmarshal(jsonContent, &jsonMap)
	if err != nil {
		return err
	}

	for key, value := range jsonMap {
		img_conf.SetValue(key, value)
	}

	return nil
}

func (img_conf *jsonConfig) DeleteKey(key string) error {
	//Workaround
	settings := img_conf.data.AllSettings()
	delete(settings, key)

	viperWithoutKey := newViperConfig()

	if err := viperWithoutKey.MergeConfigMap(settings); err != nil {
		log.WithFields(log.Fields{
			"settings":      settings,
			"viperInstance": viperWithoutKey,
		}).Errorf("jsonConfg.DeleteKey: %s", err)

		return err
	}

	img_conf.data = viperWithoutKey

	return nil
}

func (img_conf *jsonConfig) Copy() (Config, error) {
	copyData := newViperConfig()
	copyData.SetKeysCaseSensitive(true)

	settings := img_conf.data.AllSettings()

	if err := copyData.MergeConfigMap(settings); err != nil {
		log.WithFields(log.Fields{
			"settings":      settings,
			"viperInstance": copyData,
		}).Errorf("jsonConfg.Copy: %s", err)

		return nil, err
	}

	return &jsonConfig{data: copyData}, nil
}

func (img_conf *jsonConfig) Print() {
	img_conf.data.Debug()
}
