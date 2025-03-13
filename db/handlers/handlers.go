package handlers

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	config "github.com/manish-singh-bisht/Redis-From-Scratch/db/persistence"
	RESP "github.com/manish-singh-bisht/Redis-From-Scratch/db/resp"
	store "github.com/manish-singh-bisht/Redis-From-Scratch/db/store"
)

type commandHandler func(writer *RESP.Writer, args []RESP.RESPMessage) error

var (
	handlers = map[string]commandHandler{
		"PING":   handlePing, // responds with "PONG"
		"ECHO":   handleEcho, // echoes a message, that is return what is passed in
		"SET":    handleSet,  // sets a key to a value, with optional expiration time, updates the value if the key already exists
		"GET":    handleGet,  // gets a value from a key
		"CONFIG": handleConfig,
		// gets the configuration of the server,
		//--------currently only dir and dbfilename are supported--------

		"KEYS": handleKeys,
		// returns all the keys that match the pattern,
		//--------currently only * is supported--------

		"TYPE":   handleType, // returns the type of the key
		"XADD":   handleXAdd, // adds a new entry to a stream, creates a stream if it doesn't exist
		"XRANGE": handleXRange,
		// gets a range of entries from a stream,
		// inclusive of the start and end IDs,
		// takes in start and end IDs as arguments,
		// cannot read from multiple streams
		// --------currently only xrange is supported with + and - --------

		"XREAD": handleXRead,
		// gets a range of entries from a stream
		// that are strictly greater than the start id,
		// exclusive of start id, takes in start id as argument,
		// can also read from multiple streams(this is good when we want to read from multiple streams using just one command)
		// also has blocking options(that is the command is blocked until the given time specified in command and during that time if entries come they will be listened nearly instantly.)
		// --------currently only xread, blocking with and without timeout is supported, $ as id--------
	}
)

/*
 	* handlePing handles the PING command, returns "PONG"
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @return error - the error if there is one
	* @return simple string "PONG"
*/
func handlePing(writer *RESP.Writer, args []RESP.RESPMessage) error {
	return writer.Encode(&RESP.RESPMessage{
		Type:  RESP.SimpleString,
		Value: []byte("PONG"),
	})
}

/*
 	* handleEcho handles the ECHO command, echoes a message
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @return error - the error if there is one
	* @return bulk string - the message to echo
*/
func handleEcho(writer *RESP.Writer, args []RESP.RESPMessage) error {

	if len(args) > 0 {

		return writer.Encode(&RESP.RESPMessage{
			Type:  RESP.BulkString,
			Value: args[0].Value,
			Len:   (args[0].Len),
		})
	}

	return HandleError(writer, []byte("ERR wrong number of arguments for 'ECHO' command"))

}

/*
 	* handleSet handles the SET command, sets a key to a value
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @return error - the error if there is one
	* @return simple string "OK"
*/
func handleSet(writer *RESP.Writer, args []RESP.RESPMessage) error {

	if len(args) < 2 {
		return HandleError(writer, []byte("ERR wrong number of arguments for 'SET' command"))

	}

	key := string(args[0].Value)
	value := args[1].Value

	var expiration time.Duration = 0

	// starting from 2 because 0 and 1 will be key and value respectively
	for i := 2; i < len(args); i++ {
		option := strings.ToUpper(string(args[i].Value))

		switch option {
		case "EX":
			if i+1 < len(args) {
				seconds, err := strconv.Atoi(string(args[i+1].Value))
				if err != nil {
					return HandleError(writer, []byte("ERR invalid expire time"))

				}
				expiration = time.Duration(seconds) * time.Second
				i++ // skip the next item, which will be the "value" for "EX"
			}
		case "PX":
			if i+1 < len(args) {
				milliseconds, err := strconv.Atoi(string(args[i+1].Value))
				if err != nil {
					return HandleError(writer, []byte("ERR invalid expire time"))

				}
				expiration = time.Duration(milliseconds) * time.Millisecond
				i++ // skip the next item, which will be the "value" for "PX"
			}
		case "NX":
			_, exists := store.Store.Get(key)
			if exists {
				return writer.EncodeNil()
			}
		case "XX":
			_, exists := store.Store.Get(key)
			if !exists {
				return writer.EncodeNil()
			}
		default:
			return HandleError(writer, []byte("ERR syntax error"))

		}
	}

	store.Store.Set(key, value, expiration)

	return writer.Encode(&RESP.RESPMessage{
		Type:  RESP.SimpleString,
		Value: []byte("OK"),
	})
}

