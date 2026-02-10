#!/bin/bash
set -e

# Go Mini RMM - Server Setup Script
# Bu script sunucuda calistirilir

PORT="${1:-9090}"
INSTALL_DIR="/opt/rmm"

echo "[RMM] Kurulum basliyor..."
echo "[RMM] Port: $PORT"

# Dizin olustur
mkdir -p "$INSTALL_DIR/binaries"

# Binary'leri tasi
if [ -f /tmp/server-linux-amd64 ]; then
    mv /tmp/server-linux-amd64 "$INSTALL_DIR/server"
    chmod +x "$INSTALL_DIR/server"
    echo "[RMM] Server binary kuruldu"
fi

if [ -f /tmp/agent-linux-amd64 ]; then
    cp /tmp/agent-linux-amd64 "$INSTALL_DIR/binaries/agent-linux-amd64"
    mv /tmp/agent-linux-amd64 "$INSTALL_DIR/agent"
    chmod +x "$INSTALL_DIR/agent" "$INSTALL_DIR/binaries/agent-linux-amd64"
    echo "[RMM] Agent binary kuruldu"
fi

# Server systemd servisi
cat > /etc/systemd/system/rmm-server.service << EOF
[Unit]
Description=Go Mini RMM Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/server -addr :${PORT} -db ${INSTALL_DIR}/rmm.db
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Agent systemd servisi (sunucunun kendisi icin)
cat > /etc/systemd/system/rmm-agent.service << EOF
[Unit]
Description=Go Mini RMM Agent
After=network-online.target rmm-server.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/agent -server http://127.0.0.1:${PORT} -key sunucu-agent
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Baslat
systemctl daemon-reload
systemctl enable rmm-server rmm-agent
systemctl restart rmm-server
sleep 2
systemctl restart rmm-agent

echo ""
echo "================================================"
echo "[RMM] Kurulum tamamlandi!"
echo "[RMM] Dashboard: http://$(curl -s ifconfig.me):${PORT}"
echo "[RMM] Server log: journalctl -u rmm-server -f"
echo "[RMM] Agent log:  journalctl -u rmm-agent -f"
echo ""
echo "[RMM] Baska makineden agent kurmak icin:"
echo "  curl -sSL http://$(curl -s ifconfig.me):${PORT}/install.sh | sudo bash -s -- AGENT_KEY"
echo "================================================"
