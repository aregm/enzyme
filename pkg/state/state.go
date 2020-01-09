package state

import (
	log "github.com/sirupsen/logrus"

	"Rhoc/pkg/storage"
)

// Entry is an interface that describes persistent part of an Rhoc object
type Entry interface {
	Hierarchy() ([]string, error)
	ToPublic() (interface{}, error)
	FromPublic(v interface{}) (Entry, error)
}

// Fetcher is a helper struct that wraps a Chest to actually store and
// load Entries which are describing the state of any Rhoc persistent object
type Fetcher struct {
	Chest Chest
}

func (fetcher Fetcher) loadFromPath(entry Entry, path string) (Entry, error) {
	blank, err := entry.ToPublic()
	if err != nil {
		return nil, err
	}

	if !fetcher.Chest.has(path) {
		log.WithFields(log.Fields{
			"entry": entry,
			"path":  path,
		}).Info("Load: no state on disk")

		return nil, nil
	}

	if blank, err = fetcher.Chest.get(blank, path); err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"entry": entry,
		"path":  path,
	}).Infof("Load: loaded entry from disk as %s", blank)

	result, err := entry.FromPublic(blank)
	if err != nil {
		log.WithFields(log.Fields{
			"entry": entry,
			"path":  path,
		}).Errorf("Load: failed FromPublic(): %s", err)

		return nil, err
	}

	return result, err
}

// Load uses fetcher's Chest to load the stored state of given entry
func (fetcher Fetcher) Load(entry Entry) (Entry, error) {
	path, err := getPath(entry)
	if err != nil {
		return nil, err
	}

	return fetcher.loadFromPath(entry, path)
}

// Save uses fetcher's Chest to save the state of given entry
func (fetcher Fetcher) Save(entry Entry) error {
	public, err := entry.ToPublic()
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"entry": entry,
	}).Info("Save: saving entry")

	path, err := getPath(entry)
	if err != nil {
		return err
	}

	return fetcher.Chest.put(public, path)
}

func getPath(entry Entry) (string, error) {
	hier, err := entry.Hierarchy()
	if err != nil {
		return "", err
	}

	return storage.MakeStorageFilename(category, hier, stateExt), nil
}
