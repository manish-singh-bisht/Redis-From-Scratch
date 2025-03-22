package transactions

import (
	"log"
	"sync"
)

type clientWatches struct {
	mu                sync.RWMutex
	watches           map[string]map[string]uint64 // clientID->localKey->localVersion(only the first version)
	globalKeyVersions *globalKeyVersions
}

func NewClientWatches(global *globalKeyVersions) *clientWatches {
	return &clientWatches{
		watches:           make(map[string]map[string]uint64),
		globalKeyVersions: global,
	}
}

func (cw *clientWatches) startWatch(clientID string, key string) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	_, exists := cw.watches[clientID]
	if !exists {
		cw.watches[clientID] = make(map[string]uint64)
	}

	// store the current global version as the first seen version
	latestVer, exists := cw.globalKeyVersions.getGlobalVersion(key)
	if !exists {
		latestVer = cw.globalKeyVersions.upsertGlobalVersion(key)
	}

	cw.watches[clientID][key] = latestVer
}

func (cw *clientWatches) unwatch(clientID string) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	// should we remove key-versions from global as well??
	delete(cw.watches, clientID)
}

// checkWatches compares local versions with the current global version and if any differences than transaction invalid, CAS
func (cw *clientWatches) checkWatches(clientID string) (bool, error) {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	clientVersions, exists := cw.watches[clientID]
	if !exists {
		return true, nil // no watches,transaction is valid
	}

	for key, localVersion := range clientVersions {
		currentGlobalVersion, exists := cw.globalKeyVersions.getGlobalVersion(key)
		if !exists {
			// if a local keyVersion is being added then a global is also added or previously present
			return false, inconsistency(key)
		}

		// if the current global version is greater,it was modified
		if currentGlobalVersion > localVersion {
			log.Print("aborting transactions, keys were changed")
			return false, nil
		}
	}
	return true, nil
}
