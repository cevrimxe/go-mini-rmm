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

	// Insert audit log
	user := GetUserFromContext(r)
	username := "system"
	if user != nil {
		username = user.Username
	}
	details := `{"command": "` + req.Command + `"}`
	if err := h.Store.InsertAuditLog(username, "command_execution", agentID, details); err != nil {
		slog.Error("failed to insert audit log", "error", err)
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

type bulkSendRequest struct {
	AgentIDs []string `json:"agent_ids"`
	Command  string   `json:"command"`
	Label    string   `json:"label"` // optional preset label for audit log
}

type bulkSendResult struct {
	AgentID   string `json:"agent_id"`
	CommandID int64  `json:"command_id,omitempty"`
	Queued    bool   `json:"queued"`
	Error     string `json:"error,omitempty"`
}

// BulkSend dispatches the same command to many agents at once
func (h *CommandHandler) BulkSend(w http.ResponseWriter, r *http.Request) {
	var req bulkSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if req.Command == "" {
		http.Error(w, "command required", http.StatusBadRequest)
		return
	}
	if len(req.AgentIDs) == 0 {
		http.Error(w, "agent_ids required", http.StatusBadRequest)
		return
	}

	user := GetUserFromContext(r)
	username := "system"
	if user != nil {
		username = user.Username
	}

	results := make([]bulkSendResult, 0, len(req.AgentIDs))
	for _, agentID := range req.AgentIDs {
		res := bulkSendResult{AgentID: agentID}

		agent, err := h.Store.GetAgent(agentID)
		if err != nil || agent == nil {
			res.Error = "agent not found"
			results = append(results, res)
			continue
		}

		cmd, err := h.Store.CreateCommand(agentID, req.Command)
		if err != nil {
			slog.Error("bulk create command failed", "agent_id", agentID, "error", err)
			res.Error = "create command failed"
			results = append(results, res)
			continue
		}
		res.CommandID = cmd.ID

		msg := models.WSMessage{
			Type: "command",
			Payload: map[string]interface{}{
				"command_id": cmd.ID,
				"command":    req.Command,
			},
		}
		if err := h.Hub.SendToAgent(agentID, msg); err != nil {
			res.Error = "agent not connected"
		} else {
			res.Queued = true
		}
		results = append(results, res)
	}

	label := req.Label
	if label == "" {
		label = "custom"
	}
	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"label":   label,
		"command": req.Command,
		"agents":  req.AgentIDs,
	})
	if err := h.Store.InsertAuditLog(username, "bulk_command_execution", "", string(detailsJSON)); err != nil {
		slog.Error("failed to insert audit log", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(results)
}

func (h *CommandHandler) Get(w http.ResponseWriter, r *http.Request) {
	cmdIDStr := chi.URLParam(r, "cmdID")
	cmdID, err := strconv.ParseInt(cmdIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid command id", http.StatusBadRequest)
		return
	}
	cmd, err := h.Store.GetCommand(cmdID)
	if err != nil || cmd == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
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
