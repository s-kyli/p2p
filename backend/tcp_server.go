package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

type Message struct {
	from    string
	payload []byte
}

type Server struct {
	listenAddr string
	ln         net.Listener
	quitch     chan struct{}
	msgch      chan Message
	peerch     chan net.Conn
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		msgch:      make(chan Message, 10),
		peerch:     make(chan net.Conn),
	}
}

func (s *Server) Dial(peerAddress string) error {
	conn, err := net.Dial("tcp", peerAddress)
	if err != nil {
		return err
	}

	fmt.Println("connected to:", peerAddress)
	s.peerch <- conn
	go s.readLoop(conn)
	return nil
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}

	defer ln.Close()
	s.ln = ln

	go s.acceptLoop()

	<-s.quitch
	close(s.msgch)

	return nil

}

func (s *Server) acceptLoop() {
	for {

		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}

		fmt.Println("new connection to server:", conn.RemoteAddr())
		s.peerch <- conn
		go s.readLoop(conn)

	}
}

func (s *Server) readLoop(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println(conn.RemoteAddr(), "disconnected successfully:")
				return
			}
			fmt.Println("read error:", err)
			return
		}

		s.msgch <- Message{
			from:    conn.RemoteAddr().String(),
			payload: buf[:n],
		}

		// conn.Write([]byte("Recieved message!"))
	}
}

func (server *Server) processRecieve(msg Message, peers map[string]net.Conn) {

	peer, exist := peers[msg.from]

	if exist {
		fmt.Println("Recieved message from:", msg.from, ":", string(msg.payload))
	} else {
		fmt.Println(peer.RemoteAddr())
		fmt.Println("Recieved unsolicited message from:", msg.from, ":", string(msg.payload))

	}

}

func (server *Server) processInput(text string, peers map[string]net.Conn) {
	if strings.HasPrefix(text, "/connect") {
		parts := strings.Split(text, " ")
		if len(parts) == 2 {
			go server.Dial(parts[1])
		}
	} else if strings.HasPrefix(text, "/get-peers") {
		for address := range peers {
			fmt.Println(address)
		}
	} else if strings.HasPrefix(text, "/chat") {
		parts := strings.SplitN(text, " ", 3)
		if len(parts) >= 3 {

			peer, exist := peers[parts[1]]
			if exist {
				_, err := peer.Write([]byte(parts[2]))
				if err != nil {
					fmt.Println("send error:", err)
				}
			} else {
				fmt.Println(parts[1], "is not a valid peer")
			}

		}
	} else if strings.HasPrefix(text, "/disconnect") {
		parts := strings.Split(text, " ")
		if len(parts) == 2 {

		}
	}
}

// learning how to use go lang and tcp server
func main() {

	if len(os.Args) <= 1 {
		fmt.Println("usage: go run tcp_sever.go [port]")
		return
	}

	port := os.Args[1]
	server := NewServer(port)

	fmt.Println("server started on ", port)
	fmt.Println("type: /connect :[port number] to chat")
	peers := make(map[string]net.Conn)

	inputCh := make(chan string)

	go func() {
		log.Fatal(server.Start())
	}()

	go func() {
		for {
			select {
			case peer := <-server.peerch:
				peers[peer.RemoteAddr().String()] = peer
				fmt.Printf("new chat connected %s", peer.RemoteAddr())
				fmt.Println()
				fmt.Println("Total peers:", len(peers))
				fmt.Println()

			case msg := <-server.msgch:
				server.processRecieve(msg, peers)
				fmt.Println()

			case text := <-inputCh:
				server.processInput(text, peers)
				fmt.Println()
			}
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	// fmt.Print("-- ")
	for scanner.Scan() {
		inputCh <- scanner.Text()
		// fmt.Print("-- ")
	}
}
