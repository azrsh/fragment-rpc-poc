package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/azrsh/fragment-colocation-with-grpc/internal/demo"
)

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
func run() error {
	static := flag.String("static", "dist", "directory containing the built web client")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server, err := demo.Start("127.0.0.1:"+env("PORT", "3000"), "127.0.0.1:"+env("GRPC_PORT", "50051"), *static, func(service, id string) { fmt.Printf("gRPC %s(%q)\n", service, id) })
	if err != nil {
		return err
	}
	fmt.Printf("Demo: %s\nGenerated gRPC API: %s/app.v1.AppService\n", server.Web.URL, server.GRPC.URL)
	<-ctx.Done()
	return server.Close()
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
