package backend

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"onion-chat-app/client"

	"github.com/stretchr/testify/assert"
)

func TestReceiveAndFetch(t *testing.T) {
	os.Setenv("REDIS_ADDR", "localhost:6379")
	os.Setenv("REDIS_PASSWORD", "")

	server := NewServer()

	// TESTING REDIS PING
	_, err := server.redisClient.Ping(server.ctx).Result()
	assert.NoError(t, err, "redis client ping should not throw an error")

	// ed25519 keys for identity verification
	jasonEdPub, jasonEdPriv, _ := ed25519.GenerateKey(rand.Reader)
	jasonEdPubHex := hex.EncodeToString(jasonEdPub)

	server.redisClient.Del(server.ctx, jasonEdPubHex)

	// x25519 keys for e2ee
	jasonXPriv, _ := ecdh.X25519().GenerateKey(rand.Reader)
	jasonXPub := jasonXPriv.PublicKey()

	aliceXPriv, _ := ecdh.X25519().GenerateKey(rand.Reader)
	aliceXPub := aliceXPriv.PublicKey()
	aliceXPubHex := hex.EncodeToString(aliceXPub.Bytes())

	aliceSharedSecret, _ := aliceXPriv.ECDH(jasonXPub)
	jasonSharedSecret, _ := jasonXPriv.ECDH(aliceXPub)

	// MAKING JSON EXAMPLE FOR TESTING. in production this will be called on client.

	rawMessages := []string{
		"JASON!!! We should hack the pentagon!",
		"Pleasee?? Think of all the arguments we could win on war thunder!!",
		"Fine! I'll do it myself then! >:(",
	}

	var jsonBytes [][]byte // jsonbytes is what will be "sent" to the server from alice
	for _, msgText := range rawMessages {
		ciphertext, err := client.EncryptPayload(aliceSharedSecret, msgText)
		assert.NoError(t, err, "encryption should not fail")

		jsonByte, err := client.MakeJsonByte(aliceXPubHex, jasonEdPubHex, ciphertext)
		assert.NoError(t, err, "makeJsonByte should not throw an error")
		jsonBytes = append(jsonBytes, jsonByte)
	}

	// server, upon receiving message from alice
	// testing receiveAndHold. in production this will be called on server.
	for _, jsonByte := range jsonBytes {
		// fmt.Println(string(jsonByte))
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
		err := json.Unmarshal(jsonByte, &messages[i])
		assert.NoError(t, err, "Unmarshal should not throw an error")
		messages[i].Payload, err = client.DecryptPayload(jasonSharedSecret, messages[i].Payload)
		assert.NoError(t, err, "decryptPayload should not throw an error")
		assert.Equal(t, rawMessages[i], string(messages[i].Payload), "raw messages and decrypted messages must match")
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
