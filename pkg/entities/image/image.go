package image

import (
	"crypto/md5"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	"enzyme/pkg/config"
	"enzyme/pkg/controller"
	"enzyme/pkg/provider"
	"enzyme/pkg/state"
	"enzyme/pkg/storage"
)

// Status describes status of the image
type Status int

const (
	// Nothing - nothing has been done to the image
	Nothing Status = iota
	// Configured - image config file has been created
	Configured
	// Created - image has been created in the cloud and is ready to use
	Created
)

const (
	category  = "image-configs"
	configExt = ".json"
)

// Satisfies being true means this status satisfies required "other" status
func (s Status) Satisfies(other controller.Status) bool {
	if casted, ok := other.(Status); ok {
		return s >= casted
	}

	return false
}

// Equals is only true if "other" status is exactly equal to this status
func (s Status) Equals(other controller.Status) bool {
	if casted, ok := other.(Status); ok {
		return s == casted
	}

	return false
}

var (
	statusToString = map[Status]string{
		Nothing:    "nothing",
		Configured: "configured",
		Created:    "created",
	}
	transitions = map[Status][]controller.Status{
		Nothing:    { /*Configured - this transition is not yet implemented*/ },
		Configured: {Nothing, Created},
		Created:    {Configured},
	}
)

func (s Status) String() string {
	result, ok := statusToString[s]
	if !ok {
		return "unknown"
	}

	return result
}

type imgState struct {
	status        Status
	name          string
	provider      provider.Provider
	templatePath  string
	configPath    string
	userVariables config.Config

	fetcher           state.Fetcher
	serviceParameters config.ServiceParams
}

func (img *imgState) String() string {
	return fmt.Sprintf("Image(name=%s, status=%s)", img.name, img.status)
}

func (img *imgState) GetDestroyedTarget() controller.Target {
	return controller.Target{
		Thing:         img,
		DesiredStatus: Configured,
		MatchExact:    true,
	}
}

func (img *imgState) Status() controller.Status {
	return img.status
}

func (img *imgState) SetStatus(status controller.Status) error {
	log.Info("Image.SetStatus called")

	casted, ok := status.(Status)
	if !ok {
		return fmt.Errorf("cannot set status of image - wrong type")
	}

	img.status = casted

	err := img.fetcher.Save(img)

	log.WithFields(log.Fields{
		"image":       *img,
		"new-status":  casted,
		"save-result": err,
	}).Info("status saved")

	return err
}

func (img *imgState) GetTransitions(to controller.Status) ([]controller.Status, error) {
	casted, ok := to.(Status)
	if !ok {
		return nil, fmt.Errorf("cannot get transitions to status %v - not an image status", to)
	}

	result, ok := transitions[casted]
	if !ok {
		return nil, fmt.Errorf("unexpected image status %v", to)
	}

	return result, nil
}

func (img *imgState) Equals(other controller.Thing) bool {
	casted, ok := other.(*imgState)
	if !ok {
		return false
	}

	return img.status.Equals(casted.status) &&
		img.name == casted.name &&
		img.provider.Equals(casted.provider) &&
		img.templatePath == casted.templatePath &&
		img.configPath == casted.configPath &&
		img.getUserVars() == casted.getUserVars()
}

func (img *imgState) GetAction(current controller.Status,
	target controller.Status) (controller.Action, error) {
	currentStatus, ok := current.(Status)
	if !ok {
		return nil, fmt.Errorf("current status %v is not image status", current)
	}

	targetStatus, ok := target.(Status)
	if !ok {
		return nil, fmt.Errorf("target status %v is not image status", target)
	}

	switch currentStatus {
	case Nothing:
		if targetStatus == Configured {
			return &makeConfig{img: img}, nil
		}
	case Configured:
		if targetStatus == Created {
			return &buildImage{img: img}, nil
		}
	case Created:
		if targetStatus == Configured {
			return &destroyImage{img: img}, nil
		}
	}

	return nil, fmt.Errorf("unsupported transition of (%v => %v)", currentStatus, targetStatus)
}

func (img *imgState) getConfigHash() (string, error) {
	configVars := img.getUserVars()

	packed, err := json.Marshal(configVars)
	if err != nil {
		log.WithFields(log.Fields{
			"vars": configVars,
		}).Errorf("Image.getConfigHash: cannot pack config vars to JSON: %s", err)

		return "", err
	}

	hasher := md5.New()
	if _, err = hasher.Write(packed); err != nil {
		log.WithFields(log.Fields{
			"vars": configVars,
		}).Errorf("Image.getConfigHash: cannot compute config hash: %s", err)

		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func (img *imgState) makeToolLogPrefix(tool string) (string, error) {
	hier, err := getHierarchy(img.provider, img.name)
	if err != nil {
		log.WithFields(log.Fields{
			"image": img,
		}).Errorf("Image.makeToolLogPrefix: cannot compute hierarchy: %s", err)

		return "", err
	}

	return storage.MakeStorageFilename(storage.LogCategory, append(hier, tool), ""), nil
}

// CreateImageTarget creates a Thing for controller package that represents the image used by cluster and
/// described by userVariables
func CreateImageTarget(prov provider.Provider, userVariables config.Config,
	serviceParams config.ServiceParams, fetcher state.Fetcher) (controller.Thing, error) {
	if err := prov.CheckUserVars(userVariables); err != nil {
		log.Errorf("Image.CreateImageTarget: user variables aren't supported or correct")

		return nil, err
	}

	imageTemplatePath, err := provider.GetDefaultTemplate(prov.GetName(), provider.ImageDescriptor)
	if err != nil {
		log.WithFields(log.Fields{
			"providerName": prov.GetName(),
		}).Errorf("CreateImage: %s", err)

		return nil, err
	}

	name, err := userVariables.GetString("image_name")
	if err != nil {
		template, err := config.CreateJSONConfigFromFile(imageTemplatePath)
		if err != nil {
			log.WithFields(log.Fields{
				"config-path": imageTemplatePath,
			}).Errorf("CreateImage: cannot read default config: %s", err)

			return nil, err
		}

		varname := "variables.image_name"

		name, err = template.GetString(varname)
		if err != nil {
			log.WithFields(log.Fields{
				"config-path": imageTemplatePath,
			}).Errorf("CreateImage: cannot read default %s: %s", varname, err)

			return nil, err
		}
	}

	hier, err := getHierarchy(prov, name)
	if err != nil {
		return nil, err
	}

	imageConfigPath := storage.MakeStorageFilename(category, append(hier, "config"), configExt)
	image := imgState{
		status:            Nothing,
		name:              name,
		provider:          prov,
		templatePath:      imageTemplatePath,
		configPath:        imageConfigPath,
		userVariables:     userVariables,
		fetcher:           fetcher,
		serviceParameters: serviceParams,
	}

	if imageFromDisk, err := image.fetcher.Load(&image); err == nil {
		if imageFromDisk != nil {
			image = *imageFromDisk.(*imgState)

			log.WithFields(log.Fields{
				"name": name,
			}).Info("CreateImage: image state loaded from disk")
		} else {
			log.WithFields(log.Fields{
				"name": name,
			}).Info("CreateImage: image state not found on disk")
		}
	} else {
		log.WithFields(log.Fields{
			"name": name,
		}).Errorf("CreateImage: cannot load image state: %s", err)

		return nil, err
	}

	return &image, nil
}
