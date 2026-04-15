package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cevrimxe/go-mini-rmm/internal/models"
	"github.com/cevrimxe/go-mini-rmm/internal/server/db"
	"github.com/cevrimxe/go-mini-rmm/internal/server/ws"
	"github.com/go-chi/chi/v5"
)

const maxUploadSize = 50 << 20 // 50 MB

type FileTransferHandler struct {
	Store      *db.Store
	Hub        *ws.Hub
	UploadDir  string
}

func NewFileTransferHandler(store *db.Store, hub *ws.Hub, uploadDir string) *FileTransferHandler {
	os.MkdirAll(uploadDir, 0755)
	return &FileTransferHandler{Store: store, Hub: hub, UploadDir: uploadDir}
}

// Upload handles multipart file upload from dashboard → server → agent
func (h *FileTransferHandler) Upload(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")

	// Check agent exists
	agent, err := h.Store.GetAgent(agentID)
	if err != nil || agent == nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "file too large (max 50MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	remotePath := r.FormValue("remote_path")
	if remotePath == "" {
		http.Error(w, "remote_path required", http.StatusBadRequest)
		return
	}

	// Save file to upload dir
	storageName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	storagePath := filepath.Join(h.UploadDir, storageName)

	dst, err := os.Create(storagePath)
	if err != nil {
		slog.Error("create upload file failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		slog.Error("write upload file failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Create DB record
	ft, err := h.Store.CreateFileTransfer(agentID, header.Filename, written, models.TransferToAgent, storagePath, remotePath)
	if err != nil {
		slog.Error("create file transfer failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Audit log
	user := GetUserFromContext(r)
	username := "system"
	if user != nil {
		username = user.Username
	}
	details := fmt.Sprintf(`{"file":"%s","remote_path":"%s","size":%d}`, header.Filename, remotePath, written)
	if err := h.Store.InsertAuditLog(username, "file_upload", agentID, details); err != nil {
		slog.Error("failed to insert audit log", "error", err)
	}

	// Send download command to agent via WebSocket
	msg := models.WSMessage{
		Type: "file_download",
		Payload: map[string]interface{}{
			"transfer_id": ft.ID,
			"file_name":   header.Filename,
			"remote_path": remotePath,
			"file_size":   written,
		},
	}
	if err := h.Hub.SendToAgent(agentID, msg); err != nil {
		slog.Warn("agent not connected via ws for file transfer", "agent_id", agentID, "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ft)
}

// RequestDownload requests a file from the agent
func (h *FileTransferHandler) RequestDownload(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")

	agent, err := h.Store.GetAgent(agentID)
	if err != nil || agent == nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	var req struct {
		RemotePath string `json:"remote_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if req.RemotePath == "" {
		http.Error(w, "remote_path required", http.StatusBadRequest)
		return
	}

	fileName := filepath.Base(req.RemotePath)
	ft, err := h.Store.CreateFileTransfer(agentID, fileName, 0, models.TransferFromAgent, "", req.RemotePath)
	if err != nil {
		slog.Error("create file transfer failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Audit log
	user := GetUserFromContext(r)
	username := "system"
	if user != nil {
		username = user.Username
	}
	details := fmt.Sprintf(`{"remote_path":"%s"}`, req.RemotePath)
	if err := h.Store.InsertAuditLog(username, "file_download_request", agentID, details); err != nil {
		slog.Error("failed to insert audit log", "error", err)
	}

	// Send upload command to agent
	msg := models.WSMessage{
		Type: "file_upload",
		Payload: map[string]interface{}{
			"transfer_id": ft.ID,
			"remote_path": req.RemotePath,
		},
	}
	if err := h.Hub.SendToAgent(agentID, msg); err != nil {
		slog.Warn("agent not connected via ws for file transfer", "agent_id", agentID, "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ft)
}

// List returns file transfer history for an agent
func (h *FileTransferHandler) List(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	transfers, err := h.Store.GetFileTransfersByAgent(agentID, limit)
	if err != nil {
		slog.Error("list file transfers failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if transfers == nil {
		transfers = []models.FileTransfer{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transfers)
}

// ServeFile serves a file for agent to download (agent pulls from server)
func (h *FileTransferHandler) ServeFile(w http.ResponseWriter, r *http.Request) {
	transferID, err := strconv.ParseInt(chi.URLParam(r, "transferID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid transfer id", http.StatusBadRequest)
		return
	}

	ft, err := h.Store.GetFileTransfer(transferID)
	if err != nil || ft == nil {
		http.Error(w, "transfer not found", http.StatusNotFound)
		return
	}

	if ft.StoragePath == "" {
		http.Error(w, "file not available", http.StatusNotFound)
		return
	}

	// Update status to transferring
	h.Store.UpdateFileTransferStatus(transferID, models.TransferTransferring, "")

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, ft.FileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, ft.StoragePath)
}

// ReceiveFile handles file upload from agent (agent pushes to server)
func (h *FileTransferHandler) ReceiveFile(w http.ResponseWriter, r *http.Request) {
	transferID, err := strconv.ParseInt(chi.URLParam(r, "transferID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid transfer id", http.StatusBadRequest)
		return
	}

	ft, err := h.Store.GetFileTransfer(transferID)
	if err != nil || ft == nil {
		http.Error(w, "transfer not found", http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "file too large (max 50MB)", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	storageName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), ft.FileName)
	storagePath := filepath.Join(h.UploadDir, storageName)

	dst, err := os.Create(storagePath)
	if err != nil {
		slog.Error("create receive file failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		slog.Error("write receive file failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.Store.UpdateFileTransferStorage(transferID, storagePath, written)
	h.Store.UpdateFileTransferStatus(transferID, models.TransferDone, "")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// DownloadFile serves a received file to the dashboard user
func (h *FileTransferHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	transferID, err := strconv.ParseInt(chi.URLParam(r, "transferID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid transfer id", http.StatusBadRequest)
		return
	}

	ft, err := h.Store.GetFileTransfer(transferID)
	if err != nil || ft == nil {
		http.Error(w, "transfer not found", http.StatusNotFound)
		return
	}

	if ft.Status != models.TransferDone || ft.StoragePath == "" {
		http.Error(w, "file not available yet", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, ft.FileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, ft.StoragePath)
}
