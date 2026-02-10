package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cevrimxe/go-mini-rmm/internal/models"
	"github.com/cevrimxe/go-mini-rmm/internal/server/alert"
	"github.com/cevrimxe/go-mini-rmm/internal/server/db"
	"github.com/go-chi/chi/v5"
)

type AlertHandler struct {
	Store  *db.Store
	Engine *alert.Engine
}

func (h *AlertHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.Store.ListAlerts(100)
	if err != nil {
		slog.Error("list alerts failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if alerts == nil {
		alerts = []models.Alert{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

func (h *AlertHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.Store.ListAlertRules()
	if err != nil {
		slog.Error("list alert rules failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if rules == nil {
		rules = []models.AlertRule{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

func (h *AlertHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var req models.AlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	// Validate
	validMetrics := map[string]bool{"cpu_percent": true, "memory_percent": true, "disk_percent": true}
	if !validMetrics[req.Metric] {
		http.Error(w, "invalid metric (cpu_percent, memory_percent, disk_percent)", http.StatusBadRequest)
		return
	}
	validOps := map[string]bool{">": true, "<": true, ">=": true, "<=": true, "==": true}
	if !validOps[req.Operator] {
		http.Error(w, "invalid operator (>, <, >=, <=, ==)", http.StatusBadRequest)
		return
	}

	rule, err := h.Store.CreateAlertRule(req)
	if err != nil {
		slog.Error("create alert rule failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

func (h *AlertHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.Store.DeleteAlertRule(id); err != nil {
		slog.Error("delete alert rule failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
