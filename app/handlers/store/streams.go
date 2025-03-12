package store

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

var StreamManager = newStreamsManager()

const (
	DefaultStreamMaxLen = 1000
)

type StreamRecord struct {
	Id               string // store ID (Milliseconds-SequenceNumber), this combination is most probably done to make it monotonically increasing, not completely dependent on the time(due to time-of-the-day clock skew), not sure though
	MillisecondsTime int64
	SequenceNumber   int
	Data             map[string][]byte
}

type Stream struct {
	mu sync.RWMutex
	// Maps record ID to its corresponding list element for O(1) lookups
	recordMap   map[string]*list.Element   // Fast lookup of records by ID
	recordList  *list.List                 // Doubly linked list for ordered storage
	maxLen      int                        // Maximum number of entries to keep, also in REDIS
	subscribers map[chan struct{}]struct{} // map of channels to notify of new entries, for faster lookups during sending and removing of subscribers

}

type StreamsManager struct {
	Streams map[string]*Stream // Map of stream names to Stream objects, for faster lookups
}

func newStream() *Stream {
	return &Stream{
		recordMap:   make(map[string]*list.Element),
		recordList:  list.New(),
		maxLen:      DefaultStreamMaxLen,
		subscribers: make(map[chan struct{}]struct{}),
	}
}

func newStreamsManager() *StreamsManager {
	return &StreamsManager{
		Streams: make(map[string]*Stream),
	}
}

/*
 	* xAdd adds a new entry to a stream
	* @param streamName string - the name of the stream
	* @param id string - the ID of the new entry
	* @param data map[string][]byte - the data for the new entry
	* @return StreamRecord - the new entry
	* @return bool - true if the entry was added successfully, false otherwise
	* @return error - the error if there is one
*/
func (sm *StreamsManager) XAdd(streamName, id string, data map[string][]byte) (StreamRecord, bool, error) {

	if !sm.IsStreamKey(streamName) {
		sm.Streams[streamName] = newStream()
	}
	valid, err := sm.verifyStreamId(streamName, id)
	if !valid {
		return StreamRecord{}, false, err
	}

	wildcardNum, millisecondsTime, sequenceNumber, err := sm.parseStreamId(id)
	if err != nil {
		return StreamRecord{}, false, err
	}

	var newId string = id
	var newMillisecondsTime int64 = millisecondsTime
	var newSequenceNumber int = sequenceNumber

	if wildcardNum == AutoGeneratedTimeAndSeq {
		newId, newMillisecondsTime, newSequenceNumber, err = sm.generateStreamId(streamName, id)
		if err != nil {
			return StreamRecord{}, false, err
		}
	}

	if wildcardNum == AutoGeneratedSeq {
		newId, newMillisecondsTime, newSequenceNumber, err = sm.generateStreamId(streamName, id)
		if err != nil {
			return StreamRecord{}, false, err
		}
	}

	stream := sm.Streams[streamName]
	stream.mu.Lock()
	defer stream.mu.Unlock()

	newStreamRecord := StreamRecord{
		Id:               newId,
		MillisecondsTime: newMillisecondsTime,
		SequenceNumber:   newSequenceNumber,
		Data:             data,
	}
	element := stream.recordList.PushBack(&newStreamRecord)
	stream.recordMap[newId] = element

	// remove the oldest entries when len exceeds the specified max length for a stream
	if stream.recordList.Len() > stream.maxLen {
		numToRemove := stream.recordList.Len() - stream.maxLen
		for i := 0; i < numToRemove; i++ {
			oldest := stream.recordList.Front()
			if oldest != nil {
				record := oldest.Value.(*StreamRecord)
				delete(stream.recordMap, record.Id)
				stream.recordList.Remove(oldest)
			}
		}
	}

	stream.notifySubscribers()

	return newStreamRecord, true, nil
}

