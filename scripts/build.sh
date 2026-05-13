#!/bin/bash
set -e

VERSION=${1:-"dev"}
OUTPUT_DIR="./bin"
BINARY_DIR="./binaries"

mkdir -p "$OUTPUT_DIR" "$BINARY_DIR"

echo "Building Go Mini RMM v${VERSION}..."

# Server (version embedded so agents know the latest version via /api/v1/update/check)
echo "  -> server (linux/amd64)"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X github.com/cevrimxe/go-mini-rmm/internal/server/update.LatestVersion=${VERSION}" \
    -o "${OUTPUT_DIR}/server-linux-amd64" ./cmd/server

# Agent - multi-platform
PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")

for PLATFORM in "${PLATFORMS[@]}"; do
    OS="${PLATFORM%/*}"
    ARCH="${PLATFORM#*/}"
    EXT=""
    if [ "$OS" = "windows" ]; then
        EXT=".exe"
    fi
    AGENT_NAME="agent-${OS}-${ARCH}${EXT}"
    OUTPUT="${OUTPUT_DIR}/${AGENT_NAME}"
    echo "  -> agent (${OS}/${ARCH})"
    GOOS=$OS GOARCH=$ARCH CGO_ENABLED=0 go build \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        -o "$OUTPUT" ./cmd/agent
    # Copy to binaries/ so the server can serve them for auto-update
    cp "$OUTPUT" "${BINARY_DIR}/${AGENT_NAME}"
done

echo ""
echo "Build complete! Binaries in ${OUTPUT_DIR}/"
ls -lh "$OUTPUT_DIR"/