/*
 	* handleGet handles the GET command, gets a value from a key
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @return error - the error if there is one
	* @return bulk string - the value of the key
*/
func handleGet(writer *RESP.Writer, args []RESP.RESPMessage) error {

	if len(args) < 1 {
		return HandleError(writer, []byte("ERR wrong number of arguments for 'GET' command"))

	}

	key := string(args[0].Value)
	value, exists := store.Store.Get(key)

	if !exists {

		return writer.EncodeNil()
	}

	return writer.Encode(&RESP.RESPMessage{
		Type:  RESP.BulkString,
		Value: value,
		Len:   len(value),
	})
}

/*
 	* handleConfig handles the CONFIG command, gets the configuration of the server
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @return error - the error if there is one
	* @return array - the configuration of the server
*/
func handleConfig(writer *RESP.Writer, args []RESP.RESPMessage) error {

	if len(args) < 2 {
		return HandleError(writer, []byte("ERR wrong number of arguments for 'CONFIG' command"))

	}

	dir, dbFilename := config.GetConfig()
	subCommand := strings.ToUpper(string(args[0].Value))
	parameter := strings.ToLower(string(args[1].Value))

	switch subCommand {
	case "GET":
		var response []RESP.RESPMessage

		switch parameter {
		case "dir":
			response = []RESP.RESPMessage{
				{Type: RESP.BulkString, Len: 3, Value: []byte("dir")},
				{Type: RESP.BulkString, Len: len(dir), Value: []byte(dir)},
			}

		case "dbfilename":
			response = []RESP.RESPMessage{
				{Type: RESP.BulkString, Len: 10, Value: []byte("dbfilename")},
				{Type: RESP.BulkString, Len: len(dbFilename), Value: []byte(dbFilename)},
			}

		default:

			response = []RESP.RESPMessage{}
		}

		return writer.Encode(&RESP.RESPMessage{
			Type:      RESP.Array,
			Len:       len(response),
			ArrayElem: response,
		})

	default:
		return HandleError(writer, []byte("ERR unknown command"))

	}
}

/*
 	* handleKeys returns all the keys that match the pattern
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @return error - the error if there is one
	* @return array - the keys that match the pattern
*/
func handleKeys(writer *RESP.Writer, args []RESP.RESPMessage) error {

	if len(args) != 1 {
		return HandleError(writer, []byte("ERR wrong number of arguments for 'KEYS' command"))
	}

	pattern := string(args[0].Value)
	keys := store.Store.GetKeys(pattern)

	// Create response array
	response := make([]RESP.RESPMessage, len(keys))
	for i, key := range keys {
		response[i] = RESP.RESPMessage{
			Type:  RESP.BulkString,
			Value: []byte(key),
			Len:   len(key),
		}
	}

	return writer.Encode(&RESP.RESPMessage{
		Type:      RESP.Array,
		Len:       len(response),
		ArrayElem: response,
	})
}

