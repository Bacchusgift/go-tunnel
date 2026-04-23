package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Bacchusgift/go-tunnel/internal/protocol"
	"github.com/gorilla/websocket"
)

// Client manages a single WebSocket tunnel connection.
type Client struct {
	ServerURL string
	Port      int
	Prefix    string
	Domain    string

	conn    *websocket.Conn
	writeMu sync.Mutex
	closed  bool
	closeMu sync.Mutex

	onRegistered func(domain string)
	registeredCh chan struct{}
}

// New creates a new tunnel client.
func New(serverURL string, port int, prefix string) *Client {
	return &Client{
		ServerURL:    serverURL,
		Port:         port,
		Prefix:       prefix,
		registeredCh: make(chan struct{}),
	}
}

// OnRegistered sets a callback fired when the server assigns a domain.
func (c *Client) OnRegistered(cb func(domain string)) {
	c.onRegistered = cb
}

// Registered returns a channel that's closed after successful registration.
func (c *Client) Registered() <-chan struct{} {
	return c.registeredCh
}

// Connect establishes the WebSocket connection and starts the read loop.
// It blocks until the connection is lost, then returns the error.
func (c *Client) Connect() error {
	wsURL := strings.Replace(c.ServerURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	c.conn = conn
	defer c.cleanup()

	// Register
	if err := c.send(protocol.Message{
		Type:   protocol.TypeRegister,
		Prefix: c.Prefix,
		Port:   c.Port,
	}); err != nil {
		return fmt.Errorf("register: %w", err)
	}

	// Start heartbeat
	done := make(chan struct{})
	defer close(done)
	go c.heartbeat(done)

	// Read loop
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var m protocol.Message
		if err := json.Unmarshal(msg, &m); err != nil {
			continue
		}

		switch m.Type {
		case protocol.TypeRegistered:
			c.Domain = m.Domain
			if idx := strings.Index(m.Domain, "."); idx > 0 {
				c.Prefix = m.Domain[:idx]
			}
			select {
			case <-c.registeredCh:
				// already closed
			default:
				close(c.registeredCh)
			}
			if c.onRegistered != nil {
				c.onRegistered(m.Domain)
			}

		case protocol.TypeRequest:
			go c.handleRequest(&m)

		case protocol.TypePing:
			c.send(protocol.Message{Type: protocol.TypePong})

		case protocol.TypePong:
		}
	}
}

// Close shuts down the connection.
func (c *Client) Close() {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) cleanup() {
	c.Close()
}

func (c *Client) heartbeat(done chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := c.send(protocol.Message{Type: protocol.TypePing}); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

func (c *Client) handleRequest(m *protocol.Message) {
	var body io.Reader
	if m.Body != "" {
		b, err := base64.StdEncoding.DecodeString(m.Body)
		if err != nil {
			c.sendResponse(m.ID, 500, nil, []byte("base64 decode error"))
			return
		}
		body = bytes.NewReader(b)
	}

	url := fmt.Sprintf("http://localhost:%d%s", c.Port, m.Path)
	req, err := http.NewRequest(m.Method, url, body)
	if err != nil {
		c.sendResponse(m.ID, 500, nil, []byte(err.Error()))
		return
	}
	for k, v := range m.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.sendResponse(m.ID, 502, nil, []byte(err.Error()))
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	respHeaders := make(map[string]string)
	for k, vs := range resp.Header {
		if len(vs) > 0 {
			respHeaders[k] = vs[0]
		}
	}

	c.sendResponse(m.ID, resp.StatusCode, respHeaders, respBody)
}

func (c *Client) sendResponse(id string, status int, headers map[string]string, body []byte) {
	msg := protocol.Message{
		Type:    protocol.TypeResponse,
		ID:      id,
		Status:  status,
		Headers: headers,
		Body:    base64.StdEncoding.EncodeToString(body),
	}
	if err := c.send(msg); err != nil {
		log.Printf("Failed to send response for %s: %v", id, err)
	}
}

func (c *Client) send(m protocol.Message) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}
