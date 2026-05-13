package api

import (
	"log/slog"
	"net/http"

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