/*
 	* handleType returns the type of the key
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @return error - the error if there is one
	* @return simple string - the type of the key
*/
func handleType(writer *RESP.Writer, args []RESP.RESPMessage) error {

	if len(args) != 1 {
		return HandleError(writer, []byte("ERR wrong number of arguments for 'TYPE' command"))
	}

	inputKey := string(args[0].Value)
	keys := store.Store.GetKeys("*")

	var typeOfKey string

	for _, key := range keys {

		if inputKey == key {
			typeOfKey = reflect.TypeOf(key).String()
		}

	}

	if typeOfKey == "" {
		isStream := store.StreamManager.IsStreamKey(inputKey)
		if isStream {
			typeOfKey = "stream"
		}
	}

	if typeOfKey == "" {
		typeOfKey = "none"
	}

	return writer.Encode(&RESP.RESPMessage{
		Type:  RESP.SimpleString,
		Len:   len(typeOfKey),
		Value: []byte(typeOfKey),
	})
}

/*
 	* handleXAdd handles the XADD command, adds a new entry to a stream
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @return error - the error if there is one
	* @return bulk string - the ID of the new entry
*/
func handleXAdd(writer *RESP.Writer, args []RESP.RESPMessage) error {

	// Check minimum required arguments (stream name, ID, and at least one field-value pair)
	if len(args) < 4 {
		return HandleError(writer, []byte("ERR wrong number of arguments for 'XADD' command"))
	}

	streamName := string(args[0].Value)
	id := string(args[1].Value)

	// Validate that we have an even number of field-value pairs
	if (len(args)-2)%2 != 0 {
		return HandleError(writer, []byte("ERR wrong number of arguments for 'XADD' command"))
	}

	dataMap := make(map[string][]byte)
	for i := 2; i < len(args); i += 2 {
		key := string(args[i].Value)
		value := args[i+1].Value
		dataMap[key] = value
	}

	streamRecord, ok, err := store.StreamManager.XAdd(streamName, id, dataMap)

	if err != nil {
		return HandleError(writer, []byte(err.Error()))
	}

	if !ok {
		return HandleError(writer, []byte("ERR failed to add entry to stream"))
	}

	return writer.Encode(&RESP.RESPMessage{
		Type:  RESP.BulkString,
		Len:   len(streamRecord.Id),
		Value: []byte(streamRecord.Id),
	})
}

/*
 	* handleXRange handles the XRANGE command, gets a range of entries from a stream
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @return error - the error if there is one
	* @return array - The actual return value is a RESP Array of arrays. Each inner array represents an entry.The first item in the inner array is the ID of the entry.The second item is a list of key value pairs, where the key value pairs are represented as a list of strings.The key value pairs are in the order they were added to the entry.
*/
func handleXRange(writer *RESP.Writer, args []RESP.RESPMessage) error {
	if len(args) != 3 {
		return HandleError(writer, []byte("ERR wrong number of arguments for 'XRANGE' command"))
	}

	streamName := string(args[0].Value)
	startId := string(args[1].Value)
	endId := string(args[2].Value)

	streamRecords, err := store.StreamManager.XRange(streamName, startId, endId)
	if err != nil {
		return HandleError(writer, []byte(err.Error()))
	}
	// final response
	// [
	//   [
	//     "0-2",
	//     [
	//       "bar",
	//       "baz"
	//     ]
	//   ],
	//   [
	//     "0-3",
	//     [
	//       "baz",
	//       "foo"
	//     ]
	//   ]
	// ]

	entries := store.StreamManager.CreateStreamMessages(streamRecords)
	return writer.Encode(&RESP.RESPMessage{
		Type:      RESP.Array,
		Len:       len(entries),
		ArrayElem: entries,
	})
}

