package store

import (
	"fmt"
	"strconv"
	"strings"
)

/*
 	* IsStreamKey checks if a stream key exists
	* @param key string - the key to check
	* @return bool - true if the key exists, false otherwise
*/
func (sm *StreamsManager) IsStreamKey(key string) bool {
	_, exists := sm.Streams[key]
	return exists
}

/*
 	* The ID should be greater than the ID of the last entry in the stream.
	* The millisecondsTime part of the ID should be greater than or equal to the millisecondsTime of the last entry.
	* If the millisecondsTime part of the ID is equal to the millisecondsTime of the last entry, the sequenceNumber part of the ID should be greater than the sequenceNumber of the last entry.
	* If the stream is empty, the ID should be greater than 0-0
*/
func (sm *StreamsManager) verifyStreamId(streamName, id string) (bool, error) {
	if !sm.IsStreamKey(streamName) {
		return false, fmt.Errorf("the stream specified does not exist")
	}

	stream := sm.Streams[streamName]

	lastEntry := stream.recordList.Back()
	if lastEntry == nil {
		if id > "0-0" {
			return true, nil
		}
		return false, fmt.Errorf(" ERR The ID specified must be greater than 0-0")
	}

	if id == "0-0" {
		return false, fmt.Errorf("ERR The ID specified must be greater than 0-0")
	}

	lastEntryRecord, ok := lastEntry.Value.(*streamRecord)
	if !ok {
		return false, fmt.Errorf("ERR Unexpected type in list element")
	}

	millisecondsTime, sequenceNumber, ok, err := sm.parseStreamId(id)
	if !ok && err != nil {
		return false, err
	}

	if lastEntryRecord.millisecondsTime > millisecondsTime {
		return false, fmt.Errorf("ERR The ID specified is equal or smaller than the target stream top item")
	}
	if lastEntryRecord.millisecondsTime == millisecondsTime && lastEntryRecord.sequenceNumber >= sequenceNumber {
		return false, fmt.Errorf("ERR The ID specified is equal or smaller than the target stream top item")
	}

	return true, nil
}

/*
 	* parseStreamId parses a stream ID into millisecondsTime and sequenceNumber
	* @param id string - the ID to parse
	* @return int64 - the millisecondsTime part of the ID
	* @return int - the sequenceNumber part of the ID
	* @return bool - true if the ID is valid, false otherwise
	* @return error - the error if there is one
*/
func (sm *StreamsManager) parseStreamId(id string) (int64, int, bool, error) {
	idParts := strings.Split(id, "-")
	millisecondsTime, err := strconv.ParseInt(idParts[0], 10, 64)
	if err != nil {
		return 0, 0, false, fmt.Errorf("ERR The millisecondsTime part of the ID specified is invalid")
	}
	sequenceNumber, err := strconv.Atoi(idParts[1])
	if err != nil {
		return 0, 0, false, fmt.Errorf("ERR The sequenceNumber part of the ID specified is invalid")
	}
	return millisecondsTime, sequenceNumber, true, nil
}
