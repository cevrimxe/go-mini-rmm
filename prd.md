# Go Mini RMM - Product Requirements Document

## Proje Özeti
Go ile yazılmış, agent-server mimarisinde mini bir Remote Monitoring and Management (RMM) aracı.

## Özellikler

### Tamamlanan
- [x] Proje iskeleti ve module init
- [x] Ortak veri modelleri (Agent, Metric, Command, Alert)
- [x] SQLite veritabanı katmanı + migration
- [x] Agent: Sistem bilgisi collector (CPU, RAM, Disk, Host)
- [x] Agent: Periyodik heartbeat (30s)
- [x] Agent: Uzaktan komut çalıştırma (WebSocket)
- [x] Agent: Auto-update mekanizması
- [x] Server: REST API (heartbeat, agent CRUD, komut, alert)
- [x] Server: WebSocket hub (gerçek zamanlı komut iletimi)
- [x] Server: Alert engine (kural tabanlı + offline algılama)
- [x] Server: Embedded Web UI (htmx + PicoCSS, dark theme)
- [x] Server: Agent update endpoint'leri
- [x] Docker Compose deployment
- [x] Multi-stage Dockerfile'lar (server + agent)
- [x] Cross-compile build script

### Gelecek İyileştirmeler
- [ ] Agent authentication (JWT/API key validation)
- [ ] HTTPS/TLS support
- [ ] Prometheus metrics exporter
- [ ] Grafana dashboard template
- [ ] Agent grouping / tagging
- [ ] Scheduled tasks (cron-like)
- [ ] File transfer (upload/download)
- [ ] Multi-tenant support
- [ ] Rate limiting
- [ ] Audit logging

## Teknoloji Stack
- **Dil**: Go 1.21+
- **Router**: go-chi/chi v5
- **DB**: SQLite (modernc.org/sqlite - pure Go)
- **Metrics**: shirou/gopsutil v3
- **WebSocket**: gorilla/websocket
- **UI**: htmx + PicoCSS (dark theme)
- **Logging**: log/slog (stdlib)
- **Deployment**: Docker Compose
