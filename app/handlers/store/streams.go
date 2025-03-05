package store

import (
	"container/list"
	"sync"
)

var StreamManager = newStreamsManager()

type StreamsStore interface {
	XAdd(stream, id string, data map[string]string) (streamRecord, error)
	XRange(stream, startId, endId string) ([]streamRecord, error)
	XGetStream(stream string) (*stream, bool)
	XRead(stream, id string) (streamRecord, bool)
}

type streamRecord struct {
	Id               string // store ID (Milliseconds-SequenceNumber), this is most probably done to make it monotonically increasing, not completely dependent on the time(due to time-of-the-day clock skew), not sure though
	millisecondsTime int64
	sequenceNumber   int
	data             map[string][]byte // Key-value data for the record
}

type stream struct {
	mu sync.RWMutex
	// Maps record ID to its corresponding list element for O(1) lookups
	recordMap  map[string]*list.Element // Fast lookup of records by ID
	recordList *list.List               // Doubly linked list for ordered storage
}

// Maps stream names to corresponding stream object
type StreamsManager struct {
	Streams map[string]*stream // Map of stream names to Stream objects, for faster lookups
}

func newStream() *stream {
	return &stream{
		recordMap:  make(map[string]*list.Element),
		recordList: list.New(),
	}
}

func newStreamsManager() *StreamsManager {
	return &StreamsManager{
		Streams: make(map[string]*stream),
	}
}

/*
 	* xAdd adds a new entry to a stream
	* @param streamName string - the name of the stream
	* @param id string - the ID of the new entry
	* @param data map[string][]byte - the data for the new entry
	* @return streamRecord - the new entry
	* @return bool - true if the entry was added successfully, false otherwise
	* @return error - the error if there is one
*/
func (sm *StreamsManager) XAdd(streamName, id string, data map[string][]byte) (streamRecord, bool, error) {

	if !sm.IsStreamKey(streamName) {
		sm.Streams[streamName] = newStream()
	}
	valid, err := sm.verifyStreamId(streamName, id)
	if !valid {
		return streamRecord{}, false, err
	}

	millisecondsTime, sequenceNumber, ok, err := sm.parseStreamId(id)
	if !ok && err != nil {
		return streamRecord{}, false, err
	}

	stream := sm.Streams[streamName]
	stream.mu.Lock()
	defer stream.mu.Unlock()

	streamRecord := streamRecord{
		Id:               id,
		millisecondsTime: millisecondsTime,
		sequenceNumber:   sequenceNumber,
		data:             data,
	}
	element := stream.recordList.PushBack(&streamRecord)
	stream.recordMap[id] = element

	return streamRecord, true, nil
}
