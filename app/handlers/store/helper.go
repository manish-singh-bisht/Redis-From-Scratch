package store

import (
	"fmt"
	"strconv"
	"strings"
	"time"
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

	if id == "0-0" {
		return false, fmt.Errorf("ERR The ID specified in XADD must be greater than 0-0")
	}

	lastEntry := stream.recordList.Back()
	if lastEntry == nil {
		if id > "0-0" || id == "0-*" || id == "*" {
			return true, nil
		}
		return false, fmt.Errorf("ERR The ID specified in XADD must be greater than 0-0")
	}
	if id == "0-0" {
		return false, fmt.Errorf("ERR The ID specified must be greater than 0-0")
	}

	lastEntryRecord, ok := lastEntry.Value.(*streamRecord)
	if !ok {
		return false, fmt.Errorf("ERR Unexpected type in list element")
	}

	isFullAutoGenerated, millisecondsTime, sequenceNumber, err := sm.parseStreamId(id)
	if err != nil {
		return false, err
	}

	if isFullAutoGenerated || sequenceNumber == -1 {
		return true, nil
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
	* -1 means that the sequenceNumber is auto generated
	* -2 means that the millisecondsTime is auto generated
	* @param id string - the ID to parse
	* @return bool - true if the ID is *
	* @return int64 - the millisecondsTime part of the ID
	* @return int - the sequenceNumber part of the ID
	* @return error - the error if there is one

*/
func (sm *StreamsManager) parseStreamId(id string) (bool, int64, int, error) {

	if id == "*" {
		return true, -2, -1, nil
	}

	idParts := strings.Split(id, "-")
	millisecondsTime, err := strconv.ParseInt(idParts[0], 10, 64)
	if err != nil {
		return false, 0, 0, fmt.Errorf("ERR The millisecondsTime part of the ID specified is invalid")
	}

	if idParts[1] == "*" {
		return false, millisecondsTime, -1, nil // -1 means that sequenceNumber is auto generated
	}

	sequenceNumber, err := strconv.Atoi(idParts[1])
	if err != nil {
		return false, 0, 0, fmt.Errorf("ERR The sequenceNumber part of the ID specified is invalid")
	}
	return false, millisecondsTime, sequenceNumber, nil
}

/*
 	* generateStreamId generates a new stream ID
	* @param streamName string - the name of the stream
	* @param id string - the ID of the new entry
	* @param halfAutoGenerated bool - true if the ID is half auto generated, false otherwise, when false , it is full auto generated, halfgenerated means that the sequenceNumber is auto generated, fullgenerated means that the millisecondsTime and sequenceNumber are auto generated
	* @return string - the new stream ID
	* @return int64 - the millisecondsTime part of the new stream ID
	* @return int - the sequenceNumber part of the new stream ID
	* @return error - the error if there is one
*/
func (sm *StreamsManager) generateStreamId(streamName string,id string, halfAutoGenerated bool) (string, int64, int, error) {
	
	if !sm.IsStreamKey(streamName) {
		return "", 0, 0, fmt.Errorf("ERR The stream specified does not exist")
	}

	stream := sm.Streams[streamName]

	_, millisecondsTime, sequenceNumber, err := sm.parseStreamId(id)
	if err != nil {
		return "", 0, 0, err
	}

	lastEntry := stream.recordList.Back()

	if lastEntry == nil {
		if sequenceNumber == -1 && millisecondsTime != 0 {
			return fmt.Sprintf("%d-%d", millisecondsTime, 0), millisecondsTime, 0, nil
		}else if millisecondsTime == 0 && sequenceNumber == -1 {
			return fmt.Sprintf("%d-%d", millisecondsTime, 0), millisecondsTime, 0, nil
		}else if millisecondsTime == -2 && sequenceNumber == -1 {
			currentTime := time.Now().UnixNano() / int64(time.Millisecond)
			return fmt.Sprintf("%d-%d", currentTime, 0), currentTime, 0, nil
		}
		return fmt.Sprintf("%d-%d", millisecondsTime, sequenceNumber), millisecondsTime, sequenceNumber, nil
	}
	
	lastEntryRecord, ok := lastEntry.Value.(*streamRecord)
	if !ok {
		return "", 0, 0, fmt.Errorf("ERR Unexpected type in list element")
	}

	

	var newSequenceNumber int
	var newMillisecondsTime int64



	if halfAutoGenerated {

		if millisecondsTime == lastEntryRecord.millisecondsTime {
			newSequenceNumber = lastEntryRecord.sequenceNumber + 1
		} else {
			newSequenceNumber = 0
		}
		newMillisecondsTime = millisecondsTime
	} else {
		// if not half auto generated, then it is full auto generated

		if millisecondsTime == lastEntryRecord.millisecondsTime {
			newSequenceNumber = lastEntryRecord.sequenceNumber + 1
		} else {
			newSequenceNumber = 0
		}
		newMillisecondsTime = time.Now().UnixNano() / int64(time.Millisecond)
	}



	return fmt.Sprintf("%d-%d", newMillisecondsTime, newSequenceNumber), newMillisecondsTime, newSequenceNumber, nil
}
