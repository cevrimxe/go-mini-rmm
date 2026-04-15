package models

import "time"

type TransferDirection string
type TransferStatus string

const (
	TransferToAgent   TransferDirection = "to_agent"   // Server → Agent (upload to agent)
	TransferFromAgent TransferDirection = "from_agent"  // Agent → Server (download from agent)

	TransferPending     TransferStatus = "pending"
	TransferTransferring TransferStatus = "transferring"
	TransferDone        TransferStatus = "done"
	TransferFailed      TransferStatus = "failed"
)

type FileTransfer struct {
	ID          int64             `json:"id"`
	AgentID     string            `json:"agent_id"`
	FileName    string            `json:"file_name"`
	FileSize    int64             `json:"file_size"`
	Direction   TransferDirection `json:"direction"`
	Status      TransferStatus    `json:"status"`
	StoragePath string            `json:"storage_path"` // server-side path
	RemotePath  string            `json:"remote_path"`  // agent-side path
	Error       string            `json:"error"`
	CreatedAt   time.Time         `json:"created_at"`
}
