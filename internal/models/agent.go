package models

import "time"

type AgentStatus string

const (
	AgentOnline  AgentStatus = "online"
	AgentOffline AgentStatus = "offline"
)

type Agent struct {
	ID            string      `json:"id"`
	Hostname      string      `json:"hostname"`
	OS            string      `json:"os"`
	IP            string      `json:"ip"`
	Version       string      `json:"version"`
	LastHeartbeat time.Time   `json:"last_heartbeat"`
	Status        AgentStatus `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
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
	Hostname      string  `json:"hostname"`
	OS            string  `json:"os"`
	IP            string  `json:"ip"`
	Version       string  `json:"version"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
}
