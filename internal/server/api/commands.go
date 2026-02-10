package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cevrimxe/go-mini-rmm/internal/models"
	"github.com/cevrimxe/go-mini-rmm/internal/server/db"
	"github.com/cevrimxe/go-mini-rmm/internal/server/ws"
	"github.com/go-chi/chi/v5"
)

type CommandHandler struct {
	Store *db.Store
	Hub   *ws.Hub
}

func (h *CommandHandler) Send(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")

	var req models.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		http.Error(w, "command required", http.StatusBadRequest)
		return
	}

	// Check agent exists
	agent, err := h.Store.GetAgent(agentID)
	if err != nil || agent == nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	// Create command record
	cmd, err := h.Store.CreateCommand(agentID, req.Command)
	if err != nil {
		slog.Error("create command failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Send via WebSocket
	msg := models.WSMessage{
		Type: "command",
		Payload: map[string]interface{}{
			"command_id": cmd.ID,
			"command":    req.Command,
		},
	}
	if err := h.Hub.SendToAgent(agentID, msg); err != nil {
		slog.Warn("agent not connected via ws", "agent_id", agentID, "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cmd)
}

func (h *CommandHandler) List(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	cmds, err := h.Store.GetCommandsByAgent(agentID, limit)
	if err != nil {
		slog.Error("list commands failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if cmds == nil {
		cmds = []models.Command{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cmds)
}
