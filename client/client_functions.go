package client

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

func getRedisKey(publicKey ed25519.PublicKey) string {
	return hex.EncodeToString(publicKey)
}
