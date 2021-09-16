package state

import (
	"encoding/json"
	"os"
	"reflect"

	log "github.com/sirupsen/logrus"

	"enzyme/pkg/storage"
)

// Chest is an interface that actually stores and retrieves state objects,
// could be storing them on disk or in memory
type Chest interface {
	put(v interface{}, path string) error
	get(blank interface{}, path string) (interface{}, error)
	has(path string) bool
}

var (
	jsonChest Chest = &JSONChest{}
)

// JSONChest keeps objects as .json files
type JSONChest struct {
}

func (ch *JSONChest) put(v interface{}, path string) error {
	if err := storage.CreateDirForFile(path); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)

	return enc.Encode(v)
}

func (ch *JSONChest) get(blank interface{}, path string) (interface{}, error) {
	log.WithFields(log.Fields{
		"path":      path,
		"blueprint": blank,
	}).Info("JSONChest.get: loading entry")

	file, err := os.Open(path)
	if err != nil {
		log.WithFields(log.Fields{
			"path": path,
		}).Errorf("JSONChest.get: cannot open file: %s", err)

		return nil, err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	eptr := reflect.New(reflect.TypeOf(blank))

	if err = dec.Decode(eptr.Interface()); err != nil {
		log.WithFields(log.Fields{
			"path": path,
		}).Errorf("JSONChest.get: cannot parse json file: %s", err)

		return nil, err
	}

	return eptr.Interface(), nil
}

func (ch *JSONChest) has(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// MemChest keeps objects in memory and falls *on reading only* to
// JSONChest to see if objects were stored on disk; this is needed
// to make a read-only disk storage
type MemChest struct {
	data map[string]interface{}
}

func (ch *MemChest) put(v interface{}, path string) error {
	if ch.data == nil {
		ch.data = make(map[string]interface{})
	}

	ch.data[path] = v

	return nil
}

func (ch *MemChest) get(blank interface{}, path string) (interface{}, error) {
	mem, ok := ch.data[path]
	if ok {
		return mem, nil
	}

	return jsonChest.get(blank, path)
}

func (ch *MemChest) has(path string) bool {
	_, present := ch.data[path]
	if present {
		return true
	}

	return jsonChest.has(path)
}
