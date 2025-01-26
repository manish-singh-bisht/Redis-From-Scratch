package handlers

import (
	"strconv"
	"strings"
	"time"

	RESP "github.com/manish-singh-bisht/Redis-From-Scratch/app/resp"
)

var store = NewKeyValueStore()

type CommandHandler func(writer *RESP.Writer, args []RESP.RESPMessage) error

var Handlers = map[string]CommandHandler{
	"PING": handlePing,
	"ECHO": handleEcho,
	"SET":  handleSet,
	"GET":  handleGet,
}

func handlePing(writer *RESP.Writer, args []RESP.RESPMessage) error {
	return writer.Encode(&RESP.RESPMessage{
		Type:  RESP.SimpleString,
		Value: []byte("PONG"),
	})
}

func handleEcho(writer *RESP.Writer, args []RESP.RESPMessage) error {
	if len(args) > 0 {

		return writer.Encode(&RESP.RESPMessage{
			Type:  RESP.BulkString,
			Value: args[0].Value,
			Len:   len(args[0].Value),
		})
	}

	return writer.Encode(&RESP.RESPMessage{
		Type:  RESP.Error,
		Value: []byte("ERR wrong number of arguments for 'ECHO' command"),
	})
}

func handleSet(writer *RESP.Writer, args []RESP.RESPMessage) error {
	if len(args) < 2 {
		return writer.Encode(&RESP.RESPMessage{
			Type:  RESP.Error,
			Value: []byte("ERR wrong number of arguments for 'SET' command"),
		})
	}

	key := string(args[0].Value)
	value := args[1].Value

	var expiration time.Duration = 0

	for i := 2; i < len(args); i++ {
		option := strings.ToUpper(string(args[i].Value))

		switch option {
		case "EX":
			if i+1 < len(args) {
				seconds, err := strconv.Atoi(string(args[i+1].Value))
				if err != nil {
					return writer.Encode(&RESP.RESPMessage{
						Type:  RESP.Error,
						Value: []byte("ERR invalid expire time"),
					})
				}
				expiration = time.Duration(seconds) * time.Second
				i++
			}
		case "PX":
			if i+1 < len(args) {
				milliseconds, err := strconv.Atoi(string(args[i+1].Value))
				if err != nil {
					return writer.Encode(&RESP.RESPMessage{
						Type:  RESP.Error,
						Value: []byte("ERR invalid expire time"),
					})
				}
				expiration = time.Duration(milliseconds) * time.Millisecond
				i++
			}
		case "NX":
			_, exists := store.Get(key)
			if exists {
				return writer.Encode(&RESP.RESPMessage{
					Type: RESP.BulkString,
					Len:  -1,
				})
			}
		case "XX":
			_, exists := store.Get(key)
			if !exists {
				return writer.Encode(&RESP.RESPMessage{
					Type: RESP.BulkString,
					Len:  -1,
				})
			}
		default:
			return writer.Encode(&RESP.RESPMessage{
				Type:  RESP.Error,
				Value: []byte("ERR syntax error"),
			})
		}
	}

	store.Set(key, value, expiration)

	return writer.Encode(&RESP.RESPMessage{
		Type:  RESP.SimpleString,
		Value: []byte("OK"),
	})
}

func handleGet(writer *RESP.Writer, args []RESP.RESPMessage) error {
	if len(args) < 1 {
		return writer.Encode(&RESP.RESPMessage{
			Type:  RESP.Error,
			Value: []byte("ERR wrong number of arguments for 'GET' command"),
		})
	}

	key := string(args[0].Value)
	value, exists := store.Get(key)

	if !exists {
		return writer.Encode(&RESP.RESPMessage{
			Type: RESP.BulkString,
			Len:  -1,
		})
	}

	return writer.Encode(&RESP.RESPMessage{
		Type:  RESP.BulkString,
		Value: value,
		Len:   len(value),
	})
}

func ExecuteCommand(writer *RESP.Writer, cmd string, args []RESP.RESPMessage) error {
	// convert command to uppercase for case-insensitive matching
	cmd = strings.ToUpper(cmd)

	handler, exists := Handlers[cmd]

	if !exists {
		return writer.Encode(&RESP.RESPMessage{
			Type:  RESP.Error,
			Value: []byte("ERR unknown command"),
		})
	}

	return handler(writer, args)
}