func handleXRead(writer *RESP.Writer, args []RESP.RESPMessage) error {
	if len(args) < 3 {
		return HandleError(writer, []byte("ERR wrong number of arguments for 'XREAD' command"))
	}

	blockMs := -1
	count := -1
	streamStartIdx := -1

	for i := 0; i < len(args); i++ {
		option := strings.ToUpper(string(args[i].Value))
		switch option {
		case "BLOCK":
			if i+1 >= len(args) {
				return HandleError(writer, []byte("ERR syntax error"))
			}
			var err error
			blockMs, err = strconv.Atoi(string(args[i+1].Value))
			if err != nil || blockMs < 0 {
				return HandleError(writer, []byte("ERR invalid BLOCK time"))
			}
			i++ // skip the block value
		case "COUNT":
			if i+1 >= len(args) {
				return HandleError(writer, []byte("ERR syntax error"))
			}
			var err error
			count, err = strconv.Atoi(string(args[i+1].Value))
			if err != nil || count < 0 {
				return HandleError(writer, []byte("ERR invalid COUNT"))
			}
			i++ // skip the count value
		case "STREAMS":
			streamStartIdx = i + 1
			i = len(args) // exit the loop by setting i to length of args
		}
	}

	if streamStartIdx == -1 {
		return HandleError(writer, []byte("ERR syntax error"))
	}

	// calculate number of streams and validate argument count
	// the total args for XREAD will always be odd, and removing the arg which is "STREAMS" and even pair of "COUNT" and "BLOCK", there will just streamName(s) and id(s)
	remainingArgs := len(args) - streamStartIdx
	if remainingArgs%2 != 0 {
		return HandleError(writer, []byte("ERR wrong number of arguments for 'XREAD' command"))
	}
	numStreams := remainingArgs / 2

	// collect stream names and ids
	streamNames := args[streamStartIdx : streamStartIdx+numStreams]
	streamIds := args[streamStartIdx+numStreams:]

	var streamRecords []store.StreamRecord
	var err error

	// Process each stream
	finalResponse := make([]RESP.RESPMessage, 0, numStreams)
	for i := 0; i < numStreams; i++ {
		streamName := string(streamNames[i].Value)
		startId := string(streamIds[i].Value)

		if blockMs >= 0 {
			var noTimeout bool = false
			if blockMs == 0 {
				noTimeout = true
			}
			streamRecords, err = store.StreamManager.XReadBlock(streamName, startId, blockMs, noTimeout)
		} else {
			streamRecords, err = store.StreamManager.XRead(streamName, startId)
		}

		if err != nil {
			fmt.Print(err.Error())
			continue
		}

		// no records found in blocking mode, return nil
		if blockMs > 0 && len(streamRecords) == 0 {
			return writer.EncodeNil()
		}

		if len(streamRecords) == 0 {
			fmt.Printf("stream record length is 0 for stream: %v, id: %v", streamName, startId)
			continue
		}

		// final response
		// [
		//   [
		//     "some_key", this is the stream name
		//     [
		//       [
		//         "1526985054079-0",
		//         [
		//           "temperature",
		//           "37",
		//           "humidity",
		//           "94"
		//         ]
		//       ]
		//     ]
		//   ]
		// ]
		entries := store.StreamManager.CreateStreamMessages(streamRecords)
		streamResponse := RESP.RESPMessage{
			Type: RESP.Array,
			Len:  2,
			ArrayElem: []RESP.RESPMessage{
				{
					Type:  RESP.BulkString,
					Len:   len(streamName),
					Value: []byte(streamName),
				},
				{
					Type:      RESP.Array,
					Len:       len(entries),
					ArrayElem: entries,
				},
			},
		}
		finalResponse = append(finalResponse, streamResponse)
	}

	return writer.Encode(&RESP.RESPMessage{
		Type:      RESP.Array,
		Len:       len(finalResponse),
		ArrayElem: finalResponse,
	})
}

/*
* ExecuteCommand executes a command and returns the response
 */
func ExecuteCommand(writer *RESP.Writer, cmd string, args []RESP.RESPMessage) error {

	// convert command to uppercase for case-insensitive matching
	cmd = strings.ToUpper(cmd)

	handler, exists := handlers[cmd]
	if !exists {
		return HandleError(writer, []byte("ERR unknown command"))
	}

	return handler(writer, args)
}
