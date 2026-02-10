#!/bin/bash
set -e

VERSION=${1:-"dev"}
OUTPUT_DIR="./bin"

mkdir -p "$OUTPUT_DIR"

echo "Building Go Mini RMM v${VERSION}..."

# Server
echo "  -> server (linux/amd64)"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/server-linux-amd64" ./cmd/server

# Agent - multi-platform
PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")

for PLATFORM in "${PLATFORMS[@]}"; do
    OS="${PLATFORM%/*}"
    ARCH="${PLATFORM#*/}"
    EXT=""
    if [ "$OS" = "windows" ]; then
        EXT=".exe"
    fi
    OUTPUT="${OUTPUT_DIR}/agent-${OS}-${ARCH}${EXT}"
    echo "  -> agent (${OS}/${ARCH})"
    GOOS=$OS GOARCH=$ARCH CGO_ENABLED=0 go build \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        -o "$OUTPUT" ./cmd/agent
done

echo "Build complete! Binaries in ${OUTPUT_DIR}/"
ls -lh "$OUTPUT_DIR"/
