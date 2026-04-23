package protocol

type MessageType string

const (
	TypeRegister   MessageType = "register"
	TypeRegistered MessageType = "registered"
	TypeRequest    MessageType = "request"
	TypeResponse   MessageType = "response"
	TypePing       MessageType = "ping"
	TypePong       MessageType = "pong"
)

type Message struct {
	Type    MessageType       `json:"type"`
	ID      string            `json:"id,omitempty"`
	Prefix  string            `json:"prefix,omitempty"`
	Port    int               `json:"port,omitempty"`
	Domain  string            `json:"domain,omitempty"`
	Method  string            `json:"method,omitempty"`
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	Status  int               `json:"status,omitempty"`
	Error   string            `json:"error,omitempty"`
}
