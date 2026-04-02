package backend

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

func verifyPoW(message []byte, nonce int, difficulty int) bool {
	data := append(message, []byte(strconv.Itoa(nonce))...)
	hash := sha256.Sum256(data)

	for i := 0; i < difficulty; i++ {
		if hash[i] != 0 {
			return false
		}
	}
	return true
}

func verifyIdentity(requesterHex string, timestamp int64, signatureHex string) bool {

	publicKeyDecoded, err := hex.DecodeString(requesterHex)
	if err != nil || len(publicKeyDecoded) != ed25519.PublicKeySize {
		return false
	}

	// stop replay attacks
	if time.Now().Unix()-timestamp > 30 {
		return false
	}
	if timestamp-time.Now().Unix() > 5 {
		return false
	}

	publicKey := ed25519.PublicKey(publicKeyDecoded)
	signatureDecoded, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}
	//[]byte(fmt.Sprintf("fetch:%s:%d", jasonEdPubHex, timestamp)) <--matches this signed from client
	message := []byte(fmt.Sprintf("fetch:%s:%d", requesterHex, timestamp))
	return ed25519.Verify(publicKey, message, signatureDecoded)
}

// w http.ResponseWriter, r *http.Request
// will use redis RPUSH
// caches incoming messages for when the receiver wants to get the message.
func (server *Server) recieveAndHold(recipientPubKey string, jsonString []byte) error {

	err := server.redisClient.RPush(server.ctx, recipientPubKey, jsonString).Err()
	if err != nil {
		return err
	}

	err = server.redisClient.Expire(server.ctx, recipientPubKey, TTLDuration).Err()
	if err != nil {
		return err
	}

	return nil

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
		fmt.Println("Error getting messages from redis:", err)
		return nil, err
	}

	if len(results) == 0 {
		return []Message{}, nil
	}

	err = server.redisClient.Del(server.ctx, requesterPubKey).Err()
	if err != nil {
		fmt.Println("Error deleting messages:", err)
		return nil, err
	}

	var messages []Message

	for _, res := range results {
		var msg Message
		err := json.Unmarshal([]byte(res), &msg)
		if err != nil {
			fmt.Println("Error unmarshalling:", err)
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}
