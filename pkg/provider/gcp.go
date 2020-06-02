package provider

import (
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"Rhoc/pkg/config"
)

type providerGCP struct {
	baseFunctionality
}

func (provider *providerGCP) Equals(other Provider) bool {
	casted, ok := other.(*providerGCP)
	if !ok {
		return false
	}

	return *provider == *casted
}

func createProviderGCP(region string, zone string, credentialPath string) (Provider, error) {

	credentialAbsPath, err := filepath.Abs(credentialPath)
	if err != nil {
		log.WithFields(log.Fields{
			"credentialPath": credentialPath,
		}).Errorf("createProvider: %s", err)

		return nil, err
	}

	return &providerGCP{
		baseFunctionality{
			providerName:   GCPProviderName,
			region:         region,
			zone:           zone,
			credentialPath: credentialAbsPath,
		},
	}, nil
}

func (provider *providerGCP) GetTFImageResourceName() string {
	return "google_compute_image"
}

func (provider *providerGCP) GetTFStorageResourceName() string {
	return "google_compute_disk"
}

func (provider *providerGCP) CheckUserVars(userVars config.Config) error {
	imageOwners, err := userVars.GetString("owners")
	if err != nil {
		userVars.SetValue("owners", "")
	}

	if imageOwners != "" {
		userVars.SetValue("owners", "")

		log.WithFields(log.Fields{
			"provider":             provider.GetName(),
			"userVars.ImageOwners": imageOwners,
		}).Warnf("Provider.CheckUserVars: GCP provider doesn't contain image owners. It will be ignored.")
	}
	return nil
}

func (provider *providerGCP) MakeCreateImageConfig(imageTemplatePath string, imageVariables config.Config,
	configHash string) (config.Config, error) {
	return provider.baseFunctionality.MakeCreateImageConfig(provider, imageTemplatePath, imageVariables, configHash)
}

func (provider *providerGCP) MakeDestroyImageConfig(imageVariables config.Config) (config.Config, error) {
	configsToSet := make(map[string]interface{})
	configsToSet["provider.google.credentials"] = provider.GetCredentialPath()

	if projectName, err := imageVariables.GetString("project_name"); err == nil && projectName != "" {
		configsToSet["provider.google.project"] = projectName
	} else {
		configsToSet["provider.google.project"] = "zyme-cluster"
	}

	imageName, err := imageVariables.GetString("image_name")
	if err != nil {
		log.WithFields(log.Fields{
			"config": imageVariables,
		}).Errorf("providerGCP.MakeDestroyImageConfig: image_name variable must be defined: %s", err)
		return nil, err
	}

	configsToSet["provider.google.region"] = provider.GetRegion()
	configsToSet["provider.google.version"] = "~> 2.5"

	configsToSet["data.google_compute_image.get_image_id.name"] = imageName

	configsToSet["output.id.value"] = "${data.google_compute_image.get_image_id.self_link}"

	return makeDestroyImageConfigGeneral(configsToSet, templates[provider.GetName()]["destroyImageTemplate"])
}

func (provider *providerGCP) MakeCreateClusterConfig(clusterTemplatePath string,
	clusterVariables config.Config) (config.Config, error) {
	return provider.baseFunctionality.MakeCreateClusterConfig(provider, clusterTemplatePath, clusterVariables)
}

func (provider *providerGCP) MakeStorageNodeConfig(storageTemplatePath string,
	storageVariables config.Config) (config.Config, error) {
	return provider.baseFunctionality.MakeStorageNodeConfig(provider, storageTemplatePath, storageVariables)
}

func (provider *providerGCP) setupProviderSpecificVariables(variablesSection map[string]interface{},
	configHash string, packInDefaultSection bool) error {
	return provider.baseFunctionality.setupProviderSpecificVariables(provider, variablesSection, configHash, packInDefaultSection)
}

func (provider *providerGCP) setupSourcePath(clusterTemplate config.Config) error {
	return provider.baseFunctionality.setupSourcePath(provider, clusterTemplate)
}
