package transactions

import "sync"

var globalKeyVersionsInstance *globalKeyVersions

// tracks the latest committed version of each key
type globalKeyVersions struct {
	mu       sync.RWMutex
	versions map[string]uint64 // key->latest committed version
}

func newGlobalKeyVersions() *globalKeyVersions {
	return &globalKeyVersions{
		versions: make(map[string]uint64),
	}
}

func getGlobalKeyVersions() *globalKeyVersions {
	if globalKeyVersionsInstance == nil {
		globalKeyVersionsInstance = newGlobalKeyVersions()
	}
	return globalKeyVersionsInstance
}

/**
 * upsertGlobalVersion updates the global version for a key
 * @param key string - the key to update the global version for
 * @return uint64 - the new global version
 */
func (ver *globalKeyVersions) upsertGlobalVersion(key string) uint64 {
	ver.mu.Lock()
	defer ver.mu.Unlock()

	newVersion := getNextVersion()
	ver.versions[key] = newVersion

	return newVersion
}

/**
 * getGlobalVersion gets the global version for a key
 * @param key string - the key to get the global version for
 * @return uint64 - the global version
 * @return bool - true if the key exists, false otherwise
 */
func (ver *globalKeyVersions) getGlobalVersion(key string) (uint64, bool) {
	ver.mu.RLock()
	defer ver.mu.RUnlock()

	version, exists := ver.versions[key]
	if exists {
		return version, true
	}

	return 0, false
}
