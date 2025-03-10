package store

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	RESP "github.com/manish-singh-bisht/Redis-From-Scratch/app/resp"
)

type OperationType string

const (
	OperationTypeRange OperationType = "range"
	OperationTypeRead  OperationType = "read"
)

const (
	StreamIDMin      = "0-0"
	StreamIDWildcard = "*"
	RangeQueryStart  = "-"
	RangeQueryEnd    = "+"
)

const (
	AutoGeneratedSeq        = -1
	AutoGeneratedTimeAndSeq = -2
)

type IDValidator func(string) bool

var operationValidators = map[OperationType]IDValidator{
	OperationTypeRange: isRangeQueryMarker,
	OperationTypeRead:  func(id string) bool { return false }, // for no specific checks always return false
}

/**
 * isRangeQueryMarker checks if the id is a range query marker like "-" or "+"
 * @param id string - the ID to check
 * @return bool - true if the ID is a range query marker, false otherwise
 */
func isRangeQueryMarker(id string) bool {
	return id == RangeQueryStart || id == RangeQueryEnd
}

/**
 * isWildcard checks if the id is a wildcard i.e. "*"
 * @param id string - the ID to check
 * @return bool - true if the ID is a wildcard, false otherwise
 */
func isWildcard(id string) bool {
	return id == StreamIDWildcard
}

/**
 * isMinStreamID checks if the id is the minimum stream ID i.e. 0-0
 * @param id string - the ID to check
 * @return bool - true if the ID is the minimum stream ID, false otherwise
 */
func isMinStreamID(id string) bool {
	return id == StreamIDMin
}

/**
 * getCurrentMillisTime returns the current milliseconds time
 * @return int64 - the current milliseconds time
 */
func getCurrentMillisTime() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

/**
 * IsStreamKey checks if a stream key exists
 * @param key string - the key to check
 * @return bool - true if the key exists, false otherwise
 */
func (sm *StreamsManager) IsStreamKey(key string) bool {
	_, exists := sm.Streams[key]
	return exists
}

/**
 * IsValidStreamRecordIdExists checks if a stream record ID exists
 * @param streamName string - the name of the stream
 * @param id string - the ID to check
 * @param typeOfOperation string - the type of operation, "range" or "read" etc
 * @return bool - true if the ID exists, false otherwise
 */
func (sm *StreamsManager) IsValidStreamRecordIdExists(streamName string, id string, op OperationType) (bool, error) {
	if !sm.IsStreamKey(streamName) {
		return false, fmt.Errorf("ERR The stream specified does not exist")
	}

	// some checks for specific operations like range where we need to check the id is a range query marker like "-" or "+"
	validator, exists := operationValidators[op]
	if !exists {
		return false, fmt.Errorf("ERR Unknown operation type for id validator: %v", op)
	}

	if validator(id) {
		return true, nil
	}

	stream := sm.Streams[streamName]
	stream.mu.RLock()
	defer stream.mu.RUnlock()

	_, exists = stream.recordMap[id]
	return exists, nil
}

/**
 * The ID should be greater than the ID of the last entry in the stream.
 * The millisecondsTime part of the ID should be greater than or equal to the millisecondsTime of the last entry.
 * If the millisecondsTime part of the ID is equal to the millisecondsTime of the last entry, the sequenceNumber part of the ID should be greater than the sequenceNumber of the last entry.
 * If the stream is empty, the ID should be greater than 0-0
 */
