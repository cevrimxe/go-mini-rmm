package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

const checkInterval = 5 * time.Minute

type Updater struct {
	serverURL string
	version   string
	client    *http.Client
}

type UpdateCheckResponse struct {
	UpdateAvailable bool   `json:"update_available"`
	LatestVersion   string `json:"latest_version"`
}

func New(serverURL, version string) *Updater {
	return &Updater{
		serverURL: serverURL,
		version:   version,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (u *Updater) Run(ctx context.Context) {
	// Check immediately
	u.checkAndUpdate()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.checkAndUpdate()
		}
	}
}

func (u *Updater) checkAndUpdate() {
	resp, err := u.client.Get(fmt.Sprintf("%s/api/v1/update/check?version=%s&os=%s&arch=%s",
		u.serverURL, u.version, runtime.GOOS, runtime.GOARCH))
	if err != nil {
		slog.Debug("update check failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var check UpdateCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&check); err != nil {
		return
	}

	if !check.UpdateAvailable {
		slog.Debug("no update available")
		return
	}

	slog.Info("update available", "current", u.version, "latest", check.LatestVersion)
	if err := u.downloadAndReplace(); err != nil {
		slog.Error("update failed", "error", err)
	}
}

func (u *Updater) downloadAndReplace() error {
	resp, err := u.client.Get(fmt.Sprintf("%s/api/v1/update/download?os=%s&arch=%s",
		u.serverURL, runtime.GOOS, runtime.GOARCH))
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download status: %d", resp.StatusCode)
	}

	// Write to temp file
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	tmpPath := execPath + ".new"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	f.Close()

	// Make executable on unix
	if runtime.GOOS != "windows" {
		os.Chmod(tmpPath, 0755)
	}

	// Replace and restart
	oldPath := execPath + ".old"
	os.Remove(oldPath)

	if err := os.Rename(execPath, oldPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename old: %w", err)
	}
	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Rename(oldPath, execPath) // rollback
		return fmt.Errorf("rename new: %w", err)
	}

	slog.Info("update applied, restarting...")
	os.Remove(oldPath)

	// Restart self
	cmd := exec.Command(execPath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	os.Exit(0)

	return nil
}
