package alert

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cevrimxe/go-mini-rmm/internal/models"
	"github.com/cevrimxe/go-mini-rmm/internal/server/db"
)

const (
	checkInterval    = 60 * time.Second
	offlineThreshold = 90 * time.Second // 3 missed heartbeats
)

type Engine struct {
	store *db.Store
}

func NewEngine(store *db.Store) *Engine {
	return &Engine{store: store}
}

func (e *Engine) Run(ctx context.Context) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.checkOfflineAgents()
			e.checkMetricRules()
		}
	}
}

func (e *Engine) checkOfflineAgents() {
	count, err := e.store.MarkOfflineAgents(offlineThreshold)
	if err != nil {
		slog.Error("mark offline agents failed", "error", err)
		return
	}
	if count > 0 {
		slog.Info("marked agents offline", "count", count)
	}
}

func (e *Engine) checkMetricRules() {
	rules, err := e.store.ListAlertRules()
	if err != nil {
		slog.Error("list alert rules failed", "error", err)
		return
	}

	if len(rules) == 0 {
		return
	}

	agents, err := e.store.ListAgents()
	if err != nil {
		slog.Error("list agents failed", "error", err)
		return
	}

	for _, agent := range agents {
		if agent.Status != models.AgentOnline {
			continue
		}

		metric, err := e.store.GetLatestMetric(agent.ID)
		if err != nil || metric == nil {
			continue
		}

		for _, rule := range rules {
			if rule.AgentID != "" && rule.AgentID != agent.ID {
				continue
			}
			value := getMetricValue(metric, rule.Metric)
			if evaluate(value, rule.Operator, rule.Threshold) {
				msg := fmt.Sprintf("%s: %s %s %.1f (current: %.1f)",
					agent.Hostname, rule.Metric, rule.Operator, rule.Threshold, value)
				if err := e.store.CreateAlert(rule.ID, agent.ID, msg); err != nil {
					slog.Error("create alert failed", "error", err)
				} else {
					slog.Warn("alert triggered", "agent", agent.ID, "message", msg)
				}
			}
		}
	}
}

func getMetricValue(m *models.Metric, metric string) float64 {
	switch metric {
	case "cpu_percent":
		return m.CPUPercent
	case "memory_percent":
		return m.MemoryPercent
	case "disk_percent":
		return m.DiskPercent
	default:
		return 0
	}
}

func evaluate(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	default:
		return false
	}
}
