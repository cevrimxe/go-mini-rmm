package models

import "time"

type AlertRule struct {
	ID        int64     `json:"id"`
	Metric    string    `json:"metric"`    // cpu_percent, memory_percent, disk_percent
	Operator  string    `json:"operator"`  // >, <, >=, <=, ==
	Threshold float64   `json:"threshold"` // e.g. 90.0
	CreatedAt time.Time `json:"created_at"`
}

type Alert struct {
	ID        int64     `json:"id"`
	RuleID    int64     `json:"rule_id"`
	AgentID   string    `json:"agent_id"`
	Message   string    `json:"message"`
	Resolved  bool      `json:"resolved"`
	CreatedAt time.Time `json:"created_at"`
}

type AlertRuleRequest struct {
	Metric    string  `json:"metric"`
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
}
