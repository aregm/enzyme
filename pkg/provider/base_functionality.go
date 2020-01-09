package provider

import (
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"Rhoc/pkg/config"
)

// String provider names
const (
	GCPProviderName = "gcp"
	AWSProviderName = "aws"
)

type baseFunctionality struct {
	providerName   string
	region         string
	zone           string
	credentialPath string
}

func (baseFunctionality *baseFunctionality) Equals(other Provider) bool {
	return baseFunctionality.GetName() == other.GetName() &&
		baseFunctionality.GetRegion() == other.GetRegion() &&
		baseFunctionality.GetZone() == other.GetZone()
}

func (baseFunctionality *baseFunctionality) GetName() string {
	return baseFunctionality.providerName
}

func (baseFunctionality *baseFunctionality) GetZone() string {
	return baseFunctionality.zone
}

func (baseFunctionality *baseFunctionality) GetRegion() string {
	return baseFunctionality.region
}

func (baseFunctionality *baseFunctionality) GetCredentialPath() string {
	return baseFunctionality.credentialPath
}

func (baseFunctionality *baseFunctionality) GetImageConfigHash(data []byte) (string, error) {
	return getImageConfigHashGeneral(data)
}

func (baseFunctionality *baseFunctionality) MakeCreateImageConfig(provider Provider, imageTemplatePath string, imageVariables config.Config,
	configHash string) (config.Config, error) {
	return makeCreateImageConfigGeneral(imageTemplatePath, imageVariables, configHash, provider)
}

func (baseFunctionality *baseFunctionality) MakeCreateClusterConfig(provider Provider, clusterTemplatePath string,
	clusterVariables config.Config) (config.Config, error) {
	removeDefaultLayer := false
	return makeCreateClusterConfigGeneral(clusterTemplatePath, clusterVariables, removeDefaultLayer, provider)
}

func (baseFunctionality *baseFunctionality) MakeStorageNodeConfig(provider Provider, storageTemplatePath string,
	storageVariables config.Config) (config.Config, error) {
	removeDefaultLayer := false

	storageTemplate, storageVariablesSection, err :=
		GetStorageVariables(provider.GetName(), storageTemplatePath, removeDefaultLayer)
	if err != nil {
		return nil, err
	}

	packVariables := !removeDefaultLayer
	storageVariablesSection =
		redefinitionVariablesSection(storageVariablesSection, storageVariables, packVariables)

	if err = provider.setupProviderSpecificVariables(storageVariablesSection, "", packVariables); err != nil {
		log.WithFields(log.Fields{
			"storageVariablesSection": storageVariablesSection,
			"packVariables":           packVariables,
		}).Errorf("provider.MakeStorageNodeConfig: cannot setup provider-specific variables: %s", err)

		return nil, err
	}

	storageTemplate.SetValue(storageVariablesSectionName, storageVariablesSection)

	return storageTemplate, nil
}

func (baseFunctionality *baseFunctionality) setupProviderSpecificVariables(provider Provider, variablesSection map[string]interface{},
	configHash string, packInDefaultSection bool) error {
	//TODO implementation without modification input variable

	variablesSection["credential_path"] = packVariable(provider.GetCredentialPath(), packInDefaultSection)
	variablesSection["region"] = packVariable(provider.GetRegion(), packInDefaultSection)
	variablesSection["zone"] = packVariable(provider.GetZone(), packInDefaultSection)

	if configHash != "" {
		variablesSection["configuration_hash"] = packVariable(configHash, packInDefaultSection)
	}

	rootFolder, err := RootFolder()
	if err != nil {
		log.Errorf("provider.setupProviderSpecificVariables: cannot get root folder: %s", err)
		return err
	}

	variablesSection["root_folder"] = packVariable(rootFolder, packInDefaultSection)
	variablesSection["chmod_command"] = packVariable(chmodCommand, packInDefaultSection)

	return nil
}

func (baseFunctionality *baseFunctionality) setupSourcePath(provider Provider, clusterTemplate config.Config) error {
	rootFolder, err := RootFolder()
	if err != nil {
		log.Errorf("provider-%s.setupSourcePath: cannot get root folder: %s", provider.GetName(), err)
		return err
	}

	providerSource := filepath.Join(rootFolder, "templates", provider.GetName(), "cluster_source")
	provisionSource := filepath.Join(rootFolder, "templates/cluster_provision")

	clusterTemplate.SetValue("module."+provider.GetName()+"_provider.source", providerSource)
	clusterTemplate.SetValue("module.provision.source", provisionSource)

	return nil
}
