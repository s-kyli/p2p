package client

import (
	"bufio"
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Message struct {
	From    string `json:"From"`
	FromX   string `json:"FromX"`
	To      string `json:"To"`
	Payload []byte `json:"Payload"`
}

type Contact struct {
	publicXKeyHex  string
	publicEdKeyHex string
	alias          string
	sharedSecret   []byte
}

type Client struct {
	listenAddr     string
	publicEdKeyHex string
	privateEdKey   ed25519.PrivateKey
	publicXKey     *ecdh.PublicKey
	publicXKeyHex  string
	privateXKey    *ecdh.PrivateKey   // ermm doesnnt look so secure, changge this later
	contacts       map[string]Contact // Key is the public ed25519 key of contact
	sendServerUrl  string
	fetchServerUrl string
}

func NewClient(listenAddr string) *Client {

	pubEdKey, privEdKey, _ := ed25519.GenerateKey(rand.Reader)
	pubEdKeyHex := hex.EncodeToString(pubEdKey)

	privXKey, _ := ecdh.X25519().GenerateKey(rand.Reader)
	pubXKey := privXKey.PublicKey()

	return &Client{
		listenAddr:     listenAddr,
		publicEdKeyHex: pubEdKeyHex,
		privateEdKey:   privEdKey,
		publicXKey:     pubXKey,
		publicXKeyHex:  hex.EncodeToString(pubXKey.Bytes()),
		privateXKey:    privXKey,
		contacts:       make(map[string]Contact), // key is public ed25519 key of contact
		sendServerUrl:  "http://localhost:8080/send",
		fetchServerUrl: "http://localhost:8080/fetch",
	}
}

func (c *Client) addContact(publicXKeyHex string, publicEdKeyHex string, alias string) {

	_, exist := c.contacts[publicEdKeyHex]
	if exist {
		fmt.Println(publicEdKeyHex, "is already a contact, replacing..")
	}

	decodedPublicXKey, err := hex.DecodeString((publicXKeyHex))
	if err != nil {
		fmt.Println("Error decoding x25519 key:", err)
		return
	}

	pubXKey, err := ecdh.X25519().NewPublicKey(decodedPublicXKey)
	if err != nil {
		fmt.Println("Error converting decodedPublicXKey to ecdh.PublicKey", err)
		return
	}

	sharedSecret, err := c.privateXKey.ECDH(pubXKey)
	if err != nil {
		fmt.Println("Error creating shared secret", err)
		return
	}

	c.contacts[publicEdKeyHex] = Contact{
		publicXKeyHex:  publicXKeyHex,
		publicEdKeyHex: publicEdKeyHex,
		alias:          alias,
		sharedSecret:   sharedSecret,
	}
}

func (c *Client) sendMessage(contact Contact, msgText string) {

	ciphertext, err := EncryptPayload(contact.sharedSecret, msgText)
	if err != nil {
		fmt.Println("sendMesesage failed, encryption failed:", err)
		return
	}

	jsonByte, err := MakeJsonByte(c.publicEdKeyHex, c.publicXKeyHex, contact.publicEdKeyHex, ciphertext)
	if err != nil {
		fmt.Println("makeJsonByte error:", err)
		return
	}

	difficulty := 2
	retryLimit := 3

	for attempt := 0; attempt < retryLimit; attempt++ {
		nonce := GeneratePoW(jsonByte, difficulty)

		request, err := http.NewRequest("POST", c.sendServerUrl, bytes.NewBuffer(jsonByte))
		if err != nil {
			fmt.Println("Failed to create HTTP request:", err)
			return
		}

		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("PoWNonce", strconv.Itoa(nonce))

		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			fmt.Println("HTTP POST failed:", err)
			return
		}

		if response.StatusCode == http.StatusOK {
			fmt.Println("Message send to server successfully")
			response.Body.Close()
			return
		}

		if response.StatusCode == http.StatusPreconditionFailed {
			newDifficulty, err := strconv.Atoi(response.Header.Get("X-Required-Difficulty"))
			if err == nil {
				fmt.Printf("Server requires higher difficulty: %d\n", newDifficulty)
				response.Body.Close()
				difficulty = newDifficulty
				continue
			}
		}

		body, _ := io.ReadAll(response.Body)
		msg := string(body)

		fmt.Printf("Server rejected message send. Error: %s (status: %s)\n", msg, response.Status)
		defer response.Body.Close()
		return

	}

}

