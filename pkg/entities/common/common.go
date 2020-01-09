package common

import (
	"sync"
)

// SyncedStr is a struct that synchronizes reading/writing a string variable
type SyncedStr struct {
	value string
	mux   sync.Mutex
}

// Get returns synchronized string value
func (ss *SyncedStr) Get() string {
	ss.mux.Lock()
	defer ss.mux.Unlock()

	return ss.value
}

// Set puts new value with synchronization
func (ss *SyncedStr) Set(value string) {
	ss.mux.Lock()
	ss.value = value
	ss.mux.Unlock()
}

// Reset sets the value to empty string (is a shortcut for .Set(""))
func (ss *SyncedStr) Reset() {
	ss.Set("")
}
