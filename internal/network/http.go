package network

import (
	"net"
	"net/http"
	"sync"
)

type Server struct {
	URL    string
	server *http.Server
	done   chan error
	once   sync.Once
	err    error
}

func Serve(handler http.Handler, address string) (*Server, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	server := &http.Server{Handler: handler, Protocols: protocols}
	s := &Server{URL: "http://" + listener.Addr().String(), server: server, done: make(chan error, 1)}
	go func() { s.done <- server.Serve(listener) }()
	return s, nil
}

func (s *Server) Close() error {
	s.once.Do(func() {
		s.err = s.server.Close()
		if err := <-s.done; err != nil && err != http.ErrServerClosed && s.err == nil {
			s.err = err
		}
	})
	return s.err
}

func NewGRPCClient() *http.Client {
	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	return &http.Client{Transport: &http.Transport{Protocols: protocols}}
}
