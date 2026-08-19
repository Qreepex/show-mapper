package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Hub fans out Envelopes to all connected web clients.
// It implements core.Sink so the conductor can broadcast without importing
// the HTTP layer.
type Hub struct {
	mu            sync.Mutex
	clients       map[*wsClient]struct{}
	snapshotFn    func() Envelope
	clientHandler ClientHandler
}

// NewHub creates a Hub. Call SetSnapshot before serving clients.
func NewHub() *Hub {
	return &Hub{clients: map[*wsClient]struct{}{}}
}

// SetSnapshot registers the producer for the initial "state.snapshot"
// message every new client receives right after connecting.
func (h *Hub) SetSnapshot(fn func() Envelope) { h.snapshotFn = fn }

// SetClientHandler registers the handler for inbound "client.*" messages.
func (h *Hub) SetClientHandler(fn ClientHandler) { h.clientHandler = fn }

type wsClient struct {
	conn *websocket.Conn
	send chan []byte
}

// ClientHandler receives application-level messages from web clients
// ("client.*" types, see docs/protocols.md). Set via SetClientHandler.
type ClientHandler func(env Envelope)

// Broadcast implements core.Sink. Slow clients that can't keep up
// (full buffer) are dropped — the UI shows a reconnect state anyway.
func (h *Hub) Broadcast(msgType string, data any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.clients) == 0 {
		return
	}
	payload, err := marshalEnvelope(msgType, data)
	if err != nil {
		slog.Error("hub: marshal failed", "type", msgType, "err", err)
		return
	}
	for c := range h.clients {
		select {
		case c.send <- payload:
		default:
			slog.Warn("hub: dropping slow client")
			h.removeLocked(c)
		}
	}
}

// ClientCount returns connected client count (used by /api/health).
func (h *Hub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func (h *Hub) removeLocked(c *wsClient) {
	close(c.send)
	_ = c.conn.Close(websocket.StatusGoingAway, "")
	delete(h.clients, c)
}

// ServeWS upgrades an HTTP connection and serves it until it dies.
// Origin policy: default (Origin header, if present, must match Host).
// The frontend is served same-origin; vite dev proxies /ws, so no CORS setup
// is required. Non-browser clients (wscat) send no Origin and are accepted.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.Warn("ws accept failed", "err", err)
		return
	}
	conn.SetReadLimit(1 << 16) // clients send nothing large

	c := &wsClient{conn: conn, send: make(chan []byte, 128)}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	ctx := r.Context()

	// writer goroutine
	writeErr := make(chan error, 1)
	go func() {
		for msg := range c.send {
			wctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := conn.Write(wctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				writeErr <- err
				return
			}
		}
	}()

	// initial snapshot
	if h.snapshotFn != nil {
		snap := h.snapshotFn()
		if payload, err := marshalEnvelope(snap.Type, snap.Data); err == nil {
			select {
			case c.send <- payload:
			default:
			}
		}
	}

	// reader loop: app-level messages are dot-namespaced "client.*" and routed
	// to the registered handler; reading also keeps pongs flowing and detects
	// closure.
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			break
		}
		h.dispatchClient(data)
	}

	h.mu.Lock()
	h.removeLocked(c)
	h.mu.Unlock()

	select {
	case err := <-writeErr:
		slog.Debug("ws write failed", "err", err)
	default:
	}
}

// dispatchClient parses and routes one inbound message.
func (h *Hub) dispatchClient(data []byte) {
	if h.clientHandler == nil {
		return
	}
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return // garbage from a non-app client: ignore
	}
	if !strings.HasPrefix(env.Type, "client.") {
		return // inbound outside the client.* namespace is reserved
	}
	defer func() {
		if r := recover(); r != nil {
			slog.Error("client handler panicked", "type", env.Type, "panic", r)
		}
	}()
	h.clientHandler(env)
}

func marshalEnvelope(msgType string, data any) ([]byte, error) {
	return json.Marshal(Envelope{Type: msgType, TS: time.Now(), Data: data})
}
