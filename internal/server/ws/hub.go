package ws

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/cevrimxe/go-mini-rmm/internal/models"
	"github.com/cevrimxe/go-mini-rmm/internal/server/db"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type agentConn struct {
	conn    *websocket.Conn
	agentID string
}

type Hub struct {
	store      *db.Store
	agents     map[string]*websocket.Conn
	mu         sync.RWMutex
	register   chan *agentConn
	unregister chan string
}

func NewHub(store *db.Store) *Hub {
	return &Hub{
		store:      store,
		agents:     make(map[string]*websocket.Conn),
		register:   make(chan *agentConn),
		unregister: make(chan string),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case ac := <-h.register:
			h.mu.Lock()
			h.agents[ac.agentID] = ac.conn
			h.mu.Unlock()
			slog.Info("agent ws connected", "agent_id", ac.agentID)

		case agentID := <-h.unregister:
			h.mu.Lock()
			if conn, ok := h.agents[agentID]; ok {
				conn.Close()
				delete(h.agents, agentID)
			}
			h.mu.Unlock()
			slog.Info("agent ws disconnected", "agent_id", agentID)
		}
	}
}

func (h *Hub) HandleAgentWS(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "agent_id required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade failed", "error", err)
		return
	}

	h.register <- &agentConn{conn: conn, agentID: agentID}

	// Read loop - handles command results from agent
	go h.readPump(conn, agentID)
}

func (h *Hub) readPump(conn *websocket.Conn, agentID string) {
	defer func() {
		h.unregister <- agentID
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Warn("ws read error", "agent_id", agentID, "error", err)
			}
			return
		}

		var msg models.WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			slog.Warn("ws invalid message", "agent_id", agentID, "error", err)
			continue
		}

		switch msg.Type {
		case "command_result":
			h.handleCommandResult(msg.Payload)
		default:
			slog.Debug("ws unknown message type", "type", msg.Type)
		}
	}
}

func (h *Hub) handleCommandResult(payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	var result models.CommandResult
	if err := json.Unmarshal(data, &result); err != nil {
		slog.Warn("invalid command result", "error", err)
		return
	}

	if err := h.store.UpdateCommandResult(result.CommandID, result.Stdout, result.Stderr, result.ExitCode); err != nil {
		slog.Error("update command result failed", "error", err)
	}
}

func (h *Hub) SendToAgent(agentID string, msg models.WSMessage) error {
	h.mu.RLock()
	conn, ok := h.agents[agentID]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent %s not connected", agentID)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

func (h *Hub) IsConnected(agentID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.agents[agentID]
	return ok
}

func (h *Hub) ConnectedAgents() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.agents))
	for id := range h.agents {
		ids = append(ids, id)
	}
	return ids
}
