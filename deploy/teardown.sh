#!/bin/bash
# Sunucudaki RMM server + agent + watchtower'i tamamen kaldirir.
# Tekrar kurmak icin: ./setup-docker.sh

set -e

echo "[RMM] Konteynerler durdurulup kaldiriliyor..."
cd /opt/rmm 2>/dev/null || { echo "[RMM] /opt/rmm yok, atlanıyor"; exit 0; }

docker compose down -v 2>/dev/null || docker-compose down -v 2>/dev/null || true

# Eski container isimleri (onceki kurulumlardan kalma)
docker rm -f rmm-server rmm-agent-local watchtower 2>/dev/null || true

echo "[RMM] Volume siliniyor..."
docker volume rm rmm_rmm-data 2>/dev/null || docker volume rm rmm-data 2>/dev/null || true

echo "[RMM] Eski systemd servisleri kapatiliyor..."
systemctl stop rmm-server rmm-agent 2>/dev/null || true
systemctl disable rmm-server rmm-agent 2>/dev/null || true

echo ""
echo "[RMM] Temizlik tamam. Tekrar kurmak icin:"
echo "  curl -sSL https://raw.githubusercontent.com/cevrimxe/go-mini-rmm/main/deploy/setup-docker.sh | bash"
echo ""
