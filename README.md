# Go Mini RMM

Lightweight Remote Monitoring and Management tool built with Go.

## Features
- **Agent heartbeat** & system metrics collection (CPU, RAM, Disk)
- **Remote command execution** via WebSocket
- **Alert engine** with custom threshold rules + offline detection
- **Embedded web dashboard** (htmx + PicoCSS, dark theme)
- **Agent auto-update** mechanism
- **Docker Compose** deployment ready

## Architecture

```
Agent (lightweight binary)          Server (API + Dashboard)
┌─────────────────────┐            ┌──────────────────────────┐
│ Collector (gopsutil) │──HTTP──►  │ REST API (chi router)    │
│ Heartbeat (30s)      │           │ WebSocket Hub            │
│ Executor (commands)  │◄──WS───  │ Alert Engine             │
│ Auto-Updater         │           │ Embedded Web UI          │
└─────────────────────┘            │ SQLite DB                │
                                   └──────────────────────────┘
```

## Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose (optional)

### Build from source
```bash
# Server
go build -o bin/server ./cmd/server

# Agent
go build -o bin/agent ./cmd/agent
```

### Run locally
```bash
# Terminal 1: Start server
./bin/server -addr :8080 -db rmm.db

# Terminal 2: Start agent
./bin/agent -server http://localhost:8080 -key my-agent-1
```

Then open http://localhost:8080 for the dashboard.

### Docker Compose
```bash
cd deploy
docker-compose up -d
```

This starts the server on port 8080 with 2 demo agents.

### Cross-compile agents
```bash
chmod +x scripts/build.sh
./scripts/build.sh 1.0.0
```

Builds agent binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/heartbeat` | Agent heartbeat |
| GET | `/api/v1/agents` | List all agents |
| GET | `/api/v1/agents/{id}` | Agent details |
| DELETE | `/api/v1/agents/{id}` | Remove agent |
| GET | `/api/v1/agents/{id}/metrics` | Agent metrics history |
| POST | `/api/v1/agents/{id}/command` | Send remote command |
| GET | `/api/v1/agents/{id}/commands` | Command history |
| GET | `/api/v1/alerts` | List alerts |
| GET | `/api/v1/alerts/rules` | List alert rules |
| POST | `/api/v1/alerts/rules` | Create alert rule |
| DELETE | `/api/v1/alerts/rules/{id}` | Delete alert rule |
| GET | `/api/v1/update/check` | Check for agent updates |
| GET | `/api/v1/update/download` | Download agent binary |
| WS | `/ws/agent?agent_id=X` | Agent WebSocket connection |

## Web UI

| Page | URL |
|------|-----|
| Dashboard | `/` |
| Agent Detail | `/ui/agents/{id}` |
| Alerts | `/ui/alerts` |

## Project Structure
```
cmd/server/          - Server entry point
cmd/agent/           - Agent entry point
internal/server/api/ - REST API handlers + web UI handler
internal/server/ws/  - WebSocket hub
internal/server/db/  - SQLite data layer
internal/server/alert/ - Alert engine
internal/server/update/ - Agent update server
internal/agent/collector/ - System metrics collector
internal/agent/heartbeat/ - Periodic heartbeat
internal/agent/executor/  - Remote command executor
internal/agent/updater/   - Agent auto-updater
internal/models/     - Shared data models
web/templates/       - HTML templates (go:embed)
web/static/          - Static assets (go:embed)
deploy/              - Dockerfile + docker-compose
scripts/             - Build scripts
```

## Tech Stack
- **Go** 1.21+ with `log/slog`
- **chi** - HTTP router
- **gorilla/websocket** - WebSocket
- **gopsutil** - System metrics
- **modernc.org/sqlite** - Pure Go SQLite (no CGO)
- **htmx** + **PicoCSS** - Minimal frontend
