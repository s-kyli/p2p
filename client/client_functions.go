package client

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

func EncryptPayload(sharedSecret []byte, plaintext string) ([]byte, error) {
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertext, nil
}

func DecryptPayload(sharedSecret []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintextBytes, err := aesGCM.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintextBytes, nil
}

func MakeJsonByte(from string, fromX, to string, payload []byte) ([]byte, error) {
	jsonByte, err := json.Marshal(Message{
		From:    from,
		FromX:   fromX,
		To:      to,
		Payload: payload,
	})

	if err != nil {
		return nil, err
	}

	return jsonByte, err
}

// proof of work to prevent/mitigate DDoS attacks, using sha256
func GeneratePoW(data []byte, difficulty int) int {
	nonce := 0
	for {
		hashInput := append(data, []byte(strconv.Itoa(nonce))...)
		hash := sha256.Sum256(hashInput)

		isValid := true
		for i := 0; i < difficulty; i++ {
			if hash[i] != 0 {
				isValid = false
				break
			}
		}

		if isValid {
			return nonce
		}
		nonce++
	}
}

func getRedisKey(publicKey ed25519.PublicKey) string {
	return hex.EncodeToString(publicKey)
}
