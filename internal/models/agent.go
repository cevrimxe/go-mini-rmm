package models

import "time"

type AgentStatus string

const (
	AgentOnline  AgentStatus = "online"
	AgentOffline AgentStatus = "offline"
)

type Agent struct {
	ID            string      `json:"id"`
	DisplayName   string      `json:"display_name"` // Kullanıcının verdiği isim (kurulumda)
	Hostname      string      `json:"hostname"`
	OS            string      `json:"os"`
	IP            string      `json:"ip"`
	Version       string      `json:"version"`
	LastHeartbeat time.Time   `json:"last_heartbeat"`
	Status        AgentStatus `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
}

// Name returns display name if set, else agent ID (key).
func (a *Agent) Name() string {
	if a.DisplayName != "" {
		return a.DisplayName
	}
	return a.ID
}

type Metric struct {
	ID            int64     `json:"id"`
	AgentID       string    `json:"agent_id"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	DiskPercent   float64   `json:"disk_percent"`
	Timestamp     time.Time `json:"timestamp"`
}

type HeartbeatPayload struct {
	AgentID       string  `json:"agent_id"`
	DisplayName   string  `json:"display_name"` // Kurulumda girilen isim
	Hostname      string  `json:"hostname"`
	OS            string  `json:"os"`
	IP            string  `json:"ip"`
	Version       string  `json:"version"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
}
