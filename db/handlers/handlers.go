package handlers

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	config "github.com/manish-singh-bisht/Redis-From-Scratch/db/persistence"
	RESP "github.com/manish-singh-bisht/Redis-From-Scratch/db/resp"
	store "github.com/manish-singh-bisht/Redis-From-Scratch/db/store"
	tx "github.com/manish-singh-bisht/Redis-From-Scratch/db/transaction"
)

// passing clientID and txManager to the handler because we need for transaction related commands, other than that we are not using them, so is it a good practice?? Not sure!!
type commandHandler func(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error

/**
 * getHandlers returns the handler for the given command
 * @param cmd string - the command to get the handler for
 * @return commandHandler - the handler for the given command
 * @return bool - true if the handler was found, false otherwise
 */
func getHandlers(cmd string) (commandHandler, bool) {
	handlers := map[string]commandHandler{
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

		"INCR": handleIncr, // increments the value of a key, value is integer, by 1
		"EXIT": handleExit,

		"MULTI": handleMulti,
		// starts a transaction
		// all the commands after MULTI will be queued up and executed atomically i.e as a single unit

		"EXEC":    handleExec,    // executes a transaction
		"DISCARD": handleDiscard, // discards a transaction
		"WATCH":   handleWatch,
		// watches a key for changes
		// if the key is changed, the transaction is discarded
		// if the key is not changed, the transaction is executed
		// keys after EXEC, whether properly executed or not, are not watched
	}
	handler, exists := handlers[cmd]

	return handler, exists
}

/**
 * handlePing handles the PING command, responds with "PONG"
 * @param writer *RESP.Writer - the writer to write the response to
 * @param args []RESP.RESPMessage - the arguments for the command
 * @param store *store.Store - the store to get the data from
 * @param clientID string - the client id
 * @param txManager *tx.TxManager - the transaction manager
 * @return error - the error if there is one
 */
func handlePing(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {
	return writer.Encode(&RESP.RESPMessage{
		RESPType:  RESP.SimpleString,
		RESPValue: []byte("PONG"),
	})
}

/*
 	* handleEcho handles the ECHO command, echoes a message
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @param store *store.Store - the store to get the data from
	* @param clientID string - the client id
	* @param txManager *tx.TxManager - the transaction manager
	* @return error - the error if there is one
	* @return bulk string - the message to echo
*/
func handleEcho(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {

	if len(args) > 0 {

		return writer.Encode(&RESP.RESPMessage{
			RESPType:  RESP.BulkString,
			RESPValue: args[0].RESPValue,
			RESPLen:   (args[0].RESPLen),
		})
	}
	err := errWrongNumberOfArguments("ECHO")
	return HandleError(writer, []byte(err.Error()))

}

/*
 	* handleSet handles the SET command, sets a key to a value
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @param store *store.Store - the store to get the data from
	* @param clientID string - the client id
	* @param txManager *tx.TxManager - the transaction manager
	* @return error - the error if there is one
	* @return simple string "OK"
*/
func handleSet(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {

	if len(args) < 2 {
		err := errWrongNumberOfArguments("SET")
		return HandleError(writer, []byte(err.Error()))

	}

	key := string(args[0].RESPValue)
	value := args[1].RESPValue

	var expiration time.Duration = 0

	// starting from 2 because 0 and 1 will be key and value respectively
	for i := 2; i < len(args); i++ {
		option := strings.ToUpper(string(args[i].RESPValue))

		switch option {
		case "EX":
			if i+1 < len(args) {
				seconds, err := strconv.Atoi(string(args[i+1].RESPValue))
				if err != nil {
					return HandleError(writer, []byte("ERR invalid expire time"))

				}
				expiration = time.Duration(seconds) * time.Second
				i++ // skip the next item, which will be the "value" for "EX"
			}
		case "PX":
			if i+1 < len(args) {
				milliseconds, err := strconv.Atoi(string(args[i+1].RESPValue))
				if err != nil {
					return HandleError(writer, []byte("ERR invalid expire time"))

				}
				expiration = time.Duration(milliseconds) * time.Millisecond
				i++ // skip the next item, which will be the "value" for "PX"
			}
		case "NX":
			_, exists := store.Get(key)
			if exists {
				return writer.EncodeNil()
			}
		case "XX":
			_, exists := store.Get(key)
			if !exists {
				return writer.EncodeNil()
			}
		default:
			return HandleError(writer, []byte("ERR syntax error"))

		}
	}

	store.Set(key, value, expiration)

	// if the key is being watched, update the key's global version
	_, exists := txManager.GetGlobalKeyVersions(key)
	if exists {
		txManager.UpdateGlobalKeyVersionsMap(key)
	}
	return writer.Encode(&RESP.RESPMessage{
		RESPType:  RESP.SimpleString,
		RESPValue: []byte("OK"),
	})
}

/*
 	* handleGet handles the GET command, gets a value from a key
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @param store *store.Store - the store to get the data from
	* @param clientID string - the client id
	* @param txManager *tx.TxManager - the transaction manager
	* @return error - the error if there is one
	* @return bulk string - the value of the key
*/
func handleGet(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {

	if len(args) < 1 {
		err := errWrongNumberOfArguments("GET")
		return HandleError(writer, []byte(err.Error()))

	}

	key := string(args[0].RESPValue)
	value, exists := store.Get(key)

	if !exists {

		return writer.EncodeNil()
	}

	return writer.Encode(&RESP.RESPMessage{
		RESPType:  RESP.BulkString,
		RESPValue: value,
		RESPLen:   len(value),
	})
}

/*
 	* handleConfig handles the CONFIG command, gets the configuration of the server
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @param store *store.Store - the store to get the data from
	* @param clientID string - the client id
	* @param txManager *tx.TxManager - the transaction manager
	* @return error - the error if there is one
	* @return array - the configuration of the server
*/
func handleConfig(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {

	if len(args) < 2 {
		err := errWrongNumberOfArguments("CONFIG")
		return HandleError(writer, []byte(err.Error()))

	}

	dir, dbFilename := config.GetConfig()
	subCommand := strings.ToUpper(string(args[0].RESPValue))
	parameter := strings.ToLower(string(args[1].RESPValue))

	switch subCommand {
	case "GET":
		var response []RESP.RESPMessage

		switch parameter {
		case "dir":
			response = []RESP.RESPMessage{
				{RESPType: RESP.BulkString, RESPLen: 3, RESPValue: []byte("dir")},
				{RESPType: RESP.BulkString, RESPLen: len(dir), RESPValue: []byte(dir)},
			}

		case "dbfilename":
			response = []RESP.RESPMessage{
				{RESPType: RESP.BulkString, RESPLen: 10, RESPValue: []byte("dbfilename")},
				{RESPType: RESP.BulkString, RESPLen: len(dbFilename), RESPValue: []byte(dbFilename)},
			}

		default:

			response = []RESP.RESPMessage{}
		}

		return writer.Encode(&RESP.RESPMessage{
			RESPType:      RESP.Array,
			RESPLen:       len(response),
			RESPArrayElem: response,
		})

	default:
		return HandleError(writer, []byte("ERR unknown command"))

	}
}

/*
 	* handleKeys returns all the keys that match the pattern
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @param store *store.Store - the store to get the data from
	* @param clientID string - the client id
	* @param txManager *tx.TxManager - the transaction manager
	* @return error - the error if there is one
	* @return array - the keys that match the pattern
*/
func handleKeys(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {

	if len(args) != 1 {
		err := errWrongNumberOfArguments("KEYS")
		return HandleError(writer, []byte(err.Error()))
	}

	pattern := string(args[0].RESPValue)
	keys := store.GetKeys(pattern)

	response := make([]RESP.RESPMessage, len(keys))
	for i, key := range keys {
		response[i] = RESP.RESPMessage{
			RESPType:  RESP.BulkString,
			RESPValue: []byte(key),
			RESPLen:   len(key),
		}
	}

	return writer.Encode(&RESP.RESPMessage{
		RESPType:      RESP.Array,
		RESPLen:       len(response),
		RESPArrayElem: response,
	})
}

/*
 	* handleType returns the type of the key
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @param store *store.Store - the store to get the data from
	* @param clientID string - the client id
	* @param txManager *tx.TxManager - the transaction manager
	* @return error - the error if there is one
	* @return simple string - the type of the key
*/
func handleType(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {

	if len(args) != 1 {
		err := errWrongNumberOfArguments("TYPE")
		return HandleError(writer, []byte(err.Error()))
	}

	inputKey := string(args[0].RESPValue)
	keys := store.GetKeys("*")

	var typeOfKey string

	for _, key := range keys {

		if inputKey == key {
			typeOfKey = reflect.TypeOf(key).String()
		}

	}

	if typeOfKey == "" {
		isStream := store.IsStreamKey(inputKey)
		if isStream {
			typeOfKey = "stream"
		}
	}

	if typeOfKey == "" {
		typeOfKey = "none"
	}

	return writer.Encode(&RESP.RESPMessage{
		RESPType:  RESP.SimpleString,
		RESPLen:   len(typeOfKey),
		RESPValue: []byte(typeOfKey),
	})
}

/*
 	* handleXAdd handles the XADD command, adds a new entry to a stream
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @param store *store.Store - the store to get the data from
	* @param clientID string - the client id
	* @param txManager *tx.TxManager - the transaction manager
	* @return error - the error if there is one
	* @return bulk string - the ID of the new entry
*/
func handleXAdd(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {

	// Check minimum required arguments (stream name, ID, and at least one field-value pair)
	if len(args) < 4 {
		err := errWrongNumberOfArguments("XADD")
		return HandleError(writer, []byte(err.Error()))
	}

	streamName := string(args[0].RESPValue)
	id := string(args[1].RESPValue)

	// Validate that we have an even number of field-value pairs
	if (len(args)-2)%2 != 0 {
		err := errWrongNumberOfArguments("XADD")
		return HandleError(writer, []byte(err.Error()))
	}

	dataMap := make(map[string][]byte)
	for i := 2; i < len(args); i += 2 {
		key := string(args[i].RESPValue)
		value := args[i+1].RESPValue
		dataMap[key] = value
	}

	streamRecord, ok, err := store.XAdd(streamName, id, dataMap)

	if err != nil {
		return HandleError(writer, []byte(err.Error()))
	}

	if !ok {
		return HandleError(writer, []byte("ERR failed to add entry to stream"))
	}

	return writer.Encode(&RESP.RESPMessage{
		RESPType:  RESP.BulkString,
		RESPLen:   len(streamRecord.Id),
		RESPValue: []byte(streamRecord.Id),
	})
}

/*
 	* handleXRange handles the XRANGE command, gets a range of entries from a stream
	* @param writer *RESP.Writer - the writer to write the response to
	* @param args []RESP.RESPMessage - the arguments for the command
	* @param store *store.Store - the store to get the data from
	* @param clientID string - the client id
	* @param txManager *tx.TxManager - the transaction manager
	* @return error - the error if there is one
	* @return array - The actual return value is a RESP Array of arrays. Each inner array represents an entry.The first item in the inner array is the ID of the entry.The second item is a list of key value pairs, where the key value pairs are represented as a list of strings.The key value pairs are in the order they were added to the entry.
*/
func handleXRange(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {
	if len(args) != 3 {
		err := errWrongNumberOfArguments("XRANGE")
		return HandleError(writer, []byte(err.Error()))
	}

	streamName := string(args[0].RESPValue)
	startId := string(args[1].RESPValue)
	endId := string(args[2].RESPValue)

	streamRecords, err := store.XRange(streamName, startId, endId)
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

	entries := store.CreateStreamMessages(streamRecords)
	return writer.Encode(&RESP.RESPMessage{
		RESPType:      RESP.Array,
		RESPLen:       len(entries),
		RESPArrayElem: entries,
	})
}

func handleXRead(writer *RESP.Writer, args []RESP.RESPMessage, streamStore *store.Store, clientID string, txManager *tx.TxManager) error {
	if len(args) < 3 {
		err := errWrongNumberOfArguments("XREAD")
		return HandleError(writer, []byte(err.Error()))
	}

	blockMs := -1
	count := -1
	streamStartIdx := -1

	for i := 0; i < len(args); i++ {
		option := strings.ToUpper(string(args[i].RESPValue))
		switch option {
		case "BLOCK":
			if i+1 >= len(args) {
				return HandleError(writer, []byte("ERR syntax error"))
			}
			var err error
			blockMs, err = strconv.Atoi(string(args[i+1].RESPValue))
			if err != nil || blockMs < 0 {
				return HandleError(writer, []byte("ERR invalid BLOCK time"))
			}
			i++ // skip the block value
		case "COUNT":
			if i+1 >= len(args) {
				return HandleError(writer, []byte("ERR syntax error"))
			}
			var err error
			count, err = strconv.Atoi(string(args[i+1].RESPValue))
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
		err := errWrongNumberOfArguments("XREAD")
		return HandleError(writer, []byte(err.Error()))
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
		streamName := string(streamNames[i].RESPValue)
		startId := string(streamIds[i].RESPValue)

		if blockMs >= 0 {
			var noTimeout bool = false
			if blockMs == 0 {
				noTimeout = true
			}
			streamRecords, err = streamStore.XReadBlock(streamName, startId, blockMs, noTimeout)
		} else {
			streamRecords, err = streamStore.XRead(streamName, startId)
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

		entries := streamStore.CreateStreamMessages(streamRecords)
		streamResponse := RESP.RESPMessage{
			RESPType: RESP.Array,
			RESPLen:  2,
			RESPArrayElem: []RESP.RESPMessage{
				{
					RESPType:  RESP.BulkString,
					RESPLen:   len(streamName),
					RESPValue: []byte(streamName),
				},
				{
					RESPType:      RESP.Array,
					RESPLen:       len(entries),
					RESPArrayElem: entries,
				},
			},
		}
		finalResponse = append(finalResponse, streamResponse)
	}

	return writer.Encode(&RESP.RESPMessage{
		RESPType:      RESP.Array,
		RESPLen:       len(finalResponse),
		RESPArrayElem: finalResponse,
	})
}

func handleIncr(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {
	if len(args) != 1 {
		err := errWrongNumberOfArguments("INCR")
		return HandleError(writer, []byte(err.Error()))
	}

	key := string(args[0].RESPValue)
	value, exists := store.Get(key)

	// if the key is being watched, update the key's global version
	_, globalVersionExists := txManager.GetGlobalKeyVersions(key)
	if globalVersionExists {
		txManager.UpdateGlobalKeyVersionsMap(key)
	}

	var newValue int
	if !exists {
		// if the key doesn't exist, set to 1
		store.Set(key, []byte("1"), 0)
		return writer.Encode(&RESP.RESPMessage{
			RESPType:  RESP.Integer,
			RESPValue: []byte("1"),
		})
	}

	// if the existing value is an integer or not
	currentValue, err := strconv.Atoi(string(value))
	if err != nil {
		return HandleError(writer, []byte("ERR value is not an integer or out of range"))
	}

	newValue = currentValue + 1
	store.Set(key, []byte(strconv.Itoa(newValue)), 0)

	return writer.Encode(&RESP.RESPMessage{
		RESPType:  RESP.Integer,
		RESPValue: []byte(strconv.Itoa(newValue)),
	})
}

/*
* handleExit handles the EXIT command, exits the server
 */
func handleExit(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {
	// Signal to close the connection
	return ErrClientClosed
}

/**
 * handleMulti handles the MULTI command, starts a transaction
 * @param writer *RESP.Writer - the writer to write the response to
 * @param args []RESP.RESPMessage - the arguments for the command
 * @param store *store.Store - the store to get the data from
 * @param clientID string - the client id
 * @param txManager *tx.TxManager - the transaction manager
 * @return error - the error if there is one
 */
func handleMulti(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {
	err := txManager.Multi(clientID)
	if err != nil {
		return HandleError(writer, []byte(err.Error()))
	}
	return writer.Encode(&RESP.RESPMessage{
		RESPType:  RESP.SimpleString,
		RESPValue: []byte("OK"),
	})
}

/**
 * handleExec handles the EXEC command, executes a transaction
 * @param writer *RESP.Writer - the writer to write the response to
 * @param args []RESP.RESPMessage - the arguments for the command
 * @param store *store.Store - the store to get the data from
 * @param clientID string - the client id
 * @param txManager *tx.TxManager - the transaction manager
 * @return error - the error if there is one
 */
func handleExec(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {
	commands, err := txManager.Exec(clientID)
	if err != nil {
		return HandleError(writer, []byte(err.Error()))
	}

	if len(commands) == 0 {
		return writer.Encode(&RESP.RESPMessage{
			RESPType:      RESP.Array,
			RESPLen:       0,
			RESPArrayElem: []RESP.RESPMessage{},
		})
	}

	responses := make([]RESP.RESPMessage, 0, len(commands))

	// execute each command in the transaction and collect responses
	for _, command := range commands {
		cmd := strings.ToUpper(string(command.Cmd.RESPValue))

		handler, exists := getHandlers(cmd)
		if !exists {
			return HandleError(writer, []byte(fmt.Sprintf("ERR unknown command '%s'", cmd)))
		}

		var respBuf bytes.Buffer
		tempWriter := RESP.NewWriter(&respBuf)

		err = handler(tempWriter, command.Args, store, clientID, txManager)
		if err != nil {
			return HandleError(writer, []byte(err.Error()))
		}

		// Decode the response from the buffer
		reader := RESP.NewReader(&respBuf)
		resp, err := reader.Decode()
		if err != nil {
			return HandleError(writer, []byte("ERR failed to decode response"))
		}

		responses = append(responses, *resp)
	}

	return writer.Encode(&RESP.RESPMessage{
		RESPType:      RESP.Array,
		RESPLen:       len(responses),
		RESPArrayElem: responses,
	})
}

/**
 * handleDiscard handles the DISCARD command, discards a transaction
 * @param writer *RESP.Writer - the writer to write the response to
 * @param args []RESP.RESPMessage - the arguments for the command
 * @param store *store.Store - the store to get the data from
 * @param clientID string - the client id
 * @param txManager *tx.TxManager - the transaction manager
 * @return error - the error if there is one
 */
func handleDiscard(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {
	err := txManager.Discard(clientID)
	if err != nil {
		return HandleError(writer, []byte(err.Error()))
	}
	return writer.Encode(&RESP.RESPMessage{
		RESPType:  RESP.SimpleString,
		RESPValue: []byte("OK"),
	})

}

/**
 * handleWatch handles the WATCH command, sets keys to be watched for transaction, CAS
 * @param writer *RESP.Writer - the writer to write the response to
 * @param args []RESP.RESPMessage - the arguments for the command, each representing a key to watch
 * @param store *store.Store - the store to get the data from (unused in this function)
 * @param clientID string - the client id
 * @param txManager *tx.TxManager - the transaction manager
 * @return error - the error if there is one
 */
func handleWatch(writer *RESP.Writer, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {
	if len(args) < 1 {
		err := errWrongNumberOfArguments("WATCH")
		return HandleError(writer, []byte(err.Error()))
	}
	for _, arg := range args {
		key := string(arg.RESPValue)
		txManager.Watch(clientID, key)
	}
	return writer.Encode(&RESP.RESPMessage{
		RESPType:  RESP.SimpleString,
		RESPValue: []byte("OK"),
	})
}

/**
 * ExecuteCommand executes a command and returns the response
 * @param writer *RESP.Writer - the writer to write the response to
 * @param cmd string - the command to execute
 * @param args []RESP.RESPMessage - the arguments for the command
 * @param store *store.Store - the store to get the data from
 * @param clientID string - the client id
 * @param txManager *tx.TxManager - the transaction manager
 * @return error - the error if there is one
 */
func ExecuteCommand(writer *RESP.Writer, cmd string, args []RESP.RESPMessage, store *store.Store, clientID string, txManager *tx.TxManager) error {

	// convert command to uppercase for case-insensitive matching
	cmd = strings.ToUpper(cmd)

	handler, exists := getHandlers(cmd)
	if !exists {
		log.Printf("cmd:%v, does not exist", cmd)
		return HandleError(writer, []byte("ERR unknown command"))
	}

	// if in MULTI, queue commands except for transaction-related ones
	if cmd != "MULTI" && cmd != "EXEC" && cmd != "DISCARD" && cmd != "WATCH" && cmd != "UNWATCH" {

		err := txManager.Queue(clientID, RESP.RESPMessage{
			RESPType:  RESP.BulkString,
			RESPValue: []byte(cmd),
		}, args)

		// if there is no MULTI, then there will be an error and in that case other commands will be executed.
		if err == nil {
			log.Printf("cmd:%v, queued", cmd)
			return writer.Encode(&RESP.RESPMessage{
				RESPType:  RESP.SimpleString,
				RESPValue: []byte("QUEUED"),
			})
		}
	}

	return handler(writer, args, store, clientID, txManager)
}
