package store

import (
	"time"

	RESP "github.com/manish-singh-bisht/Redis-From-Scratch/db/resp"
)

type Store struct {
	kv      *keyValueStore
	streams *streamManager
}

func GetStore() *Store {
	return &Store{
		kv:      getKeyValueStore(),
		streams: getStreamManager(),
	}
}

func (s *Store) Get(key string) ([]byte, bool) {
	return s.kv.get(key)
}

func (s *Store) Set(key string, value []byte, expiration time.Duration) {
	s.kv.set(key, value, expiration)
}

func (s *Store) GetKeys(pattern string) []string {
	return s.kv.getKeys(pattern)
}

func (s *Store) XAdd(streamName, id string, data map[string][]byte) (StreamRecord, bool, error) {
	return s.streams.xadd(streamName, id, data)
}

func (s *Store) XRange(streamName, startId, endId string) ([]StreamRecord, error) {
	return s.streams.xrange(streamName, startId, endId)
}

func (s *Store) XRead(streamName, startId string) ([]StreamRecord, error) {
	return s.streams.xread(streamName, startId)
}

func (s *Store) XReadBlock(streamName, startId string, blockMs int, noTimeout bool) ([]StreamRecord, error) {
	return s.streams.xreadblock(streamName, startId, blockMs, noTimeout)
}

func (s *Store) CreateStreamMessages(records []StreamRecord) []RESP.RESPMessage {
	return s.streams.createStreamMessages(records)
}

func (s *Store) IsStreamKey(key string) bool {
	return s.streams.isStreamKey(key)
}
