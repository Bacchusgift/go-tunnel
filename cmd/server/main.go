package main

import (
	"flag"
	"log"

	"github.com/Bacchusgift/go-tunnel/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	domain := flag.String("domain", "autowired.cn", "base domain for subdomain routing")
	flag.Parse()

	s := server.New(*addr, *domain)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
