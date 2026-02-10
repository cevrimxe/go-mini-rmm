#!/bin/bash
set -e

# Go Mini RMM - Agent Installer
# Usage: curl -sSL http://SERVER:8080/install.sh | bash -s -- <AGENT_KEY>

AGENT_KEY="${1:-}"
SERVER_URL="{{.ServerURL}}"
INSTALL_DIR="/opt/rmm"
SERVICE_NAME="rmm-agent"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[RMM]${NC} $1"; }
warn() { echo -e "${YELLOW}[RMM]${NC} $1"; }
err()  { echo -e "${RED}[RMM]${NC} $1"; exit 1; }

# Check root
if [ "$(id -u)" -ne 0 ]; then
    err "Root olarak calistirin: curl ... | sudo bash -s -- <KEY>"
fi

# Check key
if [ -z "$AGENT_KEY" ]; then
    err "Agent key gerekli. Kullanim: curl ... | sudo bash -s -- <AGENT_KEY>"
fi

# Detect arch
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *)       err "Desteklenmeyen mimari: $ARCH" ;;
esac

OS="linux"
log "OS: $OS/$ARCH"
log "Server: $SERVER_URL"
log "Agent Key: $AGENT_KEY"

# Download
log "Agent indiriliyor..."
mkdir -p "$INSTALL_DIR"
curl -sSL -o "$INSTALL_DIR/agent" "${SERVER_URL}/api/v1/update/download?os=${OS}&arch=${ARCH}"
chmod +x "$INSTALL_DIR/agent"
log "Agent indirildi: $INSTALL_DIR/agent"

# Create systemd service
log "Systemd servisi olusturuluyor..."
cat > /etc/systemd/system/${SERVICE_NAME}.service << EOF
[Unit]
Description=Go Mini RMM Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/agent -server ${SERVER_URL} -key ${AGENT_KEY}
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Start
systemctl daemon-reload
systemctl enable ${SERVICE_NAME}
systemctl restart ${SERVICE_NAME}

log "Agent basariyla kuruldu ve baslatildi!"
log "Durum:  systemctl status ${SERVICE_NAME}"
log "Loglar: journalctl -u ${SERVICE_NAME} -f"
log "Dashboard: ${SERVER_URL}"
