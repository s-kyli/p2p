package main

import (
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

func TestReceiveAndFetch(t *testing.T) {
	os.Setenv("REDIS_ADDR", "localhost:6379")
	os.Setenv("REDIS_PASSWORD", "")

	server := NewServer()

	// TESTING REDIS PING
	_, err := server.redisClient.Ping(server.ctx).Result()
	assert.NoError(t, err, "redis client ping should not throw an error")

	testPubKey := "Jason"
	timestamp := time.Now().Unix()
	server.redisClient.Del(server.ctx, testPubKey)

	// MAKING JSON EXAMPLE FOR TESTING
	var jsonBytes [][]byte
	jsonByte1, err := makeJsonByte("Alice", testPubKey, "JASON!!! We should hack the pentagon!")
	assert.NoError(t, err, "makeJsonByte should not throw an error")
	jsonBytes = append(jsonBytes, jsonByte1)

	jsonByte2, err := makeJsonByte("Alice", testPubKey,
		"Pleasee?? Think of all the arguments we could win on war thunder!!")
	assert.NoError(t, err, "makeJsonByte should not throw an error")
	jsonBytes = append(jsonBytes, jsonByte2)

	jsonByte3, err := makeJsonByte("Alice", testPubKey, "Fine! I'll do it myself then! >:(")
	assert.NoError(t, err, "makeJsonByte should not throw an error")
	jsonBytes = append(jsonBytes, jsonByte3)

	// testing receiveAndHold
	for _, jsonByte := range jsonBytes {
		err = server.recieveAndHold(testPubKey, jsonByte)
		assert.NoError(t, err, "receiveAndHold should not throw an error")
	}

	// testing malicious signature and valid signature for fetchAndClear
	maliciousSignature := "Hi I'm gonna steal your messages now!!!"
	validSignature := testPubKey + "-Secret"

	_, err = server.fetchAndClear(testPubKey, timestamp, maliciousSignature)
	assert.Error(t, err, "malicious signature must return error")

	messages, err := server.fetchAndClear(testPubKey, timestamp, validSignature)
	assert.NoError(t, err, "fetchAndClear should not throw an error")
	assert.Len(t, messages, 3, "should be 3 messages returned")

	for i, jsonByte := range jsonBytes {
		var unmarshlledMsg Message
		err := json.Unmarshal(jsonByte, &unmarshlledMsg)
		assert.NoError(t, err, "Unmarshal should not throw an error")
		assert.Equal(t, string(unmarshlledMsg.Payload), string(messages[i].Payload), "Payload bytes must match")

	}

	queueLength := server.redisClient.LLen(server.ctx, testPubKey).Val()
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
