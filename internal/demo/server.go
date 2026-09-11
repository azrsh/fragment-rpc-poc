package demo

import (
	"errors"
	"net/http"

	"github.com/azrsh/fragment-colocation-with-grpc/generated"
	"github.com/azrsh/fragment-colocation-with-grpc/internal/gateway"
	"github.com/azrsh/fragment-colocation-with-grpc/internal/network"
)

type Server struct {
	Web, GRPC *network.Server
	Backends  *Backends
	client    *http.Client
}

func Start(webAddress, grpcAddress, staticRoot string, onCall func(string, string)) (*Server, error) {
	backends, err := StartBackends(onCall)
	if err != nil {
		return nil, err
	}
	client := network.NewGRPCClient()
	path, handler, err := generated.NewHandler(gateway.NewResolvers(client, backends.User.URL, backends.Organization.URL))
	if err != nil {
		backends.Close()
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	mux.Handle("/", http.FileServer(http.Dir(staticRoot)))
	grpc, err := network.Serve(mux, grpcAddress)
	if err != nil {
		backends.Close()
		return nil, err
	}
	web, err := network.Serve(mux, webAddress)
	if err != nil {
		grpc.Close()
		backends.Close()
		return nil, err
	}
	return &Server{Web: web, GRPC: grpc, Backends: backends, client: client}, nil
}
func (s *Server) Close() error {
	err := errors.Join(s.Web.Close(), s.GRPC.Close(), s.Backends.Close())
	s.client.CloseIdleConnections()
	return err
}
