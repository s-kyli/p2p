package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/redis/go-redis/v9"
)

type Message struct {
	From    string `json:"From"`    // names for rn, but soon will be ed25519 public key.
	To      string `json:"To"`      // same for this.
	Payload []byte `json:"Payload"` // not encrypted for now, but will be e2ee later.
}

type Server struct {
	ln          net.Listener
	redisClient *redis.Client
	ctx         context.Context
}

func verifyIdentity(requesterID string, timestamp int64, signature string) bool {

	if signature == requesterID+"-Secret" {
		return true
	}

	return false
}

func NewServer() *Server {
	return &Server{
		redisClient: redis.NewClient(&redis.Options{
			Addr:     os.Getenv("REDIS_ADDR"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       0,
		}),
		ctx: context.Background(),
	}
}

// w http.ResponseWriter, r *http.Request
// will use redis RPUSH
// caches incoming messages for when the receiver wants to get the message.
func (server *Server) recieveAndHold(recipientPubKey string, jsonString []byte) error {
	return server.redisClient.RPush(server.ctx, recipientPubKey, jsonString).Err()
}

// will use redis LPOPs
// will verify identity of the fetcher, (obviously not like this in final product)
// then return the messages

func (server *Server) fetchAndClear(requesterPubKey string, timestamp int64, signature string) ([]Message, error) {

	if !verifyIdentity(requesterPubKey, timestamp, signature) {
		return nil, fmt.Errorf("authentication failed. invalid signature.")
	}

	results, err := server.redisClient.LRange(server.ctx, requesterPubKey, 0, -1).Result()

	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return []Message{}, nil
	}

	err = server.redisClient.Del(server.ctx, requesterPubKey).Err()
	if err != nil {
		return nil, err
	}

	var messages []Message

	for _, res := range results {
		var msg Message

		err := json.Unmarshal([]byte(res), &msg)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// to start: redis-server --daemonize yes.
// to terminate: redis-cli shutdown
// to ping: redis-cli ping
// just getting used to redis client for now
func main() {

	//INITIALIZING NEW SERVER

	os.Setenv("REDIS_ADDR", "localhost:6379")
	os.Setenv("REDIS_PASSWORD", "")

	server := NewServer()

	// TESTING REDIS PING
	ping, err := server.redisClient.Ping(server.ctx).Result()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(ping)

	// adding http requests later

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
