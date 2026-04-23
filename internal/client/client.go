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

type Client struct {
	serverURL string
	port      int
	prefix    string
	domain    string

	conn   *websocket.Conn
	writeMu sync.Mutex

	pending   map[string]chan *protocol.Message
	pendingMu sync.Mutex
}

func New(serverURL string, port int, prefix string) *Client {
	return &Client{
		serverURL: serverURL,
		port:      port,
		prefix:    prefix,
		pending:   make(map[string]chan *protocol.Message),
	}
}

func (c *Client) Run() {
	for {
		err := c.connect()
		if err != nil {
			log.Printf("Connection error: %v, reconnecting in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("Disconnected, reconnecting in 5s...")
		time.Sleep(5 * time.Second)
	}
}

func (c *Client) connect() error {
	wsURL := strings.Replace(c.serverURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	c.conn = conn
	defer conn.Close()

	// Register
	if err := c.send(protocol.Message{
		Type:   protocol.TypeRegister,
		Prefix: c.prefix,
		Port:   c.port,
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
			c.domain = m.Domain
			fmt.Printf("\u2705 隧道已建立: %s \u2192 localhost:%d\n\n", m.Domain, c.port)

		case protocol.TypeRequest:
			go c.handleRequest(&m)

		case protocol.TypePing:
			c.send(protocol.Message{Type: protocol.TypePong})

		case protocol.TypePong:
			// heartbeat ack
		}
	}
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

	url := fmt.Sprintf("http://localhost:%d%s", c.port, m.Path)
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
	return c.conn.WriteMessage(websocket.TextMessage, data)
}
