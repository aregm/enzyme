package provider

import (
	"bytes"
	"fmt"

	log "github.com/sirupsen/logrus"

	action_pkg "Rhoc/pkg/action"
	"Rhoc/pkg/config"
)

// MissingKey is an error returned by ExtractOutputValues when a value cannot be found in output
type MissingKey struct {
	Key string
}

func (err MissingKey) Error() string {
	return fmt.Sprintf("no such key: %s", err.Key)
}

// ExtractOutputValues extracts values by names making sure each value exists
func ExtractOutputValues(json config.Config, names ...string) ([]string, error) {
	result := []string{}

	for _, name := range names {
		if !json.IsSet(name) {
			return result, MissingKey{
				Key: name,
			}
		}

		value, err := json.GetString(name)
		if err != nil {
			log.WithField("variable", name).Infof("ExtractOutputValues: cannot read variable: %s", err)
			return result, err
		}

		result = append(result, value)
	}

	return result, nil
}

// ParseTerraformOutputs calles "terraform output" and returns its output as parsed config
func ParseTerraformOutputs(workDir, logPrefix string, logger *log.Entry) (config.Config, error) {
	logger = logger.WithField("dir", workDir)

	var buffer bytes.Buffer

	if _, err := action_pkg.RunLoggedCmdDirOutput(logPrefix, workDir, &buffer, Terraform(),
		"output", "-no-color", "-json"); err != nil {
		logger.Errorf("ParseTerraformOutputs: cannot read terraform output: %s", err)
		return nil, err
	}

	json := config.CreateJSONConfig()

	if err := json.DeserializeBytes(buffer.Bytes()); err != nil {
		logger.WithField("buffer", buffer.String()[:30]).Errorf(
			"ParseTerraformOutputs: cannot parse terraform output: %s", err)
		return nil, err
	}

	return json, nil
}
