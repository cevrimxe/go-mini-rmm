package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
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

		switch msg.Type {
		case "command":
			go e.executeCommand(conn, msg.Payload)
		case "file_download":
			go e.handleFileDownload(conn, msg.Payload)
		case "file_upload":
			go e.handleFileUpload(conn, msg.Payload)
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

// handleFileDownload downloads a file from the server and writes it to the specified path
func (e *Executor) handleFileDownload(conn *websocket.Conn, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	var dlPayload struct {
		TransferID int64  `json:"transfer_id"`
		FileName   string `json:"file_name"`
		RemotePath string `json:"remote_path"`
		FileSize   int64  `json:"file_size"`
	}
	if err := json.Unmarshal(data, &dlPayload); err != nil {
		slog.Warn("invalid file_download payload", "error", err)
		return
	}

	slog.Info("downloading file from server", "transfer_id", dlPayload.TransferID, "remote_path", dlPayload.RemotePath)

	success := true
	errMsg := ""

	// Download from server
	downloadURL := fmt.Sprintf("%s/api/v1/files/%d/serve", e.serverURL, dlPayload.TransferID)
	resp, err := http.Get(downloadURL)
	if err != nil {
		success = false
		errMsg = fmt.Sprintf("download request failed: %v", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			success = false
			errMsg = fmt.Sprintf("download returned status %d", resp.StatusCode)
		} else {
			// Ensure directory exists
			dir := filepath.Dir(dlPayload.RemotePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				success = false
				errMsg = fmt.Sprintf("create directory failed: %v", err)
			} else {
				f, err := os.Create(dlPayload.RemotePath)
				if err != nil {
					success = false
					errMsg = fmt.Sprintf("create file failed: %v", err)
				} else {
					_, err = io.Copy(f, resp.Body)
					f.Close()
					if err != nil {
						success = false
						errMsg = fmt.Sprintf("write file failed: %v", err)
					}
				}
			}
		}
	}

	if success {
		slog.Info("file downloaded successfully", "path", dlPayload.RemotePath)
	} else {
		slog.Error("file download failed", "error", errMsg)
	}

	// Send result back
	result := models.WSMessage{
		Type: "file_download_result",
		Payload: map[string]interface{}{
			"transfer_id": dlPayload.TransferID,
			"success":     success,
			"error":       errMsg,
		},
	}
	resultData, _ := json.Marshal(result)
	if err := conn.WriteMessage(websocket.TextMessage, resultData); err != nil {
		slog.Error("failed to send file download result", "error", err)
	}
}

// handleFileUpload reads a file from the agent and uploads it to the server
func (e *Executor) handleFileUpload(conn *websocket.Conn, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	var ulPayload struct {
		TransferID int64  `json:"transfer_id"`
		RemotePath string `json:"remote_path"`
	}
	if err := json.Unmarshal(data, &ulPayload); err != nil {
		slog.Warn("invalid file_upload payload", "error", err)
		return
	}

	slog.Info("uploading file to server", "transfer_id", ulPayload.TransferID, "path", ulPayload.RemotePath)

	success := true
	errMsg := ""

	// Read the file
	f, err := os.Open(ulPayload.RemotePath)
	if err != nil {
		success = false
		errMsg = fmt.Sprintf("open file failed: %v", err)
	} else {
		defer f.Close()

		// Create multipart upload
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		part, err := writer.CreateFormFile("file", filepath.Base(ulPayload.RemotePath))
		if err != nil {
			success = false
			errMsg = fmt.Sprintf("create form file failed: %v", err)
		} else {
			if _, err := io.Copy(part, f); err != nil {
				success = false
				errMsg = fmt.Sprintf("copy file data failed: %v", err)
			} else {
				writer.Close()

				uploadURL := fmt.Sprintf("%s/api/v1/files/%d/receive", e.serverURL, ulPayload.TransferID)
				resp, err := http.Post(uploadURL, writer.FormDataContentType(), &buf)
				if err != nil {
					success = false
					errMsg = fmt.Sprintf("upload request failed: %v", err)
				} else {
					resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						success = false
						errMsg = fmt.Sprintf("upload returned status %d", resp.StatusCode)
					}
				}
			}
		}
	}

	if success {
		slog.Info("file uploaded successfully", "path", ulPayload.RemotePath)
	} else {
		slog.Error("file upload failed", "error", errMsg)
	}

	// Send result back
	result := models.WSMessage{
		Type: "file_upload_result",
		Payload: map[string]interface{}{
			"transfer_id": ulPayload.TransferID,
			"success":     success,
			"error":       errMsg,
		},
	}
	resultData, _ := json.Marshal(result)
	if err := conn.WriteMessage(websocket.TextMessage, resultData); err != nil {
		slog.Error("failed to send file upload result", "error", err)
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
