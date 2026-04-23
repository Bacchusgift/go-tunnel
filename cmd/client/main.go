package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Bacchusgift/go-tunnel/internal/client"
)

func main() {
	port := flag.Int("port", 0, "local port to forward to (required)")
	prefix := flag.String("prefix", "", "subdomain prefix (optional, random if empty)")
	server := flag.String("server", os.Getenv("TUNNEL_SERVER"), "server WebSocket URL (required)")
	flag.Parse()

	if *port == 0 {
		fmt.Fprintln(os.Stderr, "Error: -port is required")
		flag.Usage()
		os.Exit(1)
	}
	if *server == "" {
		fmt.Fprintln(os.Stderr, "Error: -server or TUNNEL_SERVER env is required")
		flag.Usage()
		os.Exit(1)
	}

	c := client.New(*server, *port, *prefix)
	c.Run()
}
