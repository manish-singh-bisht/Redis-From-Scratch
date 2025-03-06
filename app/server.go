package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	Handlers "github.com/manish-singh-bisht/Redis-From-Scratch/app/handlers"
	config "github.com/manish-singh-bisht/Redis-From-Scratch/app/persistence"
	RESP "github.com/manish-singh-bisht/Redis-From-Scratch/app/resp"
)

var PORT = 6379
var HOST = "0.0.0.0"

func main() {

	dir := flag.String("dir", ".", "RDB file directory") //name defaultValues description
	dbFilename := flag.String("dbfilename", "dump.rdb", "RDB filename")
	flag.Parse()

	// Initialize config
	config.InitConfig(*dir, *dbFilename)

	// Check if RDB file exists and load it
	rdbPath := filepath.Join(*dir, *dbFilename)
	if _, err := os.Stat(rdbPath); err == nil {
		log.Println("Loading RDB file:", rdbPath)
		parser := config.NewRDBParser()
		if err := parser.Parse(rdbPath); err != nil {
			log.Printf("Error loading RDB file: %v\n", err)
		}
	} else {
		log.Println("No RDB file found at:", rdbPath)
	}

	// Start server
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", HOST, PORT))
	if err != nil {
		fmt.Println("Failed to bind to port", PORT)
		os.Exit(1)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		// TODO make an event loop
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := RESP.NewReader(conn)
	writer := RESP.NewWriter(conn)

	for {

		msg, err := reader.Decode()
		if err != nil {
			fmt.Println("Error decoding RESP message:", err)
			Handlers.HandleError(writer, []byte("ERR bad request"))
			return

		}

		if msg.Type != RESP.Array || len(msg.ArrayElem) < 1 {
			Handlers.HandleError(writer, []byte("ERR invalid command format"))
			continue
		}

		cmd := string(msg.ArrayElem[0].Value)
		args := msg.ArrayElem[1:]

		// execute the command
		if err := Handlers.ExecuteCommand(writer, cmd, args); err != nil {
			fmt.Println("Error executing command:", err)
			return
		}

	}
}
