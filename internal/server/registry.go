package server

import (
	"crypto/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn     *websocket.Conn
	WriteMu  sync.Mutex
	Port     int
	LastPing time.Time
}

type Registry struct {
	mu      sync.RWMutex
	clients map[string]*Client
	domain  string
}

func NewRegistry(domain string) *Registry {
	return &Registry{
		clients: make(map[string]*Client),
		domain:  domain,
	}
}

func (r *Registry) Register(prefix string, conn *websocket.Conn, port int) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if prefix == "" {
		prefix = generatePrefix(6)
		for r.clients[prefix] != nil {
			prefix = generatePrefix(6)
		}
	}

	r.clients[prefix] = &Client{
		Conn:     conn,
		Port:     port,
		LastPing: time.Now(),
	}

	return prefix + "." + r.domain
}

func (r *Registry) Unregister(prefix string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, prefix)
}

func (r *Registry) Get(prefix string) (*Client, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.clients[prefix]
	return c, ok
}

func (r *Registry) UpdatePing(prefix string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.clients[prefix]; ok {
		c.LastPing = time.Now()
	}
}

func (r *Registry) CleanupStale(timeout time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for prefix, c := range r.clients {
		if now.Sub(c.LastPing) > timeout {
			c.Conn.Close()
			delete(r.clients, prefix)
		}
	}
}

func (r *Registry) StartCleanup(interval, timeout time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			r.CleanupStale(timeout)
		}
	}()
}

func generatePrefix(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}
