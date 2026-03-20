package main

import (
	"onion-chat-app/client"
	"os"
)

func main() {
	client.RunClient(os.Args[1:])
}
