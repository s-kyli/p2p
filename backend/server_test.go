package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func makeJsonByte(from string, to string, payload string) ([]byte, error) {
	jsonByte, err := json.Marshal(Message{
		From:    from,
		To:      to,
		Payload: []byte(payload),
	})

	if err != nil {
		return nil, err
	}

	return jsonByte, err
}

func getRedisKey(publicKey ed25519.PublicKey) string {
	return hex.EncodeToString(publicKey)
}

func TestReceiveAndFetch(t *testing.T) {
	os.Setenv("REDIS_ADDR", "localhost:6379")
	os.Setenv("REDIS_PASSWORD", "")

	server := NewServer()

	// TESTING REDIS PING
	_, err := server.redisClient.Ping(server.ctx).Result()
	assert.NoError(t, err, "redis client ping should not throw an error")

	jasonEdPub, jasonEdPriv, _ := ed25519.GenerateKey(nil)
	jasonEdPubHex := hex.EncodeToString(jasonEdPub)

	server.redisClient.Del(server.ctx, jasonEdPubHex)

	aliceEdPub, _, _ := ed25519.GenerateKey(rand.Reader)
	aliceEdPubHex := hex.EncodeToString(aliceEdPub)

	// MAKING JSON EXAMPLE FOR TESTING. in production this will be called on client.

	var jsonBytes [][]byte
	jsonByte1, _ := makeJsonByte(aliceEdPubHex, jasonEdPubHex, "JASON!!! We should hack the pentagon!")
	jsonBytes = append(jsonBytes, jsonByte1)

	jsonByte2, _ := makeJsonByte(aliceEdPubHex, jasonEdPubHex, "Pleasee?? Think of all the arguments we could win on war thunder!!")
	jsonBytes = append(jsonBytes, jsonByte2)

	jsonByte3, _ := makeJsonByte(aliceEdPubHex, jasonEdPubHex, "Fine! I'll do it myself then! >:(")
	jsonBytes = append(jsonBytes, jsonByte3)

	// testing receiveAndHold. in production this will be called on server.
	for _, jsonByte := range jsonBytes {
		err = server.recieveAndHold(jasonEdPubHex, jsonByte)
		assert.NoError(t, err, "receiveAndHold should not throw an error")
	}

	timestamp := time.Now().Unix()

	// testing malicious signature
	_, redTeamEdPrivKey, err := ed25519.GenerateKey(rand.Reader)
	redTeamPayload := []byte(fmt.Sprintf("fetch:%s:%d", jasonEdPubHex, timestamp))
	redTeamSignature := ed25519.Sign(redTeamEdPrivKey, redTeamPayload)
	redTeamSignatureHex := hex.EncodeToString(redTeamSignature)

	// also testing valid signature for fetchAndClear
	authPayload := []byte(fmt.Sprintf("fetch:%s:%d", jasonEdPubHex, timestamp))
	validSignature := ed25519.Sign(jasonEdPriv, authPayload)
	validSignatureHex := hex.EncodeToString(validSignature)

	_, err = server.fetchAndClear(jasonEdPubHex, timestamp, redTeamSignatureHex)
	assert.Error(t, err, "malicious signature must return error")

	messages, err := server.fetchAndClear(jasonEdPubHex, timestamp, validSignatureHex)
	assert.NoError(t, err, "fetchAndClear should not throw an error")
	assert.Len(t, messages, 3, "should be 3 messages returned")

	for i, jsonByte := range jsonBytes {
		var unmarshlledMsg Message
		err := json.Unmarshal(jsonByte, &unmarshlledMsg)
		assert.NoError(t, err, "Unmarshal should not throw an error")
		assert.Equal(t, string(unmarshlledMsg.Payload), string(messages[i].Payload),
			"Payload bytes must match")

	}

	queueLength := server.redisClient.LLen(server.ctx, jasonEdPubHex).Val()
	assert.Equal(t, int64(0), queueLength, "redis queue must be 0 after fetching")

	fmt.Println("Messages:")
	for _, msg := range messages {
		fmt.Println("Recieved message:")
		fmt.Println("Sender:", msg.From)
		fmt.Println("Receiver:", msg.To)
		fmt.Println("Payload:", string(msg.Payload))
		fmt.Println()

	}
}
