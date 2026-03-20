package client

import (
	"bufio"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	// "net/http"
)

type Message struct {
	From    string `json:"From"`
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
	contacts       map[string]Contact // REMINDER THE KEY IS ED25519 PUBLIC KEY HEX!!!!!
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
		contacts:       make(map[string]Contact), // REMINDER THE KEY IS ED25519 PUBLIC KEY HEX!!!!!
	}
}

func (c *Client) addContact(publicXKeyHex string, publicEdKeyHex string, alias string) {

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
				fmt.Println("Chatted '", parts[2], "' to", contact.alias, "aka", contact.publicEdKeyHex)
			} else {
				fmt.Println(parts[1], "is not a valid contact")
			}

		}
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
	privXKey, _ := ecdh.X25519().GenerateKey(rand.Reader)
	fmt.Println("TEST PUBLIC X25519 KEY:")
	fmt.Println(hex.EncodeToString(privXKey.PublicKey().Bytes()))

	pubEdKey, _, _ := ed25519.GenerateKey(rand.Reader)
	fmt.Println("TEST PUBLIC ED25519 KEY")
	fmt.Println(hex.EncodeToString(pubEdKey))

	client := NewClient(":" + args[0])
	fmt.Println("\nCOMMANDS:")
	fmt.Println("/add-contact [public X25519 key] [public ED25519 key] [alias]")
	fmt.Println("/chat [public ED25519 key] [message]")
	fmt.Println("/get-contacts")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(">> ")
		if !scanner.Scan() {
			break
		}
		client.processInput(scanner.Text())
	}
}