func (sm *StreamsManager) verifyStreamId(streamName, id string) (bool, error) {
	if !sm.IsStreamKey(streamName) {
		return false, fmt.Errorf("ERR The stream specified does not exist")
	}

	stream := sm.Streams[streamName]

	// check for "0-0"
	if isMinStreamID(id) {
		return false, fmt.Errorf("ERR The ID specified in XADD must be greater than %s", StreamIDMin)
	}

	wildcardNum, msTime, seqNum, err := sm.parseStreamId(id)
	if err != nil {
		return false, err
	}

	// if the id is a wildcard or auto sequence, then it is valid
	if wildcardNum == AutoGeneratedTimeAndSeq || wildcardNum == AutoGeneratedSeq {
		return true, nil
	}

	lastEntry := stream.recordList.Back()

	if lastEntry == nil {

		// if id is greater than 0-0
		if msTime > 0 || (msTime == 0 && seqNum > 0) {
			return true, nil
		}

		return false, fmt.Errorf("ERR The ID specified in XADD must be greater than %s", StreamIDMin)
	}

	lastEntryRecord, ok := lastEntry.Value.(*StreamRecord)
	if !ok {
		return false, fmt.Errorf("ERR Unexpected type in list element")
	}

	if lastEntryRecord.MillisecondsTime > msTime {
		return false, fmt.Errorf("ERR The ID specified in XADD is equal or smaller than the target stream top item")
	}

	if lastEntryRecord.MillisecondsTime == msTime && lastEntryRecord.SequenceNumber >= seqNum {
		return false, fmt.Errorf("ERR The ID specified in XADD is equal or smaller than the target stream top item")
	}

	return true, nil
}

/**
 * parseStreamId parses a stream ID into millisecondsTime and sequenceNumber
 * -1 means that the sequenceNumber is auto generated
 * -2 means that the millisecondsTime and sequenceNumber are auto generated
 * @param id string - the ID to parse
 * @return int - the type of the ID, 0 means that the ID is not auto generated, -1 means that the sequence is auto generated, -2 means that the millisecondsTime and sequenceNumber are auto generated
 * @return int64 - the millisecondsTime part of the ID
 * @return int - the sequenceNumber part of the ID
 * @return error - the error if there is one
 */
func (sm *StreamsManager) parseStreamId(id string) (int, int64, int, error) {

	// check if the id is a wildcard, i.e. id="*"
	if isWildcard(id) {
		return AutoGeneratedTimeAndSeq, 0, 0, nil
	}

	idParts := strings.Split(id, "-")
	if len(idParts) != 2 {
		return 0, 0, 0, fmt.Errorf("ERR Invalid stream ID format")
	}
	// not checking for seqNum wildcard first and parsing the correctness of msTime because we still send the msTime incase of wildcard seqNum
	msTime, err := strconv.ParseInt(idParts[0], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ERR The millisecondsTime part of the ID specified is invalid")
	}

	//  check if the sequenceNumber is a wildcard, i.e. idParts[1]="*", eg: 123-*,
	if isWildcard(idParts[1]) {
		return AutoGeneratedSeq, msTime, 0, nil
	}

	seqNum, err := strconv.Atoi(idParts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ERR The sequenceNumber part of the ID specified is invalid")
	}

	return 0, msTime, seqNum, nil
}

/**
 * generateStreamId generates a new stream ID
 * @param streamName string - the name of the stream
 * @param id string - the ID of the new entry
 * @return string - the new stream ID
 * @return int64 - the millisecondsTime part of the new stream ID
 * @return int - the sequenceNumber part of the new stream ID
 * @return error - the error if there is one
 */
func (sm *StreamsManager) generateStreamId(streamName string, id string) (string, int64, int, error) {

	if !sm.IsStreamKey(streamName) {
		return "", 0, 0, fmt.Errorf("ERR The stream specified does not exist")
	}

	stream := sm.Streams[streamName]
	wildcardNum, msTime, _, err := sm.parseStreamId(id)
	if err != nil {
		return "", 0, 0, err
	}

	lastEntry := stream.recordList.Back()
	if lastEntry == nil {
		return sm.generateFirstId(msTime, wildcardNum)
	}

	lastRecord, ok := lastEntry.Value.(*StreamRecord)
	if !ok {
		return "", 0, 0, fmt.Errorf("ERR Unexpected type in list element")
	}

	return sm.generateNextId(lastRecord, msTime, wildcardNum)
}

