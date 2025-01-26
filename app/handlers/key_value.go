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

func (kv *KeyValueStore) Get(key string) ([]byte, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	storedValue, exists := kv.store[key]
	if !exists {
		return nil, false
	}

	if !storedValue.expiration.IsZero() && time.Now().After(storedValue.expiration) {
		return nil, false
	}

	return storedValue.value, true
}
