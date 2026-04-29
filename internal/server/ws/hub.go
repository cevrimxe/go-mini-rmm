package ws

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

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

	// Directory listing request-response
	dirRequests   map[string]chan json.RawMessage
	dirRequestsMu sync.Mutex
}

func NewHub(store *db.Store) *Hub {
	return &Hub{
		store:       store,
		agents:      make(map[string]*websocket.Conn),
		register:    make(chan *agentConn),
		unregister:  make(chan string),
		dirRequests: make(map[string]chan json.RawMessage),
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
		case "file_download_result":
			h.handleFileTransferResult(msg.Payload)
		case "file_upload_result":
			h.handleFileTransferResult(msg.Payload)
		case "dir_list_result":
			h.handleDirListResult(message)
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

func (h *Hub) handleFileTransferResult(payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	var result struct {
		TransferID int64  `json:"transfer_id"`
		Success    bool   `json:"success"`
		Error      string `json:"error"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		slog.Warn("invalid file transfer result", "error", err)
		return
	}

	status := models.TransferDone
	if !result.Success {
		status = models.TransferFailed
	}

	if err := h.store.UpdateFileTransferStatus(result.TransferID, status, result.Error); err != nil {
		slog.Error("update file transfer status failed", "error", err)
	}
}

func (h *Hub) handleDirListResult(rawMessage []byte) {
	// Extract request_id from the payload to route the response
	var envelope struct {
		Payload struct {
			RequestID string `json:"request_id"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(rawMessage, &envelope); err != nil {
		slog.Warn("invalid dir_list_result", "error", err)
		return
	}

	h.dirRequestsMu.Lock()
	ch, ok := h.dirRequests[envelope.Payload.RequestID]
	if ok {
		delete(h.dirRequests, envelope.Payload.RequestID)
	}
	h.dirRequestsMu.Unlock()

	if ok {
		// Extract just the payload
		var msg struct {
			Payload json.RawMessage `json:"payload"`
		}
		json.Unmarshal(rawMessage, &msg)
		ch <- msg.Payload
	}
}

// BrowseAgent sends a dir_list command to an agent and waits for the response
func (h *Hub) BrowseAgent(agentID, path string) (json.RawMessage, error) {
	requestID := fmt.Sprintf("dir_%d", time.Now().UnixNano())

	// Create response channel
	ch := make(chan json.RawMessage, 1)
	h.dirRequestsMu.Lock()
	h.dirRequests[requestID] = ch
	h.dirRequestsMu.Unlock()

	// Cleanup on exit
	defer func() {
		h.dirRequestsMu.Lock()
		delete(h.dirRequests, requestID)
		h.dirRequestsMu.Unlock()
	}()

	// Send command
	msg := models.WSMessage{
		Type: "dir_list",
		Payload: map[string]interface{}{
			"request_id": requestID,
			"path":       path,
		},
	}
	if err := h.SendToAgent(agentID, msg); err != nil {
		return nil, err
	}

	// Wait for response with timeout
	select {
	case result := <-ch:
		return result, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for directory listing")
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
