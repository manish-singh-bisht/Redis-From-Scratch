package store

import (
	"sync"
	"time"
)

type StoredValue struct {
	value      []byte
	expiration time.Time
}

type KeyValueStore struct {
	mu    sync.RWMutex
	store map[string]StoredValue
}

var storeInstance *KeyValueStore

/*
 	* NewKeyValueStore creates a new KeyValueStore
	* @return *KeyValueStore - the new KeyValueStore
*/
func NewKeyValueStore() *KeyValueStore {
	return &KeyValueStore{
		store: make(map[string]StoredValue),
	}
}

/*
 	* GetStore returns the singleton instance of KeyValueStore
	* @return *KeyValueStore - the singleton instance of KeyValueStore
*/
func GetStore() *KeyValueStore {
	if storeInstance == nil {
		storeInstance = NewKeyValueStore()
	}
	return storeInstance
}

/*
 	* Set sets a key to a value
	* @param key string - the key to set
	* @param value []byte - the value to set
	* @param expiration time.Duration - the expiration time
*/
func (kv *KeyValueStore) Set(key string, value []byte, expiration time.Duration) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	storedValue := StoredValue{
		value: value,
	}

	if expiration > 0 {
		storedValue.expiration = time.Now().Add(expiration)
	}

	kv.store[key] = storedValue
}

/*
 	* Get gets a value from a key
	* @param key string - the key to get the value from
	* @return []byte - the value of the key
	* @return bool - true if the key exists, false otherwise
*/
func (kv *KeyValueStore) Get(key string) ([]byte, bool) {
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
 	* GetKeys returns all the keys that match the pattern
	* @param pattern string - the pattern to match
	* @return []string - the keys that match the pattern
*/
func (kv *KeyValueStore) GetKeys(pattern string) []string {
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
