package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	config "github.com/manish-singh-bisht/Redis-From-Scratch/app/config"
	Handlers "github.com/manish-singh-bisht/Redis-From-Scratch/app/handlers"
	RESP "github.com/manish-singh-bisht/Redis-From-Scratch/app/resp"
)

func main() {

	dir := flag.String("dir", ".", "RDB file directory") //name defaultValues description
	dbFilename := flag.String("dbfilename", "dump.rdb", "RDB filename")
	flag.Parse()

	config.InitConfig(*dir, *dbFilename)

	ln, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		// TODO make an event loop, use multi-plexing ,don't just do sync flow for async tasks.
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
