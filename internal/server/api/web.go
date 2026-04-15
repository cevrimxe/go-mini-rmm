package api

import (
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/cevrimxe/go-mini-rmm/internal/models"
	"github.com/cevrimxe/go-mini-rmm/internal/server/db"
	"github.com/cevrimxe/go-mini-rmm/internal/server/ws"
	"github.com/cevrimxe/go-mini-rmm/web"
	"github.com/go-chi/chi/v5"
)

type WebHandler struct {
	store     *db.Store
	hub       *ws.Hub
	templates map[string]*template.Template
}

type agentRow struct {
	Agent  models.Agent
	Metric *models.Metric
}

var funcMap = template.FuncMap{
	"timeAgo":     timeAgo,
	"metricColor": metricColor,
	"formatBytes": formatBytes,
}

func parseTemplate(name string) *template.Template {
	return template.Must(
		template.New("layout.html").Funcs(funcMap).ParseFS(
			web.TemplateFS,
			"templates/layout.html",
			"templates/"+name,
		),
	)
}

func NewWebHandler(store *db.Store, hub *ws.Hub) *WebHandler {
	templates := map[string]*template.Template{
		"dashboard":    parseTemplate("dashboard.html"),
		"agent_detail": parseTemplate("agent_detail.html"),
		"alerts":       parseTemplate("alerts.html"),
		"audit_logs":   parseTemplate("audit_logs.html"),
	}

	return &WebHandler{store: store, hub: hub, templates: templates}
}

func (h *WebHandler) render(w http.ResponseWriter, name string, data map[string]interface{}) {
	tmpl, ok := h.templates[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		slog.Error("template error", "template", name, "error", err)
	}
}

func (h *WebHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	agents, err := h.store.ListAgents()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if agents == nil {
		agents = []models.Agent{}
	}

	var rows []agentRow
	online, offline := 0, 0
	for _, a := range agents {
		m, _ := h.store.GetLatestMetric(a.ID)
		rows = append(rows, agentRow{Agent: a, Metric: m})
		if a.Status == models.AgentOnline {
			online++
		} else {
			offline++
		}
	}

	alerts, _ := h.store.ListAlerts(100)
	activeAlerts := 0
	for _, a := range alerts {
		if !a.Resolved {
			activeAlerts++
		}
	}

	h.render(w, "dashboard", map[string]interface{}{
		"Title":         "Dashboard",
		"Agents":        rows,
		"TotalAgents":   len(agents),
		"OnlineAgents":  online,
		"OfflineAgents": offline,
		"ActiveAlerts":  activeAlerts,
	})
}

func (h *WebHandler) AgentDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	agent, err := h.store.GetAgent(id)
	if err != nil || agent == nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	metric, _ := h.store.GetLatestMetric(id)
	commands, _ := h.store.GetCommandsByAgent(id, 20)
	if commands == nil {
		commands = []models.Command{}
	}

	transfers, _ := h.store.GetFileTransfersByAgent(id, 20)
	if transfers == nil {
		transfers = []models.FileTransfer{}
	}

	h.render(w, "agent_detail", map[string]interface{}{
		"Title":         agent.Hostname,
		"Agent":         agent,
		"Metric":        metric,
		"Commands":      commands,
		"FileTransfers": transfers,
	})
}

func (h *WebHandler) Alerts(w http.ResponseWriter, r *http.Request) {
	alerts, _ := h.store.ListAlerts(100)
	if alerts == nil {
		alerts = []models.Alert{}
	}

	rules, _ := h.store.ListAlertRules()
	if rules == nil {
		rules = []models.AlertRule{}
	}

	agents, _ := h.store.ListAgents()
	if agents == nil {
		agents = []models.Agent{}
	}

	h.render(w, "alerts", map[string]interface{}{
		"Title":  "Alerts",
		"Alerts": alerts,
		"Rules":  rules,
		"Agents": agents,
	})
}

func (h *WebHandler) AuditLogs(w http.ResponseWriter, r *http.Request) {
	logs, _ := h.store.GetAuditLogs(100)
	if logs == nil {
		logs = []models.AuditLog{}
	}

	h.render(w, "audit_logs", map[string]interface{}{
		"Title": "Audit Logs",
		"Logs":  logs,
	})
}

// fileServer serves static files embedded in the binary
func fileServer(r chi.Router) {
	staticFS, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		slog.Error("static fs error", "error", err)
		return
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
}

// Template helper functions

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func metricColor(pct float64) string {
	switch {
	case pct >= 90:
		return "fill-red"
	case pct >= 70:
		return "fill-yellow"
	default:
		return "fill-green"
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