/**
 * generateFirstId generates the stream ID for the first entry in the stream
 * @param msTime int64 - the millisecondsTime part of the new stream ID
 * @param wildcardNum int - the type of the ID, 0 means that the ID is not auto generated, -1 means that the sequence is auto generated, -2 means that the millisecondsTime and sequenceNumber are auto generated
 * @return string - the new stream ID
 * @return int64 - the millisecondsTime part of the new stream ID
 * @return int - the sequenceNumber part of the new stream ID
 * @return error - the error if there is one
 */
func (sm *StreamsManager) generateFirstId(msTime int64, wildcardNum int) (string, int64, int, error) {
	if wildcardNum == AutoGeneratedSeq && msTime > 0 {
		// If the seqNum is auto-generated and msTIme is provided,
		// generate an ID with seqNum 0.
		return fmt.Sprintf("%d-0", msTime), msTime, 0, nil
	} else if msTime == 0 && wildcardNum == AutoGeneratedSeq {
		// If both msTIme is 0,
		// default to 0 msTIme and seqNum 1.
		return fmt.Sprintf("%d-1", msTime), msTime, 1, nil
	} else if wildcardNum == AutoGeneratedTimeAndSeq {
		// If wildcardNum indicates automatic time and seqNum generation,
		// use the current system time and start with seqNum 0.
		currentTime := getCurrentMillisTime()
		return fmt.Sprintf("%d-0", currentTime), currentTime, 0, nil
	}

	return "", 0, 0, fmt.Errorf("ERR Invalid stream ID format")
}

/**
 * generateNextId generates the stream ID for the next entry in the stream
 * @param lastRecord *StreamRecord - the last entry in the stream
 * @param msTime int64 - the millisecondsTime part of the new stream ID
 * @param wildcardNum int - the type of the ID, 0 means that the ID is not auto generated, -1 means that the sequence is auto generated, -2 means that the millisecondsTime and sequenceNumber are auto generated
 * @return string - the new stream ID
 * @return int64 - the millisecondsTime part of the new stream ID
 * @return int - the sequenceNumber part of the new stream ID
 * @return error - the error if there is one
 */
func (sm *StreamsManager) generateNextId(lastRecord *StreamRecord, msTime int64, wildcardNum int) (string, int64, int, error) {
	var newSeqNum int
	var newMsTime int64

	if wildcardNum == AutoGeneratedSeq {

		if msTime == lastRecord.MillisecondsTime {
			newSeqNum = lastRecord.SequenceNumber + 1
		} else {
			newSeqNum = 0
		}
		newMsTime = msTime
	} else {

		if msTime == lastRecord.MillisecondsTime {
			newSeqNum = lastRecord.SequenceNumber + 1
		} else {
			newSeqNum = 0
		}
		newMsTime = time.Now().UnixNano() / int64(time.Millisecond)
	}

	return fmt.Sprintf("%d-%d", newMsTime, newSeqNum), newMsTime, newSeqNum, nil
}

/*
 	* createStreamMessages converts stream records to RESP messages
	* @param records []store.StreamRecord - the stream records to convert
	* @return []RESP.RESPMessage - the RESP messages
*/
func (sm *StreamsManager) CreateStreamMessages(records []StreamRecord) []RESP.RESPMessage {
	entries := make([]RESP.RESPMessage, len(records))
	for i, record := range records {
		// Create key-value pairs array
		kvPairs := make([]RESP.RESPMessage, 0, len(record.Data)*2)
		for key, value := range record.Data {
			kvPairs = append(kvPairs,
				RESP.RESPMessage{
					Type:  RESP.BulkString,
					Len:   len(key),
					Value: []byte(key),
				},
				RESP.RESPMessage{
					Type:  RESP.BulkString,
					Len:   len(value),
					Value: value,
				},
			)
		}

		entries[i] = RESP.RESPMessage{
			Type: RESP.Array,
			Len:  2,
			ArrayElem: []RESP.RESPMessage{
				{
					Type:  RESP.BulkString,
					Len:   len(record.Id),
					Value: []byte(record.Id),
				},
				{
					Type:      RESP.Array,
					Len:       len(kvPairs),
					ArrayElem: kvPairs,
				},
			},
		}
	}
	return entries
}
