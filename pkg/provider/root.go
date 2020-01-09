package provider

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	log "github.com/sirupsen/logrus"

	"Rhoc/pkg/config"
)

const (
	// ImageDescriptor is a image object-specific descriptor
	ImageDescriptor = "image"
	// ClusterDescriptor is a cluster object-specific descriptor
	ClusterDescriptor = "cluster"
	// StorageNodeDescriptor is a storage node object-specific descriptor
	StorageNodeDescriptor = "storage"
	// StorageAttachedDescriptor is a storage node attached to a cluster descriptor
	StorageAttachedDescriptor = "storage-attached"
)

var (
	// RhoctemplatesFolder is variable for storing of Packer and
	// Terraform templates
	RhoctemplatesFolder string

	imageVariablesSectionName   string
	clusterVariablesSectionName string
	storageVariablesSectionName string

	chmodCommand string

	// templates is a map of supported providers with their paths to templates
	templates map[string]map[string]string
)

func init() {
	RhoctemplatesFolder = "templates/"

	imageVariablesSectionName = "variables"
	clusterVariablesSectionName = "variable"
	storageVariablesSectionName = "variable"

	if runtime.GOOS == "windows" {
		chmodCommand = ""
	} else {
		chmodCommand = "chmod 600 \"%v\""
	}

	initProviders()
}

func makeAbsPath(providerName string, templatePath string) string {
	composedPath := filepath.Join(RhoctemplatesFolder, providerName, templatePath)
	result, err := filepath.Abs(composedPath)

	if err != nil {
		log.WithField(templatePath, composedPath).Fatalf("provider.makeAbsPath: cannot make abs path: %s", err)
	}

	return result
}

func initProviders() {
	templates = make(map[string]map[string]string)

	providers := getSupportedProviders()
	for _, providerName := range providers {
		templates[providerName] = make(map[string]string)
		templates[providerName]["destroyImageTemplate"] =
			makeAbsPath(providerName, "destroy_image/destroy_template.tf.json")

		templates[providerName]["imageTemplate"] = makeAbsPath(providerName, "image_template.json")
		templates[providerName]["clusterTemplate"] = makeAbsPath(providerName, "cluster_template.tf.json")

		templates[providerName]["standaloneStorageTemplate"] =
			makeAbsPath(providerName, "storage/standalone_template.tf.json")
		templates[providerName]["attachedStorageTemplate"] =
			makeAbsPath(providerName, "storage/attached_template.tf.json")
	}
}

// Provider is base interface for the provider package
type Provider interface {
	Equals(other Provider) bool

	GetName() string
	GetZone() string
	GetRegion() string
	GetCredentialPath() string
	GetTFImageResourceName() string
	GetTFStorageResourceName() string

	CheckUserVars(userVars config.Config) error

	MakeCreateImageConfig(imageTemplatePath string, imageVariables config.Config, configHash string) (
		config.Config, error)
	MakeDestroyImageConfig(imageVariables config.Config) (config.Config, error)
	MakeCreateClusterConfig(clusterTemplatePath string, clusterVariables config.Config) (config.Config, error)
	MakeStorageNodeConfig(storageTemplatePath string, storageVariables config.Config) (config.Config, error)

	GetImageConfigHash(data []byte) (string, error)
	setupProviderSpecificVariables(variablesSection map[string]interface{}, configHash string,
		packInDefaultSection bool) error
	setupSourcePath(clusterTemplate config.Config) error
}

// MemorizedID creates provider unique ID, that consists of
// provider name, region, zone and checksum of credentials data
func MemorizedID(prov Provider) (string, error) {
	hasher := md5.New()
	credentialsPath := prov.GetCredentialPath()

	credsFile, err := os.Open(credentialsPath)
	if err != nil {
		log.WithFields(log.Fields{
			"credentialsPath": credentialsPath,
		}).Errorf("MemorizedID: cannot open credentials file: %s", err)

		return "", err
	}

	if _, err := io.Copy(hasher, credsFile); err != nil {
		log.WithFields(log.Fields{
			"writer": hasher,
			"reader": credsFile,
		}).Errorf("MemorizedID: cannot read credentials file: %s", err)

		return "", err
	}

	return fmt.Sprintf("%s-%s-%s-%x", prov.GetName(), prov.GetRegion(), prov.GetZone(), hasher.Sum(nil)), nil
}

