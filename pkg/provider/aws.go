package provider

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"Rhoc/pkg/config"
)

// AWS provider is connected only through default ec2-user user
const (
	AWSUserName = "ec2-user"
)

type providerAWS struct {
	baseFunctionality
}

func createProviderAWS(region string, zone string, credentialPath string) (Provider, error) {

	credentialAbsPath, err := filepath.Abs(credentialPath)
	if err != nil {
		log.WithFields(log.Fields{
			"credentialPath": credentialPath,
		}).Errorf("createProvider: %s", err)

		return nil, err
	}

	// Packer Amazon EBS builder perform lookup in home location or via the env variable
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", credentialAbsPath)

	return &providerAWS{
		baseFunctionality{
			providerName:   AWSProviderName,
			region:         region,
			zone:           zone,
			credentialPath: credentialAbsPath,
		},
	}, nil
}

func (provider *providerAWS) GetTFImageResourceName() string {
	return "aws_ami"
}

func (provider *providerAWS) GetTFStorageResourceName() string {
	return "aws_volume_attachment"
}

func (provider *providerAWS) CheckUserVars(userVars config.Config) error {
	var err error

	userName, err := userVars.GetString("user_name")
	if err != nil {
		userVars.SetValue("user_name", AWSUserName)
	}

	if userName != AWSUserName {
		userVars.SetValue("user_name", AWSUserName)

		log.WithFields(log.Fields{
			"provider":          provider.GetName(),
			"userVars.UserName": userName,
		}).Warnf("Provider.CheckUserVars: cannot support custom user_name %s for AWS provider: only %s is supported. Replaced for correct.",
			userName, AWSUserName)
	}

	projectName, err := userVars.GetString("project_name")
	if err != nil {
		userVars.SetValue("project_name", "")
	}

	if projectName != "" {
		userVars.SetValue("project_name", "")

		log.WithFields(log.Fields{
			"provider":             provider.GetName(),
			"userVars.ProjectName": projectName,
		}).Warnf("Provider.CheckUserVars: AWS provider doesn't contain project name. It will be ignored.")
	}

	imageOwners, err := userVars.GetString("owners")
	if err != nil {
		userVars.SetValue("owners", "self")
	}

	if imageOwners == "" {
		userVars.SetValue("owners", "self")
		log.WithFields(log.Fields{
			"provider":             provider.GetName(),
			"userVars.ImageOwners": imageOwners,
		}).Warnf("Provider.CheckUserVars: Required variable owners for AWS provider isn't defined by user. Set self by default.")
	}

	return nil
}

func (provider *providerAWS) MakeCreateImageConfig(imageTemplatePath string, imageVariables config.Config,
	configHash string) (config.Config, error) {
	return provider.baseFunctionality.MakeCreateImageConfig(provider, imageTemplatePath, imageVariables, configHash)
}

func (provider *providerAWS) MakeDestroyImageConfig(imageVariables config.Config) (config.Config, error) {
	configsToSet := make(map[string]interface{})

	configsToSet["provider.aws.shared_credentials_file"] = provider.GetCredentialPath()
	configsToSet["provider.aws.region"] = provider.GetRegion()
	configsToSet["provider.aws.version"] = "~> 2.1"

	imageName, err := imageVariables.GetString("image_name")
	if err != nil {
		panic("image_name not found among other image variables")
	}

	configsToSet["data.aws_ami.get_image_id.owners"] = []string{"self"}
	configsToSet["data.aws_ami.get_image_id.filter.name"] = "name"
	configsToSet["data.aws_ami.get_image_id.filter.values"] = []string{imageName}

	configsToSet["output.id.value"] = "${data.aws_ami.get_image_id.id}"

	return makeDestroyImageConfigGeneral(configsToSet, templates[provider.GetName()]["destroyImageTemplate"])
}

func (provider *providerAWS) MakeCreateClusterConfig(clusterTemplatePath string,
	clusterVariables config.Config) (config.Config, error) {
	return provider.baseFunctionality.MakeCreateClusterConfig(provider, clusterTemplatePath, clusterVariables)
}

func (provider *providerAWS) MakeStorageNodeConfig(storageTemplatePath string,
	storageVariables config.Config) (config.Config, error) {
	return provider.baseFunctionality.MakeStorageNodeConfig(provider, storageTemplatePath, storageVariables)
}

func (provider *providerAWS) setupProviderSpecificVariables(variablesSection map[string]interface{},
	configHash string, packInDefaultSection bool) error {
	return provider.baseFunctionality.setupProviderSpecificVariables(provider, variablesSection, configHash, packInDefaultSection)
}

func (provider *providerAWS) setupSourcePath(clusterTemplate config.Config) error {
	return provider.baseFunctionality.setupSourcePath(provider, clusterTemplate)
}
