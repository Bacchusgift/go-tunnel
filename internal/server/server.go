package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Bacchusgift/go-tunnel/internal/protocol"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	pending   = make(map[string]chan *protocol.Message)
	pendingMu sync.Mutex
)

func registerPending(id string, ch chan *protocol.Message) {
	pendingMu.Lock()
	pending[id] = ch
	pendingMu.Unlock()
}

func unregisterPending(id string) {
	pendingMu.Lock()
	delete(pending, id)
	pendingMu.Unlock()
}

func dispatchResponse(m *protocol.Message) {
	pendingMu.Lock()
	ch, ok := pending[m.ID]
	if ok {
		delete(pending, m.ID)
	}
	pendingMu.Unlock()
	if ok {
		ch <- m
	}
}

type Server struct {
	addr     string
	domain   string
	registry *Registry
}

func New(addr, domain string) *Server {
	s := &Server{
		addr:     addr,
		domain:   domain,
		registry: NewRegistry(domain),
	}
	s.registry.StartCleanup(10*time.Second, 60*time.Second)
	return s
}

func (s *Server) ListenAndServe() error {
	http.Handle("/_tunnel/ws", http.HandlerFunc(s.handleWS))
	http.Handle("/", http.HandlerFunc(s.handleProxy))
	log.Printf("Server listening on %s (domain: %s)", s.addr, s.domain)
	return http.ListenAndServe(s.addr, nil)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	var prefix string
	var client *Client
	cleanup := func() {
		conn.Close()
		if prefix != "" {
			s.registry.Unregister(prefix)
			log.Printf("Client disconnected: %s.%s", prefix, s.domain)
		}
	}
	defer cleanup()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var m protocol.Message
		if err := json.Unmarshal(msg, &m); err != nil {
			continue
		}

		switch m.Type {
		case protocol.TypeRegister:
			domain := s.registry.Register(m.Prefix, conn, m.Port)
			prefix = m.Prefix
			if prefix == "" {
				prefix = strings.TrimSuffix(domain, "."+s.domain)
			}
			client, _ = s.registry.Get(prefix)
			log.Printf("Client registered: %s (port %d)", domain, m.Port)
			clientSendJSON(client, protocol.Message{
				Type:   protocol.TypeRegistered,
				Domain: domain,
			})

		case protocol.TypePong:
			if prefix != "" {
				s.registry.UpdatePing(prefix)
			}

		case protocol.TypePing:
			if prefix != "" {
				s.registry.UpdatePing(prefix)
			}
			clientSendJSON(client, protocol.Message{Type: protocol.TypePong})

		case protocol.TypeResponse:
			dispatchResponse(&m)
		}
	}
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	suffix := "." + s.domain
	if !strings.HasSuffix(host, suffix) {
		http.Error(w, "unknown host", http.StatusBadGateway)
		return
	}
	prefix := strings.TrimSuffix(host, suffix)
	if prefix == "" {
		http.Error(w, "invalid subdomain", http.StatusBadRequest)
		return
	}

	client, ok := s.registry.Get(prefix)
	if !ok {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body.Close()

	headers := make(map[string]string)
	for k, vs := range r.Header {
		if len(vs) > 0 {
			headers[k] = vs[0]
		}
	}

	reqID := generateID()
	respCh := make(chan *protocol.Message, 1)
	registerPending(reqID, respCh)
	defer unregisterPending(reqID)

	reqMsg := protocol.Message{
		Type:    protocol.TypeRequest,
		ID:      reqID,
		Method:  r.Method,
		Path:    r.URL.RequestURI(),
		Headers: headers,
		Body:    base64.StdEncoding.EncodeToString(bodyBytes),
	}

	if err := clientSendJSON(client, reqMsg); err != nil {
		http.Error(w, "tunnel write failed", http.StatusBadGateway)
		return
	}

	select {
	case resp := <-respCh:
		if resp == nil {
			http.Error(w, "tunnel error", http.StatusBadGateway)
			return
		}
		for k, v := range resp.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(resp.Status)
		if resp.Body != "" {
			body, _ := base64.StdEncoding.DecodeString(resp.Body)
			w.Write(body)
		}
	case <-time.After(30 * time.Second):
		http.Error(w, "tunnel timeout", http.StatusGatewayTimeout)
	}
}

func clientSendJSON(c *Client, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.WriteMu.Lock()
	defer c.WriteMu.Unlock()
	return c.Conn.WriteMessage(websocket.TextMessage, data)
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return strings.ReplaceAll(base64.StdEncoding.EncodeToString(b), "/", "_")
}
