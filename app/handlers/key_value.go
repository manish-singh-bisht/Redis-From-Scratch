package handlers

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

func NewKeyValueStore() *KeyValueStore {
	return &KeyValueStore{
		store: make(map[string]StoredValue),
	}
}

func (kv *KeyValueStore) set(key string, value []byte, expiration time.Duration) {
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

func (kv *KeyValueStore) get(key string) ([]byte, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	// check for key, returns nil if absent.
	storedValue, exists := kv.store[key]
	if !exists {
		return nil, false
	}

	// check whether expiration was provided and the time has passed more than the expiration, if yes return nil.
	if !storedValue.expiration.IsZero() && time.Now().After(storedValue.expiration) {
		return nil, false
	}

	// return the value
	return storedValue.value, true
}
