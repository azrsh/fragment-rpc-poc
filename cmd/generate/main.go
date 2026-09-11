package main

import (
	"flag"
	"fmt"
	"github.com/azrsh/fragment-colocation-with-grpc/internal/compiler"
	"os"
)

func main() {
	root := flag.String("root", ".", "project directory")
	flag.Parse()
	n, err := compiler.Generate(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Generated %d RPC methods from colocated fragments.\n", n)
}
