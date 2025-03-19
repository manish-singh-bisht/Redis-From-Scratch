package db

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	Handlers "github.com/manish-singh-bisht/Redis-From-Scratch/db/handlers"
	config "github.com/manish-singh-bisht/Redis-From-Scratch/db/persistence"
	RESP "github.com/manish-singh-bisht/Redis-From-Scratch/db/resp"
	store "github.com/manish-singh-bisht/Redis-From-Scratch/db/store"
)


type RedisServer struct {
	host string
	port int
	store    *store.KeyValueStore  
	streams  *store.StreamsManager
	listener net.Listener
}

func NewRedisServer(host string, port int) *RedisServer {
	return &RedisServer{
		host:    host,
		port:    port,
		store:   store.GetStore(),
		streams: store.GetStreamManager(),
	}
}

func (redisServer *RedisServer) Start(dir, dbFilename string) error {
	config.InitConfig(dir, dbFilename)

	// Load RDB file if exists
	rdbPath := filepath.Join(dir, dbFilename)

	if _, err := os.Stat(rdbPath); err == nil {
		log.Println("Loading RDB file:", rdbPath)

		parser := config.NewRDBParser()
		if err := parser.Parse(rdbPath); err != nil {
			log.Printf("Error loading RDB file: %v\n", err)
		}

	} else {
		log.Println("No RDB file found at:", rdbPath)
	}

	var err error
	redisServer.listener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", redisServer.host, redisServer.port))
	if err != nil {
		return fmt.Errorf("failed to bind to port %d: %v", redisServer.port, err)
	}

	for {
		conn, err := redisServer.listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go redisServer.handleConnection(conn)
	}
}

func (redisServer *RedisServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := RESP.NewReader(conn)
	writer := RESP.NewWriter(conn)

	for {
		msg, err := reader.Decode()
		if err != nil {
			log.Printf("Error decoding RESP message: %v", err)
			Handlers.HandleError(writer, []byte("ERR bad request"))
			return
		}

		if msg.Type != RESP.Array || len(msg.ArrayElem) < 1 {
			Handlers.HandleError(writer, []byte("ERR invalid command format"))
			continue
		}

		cmd := string(msg.ArrayElem[0].Value)
		args := msg.ArrayElem[1:]

		if err := Handlers.ExecuteCommand(writer, cmd, args); err != nil {
			log.Printf("Error executing command: %v", err)
			if strings.ToUpper(cmd) == "EXIT" {
				return 
			}
			return
		}
	}
}

const (
	HOST = "0.0.0.0"
	PORT = 6379
)

func DbStart() {
	dir := flag.String("dir", ".", "RDB file directory")
	dbFilename := flag.String("dbfilename", "dump.rdb", "RDB filename")
	flag.Parse()

	server := NewRedisServer(HOST, PORT)
	if err := server.Start(*dir, *dbFilename); err != nil {
		log.Fatalf("Failed to start Redis server: %v", err)
	}
}
