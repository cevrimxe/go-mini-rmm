package update

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cevrimxe/go-mini-rmm/web"
)

// LatestVersion can be set at build time or via config
var LatestVersion = "dev"

// BinaryDir is the directory where agent binaries are stored for download
var BinaryDir = "./binaries"

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

	tmpl, err := template.New("install").Parse(string(tmplBytes))
	if err != nil {
		http.Error(w, "template parse error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]string{"ServerURL": serverURL}); err != nil {
		http.Error(w, "template execute error", http.StatusInternalServerError)
		return
	}
	out := buf.String()
	if out == "" {
		http.Error(w, "install script produced empty output", http.StatusInternalServerError)
		return
	}
	w.Write([]byte(out))
}

func (h *Handler) InstallScript(w http.ResponseWriter, r *http.Request) {
	h.serveInstallScript(w, r, "install.sh")
}

func (h *Handler) InstallScriptPS(w http.ResponseWriter, r *http.Request) {
	h.serveInstallScript(w, r, "install.ps1")
}
