#!/bin/bash
set -e

# Go Mini RMM - Docker Auto-Deploy Setup
# Sunucuda bir kez calistirilir, sonra her push otomatik deploy olur.

GHCR_USER="${1:-}"
GHCR_TOKEN="${2:-}"

if [ -z "$GHCR_USER" ] || [ -z "$GHCR_TOKEN" ]; then
    echo "Kullanim: bash setup-docker.sh <GITHUB_USERNAME> <GITHUB_TOKEN>"
    echo ""
    echo "Token olusturmak icin: https://github.com/settings/tokens"
    echo "Gerekli izin: read:packages"
    exit 1
fi

echo "[RMM] Docker kurulumu kontrol ediliyor..."
if ! command -v docker &> /dev/null; then
    echo "[RMM] Docker kuruluyor..."
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker
    systemctl start docker
    echo "[RMM] Docker kuruldu"
else
    echo "[RMM] Docker zaten kurulu"
fi

echo "[RMM] GHCR login..."
echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USER" --password-stdin

echo "[RMM] Eski servisler durduruluyor..."
systemctl stop rmm-server rmm-agent 2>/dev/null || true
systemctl disable rmm-server rmm-agent 2>/dev/null || true

echo "[RMM] Compose dosyasi indiriliyor..."
mkdir -p /opt/rmm
curl -sSL -o /opt/rmm/docker-compose.yml \
    "https://raw.githubusercontent.com/cevrimxe/go-mini-rmm/main/deploy/docker-compose.prod.yml"

echo "[RMM] Konteynerler baslatiliyor..."
cd /opt/rmm
docker compose pull
docker compose up -d

echo ""
echo "================================================"
echo "[RMM] Auto-deploy kurulumu tamamlandi!"
echo "[RMM] Dashboard: http://$(curl -s ifconfig.me):9090"
echo ""
echo "[RMM] Watchtower her 60 saniyede yeni image kontrol edecek."
echo "[RMM] git push main yaptiginizda otomatik deploy olacak."
echo ""
echo "[RMM] Loglar:"
echo "  docker compose -f /opt/rmm/docker-compose.yml logs -f"
echo "================================================"
