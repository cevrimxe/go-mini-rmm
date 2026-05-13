package update

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cevrimxe/go-mini-rmm/web"
)

// LatestVersion is set at build time via ldflags. Can be overridden at runtime
// with the RMM_AGENT_VERSION environment variable.
var LatestVersion = "dev"

// BinaryDir is the directory where agent binaries are stored for download.
// Can be overridden with the RMM_BINARY_DIR environment variable.
var BinaryDir = "./binaries"

func init() {
	if v := os.Getenv("RMM_AGENT_VERSION"); v != "" {
		LatestVersion = v
	}
	if d := os.Getenv("RMM_BINARY_DIR"); d != "" {
		BinaryDir = d
	}
}

type UpdateCheckResponse struct {
	UpdateAvailable bool   `json:"update_available"`
	LatestVersion   string `json:"latest_version"`
}

type Handler struct{}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	currentVersion := r.URL.Query().Get("version")

	resp := UpdateCheckResponse{
		UpdateAvailable: currentVersion != "" && currentVersion != LatestVersion && LatestVersion != "dev",
		LatestVersion:   LatestVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	goos := r.URL.Query().Get("os")
	goarch := r.URL.Query().Get("arch")

	if goos == "" {
		goos = "linux"
	}
	if goarch == "" {
		goarch = "amd64"
	}

	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}

	filename := "agent-" + goos + "-" + goarch + ext
	binPath := filepath.Join(BinaryDir, filename)

	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		http.Error(w, "binary not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	http.ServeFile(w, r, binPath)
}

func (h *Handler) serveInstallScript(w http.ResponseWriter, r *http.Request, filename string) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	serverURL := scheme + "://" + r.Host
	if r.Host == "" {
		serverURL = "http://localhost:9090"
	}

	tmplBytes, err := web.TemplateFS.ReadFile("templates/" + filename)
	if err != nil {
		http.Error(w, "install script not found", http.StatusInternalServerError)
		return
	}
	if len(tmplBytes) == 0 {
		http.Error(w, "install script empty", http.StatusInternalServerError)
		return
	}

	// Placeholder replace (no Go template - script has $ and braces that confuse template engine)
	out := strings.ReplaceAll(string(tmplBytes), "__RMM_SERVER_URL__", serverURL)
	if out == "" {
		http.Error(w, "install script produced empty output", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(out))
}

func (h *Handler) InstallScript(w http.ResponseWriter, r *http.Request) {
	h.serveInstallScript(w, r, "install.sh")
}

func (h *Handler) InstallScriptPS(w http.ResponseWriter, r *http.Request) {
	h.serveInstallScript(w, r, "install.ps1")
}
