package models

import "time"

type CommandStatus string

const (
	CommandPending  CommandStatus = "pending"
	CommandRunning  CommandStatus = "running"
	CommandDone     CommandStatus = "done"
	CommandFailed   CommandStatus = "failed"
)

type Command struct {
	ID        int64         `json:"id"`
	AgentID   string        `json:"agent_id"`
	Command   string        `json:"command"`
	Stdout    string        `json:"stdout"`
	Stderr    string        `json:"stderr"`
	ExitCode  int           `json:"exit_code"`
	Status    CommandStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}

type CommandRequest struct {
	Command string `json:"command"`
}

type CommandResult struct {
	CommandID int64  `json:"command_id"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  int    `json:"exit_code"`
}

// WSMessage is the WebSocket message envelope
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
