package provider

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

var (
	packerExe    string
	terraformExe string
)

func makeToolPath(tool string) string {
	currExe, err := os.Executable()
	if err != nil {
		log.WithFields(log.Fields{
			"tool": tool,
		}).Warnf("Cannot get path to current executable: %s, using default tool", err)

		return tool
	}

	baseDir, currExe := filepath.Split(currExe)
	exeExt := filepath.Ext(currExe)
	toolPath := filepath.Join(baseDir, "tools", tool+exeExt)

	info, err := os.Stat(toolPath)
	if err != nil {
		log.WithFields(log.Fields{
			"tool":      tool,
			"tool-path": toolPath,
		}).Infof("Cannot stat tool at expected place: %s, using default tool", err)

		return tool
	}

	if info.IsDir() {
		log.WithFields(log.Fields{
			"tool":      tool,
			"tool-path": toolPath,
		}).Warn("Tool at expected place is a directory, using default tool")

		return tool
	}

	log.WithFields(log.Fields{
		"tool":      tool,
		"tool-path": toolPath,
	}).Info("Found tool path")

	return toolPath
}

// Packer returns path to Packer binary
func Packer() string {
	return packerExe
}

// Terraform returns path to Terraform binary
func Terraform() string {
	return terraformExe
}

// InitTools sets path names for packer and terraform binaries
func InitTools() {
	packerExe = makeToolPath("packer")
	terraformExe = makeToolPath("terraform")
}
