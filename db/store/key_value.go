package store

import (
	"sync"
	"time"
)

type storedValue struct {
	value      []byte
	expiration time.Time
}

type keyValueStore struct {
	mu    sync.RWMutex
	store map[string]storedValue
}

var storeInstance *keyValueStore

/*
 	* newKeyValueStore creates a new keyValueStore
	* @return *keyValueStore - the new keyValueStore
*/
func newKeyValueStore() *keyValueStore {
	return &keyValueStore{
		store: make(map[string]storedValue),
	}
}

/*
 	* getKeyValueStore returns the singleton instance of keyValueStore
	* @return *keyValueStore - the singleton instance of keyValueStore
*/
func getKeyValueStore() *keyValueStore {
	if storeInstance == nil {
		storeInstance = newKeyValueStore()
	}
	return storeInstance
}

/*
 	* set sets a key to a value
	* @param key string - the key to set
	* @param value []byte - the value to set
	* @param expiration time.Duration - the expiration time
*/
func (kv *keyValueStore) set(key string, value []byte, expiration time.Duration) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	storedValue := storedValue{
		value: value,
	}

	if expiration > 0 {
		storedValue.expiration = time.Now().Add(expiration)
	}

	kv.store[key] = storedValue
}

/*
 	* get gets a value from a key
	* @param key string - the key to get the value from
	* @return []byte - the value of the key
	* @return bool - true if the key exists, false otherwise
*/
func (kv *keyValueStore) get(key string) ([]byte, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	storedValue, exists := kv.store[key]
	if !exists {
		return nil, false
	}

	// check whether expiration was provided and the time has passed more than the expiration, if yes return nil.
	// should we delete the key from the store???
	if !storedValue.expiration.IsZero() && time.Now().After(storedValue.expiration) {
		return nil, false
	}

	return storedValue.value, true
}

/*
 	* getKeys returns all the keys that match the pattern
	* @param pattern string - the pattern to match
	* @return []string - the keys that match the pattern
*/
func (kv *keyValueStore) getKeys(pattern string) []string {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	var keys []string
	for key, value := range kv.store {

		if !value.expiration.IsZero() && time.Now().After(value.expiration) {
			continue
		}
		// For now we only support "*" pattern which matches all keys
		if pattern == "*" {
			keys = append(keys, key)
		}
	}
	return keys
}
