package api

import (
	"net/http"

	"github.com/cevrimxe/go-mini-rmm/internal/server/alert"
	"github.com/cevrimxe/go-mini-rmm/internal/server/db"
	"github.com/cevrimxe/go-mini-rmm/internal/server/update"
	"github.com/cevrimxe/go-mini-rmm/internal/server/ws"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(store *db.Store, hub *ws.Hub, alertEngine *alert.Engine) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))

	agentHandler := &AgentHandler{Store: store}
	cmdHandler := &CommandHandler{Store: store, Hub: hub}
	alertHandler := &AlertHandler{Store: store, Engine: alertEngine}
	updateHandler := &update.Handler{}
	webHandler := NewWebHandler(store, hub)
	authHandler := NewAuthHandler(store)

	// ── Public routes (no auth) ──
	r.Get("/login", authHandler.LoginPage)
	r.Post("/login", authHandler.Login)
	r.Get("/setup", authHandler.SetupPage)
	r.Post("/setup", authHandler.Setup)

	// Install scripts
	r.Get("/install.sh", updateHandler.InstallScript)
	r.Get("/install.ps1", updateHandler.InstallScriptPS)

	// Agent communication (API key, not user auth)
	r.Post("/api/v1/heartbeat", agentHandler.Heartbeat)
	r.Get("/api/v1/update/check", updateHandler.Check)
	r.Get("/api/v1/update/download", updateHandler.Download)

	// WebSocket
	r.Get("/ws/agent", func(w http.ResponseWriter, r *http.Request) {
		hub.HandleAgentWS(w, r)
	})

	// ── Protected routes (user session required) ──
	r.Group(func(r chi.Router) {
		r.Use(authHandler.RequireAuth)

		// Web UI
		r.Get("/", webHandler.Dashboard)
		r.Get("/ui/agents/{id}", webHandler.AgentDetail)
		r.Get("/ui/alerts", webHandler.Alerts)
		r.Post("/logout", authHandler.Logout)

		// Management API
		r.Get("/api/v1/agents", agentHandler.List)
		r.Get("/api/v1/agents/{id}", agentHandler.Get)
		r.Delete("/api/v1/agents/{id}", agentHandler.Delete)
		r.Get("/api/v1/agents/{id}/metrics", agentHandler.Metrics)
		r.Post("/api/v1/agents/{id}/command", cmdHandler.Send)
		r.Get("/api/v1/agents/{id}/commands", cmdHandler.List)
		r.Get("/api/v1/alerts", alertHandler.ListAlerts)
		r.Get("/api/v1/alerts/rules", alertHandler.ListRules)
		r.Post("/api/v1/alerts/rules", alertHandler.CreateRule)
		r.Delete("/api/v1/alerts/rules/{id}", alertHandler.DeleteRule)
	})

	// Static files
	fileServer(r)

	return r
}