/*
 	* XRange gets a range of entries from a stream
	* @param streamName string - the name of the stream
	* @param startId string - the ID of the start of the range
	* @param endId string - the ID of the end of the range
	* @return []StreamRecord - the range of entries, inclusive of the start and end IDs
	* @return error - the error if there is one
*/
func (sm *StreamsManager) XRange(streamName, startId, endId string) ([]StreamRecord, error) {
	if !sm.IsStreamKey(streamName) {
		return nil, fmt.Errorf("ERR The stream specified does not exist")
	}

	exists, err := sm.IsValidStreamRecordIdExists(streamName, startId, OperationTypeRange)
	if !exists {
		return nil, err
	}

	exists, err = sm.IsValidStreamRecordIdExists(streamName, endId, OperationTypeRange)
	if !exists {
		return nil, err
	}

	stream := sm.Streams[streamName]

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	var startElem *list.Element
	var endElem *list.Element

	if startId == RangeQueryEnd || endId == RangeQueryStart {
		return nil, fmt.Errorf("ERR The start or end ID is invalid")
	}

	if startId == RangeQueryStart {
		startElem = stream.recordList.Front()
	} else {
		startElem = stream.recordMap[startId]
	}

	if endId == RangeQueryEnd {
		endElem = stream.recordList.Back()
	} else {
		endElem = stream.recordMap[endId]
	}

	var result []StreamRecord
	current := startElem

	for current != nil {
		record := current.Value.(*StreamRecord)
		result = append(result, *record)

		if current == endElem {
			break
		}
		current = current.Next()
	}

	return result, nil
}

/*
 	* XRead gets a range of entries from a stream, exclusive of the start ID
	* @param streamName string - the name of the stream
	* @param startId string - the ID of the start of the range
	* @return []StreamRecord - the range of entries, exclusive of the start ID
	* @return error - the error if there is one
*/
func (sm *StreamsManager) XRead(streamName, startId string) ([]StreamRecord, error) {
	if !sm.IsStreamKey(streamName) {
		return nil, fmt.Errorf("ERR The stream specified does not exist")
	}

	_, msTime, seqNum, err := sm.parseStreamId(startId)
	if err != nil {
		return nil, err
	}

	stream := sm.Streams[streamName]

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	var result []StreamRecord

	var startElem *list.Element
	var current *list.Element

	if startId == StreamIDMin {
		startElem = stream.recordList.Front()
		current = startElem
	} else {

		// find the first record that is greater than the startId
		for rec := stream.recordList.Front(); rec != nil; rec = rec.Next() {
			record := rec.Value.(*StreamRecord)
			if record.MillisecondsTime > msTime || (record.MillisecondsTime == msTime && record.SequenceNumber > seqNum) {
				current = rec
				break
			}
		}
	}

	for current != nil {
		record := current.Value.(*StreamRecord)
		result = append(result, *record)

		current = current.Next()
	}

	return result, nil

}

/*
when a xread with block comes a new subscriber is added to the map and then it first reads from the id specified and then waits for new incoming , when a another xadd happens during that time, the notifySubscribers is basically calling all the subscribers in the map(this calling is basically a way of just saying that a new has arrived and not what has arrived) this way the blocking subsribers in the xreadblock previously will now re-read and thus display the new entry
*/
func (sm *StreamsManager) XReadBlock(streamName, startId string, blockMs int) ([]StreamRecord, error) {
	if !sm.IsStreamKey(streamName) {
		return nil, fmt.Errorf("ERR The stream specified does not exist")
	}

	stream := sm.Streams[streamName]

	notify := stream.subscribe()     // add to map, and get channel
	defer stream.unsubscribe(notify) // remove from map, and close channel

	deadline := time.NewTimer(time.Duration(blockMs) * time.Millisecond)
	defer deadline.Stop()

	for {
		records, err := sm.XRead(streamName, startId)
		if err != nil {
			return nil, err
		}
		if len(records) > 0 {
			return records, nil
		}

		// block until either notification or timeout
		select {
		case <-notify:
			continue // check for records again
		case <-deadline.C:
			return nil, nil // return nil on timeout
		}
	}
}
