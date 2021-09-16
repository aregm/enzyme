package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

const (
	// LogCategory is the descriptor for logs category
	LogCategory = "logs"
)

var (
	storageDir string
	logDir     string
)

// GetStoragePath returns category-specific directory
func GetStoragePath(category string) string {
	if category == LogCategory {
		return logDir
	}

	return filepath.Join(storageDir, category)
}

// MakeStorageFilename creates path to the file with extension ext using
// category and subpath (folder hierarchy including file name without extension) parameters;
// Resulted path will be in the next format: {category defined path}/{subpath}{ext}
func MakeStorageFilename(category string, subpath []string, ext string) string {
	args := []string{GetStoragePath(category)}
	args = append(args, subpath...)
	args[len(args)-1] = fmt.Sprintf("%s%s", args[len(args)-1], ext)

	return filepath.Join(args...)
}

// CreateDirForFile is simplified version of CreateDirForFileEx
// creating a directory with group-read and no world access
func CreateDirForFile(fileName string) error {
	return CreateDirForFileEx(fileName, 0750)
}

// CreateDirForFileEx creates a directory to the file fileName in the case if
// such a directory is specified and doesn't yet exist
func CreateDirForFileEx(fileName string, perm os.FileMode) error {
	var err error

	var dir string

	if dir, _ = filepath.Split(fileName); dir == "" {
		log.WithFields(log.Fields{
			"filename": fileName,
			"dir":      dir,
		}).Warn("CreateDirForFile: directory name is empty, not creating")

		return nil
	}

	info, err := os.Stat(dir)
	if os.IsExist(err) {
		if info.IsDir() {
			log.WithFields(log.Fields{
				"dir": dir,
			}).Info("CreateDirForFile: directory already exist")

			return nil
		}

		msg := "CreateDirForFile: target path exists and is not a directory"

		log.WithFields(log.Fields{
			"dir": dir,
		}).Error(msg)

		return fmt.Errorf("%s: %s", msg, dir)
	}

	err = os.MkdirAll(dir, perm)
	if err != nil {
		log.WithFields(log.Fields{
			"dir":        dir,
			"permission": perm,
		}).Errorf("CreateDirForFile: %s", err)
	}

	return nil
}

// CopyFile copies file contents from src to dest
func CopyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, sourceFileStat.Mode())
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)

	return err
}

func init() {
	curdir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("cannot get current dir: %v", err))
	}

	storageDir = filepath.Join(curdir, ".enzyme")
	logDir = filepath.Join(curdir, "logs")
}
