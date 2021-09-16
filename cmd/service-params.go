package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"enzyme/pkg/config"
	"enzyme/pkg/provider"
)

var (
	socksHost string = ""
	socksPort int    = 1080

	zone            string
	region          string
	providerName    string
	credentialsFile string

	parametersFile string

	vars map[string]string
)

func createArgs() (config.Config, provider.Provider, config.ServiceParams, error) {
	checkFileExists(credentialsFile)

	prov, err := provider.CreateProvider(providerName, region, zone, credentialsFile)
	if err != nil {
		log.WithFields(log.Fields{
			"name":        providerName,
			"region":      region,
			"credentials": credentialsFile,
		}).Errorf("cannot create provider: %s", err)

		return nil, nil, config.ServiceParams{}, err
	}

	log.WithFields(log.Fields{
		"name":        providerName,
		"region":      region,
		"credentials": credentialsFile,
	}).Infof("created provider: %s", prov)

	var userVariables config.Config
	if parametersFile != "" {
		userVariables, err = config.CreateJSONConfigFromFile(parametersFile)
		if err != nil {
			log.WithFields(log.Fields{
				"parameters": parametersFile,
			}).Errorf("cannot create user variables: %s", err)

			return nil, nil, config.ServiceParams{}, err
		}

		log.WithFields(log.Fields{
			"parameters": parametersFile,
		}).Infof("created user variables: %s", userVariables)
	} else {
		userVariables = config.CreateJSONConfig()
		log.Infof("created default user variables: %s", userVariables)
	}

	// Extend config by map's value
	// May be redefinition of values from parametersFile
	for key, value := range vars {
		userVariables.SetValue(key, value)
	}

	return userVariables, prov, config.ServiceParams{
		SocksProxyHost: socksHost,
		SocksProxyPort: socksPort,
	}, nil
}

func addServiceParams(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&zone, "zone", "z", "a", "public CSP zone")

	cmd.Flags().StringVarP(&region, "region", "r", "us-central1", "public CSP region")

	cmd.Flags().StringVarP(&providerName, "provider", "p", "gcp", "public CSP: {gcp}")

	cmd.Flags().StringVarP(&credentialsFile, "credentials", "c", "user_credentials/credentials.json",
		"path to credentials file")

	cmd.Flags().StringVar(&parametersFile, "parameters", "", "file with parameters")

	cmd.Flags().StringToStringVar(&vars, "vars", nil,
		"list of user's variables; for example, 'image_name=enzyme,disk_size=30'")

	if os.Getenv("enzyme_ENABLE_SOCKS") != "" {
		cmd.Flags().StringVar(&socksHost, "socks-host", socksHost, "socks-host to access the network")
		cmd.Flags().IntVar(&socksPort, "socks-port", socksPort, "socks-port to access the network")
	}
}
