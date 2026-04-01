package backend

import (
	"context"

	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	"net/http"

	"github.com/redis/go-redis/v9"
)

type Message struct {
	From    string `json:"From"`
	FromX   string `json:"FromX"`
	To      string `json:"To"`
	Payload []byte `json:"Payload"`
}

type FetchRequestBody struct {
	RequesterPubKey string `json:"RequesterPubKey"`
	Timestamp       int64  `json:"Timestamp"`
	Signature       string `json:"Signature"`
}

type Server struct {
	ln          net.Listener
	redisClient *redis.Client
	ctx         context.Context
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

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// proof of work captcha
	nonceStr := r.Header.Get("PoWNonce")
	if nonceStr == "" {
		http.Error(w, "Missing PoW nonce header", http.StatusBadRequest)
		return
	}

	nonce, err := strconv.Atoi(nonceStr)
	if err != nil {
		http.Error(w, "Invalid PoW nonce format", http.StatusBadRequest)
		return
	}

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	difficulty := 2
	if !verifyPoW(rawBody, nonce, difficulty) {
		http.Error(w, "Proof of work verification failed", http.StatusForbidden)
		return
	}

	var msg Message
	err = json.Unmarshal(rawBody, &msg)
	if err != nil {
		http.Error(w, "Something is wrong with your JSON", http.StatusBadRequest)
		return
	}

	err = s.recieveAndHold(msg.To, rawBody)
	if err != nil {
		http.Error(w, "Failed to store message. recieveAndHold failed", http.StatusInternalServerError)
		return
	}

	fmt.Println("message sent and waiting for ", msg.To)
	fmt.Println("Message:", string(msg.Payload))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Message queued successfully"))

}

func (s *Server) handleFetch(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var fetchBody FetchRequestBody

	err := json.NewDecoder(r.Body).Decode(&fetchBody)
	if err != nil {
		http.Error(w, "Something is wrong with your JSON", http.StatusBadRequest)
		return
	}

	messages, err := s.fetchAndClear(fetchBody.RequesterPubKey, fetchBody.Timestamp, fetchBody.Signature)
	if err != nil {
		if err.Error() == "authentication failed. invalid signature." {
			fmt.Println("failed to verify identity")
			http.Error(w, "failed to verify identity", http.StatusBadRequest)
		} else {
			fmt.Println("fetchandClear error")
			http.Error(w, "fetchAndClear error", http.StatusBadRequest)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(messages)
	if err != nil {
		log.Printf("jsonify error: %v", err)
	}

}

// to start: redis-server --daemonize yes.
// to terminate: redis-cli shutdown
// to ping: redis-cli ping
// to get stop listening on :8080,
// run sudo lsof -i:8080
// then run kill -9 <PID>
func RunServer(args []string) {
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

	// in the future, server will ONLY be accessible via Tor.
	http.HandleFunc("/send", server.handleSend)
	http.HandleFunc("/fetch", server.handleFetch)

	log.Println("Server listening on :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(ping)
}
