#!/bin/bash
set -e

# Go Mini RMM - Interactive Agent Installer
# Usage: curl -sSL http://SERVER:PORT/install.sh | bash

SERVER_URL="__RMM_SERVER_URL__"
INSTALL_DIR="/opt/rmm"
SERVICE_NAME="rmm-agent"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

log()  { echo -e "${GREEN}[RMM]${NC} $1"; }
warn() { echo -e "${YELLOW}[RMM]${NC} $1"; }
err()  { echo -e "${RED}[RMM]${NC} $1"; exit 1; }
ask()  { echo -en "${CYAN}[RMM]${NC} $1"; }

echo ""
echo -e "${BOLD}╔══════════════════════════════════════╗${NC}"
echo -e "${BOLD}║       Go Mini RMM - Agent Setup      ║${NC}"
echo -e "${BOLD}╚══════════════════════════════════════╝${NC}"
echo ""

# Check root
if [ "$(id -u)" -ne 0 ]; then
    err "Root olarak calistirin:\n    curl -sSL ${SERVER_URL}/install.sh | sudo bash"
fi

# If args passed, use non-interactive mode: $1 = agent name (key auto-generated)
if [ -n "$1" ]; then
    AGENT_NAME="$1"
else
    # Interactive: sadece isim sorulur, key otomatik uretilir
    if [ ! -t 0 ]; then
        warn "Pipe ile calistirildi, script indirilip tekrar calistiriliyor (terminal girisleri icin)..."
        TMPSCRIPT=$(mktemp)
        curl -sSL "${SERVER_URL}/install.sh" -o "$TMPSCRIPT"
        exec bash "$TMPSCRIPT" < /dev/tty
    fi
    DEFAULT_NAME=$(hostname)
    ask "Agent ismi [${DEFAULT_NAME}]: "
    read -r AGENT_NAME </dev/tty
    AGENT_NAME="${AGENT_NAME:-$DEFAULT_NAME}"
fi

# Key = isimden slug + rastgele (benzersiz olsun)
SLUG=$(echo "$AGENT_NAME" | tr '[:upper:]' '[:lower:]' | tr ' .' '-' | tr -cd 'a-z0-9-' | head -c 20)
RAND=$(openssl rand -hex 3 2>/dev/null || echo "$(date +%s)$$" | cksum | cut -c1-6)
AGENT_KEY="${SLUG:-agent}-${RAND}"

echo ""
log "Server:     ${SERVER_URL}"
log "Agent:      ${AGENT_NAME} (key: ${AGENT_KEY})"
echo ""

# Detect arch
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l)  ARCH="arm" ;;
    *)       err "Desteklenmeyen mimari: $ARCH" ;;
esac
OS="linux"

# Check existing installation
if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
    warn "Mevcut agent calisiyor, durduruluyor..."
    systemctl stop ${SERVICE_NAME}
fi

# Download
log "Agent indiriliyor (${OS}/${ARCH})..."
mkdir -p "$INSTALL_DIR"
HTTP_CODE=$(curl -sSL -o "$INSTALL_DIR/agent" -w "%{http_code}" "${SERVER_URL}/api/v1/update/download?os=${OS}&arch=${ARCH}")
if [ "$HTTP_CODE" != "200" ]; then
    err "Agent indirilemedi (HTTP ${HTTP_CODE}). Server calistigından emin olun."
fi
chmod +x "$INSTALL_DIR/agent"
log "Agent indirildi"

# Create systemd service
log "Systemd servisi olusturuluyor..."
cat > /etc/systemd/system/${SERVICE_NAME}.service << EOF
[Unit]
Description=Go Mini RMM Agent (${AGENT_NAME})
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/agent -server ${SERVER_URL} -key ${AGENT_KEY} -name "${AGENT_NAME}"
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Start
systemctl daemon-reload
systemctl enable ${SERVICE_NAME} >/dev/null 2>&1
systemctl start ${SERVICE_NAME}

# Verify
sleep 2
if systemctl is-active --quiet ${SERVICE_NAME}; then
    STATUS="${GREEN}aktif${NC}"
else
    STATUS="${RED}basarisiz${NC}"
fi

echo ""
echo -e "${BOLD}╔══════════════════════════════════════╗${NC}"
echo -e "${BOLD}║         Kurulum Tamamlandi!          ║${NC}"
echo -e "${BOLD}╚══════════════════════════════════════╝${NC}"
echo ""
echo -e "  Durum:     ${STATUS}"
echo -e "  Dashboard: ${CYAN}${SERVER_URL}${NC}"
echo -e "  Agent:     ${BOLD}${AGENT_NAME}${NC} (key: ${AGENT_KEY})"
echo ""
echo -e "  ${BOLD}Faydali komutlar:${NC}"
echo -e "    systemctl status ${SERVICE_NAME}     # durum"
echo -e "    journalctl -u ${SERVICE_NAME} -f     # loglar"
echo -e "    systemctl restart ${SERVICE_NAME}    # yeniden baslat"
echo ""
echo -e "  ${BOLD}Kaldirmak icin:${NC}"
echo -e "    systemctl stop ${SERVICE_NAME}"
echo -e "    systemctl disable ${SERVICE_NAME}"
echo -e "    rm /etc/systemd/system/${SERVICE_NAME}.service"
echo -e "    rm -rf ${INSTALL_DIR}"
echo ""
