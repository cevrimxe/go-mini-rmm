package collector

import (
	"log/slog"

	"github.com/cevrimxe/go-mini-rmm/internal/models"
)

// Collect gathers all system metrics and host info into a HeartbeatPayload.
func Collect(agentID, displayName, version string) models.HeartbeatPayload {
	host, err := Host()
	if err != nil {
		slog.Warn("failed to collect host info", "error", err)
	}

	cpuPct, err := CPUPercent()
	if err != nil {
		slog.Warn("failed to collect cpu", "error", err)
	}

	memPct, err := MemoryPercent()
	if err != nil {
		slog.Warn("failed to collect memory", "error", err)
	}

	diskPct, err := DiskPercent()
	if err != nil {
		slog.Warn("failed to collect disk", "error", err)
	}

	return models.HeartbeatPayload{
		AgentID:       agentID,
		DisplayName:   displayName,
		Hostname:      host.Hostname,
		OS:            host.OS,
		IP:            host.IP,
		Version:       version,
		CPUPercent:    cpuPct,
		MemoryPercent: memPct,
		DiskPercent:   diskPct,
	}
}
