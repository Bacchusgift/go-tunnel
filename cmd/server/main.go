package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Bacchusgift/go-tunnel/internal/server"
)

func main() {
	addr := flag.String("addr", envOrDefault("TUNNEL_ADDR", ":8080"), "listen address")
	domain := flag.String("domain", os.Getenv("TUNNEL_DOMAIN"), "base domain for subdomain routing (required)")
	flag.Parse()

	if *domain == "" {
		fmt.Fprintln(os.Stderr, "Error: -domain or TUNNEL_DOMAIN env is required")
		flag.Usage()
		os.Exit(1)
	}

	s := server.New(*addr, *domain)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
