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

var Store = NewKeyValueStore()

func NewKeyValueStore() *KeyValueStore {
	return &KeyValueStore{
		store: make(map[string]StoredValue),
	}
}

func (kv *KeyValueStore) Set(key string, value []byte, expiration time.Duration) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	// add value.
	storedValue := StoredValue{
		value: value,
	}
	// add expiration, if provided.
	if expiration > 0 {
		storedValue.expiration = time.Now().Add(expiration)
	}

	// finally map value to the key.
	kv.store[key] = storedValue
}

func (kv *KeyValueStore) Get(key string) ([]byte, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	// check for key, returns nil if absent.
	storedValue, exists := kv.store[key]
	if !exists {
		return nil, false
	}

	// check whether expiration was provided and the time has passed more than the expiration, if yes return nil.
	// should we delete the key from the store???
	if !storedValue.expiration.IsZero() && time.Now().After(storedValue.expiration) {
		return nil, false
	}

	// return the value
	return storedValue.value, true
}
func (kv *KeyValueStore) GetKeys(pattern string) []string {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	var keys []string
	for key, value := range kv.store {
		// Skip expired keys
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
