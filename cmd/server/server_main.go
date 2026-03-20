package main

import (
	"onion-chat-app/backend"
	"os"
)

func main() {
	backend.RunServer(os.Args[1:])
}
