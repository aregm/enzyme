package state

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"enzyme/pkg/storage"
)

// HierarchyHandler is a function that should return either a blueprint Entry that
// handles given stored object hierarchy or "nil" if it doesn't know how to handle it
type HierarchyHandler = func(hier []string, fetcher Fetcher) Entry

// RegisterHandler adds a handler to the list of all known handlers for loading
// any state by its path only
func RegisterHandler(handler HierarchyHandler) {
	handlers = append(handlers, handler)
}

const (
	category = "state"
)

var (
	stateDir string
	stateExt string = ".json"
	handlers []HierarchyHandler
)

// IsParseEntryFunc is a callback which determines whether to try parsing given entry
type IsParseEntryFunc func(id string) bool

// OnEntryFunc is a callback called for each entry Enumerate() encounters
// when walking the stored states
type OnEntryFunc func(id string, entry Entry) error

// Enumerate walks over all stored states and calls onEntry on each of them
func (fetcher Fetcher) Enumerate(isParse IsParseEntryFunc, onEntry OnEntryFunc) error {
	return filepath.Walk(stateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger := log.WithFields(log.Fields{
				"path": path,
			})
			if os.IsNotExist(err) {
				logger.Infof("Enumerate: cannot walk the path: %s", err)
				return filepath.SkipDir
			}
			logger.Errorf("Enumerate: cannot walk the path: %s", err)
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, stateExt) {
			log.WithFields(log.Fields{
				"path": path,
			}).Info("Enumerate: skipping path as it's a directory or doesn't have correct extension")
			return nil
		}
		relPath, err := filepath.Rel(stateDir, path)
		if err != nil {
			log.WithFields(log.Fields{
				"path": path,
			}).Errorf("Enumerate: cannot compute relative path: %s", err)
			return err
		}
		id := strings.TrimSuffix(filepath.ToSlash(filepath.Clean(relPath)), stateExt)
		if !isParse(id) {
			return nil
		}
		hierarchy := strings.Split(id, "/")
		for _, handler := range handlers {
			entry := handler(hierarchy, fetcher)
			if entry != nil {
				entry2, err := fetcher.loadFromPath(entry, path)
				if err != nil {
					log.WithFields(log.Fields{
						"path":    path,
						"read-as": entry,
					}).Errorf("Enumerate: cannot load path as expected entry: %s", err)
					return err
				}
				err = onEntry(id, entry2)
				if err != nil {
					return err
				}
				break
			}
		}
		return nil
	})
}

func init() {
	stateDir = storage.GetStoragePath(category)
	handlers = []HierarchyHandler{}
}
