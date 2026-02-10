package executor

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/cevrimxe/go-mini-rmm/internal/models"
	"github.com/gorilla/websocket"
)

type Executor struct {
	serverURL string
	agentKey  string
}

func New(serverURL, agentKey string) *Executor {
	return &Executor{
		serverURL: serverURL,
		agentKey:  agentKey,
	}
}

func (e *Executor) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			e.connectAndListen(ctx)
			// Reconnect after 5 seconds
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	}
}

func (e *Executor) connectAndListen(ctx context.Context) {
	wsURL := e.buildWSURL()
	slog.Info("connecting to ws", "url", wsURL)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		slog.Warn("ws connect failed", "error", err)
		return
	}
	defer conn.Close()

	slog.Info("ws connected")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			slog.Warn("ws read error", "error", err)
			return
		}

		var msg models.WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			slog.Warn("invalid ws message", "error", err)
			continue
		}

		if msg.Type == "command" {
			go e.executeCommand(conn, msg.Payload)
		}
	}
}

func (e *Executor) executeCommand(conn *websocket.Conn, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	var cmdPayload struct {
		CommandID int64  `json:"command_id"`
		Command   string `json:"command"`
	}
	if err := json.Unmarshal(data, &cmdPayload); err != nil {
		slog.Warn("invalid command payload", "error", err)
		return
	}

	slog.Info("executing command", "command_id", cmdPayload.CommandID, "command", cmdPayload.Command)

	// Execute based on OS
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", cmdPayload.Command)
	} else {
		cmd = exec.Command("sh", "-c", cmdPayload.Command)
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
			stderr.WriteString(err.Error())
		}
	}

	result := models.WSMessage{
		Type: "command_result",
		Payload: models.CommandResult{
			CommandID: cmdPayload.CommandID,
			Stdout:    stdout.String(),
			Stderr:    stderr.String(),
			ExitCode:  exitCode,
		},
	}

	resultData, _ := json.Marshal(result)
	if err := conn.WriteMessage(websocket.TextMessage, resultData); err != nil {
		slog.Error("failed to send command result", "error", err)
	}
}

func (e *Executor) buildWSURL() string {
	u, _ := url.Parse(e.serverURL)
	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}
	return scheme + "://" + u.Host + "/ws/agent?agent_id=" + url.QueryEscape(e.agentKey)
}
