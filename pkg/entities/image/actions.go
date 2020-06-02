package image

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	action_pkg "Rhoc/pkg/action"
	"Rhoc/pkg/controller"
	"Rhoc/pkg/entities/common"
	"Rhoc/pkg/provider"
)

type makeConfig struct {
	img *imgState
}

func (action makeConfig) String() string {
	return fmt.Sprintf("Configure for %s", action.img)
}

func (action makeConfig) Apply() error {
	log.WithFields(log.Fields{
		"image": action.img,
	}).Info("Image.makeConfig.Apply")

	configHash, err := action.img.getConfigHash()
	if err != nil {
		return err
	}

	imageConfig, err := action.img.provider.MakeCreateImageConfig(
		action.img.templatePath, action.img.userVariables, configHash)
	if err != nil {
		log.WithFields(log.Fields{
			"imageTemplatePath": action.img.templatePath,
		}).Errorf("Image.makeConfig: cannot create config object: %s", err)

		return err
	}

	if err = imageConfig.Serialize(action.img.configPath); err != nil {
		log.WithFields(log.Fields{
			"imageConfigPath": action.img.configPath,
		}).Errorf("Image.makeConfig: cannot save config: %s", err)

		return err
	}

	imageDestroyDir, _ := filepath.Split(action.img.configPath)

	destroyImageConfig, err := action.img.provider.MakeDestroyImageConfig(action.img.userVariables)
	if err != nil {
		log.Errorf("Image.makeConfig: cannot make destroy config: %s", err)
		return err
	}

	destroyImageConfigPath := filepath.Join(imageDestroyDir, "config.tf.json")
	if err = destroyImageConfig.Serialize(destroyImageConfigPath); err != nil {
		log.WithFields(log.Fields{
			"destroyImageConfigPath": destroyImageConfigPath,
		}).Errorf("Image.makeConfig: cannot serialize destroy config: %s", err)

		return err
	}

	tfLogPrefix, err := action.img.makeToolLogPrefix("terraform")
	if err != nil {
		log.WithFields(log.Fields{
			"image": action.img,
		}).Warnf("Image.makeConfig: cannot make logfile name: %s", err)
	}

	if logname, err :=
		action_pkg.RunLoggedCmdDir(tfLogPrefix, imageDestroyDir, provider.Terraform(), "init"); err != nil {
		log.Errorf("Image.makeConfig: error initializing: %s", err)
		fmt.Fprintf(os.Stderr, "Failed to initialize tools, see log for details: %s\n", logname)

		return err
	}

	return nil
}

func (action makeConfig) IsExclusive() bool {
	return false
}

func (action makeConfig) Prerequisites() ([]controller.Target, error) {
	return []controller.Target{}, nil
}

type buildImage struct {
	img   *imgState
	stage common.SyncedStr
}

func (action *buildImage) String() string {
	return fmt.Sprintf("Build%s for %s", action.stage.Get(), action.img)
}

func (action *buildImage) imageExists() (bool, error) {
	localConfigHash, err := action.img.getConfigHash()
	if err != nil {
		return false, err
	}

	imageDestroyDir, _ := filepath.Split(action.img.configPath)
	logger := log.WithFields(log.Fields{
		"dir":   imageDestroyDir,
		"image": action.img,
	})

	tfLogPrefix, err := action.img.makeToolLogPrefix("terraform")
	if err != nil {
		logger.Warnf("Image.imageExists: cannot make logfile name: %s", err)
	}

	action.stage.Set(":checking existence")
	defer action.stage.Reset()

	if _, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, imageDestroyDir, provider.Terraform(),
		"refresh", "-state-out=checked.tfstate", "-backup=-"); err != nil {
		if exited, ok := err.(*exec.ExitError); ok {
			if exited.ExitCode() != -1 {
				logger.Info("Image.imageExists: 'terraform refresh' failed; assuming image does not exist")
			}
			return false, nil
		}
	}

	var buffer0 bytes.Buffer
	if logname, err := action_pkg.RunLoggedCmdDirOutput(tfLogPrefix, imageDestroyDir, &buffer0, provider.Terraform(),
		"output", "-state=checked.tfstate", "id"); err != nil {
		logger.Errorf("Image.imageExists: 'terraform output' failed: %s", err)
		fmt.Fprintf(os.Stderr, "Cannot check if image exists, see log for details: %s\n", logname)

		return false, err
	}
	imageID := strings.TrimSuffix(string(buffer0.Bytes()), "\n")

	imageResourceName := action.img.provider.GetTFImageResourceName()
	if _, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, imageDestroyDir, provider.Terraform(),
		"import", "-state-out=checked.tfstate", "-backup=-", imageResourceName+".zyme_image",
		imageID); err != nil {
		if exited, ok := err.(*exec.ExitError); ok {
			if exited.ExitCode() != -1 {
				logger.Info("Image.imageExists: image does not exist")
				return false, nil
			}
		}

		logger.Errorf("Image.imageExists: cannot check for existence: %s", err)
		return false, err
	}

	var buffer bytes.Buffer

	if logname, err :=
		action_pkg.RunLoggedCmdDirOutput(tfLogPrefix, imageDestroyDir, &buffer, provider.Terraform(),
			"state", "show", "-state=checked.tfstate", imageResourceName+".zyme_image"); err != nil {
		logger.Errorf("Image.imageExists: cannot read terraform output: %s", err)
		fmt.Fprintf(os.Stderr, "Cannot check if image exists, see log for details: %s\n", logname)

		return false, err
	}

	remoteConfigHash, err := action.img.provider.GetImageConfigHash(buffer.Bytes())
	if err != nil {
		logger.Warnf("Image.imageExists: cannot read config hash: %s, assume image out of date", err)
		return false, nil
	}

	if remoteConfigHash != localConfigHash {
		log.WithFields(log.Fields{
			"image":       action.img,
			"remote-hash": remoteConfigHash,
			"local-hash":  localConfigHash,
		}).Info("Image.imageExists: local and remote image config hashes differ, rebuilding image")
	} else {
		log.WithFields(log.Fields{
			"image": action.img,
			"hash":  remoteConfigHash,
		}).Info("Image.imageExists: local and remote image config hashes are equal")
	}

	return remoteConfigHash == localConfigHash, nil
}

