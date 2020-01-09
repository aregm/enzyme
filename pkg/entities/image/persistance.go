package image

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"Rhoc/pkg/provider"
	"Rhoc/pkg/state"
)

func getHierarchy(prov provider.Provider, name string) ([]string, error) {
	memorizedID, err := provider.MemorizedID(prov)
	if err != nil {
		return []string{}, err
	}

	return []string{memorizedID, name}, nil
}

// implement state.Entry methods

func (img *imgState) Hierarchy() ([]string, error) {
	hier, err := getHierarchy(img.provider, img.name)
	if err != nil {
		return []string{}, err
	}

	return append([]string{"image"}, hier...), nil
}

type userVarsPersist struct {
	ProjectName   string
	UserName      string
	DiskSize      string
	CentosRelease string
	SourceImage   string
}

type providerPersist struct {
	Name   string
	Region string
	Zone   string
	Creds  string
}

type persistent struct {
	Status       int
	Name         string
	Provider     providerPersist
	TemplatePath string
	ConfigPath   string
	UserVars     userVarsPersist
}

func (img *imgState) getProviderVars() providerPersist {
	if img.provider == nil {
		return providerPersist{}
	}

	return providerPersist{
		img.provider.GetName(),
		img.provider.GetRegion(),
		img.provider.GetZone(),
		img.provider.GetCredentialPath(),
	}
}

func (img *imgState) getUserVars() userVarsPersist {
	if img.userVariables == nil {
		return userVarsPersist{}
	}

	var result userVarsPersist

	var err error

	if result.ProjectName, err = img.userVariables.GetString("project_name"); err != nil {
		result.ProjectName = ""
	}

	if result.UserName, err = img.userVariables.GetString("user_name"); err != nil {
		result.UserName = ""
	}

	if result.DiskSize, err = img.userVariables.GetString("disk_size"); err != nil {
		result.DiskSize = ""
	}

	if result.CentosRelease, err = img.userVariables.GetString("centos_release"); err != nil {
		result.CentosRelease = ""
	}

	if result.SourceImage, err = img.userVariables.GetString("source_image"); err != nil {
		result.SourceImage = ""
	}

	return result
}

func (img *imgState) ToPublic() (interface{}, error) {
	return persistent{
		int(img.status),
		img.name,
		img.getProviderVars(),
		img.templatePath,
		img.configPath,
		img.getUserVars(),
	}, nil
}

func (img *imgState) FromPublic(v interface{}) (state.Entry, error) {
	persist, ok := v.(persistent)
	if !ok {
		pPersist, ok := v.(*persistent)
		if !ok {
			log.WithFields(log.Fields{
				"read": v,
			}).Error("Image.FromPublic: cannot parse incoming object as imgState")

			return nil, fmt.Errorf("incompatible intermediate type")
		}

		persist = *pPersist
	}

	if img.userVariables != nil {
		userVars := img.getUserVars()
		if persist.UserVars != userVars {
			log.WithFields(log.Fields{
				"loaded-vars":  persist.UserVars,
				"current-vars": userVars,
			}).Info("Image.FromPublic: loaded user variables differ from current, invalidating stored state")

			return nil, nil
		}
	} else {
		log.WithFields(log.Fields{
			"loaded-vars": persist.UserVars,
		}).Info("Image.FromPublic: target doesn't have variables set, use with caution")
	}

	status := Status(persist.Status)
	if status < Nothing || status > Created {
		log.WithFields(log.Fields{
			"read": v,
		}).Errorf("Image.FromPublic: incoming status is unexpected: %d", persist.Status)

		return nil, fmt.Errorf("unexpected status: %d", persist.Status)
	}

	prov, err := provider.CreateProvider(persist.Provider.Name, persist.Provider.Region,
		persist.Provider.Zone, persist.Provider.Creds)
	if err != nil {
		log.WithFields(log.Fields{
			"read": v,
		}).Errorf("Image.FromPublic: cannot construct provider: %s", err)

		return nil, err
	}

	return &imgState{
		status,
		persist.Name,
		prov,
		persist.TemplatePath,
		persist.ConfigPath,
		img.userVariables,
		img.fetcher,
		img.serviceParameters,
	}, nil
}

func handler(hier []string, fetcher state.Fetcher) state.Entry {
	if len(hier) != 0 && hier[0] == "image" {
		return &imgState{
			fetcher: fetcher,
		}
	}

	return nil
}

func init() {
	state.RegisterHandler(handler)
}
