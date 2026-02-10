package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cevrimxe/go-mini-rmm/internal/models"
	"github.com/cevrimxe/go-mini-rmm/internal/server/db"
	"github.com/go-chi/chi/v5"
)

type AgentHandler struct {
	Store *db.Store
}

func (h *AgentHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var payload models.HeartbeatPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if payload.AgentID == "" {
		http.Error(w, "agent_id required", http.StatusBadRequest)
		return
	}

	if err := h.Store.UpsertAgent(payload); err != nil {
		slog.Error("upsert agent failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Store metric
	metric := models.Metric{
		AgentID:       payload.AgentID,
		CPUPercent:    payload.CPUPercent,
		MemoryPercent: payload.MemoryPercent,
		DiskPercent:   payload.DiskPercent,
	}
	if err := h.Store.InsertMetric(metric); err != nil {
		slog.Error("insert metric failed", "error", err)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	agents, err := h.Store.ListAgents()
	if err != nil {
		slog.Error("list agents failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if agents == nil {
		agents = []models.Agent{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func (h *AgentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	agent, err := h.Store.GetAgent(id)
	if err != nil {
		slog.Error("get agent failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if agent == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

func (h *AgentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.Store.DeleteAgent(id); err != nil {
		slog.Error("delete agent failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AgentHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	metrics, err := h.Store.GetLatestMetrics(id, limit)
	if err != nil {
		slog.Error("get metrics failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if metrics == nil {
		metrics = []models.Metric{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
