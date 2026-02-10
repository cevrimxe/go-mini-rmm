package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/cevrimxe/go-mini-rmm/internal/agent/collector"
)

const interval = 30 * time.Second

type Heartbeat struct {
	serverURL string
	agentKey  string
	version   string
	client    *http.Client
}

func New(serverURL, agentKey, version string) *Heartbeat {
	return &Heartbeat{
		serverURL: serverURL,
		agentKey:  agentKey,
		version:   version,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *Heartbeat) Run(ctx context.Context) {
	// Send first heartbeat immediately
	h.send()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.send()
		}
	}
}

func (h *Heartbeat) send() {
	payload := collector.Collect(h.agentKey, h.version)

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("heartbeat marshal error", "error", err)
		return
	}

	req, err := http.NewRequest("POST", h.serverURL+"/api/v1/heartbeat", bytes.NewReader(body))
	if err != nil {
		slog.Error("heartbeat request error", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Key", h.agentKey)

	resp, err := h.client.Do(req)
	if err != nil {
		slog.Warn("heartbeat send failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("heartbeat unexpected status", "status", resp.StatusCode)
		return
	}

	slog.Debug("heartbeat sent successfully")
}