func (c *Client) fetchMessages() {

	timestamp := time.Now().Unix()

	authPayload := []byte(fmt.Sprintf("fetch:%s:%d", c.publicEdKeyHex, timestamp))
	validSignature := ed25519.Sign(c.privateEdKey, authPayload)
	validSignatureHex := hex.EncodeToString(validSignature)

	requestBody := map[string]interface{}{
		"RequesterPubKey": c.publicEdKeyHex,
		"Timestamp":       timestamp,
		"Signature":       validSignatureHex,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println("json.Marshal failed:", err)
		return
	}

	response, err := http.Post(c.fetchServerUrl, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Println("HTTP POST failed:", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Println("Server rejected fetch. Status:", response.Status)
		return
	}

	var messages []Message
	err = json.NewDecoder(response.Body).Decode(&messages)
	if err != nil {
		fmt.Println("Failed to decode messages from server:", err)
		return
	}

	if len(messages) == 0 {
		fmt.Println("No messages to get")
		return
	}

	for i := range messages {
		from := messages[i].From

		contact, exist := c.contacts[from]
		if exist {
			fmt.Println("Received message from:", from, "aka:", contact.alias)
		} else {
			fmt.Println("Received new message from:", from)
			fromX := messages[i].FromX
			c.addContact(fromX, from, "Unknown")
			contact, _ = c.contacts[from]
		}

		newPayload, err := DecryptPayload(contact.sharedSecret, messages[i].Payload)
		if err != nil {
			fmt.Println("Error decrypting payload:", err)
			continue
		}
		messages[i].Payload = newPayload
		fmt.Println("Message contents:", string(newPayload))
	}

}

func (c *Client) processInput(text string) {
	text = strings.TrimSpace(text)

	if strings.HasPrefix(text, "/add-contact") {
		parts := strings.Split(text, " ")
		if len(parts) == 4 {
			c.addContact(parts[1], parts[2], parts[3])
		} else {
			fmt.Println("usage: /add-contact [public X25519 key (hex string)] [public  ED25519 key (hex string)] [alias (string)]")
		}
	} else if strings.HasPrefix(text, "/get-contacts") {
		for _, contact := range c.contacts {
			fmt.Printf("%s's contact info:\n", contact.alias)
			fmt.Println("Public X25519 key:", contact.publicXKeyHex)
			fmt.Println("Public ED25519 key:", contact.publicEdKeyHex)
			fmt.Println()
		}
	} else if strings.HasPrefix(text, "/chat") {
		parts := strings.SplitN(text, " ", 3)
		if len(parts) >= 3 {

			//just for now, adding encryption and communication to server.go later
			contact, exist := c.contacts[parts[1]]
			if exist {
				// fmt.Println("Chatted '", parts[2], "' to", contact.alias, "aka", contact.publicEdKeyHex)
				c.sendMessage(contact, parts[2])
			} else {
				fmt.Println(parts[1], "is not a valid contact")
			}

		}
	} else if strings.HasPrefix(text, "/fetch") {
		c.fetchMessages()
	} else if strings.HasPrefix(text, "/disconnect") {
		parts := strings.Split(text, " ")
		if len(parts) == 2 {
			//tbd
		}
	}
}

func RunClient(args []string) {

	if len(args) == 0 {
		fmt.Println("usage: go run client_main.go [port]")
		return
	}

	//create test public key for testing with fake contact
	// privXKey, _ := ecdh.X25519().GenerateKey(rand.Reader)
	// fmt.Println("TEST PUBLIC X25519 KEY:")
	// fmt.Println(hex.EncodeToString(privXKey.PublicKey().Bytes()))

	// pubEdKey, _, _ := ed25519.GenerateKey(rand.Reader)
	// fmt.Println("TEST PUBLIC ED25519 KEY")
	// fmt.Println(hex.EncodeToString(pubEdKey))

	client := NewClient(":" + args[0])
	fmt.Println("YOUR PUBLIC X25519 KEY:")
	fmt.Println(hex.EncodeToString(client.publicXKey.Bytes()))

	fmt.Println("YOUR PUBLIC ED25519 KEY")
	fmt.Println(client.publicEdKeyHex)

	fmt.Println("\nCOMMANDS:")
	fmt.Println("/add-contact [public X25519 key] [public ED25519 key] [alias]")
	fmt.Println("/chat [public ED25519 key] [message]")
	fmt.Println("/get-contacts")
	fmt.Println("/fetch -- fetch all new messages in your inbox.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(">> ")
		if !scanner.Scan() {
			break
		}
		client.processInput(scanner.Text())
	}
}
