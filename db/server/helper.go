package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// not sure if this is a good idea to generate client id and use it in other parts of the code. Need to read this but where???
func generateClientID() string {
	buf := make([]byte, 16)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(buf) 
}

func welcomeMessage() {
	fmt.Println(`
╔═══════════════════════════════════════════════╗
║              Redis-From-Scratch               ║
║═══════════════════════════════════════════════║
║Version: 1.0.0                                 ║
║Author: Manish Singh Bisht                     ║
╚═══════════════════════════════════════════════╝
	`)
}