//CreateProvider - facade for other packages
func CreateProvider(providerName string, region string, zone string,
	credentialsPath string) (Provider, error) {
	// TODO: when making provider check if http/https API is needed and check its availability,
	// so that if http_proxy/https_proxy are needed user gets warned about that early

	log.WithFields(log.Fields{
		"providerName":    providerName,
		"credentialsPath": credentialsPath,
		"region":          region,
		"zone":            zone,
	}).Info("provider creating ...")

	switch providerName {
	case GCPProviderName:
		return createProviderGCP(region, zone, credentialsPath)
	case AWSProviderName:
		return createProviderAWS(region, zone, credentialsPath)
	default:
		errorMessage := "provider not implemented"

		log.WithFields(log.Fields{
			"providerName": providerName,
		}).Errorf("CreateProvider: %s", errorMessage)

		return nil, fmt.Errorf(errorMessage)
	}
}

func packVariable(value interface{}, packInDefaultSection bool) interface{} {
	if packInDefaultSection {
		return map[string]interface{}{"default": value}
	}

	return value
}

func getSectionFromTemplate(providerName, templateName,
	templateType string) (config.Config, map[string]interface{}, error) {
	if templateName == "" {
		var err error

		if templateName, err = GetDefaultTemplate(providerName, templateType); err != nil {
			return nil, nil, err
		}
	}

	template, err := config.CreateJSONConfigFromFile(templateName)
	if err != nil {
		log.WithFields(log.Fields{
			"templateName": templateName,
		}).Errorf("getSectionFromTemplate: cannot create config from file: %s", err)

		return nil, nil, err
	}

	var sectionName string

	switch templateType {
	case ImageDescriptor:
		sectionName = imageVariablesSectionName
	case ClusterDescriptor:
		sectionName = clusterVariablesSectionName
	case StorageNodeDescriptor:
		sectionName = storageVariablesSectionName
	case StorageAttachedDescriptor:
		sectionName = storageVariablesSectionName
	default:
		log.WithFields(log.Fields{
			"templateType": templateType,
		}).Fatal("getSectionFromTemplate: unexpected template type")
	}

	variablesSection, err := template.GetStringMap(sectionName)
	if err != nil {
		log.WithFields(log.Fields{
			"template-type": templateType,
		}).Errorf("getSectionFromTemplate: cannot get string map: %s", err)

		return nil, nil, err
	}

	return template, variablesSection, nil
}

func removeDefaultLayerFromClusterSection(clusterSection map[string]interface{}) (
	map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range clusterSection {
		casted, ok := value.(map[string]interface{})
		if !ok {
			errorMessage := "cannot cast value to map[string]interface{}"
			log.WithFields(log.Fields{
				"value": value,
			}).Errorf("removeDefaultLayerFromClusterSection: %s", errorMessage)

			return nil, fmt.Errorf(errorMessage)
		}

		defaultValue, ok := casted["default"]
		if !ok {
			errorMessage := "casted value didn't have 'default' layer"
			log.WithFields(log.Fields{
				"casted_value": casted,
			}).Errorf("removeDefaultLayerFromClusterSection: %s", errorMessage)

			return nil, fmt.Errorf(errorMessage)
		}

		result[key] = defaultValue
	}

	return result, nil
}

// GetImageVariables returns variables section from image template
func GetImageVariables(providerName, templatePath string) (config.Config, map[string]interface{}, error) {
	return getSectionFromTemplate(providerName, templatePath, ImageDescriptor)
}

