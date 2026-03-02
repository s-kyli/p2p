package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/redis/go-redis/v9"
)

type Message struct {
	From    string `json:"From"`
	To      string `json:"To"`
	Payload []byte `json:"Payload"`
}

type Server struct {
	ln       net.Listener
	messages map[string][]Message
}

func NewServer() *Server {
	return &Server{
		messages: make(map[string][]Message),
	}
}

// will use redis rpush or set
func (server *Server) recieve(w http.ResponseWriter, r *http.Request) {

}

// will use redis get
func (server *Server) fetch(w http.ResponseWriter, r *http.Request) {

}

// just getting used to redis client for now
func main() {
	//ok
	// server := NewServer()

	os.Setenv("REDIS_ADDR", "localhost:6379")
	os.Setenv("REDIS_PASSWORD", "")
	ctx := context.Background()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	ping, err := redisClient.Ping(ctx).Result()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(ping)

	jsonString1, err := json.Marshal(Message{
		From:    "Alice",
		To:      "Jason",
		Payload: []byte(`{"text": "JASON!! We should hack the pentagon!"}`),
	})

	if err != nil {
		fmt.Println("Failed to marshal:", err.Error())
	}

	jsonString2, err := json.Marshal(Message{
		From:    "Alice",
		To:      "Jason",
		Payload: []byte(`{"text": "Pleaseee? Think about the arguments we can win on war thunder!"}`),
	})

	if err != nil {
		fmt.Println("Failed to marshal:", err.Error())
	}

	jsonString3, err := json.Marshal(Message{
		From:    "Alice",
		To:      "Jason",
		Payload: []byte(`{"text": "Fine! I'll do it myself then"}`),
	})

	if err != nil {
		fmt.Println("Failed to marshal:", err.Error())
	}

	err1 := redisClient.RPush(ctx, "Alice", jsonString1).Err()
	if err1 != nil {
		fmt.Println("Failed to append to redis:", err1.Error())
	}

	err2 := redisClient.RPush(ctx, "Alice", jsonString2).Err()
	if err2 != nil {
		fmt.Println("Failed to append to redis:", err2.Error())
	}

	err3 := redisClient.RPush(ctx, "Alice", jsonString3).Err()
	if err3 != nil {
		fmt.Println("Failed to append to redis:", err3.Error())
	}

	var messages []Message

	length := redisClient.LLen(ctx, "Alice")
	for length.Val() > 0 {
		var msg Message

		rPopResult, err := redisClient.LPop(ctx, "Alice").Result()

		if err != nil {
			panic(err)
		}

		err4 := json.Unmarshal([]byte(rPopResult), &msg)
		if err4 != nil {
			panic(err4)
		}

		messages = append(messages, msg)
		length = redisClient.LLen(ctx, "Alice")
	}

	for _, msg := range messages {
		fmt.Println("Recieved message:")
		fmt.Println("Sender:", msg.From)
		fmt.Println("Receiver:", msg.To)
		var payloadData map[string]interface{}

		err5 := json.Unmarshal(msg.Payload, &payloadData)
		if err5 != nil {
			fmt.Println("error marshalling msg payload")
		}

		fmt.Println("text:", payloadData["text"])
		fmt.Println()

	}

	// var payloadData map[string]interface{}
	// err2 := json.Unmarshal(msg.Payload, &payloadData)
	// if err2 != nil {
	// 	return
	// }

	// fmt.Println(payloadData["text"])

	// fmt.Println("Starting server.go")
	// http.HandleFunc("/msg", server.recieve)
	// http.HandleFunc("/fetch", server.fetch)

	// http.ListenAndServe(":8080", nil)

}
