# Go Mini RMM

Go ile yazılmış, hafif bir **Remote Monitoring & Management (RMM)** aracı.

## Özellikler
- **Agent heartbeat** + sistem metrikleri (CPU / RAM / Disk)
- **Remote command** (WebSocket üzerinden)
- **Alert engine** (threshold kuralları + offline agent tespiti)
- **Embedded web dashboard** (htmx + PicoCSS)
- **Agent auto-update** (binary güncelleme)
- **Docker Compose** ile deploy (prod için Watchtower ile otomatik güncelleme)

## Güvenlik
- Dashboard **login zorunlu**: ilk açılışta `/setup` ile kullanıcı oluşturulur, sonra `/login`.
- Agent haberleşmesi (heartbeat, update, ws) **login istemez** (agent key ile çalışır).

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

## Hızlı Başlangıç (Local Dev)

### Prerequisites
- Go 1.25+
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

Sonra dashboard: `http://localhost:8080`

İlk açılışta kullanıcı oluşturma ekranı gelir: `/setup`.

### Docker Compose
```bash
cd deploy
docker-compose up -d
```

This starts the server on port 8080 with 2 demo agents.

## Production Kurulum (Remote Linux Sunucu)

Bu repo public olduğu için Docker image’lar **GHCR**’a pushlanır; sunucuda **Watchtower** yeni image gelince otomatik çeker/restart eder.

### 1) Sunucuda server’ı kur (tek komut)

Sunucuda (root):

```bash
curl -sSL https://raw.githubusercontent.com/cevrimxe/go-mini-rmm/main/deploy/setup-docker.sh | bash
```

- Default port: **9090**
- Dashboard: `http://SUNUCU_IP:9090`
- İlk açılışta `/setup` ile kullanıcı oluştur, sonra `/login` ile giriş yap.

> Not: Prod compose artık **agent kurmaz**. Agent’ları sen istediğin makinelere elle kurarsın.

### 2) Agent kurulumları (elle)

#### Linux agent (interactive, tek komut)

```bash
curl -sSL http://SUNUCU_IP:9090/install.sh | sudo bash
```

#### Linux agent (non-interactive)

```bash
curl -sSL http://SUNUCU_IP:9090/install.sh | sudo bash -- "AGENT_KEY" "AGENT_NAME"
```

- **Dashboard’da “Name” olarak görünen değer `AGENT_KEY`’dir** (agent_id).
- `AGENT_NAME` şu an sadece systemd servis açıklamasında kullanılır.

#### Windows agent (PowerShell, admin)

```powershell
irm http://SUNUCU_IP:9090/install.ps1 | iex
```

### Agent update nasıl çalışıyor?
- Agent, periyodik olarak server’ın `/api/v1/update/check` ve `/api/v1/update/download` endpoint’lerini kullanarak kendini günceller.
- Server Docker image’ı, agent binary’lerini de içerir (download endpoint’i buradan servis eder).

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/heartbeat` | Agent heartbeat (**public**) |
| GET | `/api/v1/update/check` | Agent update check (**public**) |
| GET | `/api/v1/update/download` | Agent binary download (**public**) |
| WS | `/ws/agent?agent_id=X` | Agent WebSocket (**public**) |
| GET | `/api/v1/agents` | List agents (**login required**) |
| GET | `/api/v1/agents/{id}` | Agent details (**login required**) |
| DELETE | `/api/v1/agents/{id}` | Remove agent (**login required**) |
| GET | `/api/v1/agents/{id}/metrics` | Metrics history (**login required**) |
| POST | `/api/v1/agents/{id}/command` | Send command (**login required**) |
| GET | `/api/v1/agents/{id}/commands` | Command history (**login required**) |
| GET | `/api/v1/alerts` | List alerts (**login required**) |
| GET | `/api/v1/alerts/rules` | List alert rules (**login required**) |
| POST | `/api/v1/alerts/rules` | Create alert rule (**login required**) |
| DELETE | `/api/v1/alerts/rules/{id}` | Delete alert rule (**login required**) |

## Web UI

| Page | URL |
|------|-----|
| Setup (first run) | `/setup` |
| Login | `/login` |
| Logout | `/logout` (POST) |
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

## Sıfırlama / Kaldırma (Sunucu)

Her şeyi kaldırıp temiz kurulum için (db dahil):

```bash
curl -sSL https://raw.githubusercontent.com/cevrimxe/go-mini-rmm/main/deploy/teardown.sh | bash
curl -sSL https://raw.githubusercontent.com/cevrimxe/go-mini-rmm/main/deploy/setup-docker.sh | bash
```

## Tech Stack
- **Go** 1.25+ with `log/slog`
- **chi** - HTTP router
- **gorilla/websocket** - WebSocket
- **gopsutil** - System metrics
- **modernc.org/sqlite** - Pure Go SQLite (no CGO)
- **htmx** + **PicoCSS** - Minimal frontend
