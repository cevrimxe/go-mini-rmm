# Go Mini RMM

Lightweight **Remote Monitoring & Management** tool written in Go.

## Features

- Agent heartbeat + system metrics (CPU, RAM, Disk)
- Remote command execution over WebSocket
- Alert engine (threshold rules + offline detection)
- Embedded web dashboard (htmx + PicoCSS)
- Agent auto-update from server
- Docker Compose deployment; Watchtower for auto-updates on push

## Security

- Dashboard requires login: first run → `/setup` to create a user, then `/login`.
- Agent endpoints (heartbeat, update, WebSocket) are public; agents use an auto-generated key (ID).

## Architecture

```
Agent (binary)                    Server
┌─────────────────────┐          ┌──────────────────────────┐
│ Collector (gopsutil) │──HTTP──► │ REST API (chi)          │
│ Heartbeat (30s)      │          │ WebSocket Hub            │
│ Executor (commands)  │◄──WS───  │ Alert Engine             │
│ Auto-Updater         │          │ Embedded Web UI          │
└─────────────────────┘          │ SQLite DB                │
                                 └──────────────────────────┘
```

## Server setup (production)

On a Linux server (root):

```bash
curl -sSL https://raw.githubusercontent.com/cevrimxe/go-mini-rmm/main/deploy/setup-docker.sh | bash
```

- Listens on port **9090**
- Dashboard: `http://SERVER_IP:9090`
- First visit: create a user at `/setup`, then log in at `/login`
- Images are pulled from GHCR; Watchtower updates the server container when you push to `main`

Agents are **not** installed by this script; you install them on each machine separately (see below).

## Agent setup

You are only asked for an **agent name**; the key (ID) is generated automatically.

### Linux (interactive)

```bash
curl -sSL http://SERVER_IP:9090/install.sh | sudo bash
```

### Linux (non-interactive)

```bash
curl -sSL http://SERVER_IP:9090/install.sh | sudo bash -- "My Agent Name"
```

### Windows (PowerShell, run as admin)

```powershell
irm http://SERVER_IP:9090/install.ps1 | iex
```

Agents pull updates from the server; the server image includes agent binaries for the download endpoint.

## Reset server (teardown + reinstall)

```bash
curl -sSL https://raw.githubusercontent.com/cevrimxe/go-mini-rmm/main/deploy/teardown.sh | bash
curl -sSL https://raw.githubusercontent.com/cevrimxe/go-mini-rmm/main/deploy/setup-docker.sh | bash
```

## Local development

```bash
# Build
go build -o bin/server ./cmd/server
go build -o bin/agent ./cmd/agent

# Run server
./bin/server -addr :8080 -db rmm.db

# Run agent (other terminal)
./bin/agent -server http://localhost:8080 -key dev-agent-1 -name "Local"
```

Dashboard: `http://localhost:8080` (use `/setup` on first run).

### Docker Compose (local)

```bash
cd deploy
docker compose up -d
```

Runs server on 8080 with two demo agents.

## Web UI

| Page        | URL                |
|------------|--------------------|
| Setup      | `/setup`           |
| Login      | `/login`           |
| Dashboard  | `/`                |
| Agent detail | `/ui/agents/{id}` |
| Alerts     | `/ui/alerts`       |

## Tech stack

- Go 1.25+, chi, gorilla/websocket, gopsutil, modernc.org/sqlite (no CGO), htmx, PicoCSS
