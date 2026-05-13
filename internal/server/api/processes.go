package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cevrimxe/go-mini-rmm/internal/server/db"
	"github.com/cevrimxe/go-mini-rmm/internal/server/ws"
	"github.com/go-chi/chi/v5"
)

type ProcessHandler struct {
	Store *db.Store
	Hub   *ws.Hub
}

// List returns the running process list from the agent via WebSocket
func (h *ProcessHandler) List(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")

	agent, err := h.Store.GetAgent(agentID)
	if err != nil || agent == nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	if !h.Hub.IsConnected(agentID) {
		http.Error(w, "agent not connected", http.StatusBadGateway)
		return
	}

	result, err := h.Hub.ListProcesses(agentID)
	if err != nil {
		slog.Warn("list processes failed", "agent_id", agentID, "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(result)
}

// Connections returns the network connections (netstat-style) from the agent via WebSocket
func (h *ProcessHandler) Connections(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")

	agent, err := h.Store.GetAgent(agentID)
	if err != nil || agent == nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	if !h.Hub.IsConnected(agentID) {
		http.Error(w, "agent not connected", http.StatusBadGateway)
		return
	}

	result, err := h.Hub.ListConnections(agentID)
	if err != nil {
		slog.Warn("list connections failed", "agent_id", agentID, "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(result)
}

// Kill terminates a process on the agent by PID
func (h *ProcessHandler) Kill(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")
	pidStr := chi.URLParam(r, "pid")

	pid64, err := strconv.ParseInt(pidStr, 10, 32)
	if err != nil || pid64 <= 0 {
		http.Error(w, "invalid pid", http.StatusBadRequest)
		return
	}
	pid := int32(pid64)

	agent, err := h.Store.GetAgent(agentID)
	if err != nil || agent == nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	if !h.Hub.IsConnected(agentID) {
		http.Error(w, "agent not connected", http.StatusBadGateway)
		return
	}

	result, err := h.Hub.KillProcess(agentID, pid)
	if err != nil {
		slog.Warn("kill process failed", "agent_id", agentID, "pid", pid, "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	var parsed struct {
		PID     int32  `json:"pid"`
		Name    string `json:"name"`
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	_ = json.Unmarshal(result, &parsed)

	user := GetUserFromContext(r)
	username := "system"
	if user != nil {
		username = user.Username
	}
	details := fmt.Sprintf(`{"pid":%d,"name":"%s","success":%t,"error":"%s"}`, parsed.PID, parsed.Name, parsed.Success, parsed.Error)
	if err := h.Store.InsertAuditLog(username, "process_kill", agentID, details); err != nil {
		slog.Error("failed to insert audit log", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if !parsed.Success {
		w.WriteHeader(http.StatusBadGateway)
	}
	w.Write(result)
}
