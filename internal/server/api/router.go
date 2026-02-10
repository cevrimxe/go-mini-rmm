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

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/heartbeat", agentHandler.Heartbeat)

		r.Get("/agents", agentHandler.List)
		r.Get("/agents/{id}", agentHandler.Get)
		r.Delete("/agents/{id}", agentHandler.Delete)
		r.Get("/agents/{id}/metrics", agentHandler.Metrics)

		r.Post("/agents/{id}/command", cmdHandler.Send)
		r.Get("/agents/{id}/commands", cmdHandler.List)

		r.Get("/alerts", alertHandler.ListAlerts)
		r.Get("/alerts/rules", alertHandler.ListRules)
		r.Post("/alerts/rules", alertHandler.CreateRule)
		r.Delete("/alerts/rules/{id}", alertHandler.DeleteRule)

		r.Get("/update/check", updateHandler.Check)
		r.Get("/update/download", updateHandler.Download)
	})

	// WebSocket
	r.Get("/ws/agent", func(w http.ResponseWriter, r *http.Request) {
		hub.HandleAgentWS(w, r)
	})

	// Install script (curl one-liner)
	r.Get("/install.sh", updateHandler.InstallScript)

	// Web UI routes (under /ui to avoid conflicts with API)
	r.Get("/", webHandler.Dashboard)
	r.Get("/ui/agents/{id}", webHandler.AgentDetail)
	r.Get("/ui/alerts", webHandler.Alerts)

	// Static files
	fileServer(r)

	return r
}
