package main

import (
	"fmt"
	"net"
)

type Server struct {
	listenAddr string
	ln         net.Listener
	quitch     chan struct{}
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tpc", s.listenAddr)
	if err != nil {
		return err
	}

	defer ln.Close()
	s.ln = ln

	<-s.quitch

	return nil

}

// learning how to use go lang and tcp server
func main() {
	fmt.Println("hello world")

}