func (action *buildImage) Apply() error {
	log.WithFields(log.Fields{
		"image": action.img,
	}).Info("Image.buildImage.Apply")

	exists, err := action.imageExists()
	if err != nil {
		return err
	}

	if exists {
		log.WithFields(log.Fields{
			"image": action.img,
		}).Info("Image.buildImage: image already exists, skip building")

		return nil
	}

	commandArg := []string{"build", "-force"}

	packerParams := map[string]string{
		"ssh_socks_proxy_host": action.img.serviceParameters.SocksProxyHost,
		"ssh_socks_proxy_port": strconv.Itoa(action.img.serviceParameters.SocksProxyPort),
	}
	for key, value := range packerParams {
		commandArg = append(commandArg, "-var", fmt.Sprintf("%s=%s", key, value))
	}

	commandArg = append(commandArg, action.img.configPath)

	packerLogPrefix, err := action.img.makeToolLogPrefix("packer")
	if err != nil {
		log.WithFields(log.Fields{
			"image": action.img,
		}).Warnf("Image.buildImage: cannot make logfile name: %s", err)
	}

	if logname, err := action_pkg.RunLoggedCmd(packerLogPrefix, provider.Packer(), commandArg...); err != nil {
		log.Errorf("Image.buildImage: error building: %s", err)
		fmt.Fprintf(os.Stderr, "Cannot build image, see log for details: %s\n", logname)

		return err
	}

	return nil
}

func (action *buildImage) IsExclusive() bool {
	return false
}

func (action *buildImage) Prerequisites() ([]controller.Target, error) {
	return []controller.Target{}, nil
}

type destroyImage struct {
	img *imgState
}

func (action destroyImage) String() string {
	return fmt.Sprintf("Destroy for %s", action.img)
}

func (action destroyImage) Apply() error {
	log.WithFields(log.Fields{
		"image": action.img,
	}).Info("Image.destroyImage.Apply")

	if !provider.IsProviderSupported(action.img.provider.GetName()) {
		log.WithFields(log.Fields{
			"provider": action.img.provider.GetName(),
		}).Error("[FIXME] unsupported provider, won't be able to delete images")

		return fmt.Errorf("unsupported provider for deletion: %s", action.img.provider.GetName())
	}

	imageDestroyDir, _ := filepath.Split(action.img.configPath)

	tfLogPrefix, err := action.img.makeToolLogPrefix("terraform")
	if err != nil {
		log.WithFields(log.Fields{
			"image": action.img,
		}).Warnf("Image.destroyImage: cannot make logfile name: %s", err)
	}

	imageResourceName := action.img.provider.GetTFImageResourceName()
	if logname, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, imageDestroyDir, provider.Terraform(),
		"import", "-state-out=imported.tfstate", "-backup=-", imageResourceName+".zyme_image",
		action.img.name); err != nil {
		log.Errorf("Image.destroyImage: error importing image: %s", err)
		fmt.Fprintf(os.Stderr, "Cannot destroy image, see log for details: %s\n", logname)

		return err
	}

	if logname, err := action_pkg.RunLoggedCmdDir(tfLogPrefix, imageDestroyDir, provider.Terraform(),
		"destroy", "-state=imported.tfstate", "-backup=-", "-force"); err != nil {
		log.Errorf("Image.destroyImage: error destroying image: %s", err)
		fmt.Fprintf(os.Stderr, "Cannot destroy image, see log for details: %s\n", logname)

		return err
	}

	return nil
}

func (action destroyImage) IsExclusive() bool {
	return false
}

func (action destroyImage) Prerequisites() ([]controller.Target, error) {
	return []controller.Target{}, nil
}