// GetClusterVariables returns variables section from cluster template;
// if removeDefaultLayer is false clusterSection with variables returns without modification,
// i.e. variable.[key].default.[value]
// if removeDefaultLayer is true clusterSection returns in next format: variable.[key].[default]
func GetClusterVariables(providerName, templatePath string,
	removeDefaultLayer bool) (config.Config, map[string]interface{}, error) {
	clusterTemplate, clusterSection, err := getSectionFromTemplate(providerName, templatePath, ClusterDescriptor)
	if err != nil {
		return nil, nil, err
	}

	if removeDefaultLayer {
		if clusterSection, err = removeDefaultLayerFromClusterSection(clusterSection); err != nil {
			return nil, nil, err
		}
	}

	return clusterTemplate, clusterSection, err
}

// GetStorageVariables returns variables section from storage template;
// if removeDefaultLayer is false storageSection with variables is returned without modification,
// i.e. variable.[key].default.[value]
// if removeDefaultLayer is true storageSection is returned in the following format: variable.[key].[default]
func GetStorageVariables(providerName, templatePath string,
	removeDefaultLayer bool) (config.Config, map[string]interface{}, error) {
	storageTemplate, storageSection, err :=
		getSectionFromTemplate(providerName, templatePath, StorageNodeDescriptor)
	if err != nil {
		return nil, nil, err
	}

	if removeDefaultLayer {
		if storageSection, err = removeDefaultLayerFromClusterSection(storageSection); err != nil {
			return nil, nil, err
		}
	}

	return storageTemplate, storageSection, nil
}

// GetTaskVariables returns combine variables from imageSection, clusterSection and storageSection
// (with removing default layer)
func GetTaskVariables(providerName string) (map[string]interface{}, error) {
	_, imageVariables, err := GetImageVariables(providerName, "")
	if err != nil {
		return nil, err
	}

	_, clusterVariables, err := GetClusterVariables(providerName, "", true)
	if err != nil {
		return nil, err
	}

	_, storageVariables, err := GetStorageVariables(providerName, "", true)
	if err != nil {
		return nil, err
	}

	// Merge all maps
	allVariables := imageVariables
	doMerge := func(input map[string]interface{}, name string) {
		for key, value := range input {
			if oldValue, ok := allVariables[key]; ok {
				if oldValue != value {
					log.WithFields(log.Fields{
						"key":       key,
						"old-value": oldValue,
						"new-value": value,
					}).Warnf("GetTaskVariables: overwriting already set value by value from %s", name)
				}
			}

			allVariables[key] = value
		}
	}

	doMerge(clusterVariables, "cluster")
	doMerge(storageVariables, "storage")

	return allVariables, nil
}

// GetDefaultTemplate returns provider specific Packer or Terraform template file
func GetDefaultTemplate(providerName string, templateType string) (string, error) {
	if !IsProviderSupported(providerName) {
		errorMessage := "provider not implemented"

		log.WithFields(log.Fields{
			"providerName": providerName,
		}).Errorf("GetDefaultTemplate: %s", errorMessage)

		return "", fmt.Errorf(errorMessage)
	}

	var template string

	switch templateType {
	case ImageDescriptor:
		template = templates[providerName]["imageTemplate"]
	case ClusterDescriptor:
		template = templates[providerName]["clusterTemplate"]
	case StorageNodeDescriptor:
		template = templates[providerName]["standaloneStorageTemplate"]
	case StorageAttachedDescriptor:
		template = templates[providerName]["attachedStorageTemplate"]
	}

	return template, nil
}

// GetSupportedProviders returnes names of all supported providers
func getSupportedProviders() []string {
	providers := make([]string, 2)
	providers[0] = GCPProviderName
	providers[1] = AWSProviderName
	return providers
}

// IsProviderSupported checks that provider is supported and returns result
func IsProviderSupported(providerName string) bool {
	if _, ok := templates[providerName]; ok {
		return true
	}
	return false
}

// RootFolder returns root folder of Rhoc project
func RootFolder() (string, error) {
	exProgram, err := os.Executable()
	if err != nil {
		log.Errorf("RootFolder: %s", err)
		return "", err
	}

	return filepath.Dir(exProgram), nil
}
