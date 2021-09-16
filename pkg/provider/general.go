package provider

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"enzyme/pkg/config"
)

const imageConfigHashPrefix = `ConfigHash=[`

// getImageConfigHashGeneral parses config hash from image state as presented by Terraform
func getImageConfigHashGeneral(data []byte) (string, error) {
	tfState := string(data)

	start := strings.Index(tfState, string(imageConfigHashPrefix))
	if start == -1 {
		log.WithFields(log.Fields{
			"data": tfState,
		}).Warnf("GetImageConfigHash: cannot find config hash start")

		return "", fmt.Errorf("cannot find config hash start")
	}

	strippedStatus := tfState[start+len(imageConfigHashPrefix):]

	stop := strings.Index(strippedStatus, "]")
	if stop == -1 {
		log.WithFields(log.Fields{
			"data": tfState,
		}).Warnf("GetImageConfigHash: cannot find config hash stop")

		return "", fmt.Errorf("cannot find config hash stop")
	}

	return strippedStatus[:stop], nil
}

// makeCreateImageConfigGeneral returnes provider-specific config object for image creation
func makeCreateImageConfigGeneral(imageTemplatePath string, imageVariables config.Config,
	configHash string, provider Provider) (config.Config, error) {
	imageTemplate, imageVariablesSection, err := GetImageVariables(provider.GetName(), imageTemplatePath)
	if err != nil {
		return nil, err
	}

	packVariables := false
	imageVariablesSection = redefinitionVariablesSection(imageVariablesSection, imageVariables, packVariables)

	if err =
		provider.setupProviderSpecificVariables(imageVariablesSection, configHash, packVariables); err != nil {
		log.WithFields(log.Fields{
			"variablesSection": imageVariablesSection,
			"packVariables":    packVariables,
		}).Errorf("provider-%s.MakeCreateImageConfig: cannot setup prodiver-specific variables: %s",
			provider.GetName(), err)

		return nil, err
	}

	imageTemplate.SetValue(imageVariablesSectionName, imageVariablesSection)

	return imageTemplate, nil
}

// makeDestroyImageConfigGeneral returnes provider-specific config object for image destruction
func makeDestroyImageConfigGeneral(configsToSet map[string]interface{},
	destroyTemplatePath string) (config.Config, error) {
	destroyImageConfig, err := config.CreateJSONConfigFromFile(destroyTemplatePath)
	if err != nil {
		log.WithFields(log.Fields{
			"template-path": destroyTemplatePath,
		}).Errorf("Provider.DestroyImageConfig: cannot parse destruction template: %s", err)

		return nil, err
	}

	for key, value := range configsToSet {
		destroyImageConfig.SetValue(key, value)
	}

	return destroyImageConfig, nil
}

// makeCreateClusterConfigGeneral returnes provider-specific config object for cluster creation
func makeCreateClusterConfigGeneral(clusterTemplatePath string, clusterVariables config.Config,
	removeDefaultLayer bool, provider Provider) (config.Config, error) {
	clusterTemplate, clusterVariablesSection, err :=
		GetClusterVariables(provider.GetName(), clusterTemplatePath, removeDefaultLayer)
	if err != nil {
		return nil, err
	}

	packVariables := !removeDefaultLayer
	clusterVariablesSection =
		redefinitionVariablesSection(clusterVariablesSection, clusterVariables, packVariables)

	if err = provider.setupProviderSpecificVariables(clusterVariablesSection, "", packVariables); err != nil {
		log.WithFields(log.Fields{
			"clusterVariablesSection": clusterVariablesSection,
			"packVariables":           packVariables,
		}).Errorf("provider-%s.MakeCreateClusterConfig: cannot setup provider-specific variables: %s",
			provider.GetName(), err)

		return nil, err
	}

	clusterTemplate.SetValue(clusterVariablesSectionName, clusterVariablesSection)

	if err := provider.setupSourcePath(clusterTemplate); err != nil {
		log.WithField("clusterTemplate", clusterTemplate).Fatalf(
			"provider-%s.MakeCreateClusterConfig: cannot setup source path: %s", provider.GetName(), err)
	}

	return clusterTemplate, nil
}

// redefinitionVariablesSection overrides default values with user-defined values
func redefinitionVariablesSection(variablesSection map[string]interface{},
	variables config.Config,
	pack bool) map[string]interface{} {
	redefinedVariables := make(map[string]interface{})

	for key, oldValue := range variablesSection {
		newValue, err := variables.GetValue(key)
		if err == nil {
			redefinedVariables[key] = packVariable(newValue, pack)
		} else {
			redefinedVariables[key] = oldValue
		}
	}

	return redefinedVariables
}
